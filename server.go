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
  "sync"
  "html/template"
  _ "github.com/lib/pq"
  "github.com/codegangsta/martini"
  "github.com/codegangsta/martini-contrib/binding"
  "github.com/fzzy/radix/redis"
  "github.com/jmoiron/sqlx"
  "./workers"// }}}
)

var (
  connectionString =  "user=temperature password=TemperatuRe dbname="
  validPath = regexp.MustCompile("^*$")
  templates = template.Must(template.ParseFiles("views/index.html"))
)

type (// {{{

  RootViewModel struct {
    Sensor string
    Value float32
    CreatedAt string
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

// ------- HELPERS (END) ---


func main() {
  // Parameter.
  port :=       flag.Int(   "port", 9001,                       "Port auf dem der Server hören soll.")
  dbName :=     flag.String("db",   "temperature_development",  "Zu verwendende Datenbank.")
  redisUrl :=   flag.String("redis","127.0.0.1:6379",           "Url zum Redis-Server.")
  pause :=      flag.Int(   "pause", 10,                        "Pause in Sekunden zwischen den Downsample-Vorängen.")
  flag.Parse()

  // Connectionstring zusammenbauen.
  connectionString += *dbName

  // DB-Connection erstellen.
  db := sqlx.MustConnect("postgres", connectionString)
  // ...DB am Ende der Funktion wieder schließen.
  defer db.Close()


  // Setup routes
  m := martini.Classic()
  // m.Get("/", rootHandler)
  m.Post("/api/measurements", binding.Json(JsonMeasurement{}), func(mm JsonMeasurement, err binding.Errors, res http.ResponseWriter) string {
    if err.Count() > 0 {
      res.WriteHeader(http.StatusBadRequest)
    }
    return postMeasurementHandler(mm)
  })


  // Quit-Channel: Hier schickt der Downsampler.
  quit := make(chan bool)

  // Downsampler parallel starten...
  go downsampleAll(db, *redisUrl, *pause, quit)

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
