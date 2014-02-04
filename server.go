package main

import (
  "log"
  "flag"
  "strconv"
  "net/http"
  "regexp"
  "os"
  "os/signal"
  _ "github.com/lib/pq"
  "github.com/codegangsta/martini"
  "github.com/codegangsta/martini-contrib/binding"
  "github.com/jmoiron/sqlx"
  "github.com/atitsbest/temperature_service/workers"
  "github.com/atitsbest/temperature_service/handlers"
)

var (
  connectionString =  "user=temperature password=TemperatuRe dbname="
  validPath = regexp.MustCompile("^*$")
)

type (

  RootViewModel struct {
    Sensor string
    Value float32
    CreatedAt string
  }
)

func main() {
  // Parameter.
  port :=       flag.Int(   "port", 9001,                       "Port auf dem der Server hören soll.")
  dbName :=     flag.String("db",   "temperature_development",  "Zu verwendende Datenbank.")
  redisUrl :=   flag.String("redis","127.0.0.1:6379",           "Url zum Redis-Server.")
  pause :=      flag.Int(   "pause", 1000,                      "Pause in Sekunden zwischen den Downsample-Vorängen.")
  flag.Parse()

  // Connectionstring zusammenbauen.
  connectionString += *dbName

  // DB-Connection erstellen.
  db := sqlx.MustConnect("postgres", connectionString)
  // ...DB am Ende der Funktion wieder schließen.
  defer db.Close()


  // Setup routes
  m := martini.Classic()
  m.Get("/", handlers.RootHandler)
  m.Get("/api/measurements.json", handlers.GetMeasurementsHandler(redisUrl))
  m.Post("/api/measurements", binding.Json(handlers.JsonMeasurement{}), func(mm handlers.JsonMeasurement, err binding.Errors, res http.ResponseWriter) string {
    if err.Count() > 0 {
      res.WriteHeader(http.StatusBadRequest)
    }
    return handlers.PostMeasurementHandler(connectionString, mm)
  })


  // Quit-Channel: Hier schickt der Downsampler.
  quit := make(chan bool)

  // Downsampler parallel starten...
  go workers.DownsampleAll(db, *redisUrl, *pause, quit)

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
