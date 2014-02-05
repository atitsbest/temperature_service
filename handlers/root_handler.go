package handlers

import (
  "log"
  "net/http"
  "html/template"
)

var (
  templates = template.Must(template.ParseFiles("views/index.html"))
)

func RootHandler(res http.ResponseWriter) {
  err := templates.ExecuteTemplate(res, "index.html", nil)
  if err != nil {
    res.WriteHeader(http.StatusInternalServerError)
    log.Panic(err)
  }
}
