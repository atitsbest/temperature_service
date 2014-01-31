package handlers

import (
  "log"
  "time"
  "strconv"
  "database/sql"
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
func postMeasurementHandler(mm JsonMeasurement) string {
  // Json-Payload aus dem Request lesen.
  log.Printf("Payload => %#v", mm)

  // Derzeit müssen wir den Wert noch von String in Int konvertieren.
  val, err := strconv.ParseInt(mm.Measurement.Value, 10, 16)
  if err != nil { panicOnError(err) }

  // In die DB einfügen.
  db, err := sql.Open("postgres", connectionString)
  panicOnError(err)
  defer db.Close()

  _, err = db.Exec("insert into measurements(sensor, value, created_at) values(($1),($2),($3))",
    mm.Measurement.Sensor,
    val,
    time.Now())
  panicOnError(err)

  return "OK"
}

