package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/zorcal/its-a-me-zorcal/internal/termui"
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
	History       []terminalSessionEntry
	CurrentPrompt string
}

func indexHandler(log *slog.Logger, sessAdapter *sessionAdapter, ghFetcher *cachedGitHubFetcher) httprouter.Handler {
	tmpl, err := template.ParseFS(templatesFS, "templates/base.html", "templates/index.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("parse template fs for index handler: %w", err)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		sessionID := getSessionID(r)
		sess := sessAdapter.mgr.GetOrCreateSession(sessionID)

		if sessionID == "" {
			setSessionCookie(w, sess.ID())
			sessionID = sess.ID()
		}

		data := IndexData{
			Repos:         ghFetcher.FetchRepositories(r.Context(), log),
			WelcomeBanner: template.HTML(welcomeBannerHTML),
			History:       sess.History(),
			CurrentPrompt: termui.GeneratePrompt(sessAdapter.GetCurrentDir(sessionID)),
		}

		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			return fmt.Errorf("exec template: %w", err)
		}

		return nil
	}
}
