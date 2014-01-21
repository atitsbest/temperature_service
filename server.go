package main

import (
  "log"
  "flag"
  "time"
  "strconv"
  "net/http"
  "regexp"
  "html/template"
  _ "github.com/lib/pq"
)

var connectionString =  "user=temperature password=TemperatuRe dbname="

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
  port := flag.Int("port", 9001, "Port auf dem der Server hören soll.")
  dbName := flag.String("db", "temperature_development", "Zu verwendende Datenbank.")
  flag.Parse()

  // Connectionstring zusammenbauen.
  connectionString += *dbName

  http.HandleFunc("/", makeHandler(rootHandler))
  http.HandleFunc("/api/measurements/", makeHandler(measurementHandler))

  log.Printf("Running on Port %d and using DB %s...", *port, *dbName)
  log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
