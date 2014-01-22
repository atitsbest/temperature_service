package main

import (
  "log"
  "flag"
  "strconv"
  "net/http"
  "regexp"
  "html/template"
  _ "github.com/lib/pq"
  "github.com/codegangsta/martini"
  "github.com/codegangsta/martini-contrib/binding"
)

var (
  connectionString =  "user=temperature password=TemperatuRe dbname="
  validPath = regexp.MustCompile("^*$")
  templates = template.Must(template.ParseFiles("views/index.html"))
)

type (
  JsonMeasurement struct {
    Measurement Measurement `json: measurement`
  }

  Measurement struct {
    Sensor string `json:sensor`
    Value int `json:value`
  }

  RootViewModel struct {
    Sensor string
    Value float32
    CreatedAt string
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

// ------- HELPERS (END) ---


func main() {
  port := flag.Int("port", 9001, "Port auf dem der Server hören soll.")
  dbName := flag.String("db", "temperature_development", "Zu verwendende Datenbank.")
  flag.Parse()

  // Connectionstring zusammenbauen.
  connectionString += *dbName

  m := martini.Classic()

  // Setup routes
  m.Get("/", rootHandler)
  m.Post("/api/measurements", binding.Json(JsonMeasurement{}), func(mm JsonMeasurement, err binding.Errors, res http.ResponseWriter) string {
    if err.Count() > 0 {
      res.WriteHeader(http.StatusBadRequest)
    }
    return postMeasurementHandler(mm)
  })

  log.Printf("Running on Port %d and using DB %s...", *port, *dbName)
  log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), m))
}
