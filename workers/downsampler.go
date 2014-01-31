package workers

import (
  "log"// {{{
  "fmt"
  "time"
  "math"
  "sync"
  _ "github.com/lib/pq"
  "github.com/fzzy/radix/redis"
  "github.com/jmoiron/sqlx"// }}}
)

type (
  Chunk struct {
    Sensor string
    Value int
    CreatedAt time.Time `db:"created_at"`
  }

  Sensor struct {
    Sensor string
  }
)

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

// Einen Sensor downsamplen und in Redis eintragen.
func downsampleSensor(db *sqlx.DB, redisUrl string, sensor string, wg *sync.WaitGroup) {
  const MAX_MEASUREMENTS = 500.0
  defer wg.Done()

  // Verbindung zu Redis herstellen.
  con, err := redis.DialTimeout("tcp", redisUrl, time.Duration(10)*time.Second)
  if err != nil { log.Fatal(err) }
  defer con.Close()

  redisKey := fmt.Sprintf("%s:all", sensor)
  redisTmpKey := fmt.Sprintf("%s:%d", redisKey, time.Now().Unix())

  count := sensorMeasurementCount(db, sensor)
  if count == 0 { return  } // Gibt es keine Einträge für den Sensor müssen wir auch nicht weiter machen.

  // Damit auch der Rest mitgenommen wird, dividieren wir floats
  // und runden auf.
  chunkSize := int(math.Ceil(float64(count) / MAX_MEASUREMENTS))
  log.Printf("%s #%d/%d", sensor, count, chunkSize)

  for offset := 0; offset <= count; offset += chunkSize {
    chunks := sensorMeasurementChunk(db, sensor, chunkSize, offset)
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
    if r.Err != nil {log.Print("REDIS"); log.Fatal(r.Err) }
  }

  // Alte gegen neue Werte austauschen.
  con.Cmd("multi")
  con.Cmd("del", redisKey)
  con.Cmd("rename", redisTmpKey, redisKey)
  r := con.Cmd("exec")
  if r.Err != nil { log.Fatal(r.Err) }
}

// Alle Sensoren downsamplen und in Redis eintragen.
func downsampleAll(db *sqlx.DB, redisUrl string, pause int, quit chan bool) {
  // Bestätigen, wenn wir fertig sind.
  defer func() { quit<-true }()

  log.Print("Starte Downsampler...")

  // Endlosschleife
  for {
    log.Print("Starte neuen Durchlauf...")
    startTime := time.Now()

    sensors := sensorNames(db)
    wg := new(sync.WaitGroup)

    for _,sensor := range sensors {
      wg.Add(1)
      go downsampleSensor(db, redisUrl, sensor.Sensor, wg)
    }

    // Warten bis alle Sensoren
    wg.Wait()

    log.Printf("Durchgang fertig (%s)!", time.Since(startTime))

    log.Printf("Pause!")
    // timer := time.NewTimer(time.Second * 10)
    select {
      case <- quit: return // defer quit<-true
      case <- time.After(time.Second * time.Duration(pause)): break // Pause zu ende.
    }
  }
}


