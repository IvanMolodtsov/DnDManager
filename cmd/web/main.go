package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"dndmanager/internal/app"
	"dndmanager/internal/platform"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	srv, err := app.New()
	if err != nil {
		slog.Error("startup", "err", err)
		os.Exit(1)
	}
	defer srv.Close()

	addr := platform.ListenAddr()
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("listening", "addr", addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
