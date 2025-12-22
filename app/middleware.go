package app

import (
	"cmp"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/zorcal/me/pkg/httprouter"
	"github.com/zorcal/me/pkg/slogctx"
	"github.com/zorcal/me/pkg/tracectx"
)

func traceMiddleware() httprouter.Middleware {
	return func(next httprouter.Handler) httprouter.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx := r.Context()

			traceparent := cmp.Or(r.Header.Get("traceparent"), uuid.NewString())

			ctx = tracectx.Set(ctx, traceparent)
			ctx = slogctx.Attach(ctx, "traceparent", traceparent)

			return next(w, r.WithContext(ctx))
		}
	}
}

func loggingMiddleware(log *slog.Logger) httprouter.Middleware {
	return func(next httprouter.Handler) httprouter.Handler {
		return func(w http.ResponseWriter, r *http.Request) (retErr error) {
			now := time.Now()

			rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			defer func() {
				attrs := []slog.Attr{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("remote_addr", r.RemoteAddr),
					slog.String("x_forwarded_for", r.Header.Get("X-Forwarded-For")),
					slog.Int("status_code", rr.statusCode),
					slog.String("user_agent", r.Header.Get("User-Agent")),
					slog.Time("started_at", now),
					slog.Int64("took_ms", time.Since(now).Milliseconds()),
				}
				if retErr != nil {
					attrs = append(attrs, slog.String("error", retErr.Error()))
				}
				log.LogAttrs(r.Context(), logLevel(rr.statusCode), "HTTP request completed", attrs...)
			}()

			return next(rr, r)
		}
	}
}

// responseRecorder is a wrapper around http.ResponseWriter to capture the
// status code.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseRecorder) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

func logLevel(statusCode int) slog.Level {
	switch {
	case statusCode >= 400:
		return slog.LevelWarn
	case statusCode >= 500:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func panicRecovery(log *slog.Logger) httprouter.Middleware {
	return func(next httprouter.Handler) httprouter.Handler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()

					log.ErrorContext(r.Context(), "Panic recovered",
						"panic", rec,
						"stack", string(stack),
						"method", r.Method,
						"path", r.URL.Path,
					)

					err = fmt.Errorf("PANIC: %v", rec)
				}
			}()

			return next(w, r)
		}
	}
}

func errorMiddleware(log *slog.Logger) httprouter.Middleware {
	tmpl, err := template.ParseFS(templatesFS, "templates/base.html", "templates/error.html")
	if err != nil {
		log.ErrorContext(context.Background(), "Failed to parse template fs for error middleware", "error", err)
		return func(next httprouter.Handler) httprouter.Handler {
			return func(w http.ResponseWriter, r *http.Request) error {
				if err := next(w, r); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
				}
				return nil
			}
		}
	}

	return func(next httprouter.Handler) httprouter.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			err := next(w, r)
			if err == nil {
				return nil
			}

			data := struct {
				CorrelationID string
			}{
				CorrelationID: tracectx.Get(r.Context()),
			}

			w.WriteHeader(http.StatusInternalServerError)

			if templateErr := tmpl.ExecuteTemplate(w, "error.html", data); templateErr != nil {
				log.ErrorContext(r.Context(), "Failed to render error template", "error", templateErr)
				w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			}

			return nil
		}
	}
}

func htmlContentTypeMiddleware() httprouter.Middleware {
	return func(next httprouter.Handler) httprouter.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			return next(w, r)
		}
	}
}
