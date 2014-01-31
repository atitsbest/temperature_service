package main

import (
  "log"// {{{
  "fmt"
  "time"
  "flag"
  "math"
  "strconv"
  "net/http"
  "regexp"
  "os"
  "os/signal"
  "html/template"
  _ "github.com/lib/pq"
  "github.com/codegangsta/martini"
  "github.com/codegangsta/martini-contrib/binding"
  "github.com/fzzy/radix/redis"
  "github.com/jmoiron/sqlx"// }}}
)

var (
  connectionString =  "user=temperature password=TemperatuRe dbname="
  validPath = regexp.MustCompile("^*$")
  templates = template.Must(template.ParseFiles("views/index.html"))
)

type (// {{{
  JsonMeasurement struct {
    Measurement Measurement
  }

  Measurement struct {
    Sensor string
    Value string
  }

  RootViewModel struct {
    Sensor string
    Value float32
    CreatedAt string
  }

  Chunk struct {
    Sensor string
    Value int
    CreatedAt time.Time `db:"created_at"`
  }

  Sensor struct {
    Sensor string
  }
)// }}}

// ------- HELPERS --------

func panicOnError(e error) {// {{{
  if e != nil { log.Fatal(e) }
}

func renderTemplate(w http.ResponseWriter, tmpl string, ms []RootViewModel) {
  err := templates.ExecuteTemplate(w, tmpl+".html", ms)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}// }}}

// Liefert alle Sensorennamen.
func sensorMeasurementCount(db *sqlx.DB, sensor string) int {
  // Letzten Eintrag abfragen.
  row := db.QueryRow("SELECT count(*) FROM measurements WHERE sensor = $1", sensor)

  var count int
  e := row.Scan(&count)
  panicOnError(e)

  return count
}


// Liefert alle Sensorennamen.
func sensorNames(db *sqlx.DB) []Sensor {
  // Letzten Eintrag abfragen.
  sensors := []Sensor{}
  db.Selectv(&sensors, "SELECT DISTINCT sensor FROM measurements")

  return sensors
}


// Liefert alle Sensorennamen.
func sensorMeasurementChunk(db *sqlx.DB, sensor string, chunkSize int, offset int) []Chunk {
  // Letzten Eintrag abfragen.
  chunks := []Chunk{}
  db.Selectv(&chunks,
  "SELECT * FROM measurements WHERE sensor = $1 LIMIT $2 OFFSET $3",
  sensor, chunkSize, offset)

  return chunks
}

func downsampleData(db *sqlx.DB, redisUrl string, quit chan bool) {
  const MAX_MEASUREMENTS = 500.0

  // Bestätigen, wenn wir fertig sind.
  defer func() { quit<-true }()

  log.Print("Starte Downsampler...")

  // Verbindung zu Redis herstellen.
  con, err := redis.DialTimeout("tcp", redisUrl, time.Duration(10)*time.Second)
  if err != nil { log.Fatal(err) }
  defer con.Close()

  // Endlosschleife
  for {
    log.Print("Starte neuen Durchlauf...")
    startTime := time.Now()

    sensors := sensorNames(db)

    for _,sensor := range sensors {
      redisKey := fmt.Sprintf("%s:all", sensor.Sensor)
      redisTmpKey := redisKey + ":" + string(time.Now().Unix())

      count := sensorMeasurementCount(db, sensor.Sensor)
      log.Printf("%s #%d", sensor.Sensor, count)
      if count == 0 { continue  } // Gibt es keine Einträge für den Sensor
      // müssen wir auch nicht weiter machen.

      // Damit auch der Rest mitgenommen wird, dividieren wir floats
      // und runden auf.
      chunkSize := int(math.Ceil(float64(count) / MAX_MEASUREMENTS))
      log.Printf("Chunksize = %d", chunkSize)

      for offset := 0; offset <= count; offset += chunkSize {
        chunks := sensorMeasurementChunk(db, sensor.Sensor, chunkSize, offset)
        if len(chunks) == 0 { continue }

        // Durchschntl. Temperature.
        avgValue := 0
        for _,c := range chunks { avgValue += c.Value }
        avgValue /= len(chunks)

        // Durchschnittliches Datum.
        var avgUnixDate int64 = 0
        for _,c := range chunks { avgUnixDate += c.CreatedAt.Unix() }
        avgUnixDate /= int64(len(chunks))

        // Den Eintrag (also das "value" aus "key/value" in Redis) erstellen.
        entry := fmt.Sprintf("{\"d\":%d, \"v\":%d}", avgUnixDate, avgValue)

        // Neuen Eintrag in Redis speichern.
        r := con.Cmd("zadd", redisTmpKey, avgUnixDate, entry)
        if r.Err != nil { log.Fatal(r.Err) }
      }

      // Alte gegen neue Werte austauschen.
      con.Cmd("multi")
      con.Cmd("del", redisKey)
      con.Cmd("rename", redisTmpKey, redisKey)
      r := con.Cmd("exec")
      if r.Err != nil { log.Fatal(err) }


    }

    log.Printf("Durchgang fertig (%s)!", time.Since(startTime))

    log.Printf("Pause!")

    timer := time.NewTimer(time.Second * 10)

    for {
      select {
        case <- quit: return // defer quit<-true
        case <- timer.C: break
      }
    }
  }
}

// ------- HELPERS (END) ---


func main() {
  port := flag.Int("port", 9001, "Port auf dem der Server hören soll.")
  dbName := flag.String("db", "temperature_development", "Zu verwendende Datenbank.")
  redisUrl := flag.String("redis", "127.0.0.1:6379", "Url zum Redis-Server.")
  flag.Parse()

  // Connectionstring zusammenbauen.
  connectionString += *dbName

  // DB-Connection erstellen.
  db := sqlx.MustConnect("postgres", connectionString)
  // ...DB am Ende der Funktion wieder schließen.
  defer db.Close()


  // Setup routes
  // m.Get("/", rootHandler)
  m := martini.Classic()
  m.Post("/api/measurements", binding.Json(JsonMeasurement{}), func(mm JsonMeasurement, err binding.Errors, res http.ResponseWriter) string {
    if err.Count() > 0 {
      res.WriteHeader(http.StatusBadRequest)
    }
    return postMeasurementHandler(mm)
  })


  // Quit-Chanel: Hier schickt der Downsampl
  quit := make(chan bool)

  // Downsampler parallel starten...
  go downsampleData(db, *redisUrl, quit)

  // Wir fangen Ctrl-C ab und geben dem Downsample Bescheid,
  // dass er sich beenden soll.
  ctrlc := make(chan os.Signal, 1)
  signal.Notify(ctrlc, os.Interrupt)
  go func(){
    for sig := range ctrlc {
      // sig is a ^C, handle it
      log.Printf("%v angefangen; warten auf Downsampler...", sig)
      quit <- true; <-quit
      log.Printf("FERTIG!")
      os.Exit(1)
    }
  }()

  // Http-Server starten.
  log.Printf("Running on Port %d and using DB %s and Redis %s...", *port, *dbName, *redisUrl)
  log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), m))

}
