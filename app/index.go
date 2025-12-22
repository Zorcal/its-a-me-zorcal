package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/zorcal/its-a-me-zorcal/pkg/github"
	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
)

const welcomeBannerHTML = `<div class="welcome">
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║                     It's a me, Zorcal!                       ║
║                                                              ║
║  Available commands: cd, ls, pwd, open, cat, clear, help     ║
║  Navigate to /home/zorcal/projects to explore my work        ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
</div>`

type IndexData struct {
	Repos         []github.Repository
	WelcomeBanner template.HTML
}

func indexHandler(log *slog.Logger) httprouter.Handler {
	tmpl, err := template.ParseFS(templatesFS, "templates/base.html", "templates/index.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("parse template fs for index handler: %w", err)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		data := IndexData{
			Repos:         fetchGitHubRepos(r.Context(), log),
			WelcomeBanner: template.HTML(welcomeBannerHTML),
		}

		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			return fmt.Errorf("exec template: %w", err)
		}

		return nil
	}
}
