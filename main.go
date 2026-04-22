package main

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/dagnull/cRook/internal/config"
	"github.com/dagnull/cRook/internal/room"
	"github.com/dagnull/cRook/internal/session"
	"github.com/dagnull/cRook/internal/web"
)

//go:embed ui
var uiFS embed.FS

func main() {
	cfg := config.Load()

	sessions := session.NewManager(cfg.SecretKey)
	mgr := room.NewManager()

	router := web.NewRouter(mgr, sessions, uiFS)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("cRook listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	srv.Shutdown(context.Background())
}
