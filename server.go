package main

import (
  "log"
  "time"
  "net/http"
  "regexp"
  "html/template"
  _ "github.com/lib/pq"
)

var connectionString =  "user=temperature password=TemperatuRe dbname=temperature_development"

var validPath = regexp.MustCompile("^*$")

var templates = template.Must(template.ParseFiles("views/index.html"))

type JsonMeasurement struct {
  Measurement Measurement
}

type Measurement struct {
  Sensor string
  Value int
}

type RootViewModel struct {
  Sensor string
  Value float32
  CreatedAt string
}

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


func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    start := time.Now()

    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
      http.NotFound(w, r)
      return
    }
    fn(w, r)

    // Dauer berechnen und anzeigen.
    duration := time.Since(start)
    log.Printf("Duration: %s", duration)
  }
}

func main() {
  http.HandleFunc("/", makeHandler(rootHandler))
  http.HandleFunc("/measurements/", makeHandler(measurementHandler))
  log.Fatal(http.ListenAndServe(":9001", nil))
}
