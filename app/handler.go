package app

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/zorcal/me/pkg/httprouter"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

func NewHandler(log *slog.Logger, appVersion string) (http.Handler, error) {
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("create sub-filesystem for static files: %w", err)
	}

	r := httprouter.New(
		traceMiddleware(),
		errorMiddleware(log),
		loggingMiddleware(log),
		panicRecovery(log),
	)

	r.SetNotFoundHandler(notFoundHandler(), htmlContentTypeMiddleware())
	r.Handle("/static/", staticHandler(static, appVersion))
	r.Handle("/{$}", indexHandler(), htmlContentTypeMiddleware())

	return r, nil
}
