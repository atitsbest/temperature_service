package handlers

import (
  "net/http"
  "html/template"
)

var (
  templates = template.Must(template.ParseFiles("views/index.html"))
)

func RootHandler(res http.ResponseWriter) {
  templates.ExecuteTemplate(res, "index.html", nil)
}
