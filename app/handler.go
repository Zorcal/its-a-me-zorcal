package app

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

func NewHandler(log *slog.Logger, appVersion string, disableStaticCache bool) (http.Handler, error) {
	sessionMgr := newSessionManager()
	startSessionCleanupTicker(sessionMgr)

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("create sub-filesystem for static files: %w", err)
	}

	r := httprouter.New(
		traceMiddleware(),
		errorMiddleware(log, sessionMgr),
		loggingMiddleware(log),
		panicRecovery(log),
	)

	r.SetNotFoundHandler(notFoundHandler(), htmlContentTypeMiddleware())
	r.Handle("/static/", staticHandler(static, appVersion, disableStaticCache))
	r.Handle("POST /command", commandHandler(sessionMgr), htmxMiddleware(), htmlContentTypeMiddleware())
	r.Handle("GET /{$}", indexHandler(log, sessionMgr), htmlContentTypeMiddleware())

	return r, nil
}
