// Package tracectx provides functions for setting and retrieving trace IDs in
// context.Context.
package tracectx

import (
	"context"
)

type ctxKey struct{}

var contextKey = ctxKey{}

// Set sets given trace ID into the context.
func Set(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, contextKey, traceID)
}

// Get retrieves the trace ID from the context.
func Get(ctx context.Context) string {
	if traceID, ok := ctx.Value(contextKey).(string); ok {
		return traceID
	}
	return ""
}
