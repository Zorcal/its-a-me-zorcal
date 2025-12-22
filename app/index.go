package app

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/zorcal/me/pkg/httprouter"
)

func indexHandler() httprouter.Handler {
	tmpl, err := template.ParseFS(templatesFS, "templates/base.html", "templates/index.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("parse template fs for index handler: %w", err)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
			return fmt.Errorf("exec template: %w", err)
		}

		return nil
	}
}
