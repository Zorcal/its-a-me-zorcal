package app

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/zorcal/its-a-me-zorcal/core/termfs"
	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

func NewHandler(log *slog.Logger, appVersion string, disableStaticCache bool) (http.Handler, error) {
	sessMgr := newSessionManager()
	startSessionCleanupTicker(sessMgr)

	// Fetch GitHub repositories and create filesystem
	ctx := context.Background()
	repos := fetchGitHubRepos(ctx, log)
	tfs := termfs.New(repos)

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("create sub-filesystem for static files: %w", err)
	}

	r := httprouter.New(
		traceMiddleware(),
		errorMiddleware(log, sessMgr),
		loggingMiddleware(log),
		panicRecovery(log),
	)

	r.SetNotFoundHandler(notFoundHandler(), htmlContentTypeMiddleware())
	r.Handle("/static/", staticHandler(static, appVersion, disableStaticCache))
	r.Handle("POST /command", commandHandler(sessMgr, tfs), htmxMiddleware(), htmlContentTypeMiddleware())
	r.Handle("GET /{$}", indexHandler(log, sessMgr), htmlContentTypeMiddleware())

	return r, nil
}
