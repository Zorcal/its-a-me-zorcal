// Package httprouter is a thin wrapper around http.ServeMux with support for
// middleware. Defines a custom handler type that returns an error.
package httprouter

import (
	"net/http"
)

// Handler is a function that can be registered to a route to handle HTTP
// requests.
type Handler func(w http.ResponseWriter, r *http.Request) error

// Middleware is a handler function designed to run code before and/or after
// another Handler.
type Middleware func(Handler) Handler

// Router is a thin wrapper around http.ServeMux that allows for registering
// handlers with middleware for different HTTP methods and patterns.
type Router struct {
	mw              []Middleware
	m               *http.ServeMux
	notFoundHandler Handler
}

// New returns a new HTTP Router. Middleware are executed in the order they are
// provided and before any handler-specific middleware.
func New(mw ...Middleware) *Router {
	return &Router{
		mw: mw,
		m:  http.NewServeMux(),
	}
}

// Handle registers a new handler with given path pattern. Middleware are
// executed in the order they are provided and after any global middleware.
// Responds to the client with a 500 status code if the handler returns a
// non-nil error.
func (r *Router) Handle(pattern string, h Handler, mw ...Middleware) {
	h = wrapMiddleware(mw, h)
	h = wrapMiddleware(r.mw, h)
	r.m.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		if err := h(w, req); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

// SetNotFoundHandler sets a custom handler for 404 responses.
func (r *Router) SetNotFoundHandler(h Handler, mw ...Middleware) {
	h = wrapMiddleware(mw, h)
	r.notFoundHandler = wrapMiddleware(r.mw, h)
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.notFoundHandler != nil {
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK, headerWritten: false}
		r.m.ServeHTTP(rr, req)
		if rr.statusCode == http.StatusNotFound && !rr.headerWritten {
			if err := r.notFoundHandler(w, req); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
		return
	}
	r.m.ServeHTTP(w, req)
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (rec *responseRecorder) WriteHeader(statusCode int) {
	rec.statusCode = statusCode
	if statusCode != http.StatusNotFound {
		rec.headerWritten = true
		rec.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if rec.statusCode == http.StatusNotFound {
		// Don't write 404 content, we'll use our custom handler
		return len(b), nil
	}
	rec.headerWritten = true
	return rec.ResponseWriter.Write(b)
}

// HandlerFromStd converts a handler from the standard library to a Handler.
func HandlerFromStd(h http.Handler) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		h.ServeHTTP(w, r)
		return nil
	}
}

func wrapMiddleware(mw []Middleware, h Handler) Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		wrap := mw[i]
		if wrap != nil {
			h = wrap(h)
		}
	}
	return h
}
