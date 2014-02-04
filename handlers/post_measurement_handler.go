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
func GetMeasurementsHandler(redisUrl string) string {
}
