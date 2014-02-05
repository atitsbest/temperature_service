package handlers

import (
  "fmt"
  "log"
  "time"
  "strconv"
  "strings"
  "net/http"
  "database/sql"
  // "encoding/json"
  "github.com/fzzy/radix/redis"
)

type (
  JsonMeasurement struct {
    Measurement Measurement
  }

  Measurement struct {
    Sensor string
    Value string
  }
)

// Neue Messung in die DB schreiben.
func PostMeasurementHandler(connectionString string, mm JsonMeasurement) string {
  // Json-Payload aus dem Request lesen.
  log.Printf("Payload => %#v", mm)

  // Derzeit müssen wir den Wert noch von String in Int konvertieren.
  val, err := strconv.ParseInt(mm.Measurement.Value, 10, 16)
  if err != nil { log.Panic(err) }

  // In die DB einfügen.
  db, err := sql.Open("postgres", connectionString)
  if err != nil { log.Panic(err) }
  defer db.Close()

  _, err = db.Exec("insert into measurements(sensor, value, created_at) values(($1),($2),($3))",
  mm.Measurement.Sensor,
  val,
  time.Now())
  if err != nil { log.Panic(err) }

  return "OK"
}

// Alles Messungen als Json aus Redis liefern.
func GetMeasurementsHandler(redisUrl *string) func(w http.ResponseWriter) {
  return func(w http.ResponseWriter) {
    // Verbindung zu Redis herstellen.
    con, err := redis.DialTimeout("tcp", *redisUrl, time.Duration(10)*time.Second)
    if err != nil { log.Fatal(err) }
    defer con.Close()

    // Sensoren auslesen.
    ss, err := con.Cmd("keys", "*:all").List()
    if err != nil { log.Fatal(err) }

    // Hier werden die einzelnen Sensorendaten gepseichert.
    rows := make([]string, len(ss))

    for i,sensor := range ss {
      // Daten für den Sensor holen.
      ms, err := con.Cmd("zrange", sensor, 0, -1).List()
      if err != nil { log.Fatal(err) }

      rows[i] = fmt.Sprintf(`"%s":[%s]`, sensor, strings.Join(ms, ","))
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    fmt.Fprintf(w, "{%s}", strings.Join(rows, ","))
  }
}
