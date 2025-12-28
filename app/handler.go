package app

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/zorcal/its-a-me-zorcal/internal/termfs"
	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

func NewHandler(log *slog.Logger, appVersion string, disableStaticCache bool) (http.Handler, error) {
	sessMgr := newSessionManager()
	startSessionCleanupTicker(sessMgr)

	sessAdapter := newSessionAdapter(sessMgr)

	ghFetcher := newCachedGitHubFetcher("Zorcal", 24*time.Hour)

	repos := ghFetcher.FetchRepositories(context.Background(), log)
	tfs := termfs.New(repos)

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("create sub-filesystem for static files: %w", err)
	}

	r := httprouter.New(
		traceMiddleware(),
		errorMiddleware(log, sessAdapter),
		loggingMiddleware(log),
		panicRecovery(log),
	)

	r.SetNotFoundHandler(notFoundHandler(), htmlContentTypeMiddleware())
	r.Handle("/static/", staticHandler(static, appVersion, disableStaticCache))
	r.Handle("POST /command", commandHandler(sessAdapter, tfs), htmxMiddleware(), htmlContentTypeMiddleware())
	r.Handle("POST /newline", newlineHandler(sessAdapter), htmlContentTypeMiddleware())
	r.Handle("GET /history", historyHandler(sessMgr))
	r.Handle("GET /{$}", indexHandler(log, sessAdapter, ghFetcher), htmlContentTypeMiddleware())

	return r, nil
}
