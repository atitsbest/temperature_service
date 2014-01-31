package main

// import (
//   "log"
//   "strconv"
//   "time"
//   "net/http"
//   "database/sql"
// )
// 
// func rootHandler(res http.ResponseWriter, req *http.Request) {
//   var (
//     sensor string
//     value int
//     created_at time.Time
//   )
//   // DB öffnen...
//   db, err := sql.Open("postgres", connectionString)
//   panicOnError(err)
//   // ...DB am Ende der Funktion wieder schließen.
//   defer db.Close()
// 
//   // Letzten Eintrag abfragen.
//   rows, err := db.Query(`
//     select m1.*
//     from measurements m1 
//     left outer join measurements m2
//     on (m1.sensor = m2.sensor and m1.created_at < m2.created_at)
//     where m2.sensor is null`)
//   panicOnError(err)
// 
//   var ms []RootViewModel
// 
//   // Eintrag lesen.
//   for rows.Next() {
//     err = rows.Scan(&sensor, &value, &created_at)
//     panicOnError(err)
//     ms = append(ms, RootViewModel{
//       Sensor:sensor,
//       Value: float32(value) / 100.0,
//       CreatedAt:created_at.Format("am Mo 2. Jan 2006 um 15:04:05")  })
//   }
// 
//   renderTemplate(res, "index", ms)
// }
