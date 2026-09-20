package handler

import (
	"log/slog"
	"net/http"
	"os"
	"sync"

	"dndmanager/app"
)

// Handler is the Vercel Go function entry (Fluid Compute, not Edge).
// Imports dndmanager/app (public): this file is compiled outside the module, so
// dndmanager/internal/* is illegal. Local Air still uses cmd/web.
// SSE max duration is 300s (vercel.json).
var (
	once    sync.Once
	handler http.Handler
	initErr error
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
		srv, err := app.New()
		if err != nil {
			initErr = err
			slog.Error("vercel startup", "err", err)
			return
		}
		handler = srv.Handler
	})
	if initErr != nil {
		http.Error(w, "startup failed", http.StatusInternalServerError)
		return
	}
	handler.ServeHTTP(w, r)
}
