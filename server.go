package main

import (
  "log"
  "time"
  "flag"
  "math"
  "strconv"
  "net/http"
  "regexp"
  "html/template"
  _ "github.com/lib/pq"
  "github.com/codegangsta/martini"
  "github.com/codegangsta/martini-contrib/binding"
  "github.com/fzzy/radix/redis"
  "github.com/jmoiron/sqlx"
)

var (
  connectionString =  "user=temperature password=TemperatuRe dbname="
  validPath = regexp.MustCompile("^*$")
  templates = template.Must(template.ParseFiles("views/index.html"))
)

type (
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
    Value int
    CreatedAt time.Time
  }
)

// ------- HELPERS --------

func panicOnError(e error) {
  if e != nil { log.Fatal(e) }
}

func renderTemplate(w http.ResponseWriter, tmpl string, ms []RootViewModel) {
  err := templates.ExecuteTemplate(w, tmpl+".html", ms)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

// Liefert alle Sensorennamen.
func sensorMeasurementCount(sensor string) int64 {
  // DB öffnen...
  db, err := sqlx.Open("postgres", connectionString)
  panicOnError(err)
  // ...DB am Ende der Funktion wieder schließen.
  defer db.Close()

  // Letzten Eintrag abfragen.
  var count int64
  db.Select(&count, "SELECT count(*) FROM measurements WHERE sensor = $1", sensor)

  return count
}


// Liefert alle Sensorennamen.
func sensorNames() []string {
  // DB öffnen...
  db, err := sqlx.Open("postgres", connectionString)
  panicOnError(err)
  // ...DB am Ende der Funktion wieder schließen.
  defer db.Close()

  // Letzten Eintrag abfragen.
  sensors := []string{}
  db.Select(&sensors, "SELECT sensor FROM measurements")

  return sensors
}


// Liefert alle Sensorennamen.
func sensorMeasurementChunk(sensor string, chunkSize float64, offset float64) []Chunk {
  // DB öffnen...
  db, err := sqlx.Open("postgres", connectionString)
  panicOnError(err)
  // ...DB am Ende der Funktion wieder schließen.
  defer db.Close()

  // Letzten Eintrag abfragen.
  chunks := []Chunk{}
  db.Select(&chunks,
    "SELECT * FROM measurements WHERE sensor = $1 LIMIT $2 OFFSET $3",
    sensor, chunkSize, offset)

  return chunks
}

func downsampleData(quit chan bool) {
  const MAX_MEASUREMENTS = 500.0

  log.Print("Starte Downsampler...")

  // Verbindung zu Redis herstellen.
  con, err := redis.DialTimeout("tcp", "127.0.0.1:6379", time.Duration(10)*time.Second)
  if err != nil { log.Fatal(err) }
  defer con.Close()

  // Endlosschleife
  for {
    log.Print("Starte neuen Durchlauf...")

    sensors := sensorNames()
    log.Printf("%v", sensors)

    for _,sensor := range sensors {
      redisKey := "KEY_" + sensor // TODO
      redisTmpKey := redisKey + ":" + string(time.Now().Unix())

      count := float64(sensorMeasurementCount(sensor))
      log.Printf("%s #%d", sensor, count)

      chunkSize := math.Ceil(count / MAX_MEASUREMENTS)

      for offset := 0.0; offset <= count; offset += chunkSize {
        chunks := sensorMeasurementChunk(sensor, chunkSize, offset)

        // Durchschntl. Temperature.
        avgValue := 0
        for _,c := range chunks { avgValue += c.Value }
        avgValue /= len(chunks)

        // Durchschnittliches Datum.
        var avgUnixDate int64 = 0
        for _,c := range chunks { avgUnixDate += c.CreatedAt.Unix() }
        avgUnixDate /= int64(len(chunks))
        avgDate := time.Unix(avgUnixDate, 0)

        // Den Eintrag (also das "value" aus "key/value" in Redis) erstellen.
        entry := "{'d':" + string(avgUnixDate) + "'v':" + string(avgValue) + "}"

        // Neuen Eintrag in Redis speichern.
        r := con.Cmd("zadd", redisTmpKey, avgDate, entry)
        if r.Err != nil { log.Fatal(err) }

      }

      // Alte gegen neue Werte austauschen.
      con.Cmd("multi")
      con.Cmd("del", redisKey)
      con.Cmd("rename", redisTmpKey, redisKey)
      r := con.Cmd("exec")
      if r.Err != nil { log.Fatal(err) }

      log.Printf("Durchgang fertig!")

    }

    // Wir warten ganze x Sekunden vor dem nächsten Durchlauf.
    log.Printf("Pause!")
    time.Sleep(time.Second * 10)
    // No-blocking: sollen wir beenden?
    log.Printf("Beenden?")
    select {
      case <-quit: break
      default:
    }
  }

  // Wir sind fertig.
  log.Printf("Beenden!")
  quit <- true
}

// ------- HELPERS (END) ---


func main() {
  port := flag.Int("port", 9001, "Port auf dem der Server hören soll.")
  dbName := flag.String("db", "temperature_development", "Zu verwendende Datenbank.")
  flag.Parse()

  // Connectionstring zusammenbauen.
  connectionString += *dbName

  m := martini.Classic()

  // Setup routes
  // m.Get("/", rootHandler)
  m.Post("/api/measurements", binding.Json(JsonMeasurement{}), func(mm JsonMeasurement, err binding.Errors, res http.ResponseWriter) string {
    if err.Count() > 0 {
      res.WriteHeader(http.StatusBadRequest)
    }
    return postMeasurementHandler(mm)
  })


  // Quit-Chanel: Hier schickt der Downsample
  quit := make(chan bool)

  go downsampleData(quit)


  log.Printf("Running on Port %d and using DB %s...", *port, *dbName)
  log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), m))

  // Downsampler soll sich beenden.
  log.Printf("Downsampler soll beenden...")
  quit <- true
  // Warten bis der Downsampler fertig ist.
  log.Printf("Warten auf Downsampler...")
  <-quit
  log.Printf("FERTIG!")
}
