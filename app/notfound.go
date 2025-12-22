package app

import (
	"html/template"
	"net/http"

	"github.com/zorcal/me/pkg/httprouter"
)

func notFoundHandler() httprouter.Handler {
	tmpl, err := template.ParseFS(templatesFS, "templates/base.html", "templates/404.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) error {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(http.StatusText(http.StatusNotFound)))
			return nil
		}
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusNotFound)
		if err := tmpl.ExecuteTemplate(w, "404.html", nil); err != nil {
			w.Write([]byte(http.StatusText(http.StatusNotFound)))
		}
		return nil
	}
}
