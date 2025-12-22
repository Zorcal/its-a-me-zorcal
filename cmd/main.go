package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/zorcal/its-a-me-zorcal/app"
	"github.com/zorcal/its-a-me-zorcal/pkg/slogctx"
)

// appVersion should be set at build time using -ldflags
var appVersion = "dev"

type Config struct {
	conf.Version
	Web struct {
		ReadTimeout        time.Duration `conf:"default:5s"`
		WriteTimeout       time.Duration `conf:"default:10s"`
		IdleTimeout        time.Duration `conf:"default:120s"`
		ShutdownTimeout    time.Duration `conf:"default:20s"`
		Addr               string        `conf:"default:127.0.0.1:5042"`
		DisableStaticCache bool          `conf:"default:true"`
	}
}

func main() {
	ctx := context.Background()

	cfg := Config{
		Version: conf.Version{
			Build: appVersion,
			Desc:  "My personal website.",
		},
	}
	if help, err := conf.Parse("ME", &cfg); err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Fprint(os.Stdout, help)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	log := slog.New(slogctx.NewHandler(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(ctx, cfg, log); err != nil {
		log.ErrorContext(ctx, "Run error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg Config, log *slog.Logger) (retErr error) {
	strCfg, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generate config for output: %w", err)
	}

	log.InfoContext(ctx, "Starting...", "config", strCfg)

	appHandler, err := app.NewHandler(log, appVersion, cfg.Web.DisableStaticCache)
	if err != nil {
		return fmt.Errorf("create app handler: %w", err)
	}

	srv := http.Server{
		Addr:         cfg.Web.Addr,
		Handler:      appHandler,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelInfo),
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	srvErrs := make(chan error, 1)

	go func() {
		log.InfoContext(ctx, "HTTP server started", "host", srv.Addr)

		if err := srv.ListenAndServe(); err != nil {
			srvErrs <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	defer log.InfoContext(ctx, "HTTP server stopped")

	select {
	case err := <-srvErrs:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.InfoContext(ctx, "Graceful shutdown started", "signal", sig)
		defer log.InfoContext(ctx, "Shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
			return fmt.Errorf("could not stop HTTP server gracefully: %w", err)
		}
	}

	return nil
}
