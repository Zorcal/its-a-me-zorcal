package slogctx_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/zorcal/me/pkg/slogctx"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	buf := bytes.Buffer{}
	h := slogctx.NewHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		// Ignore time.
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "time" {
				return slog.Attr{}
			}
			return a
		},
	}))
	l := slog.New(h)

	l = l.With("k1", "v1")
	ctx = slogctx.Attach(ctx, slog.String("k2", "v2"), slog.Int("k3", 3))
	l.InfoContext(ctx, "test")

	want := `{"level":"INFO","msg":"test","k1":"v1","k2":"v2","k3":3}
`
	if got := buf.String(); got != want {
		t.Errorf("\ngot: %q\nwant %q", got, want)
	}
}
