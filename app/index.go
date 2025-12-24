package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/zorcal/its-a-me-zorcal/pkg/github"
	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
	"github.com/zorcal/its-a-me-zorcal/pkg/session"
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
}

func indexHandler(log *slog.Logger, sessionMgr *session.Manager[terminalSessionEntry]) httprouter.Handler {
	tmpl, err := template.ParseFS(templatesFS, "templates/base.html", "templates/index.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("parse template fs for index handler: %w", err)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		sessionID := getSessionID(r)
		sess := sessionMgr.GetOrCreateSession(sessionID)

		if sessionID == "" {
			setSessionCookie(w, sess.ID())
		}

		data := IndexData{
			Repos:         fetchGitHubRepos(r.Context(), log),
			WelcomeBanner: template.HTML(welcomeBannerHTML),
			History:       sess.History(),
		}

		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			return fmt.Errorf("exec template: %w", err)
		}

		return nil
	}
}

func getSessionID(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 24 hours
	}
	http.SetCookie(w, cookie)
}
