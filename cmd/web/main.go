package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"dndmanager/internal/battles"
	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/characters"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
	"dndmanager/internal/users"
)

// Catalog lives in internal/catalog. Future mount points:
//   - internal/items     — inventory
//   - internal/abilities — spells and remaining class options
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	dataDir := getenv("DATA_DIR", "data")
	db, err := platform.OpenDB(dataDir)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := platform.Migrate(db, "migrations"); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	bundle, err := platform.LoadBundle("locales")
	if err != nil {
		slog.Error("i18n", "err", err)
		os.Exit(1)
	}
	renderer, err := platform.NewRenderer("web/templates", bundle)
	if err != nil {
		slog.Error("templates", "err", err)
		os.Exit(1)
	}

	sessions := &platform.SessionStore{DB: db}
	userSvc := &users.Service{Repo: &users.Repository{DB: db}}
	campSvc := &campaigns.Service{Repo: &campaigns.Repository{DB: db}}
	catalogSvc := &catalog.Service{Repo: &catalog.Repository{DB: db}}
	charSvc := &characters.Service{
		Repo:      &characters.Repository{DB: db},
		Campaigns: campSvc,
		Catalog:   catalogSvc,
		Rules:     &rules.Engine{Catalog: catalogSvc},
		Events:    characters.NewVitalsHub(),
	}
	battleSvc := &battles.Service{
		Repo:       &battles.Repository{DB: db},
		Campaigns:  campSvc,
		Catalog:    catalogSvc,
		Characters: charSvc,
		Events:     battles.NewHub(),
	}
	charSvc.OnCampaign = battleSvc.Events.Broadcast

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	auth := platform.RequireAuth
	(&users.Controller{Svc: userSvc, Sessions: sessions, Render: renderer}).Mount(mux)
	(&campaigns.Controller{Svc: campSvc, Characters: charSvc, Battles: battleSvc, Render: renderer}).Mount(mux, auth)
	(&characters.Controller{Svc: charSvc, Campaigns: campSvc, Catalog: catalogSvc, Render: renderer}).Mount(mux, auth)
	(&battles.Controller{Svc: battleSvc, Campaigns: campSvc, Characters: charSvc, Catalog: catalogSvc, Render: renderer}).Mount(mux, auth)

	home := &homeController{Users: userSvc, Campaigns: campSvc, Characters: charSvc, Render: renderer}
	mux.HandleFunc("GET /{$}", home.index)
	mux.Handle("GET /admin", auth(http.HandlerFunc(home.admin)))

	handler := platform.Chain(mux,
		platform.Recover,
		platform.Sessions(sessions, userSvc.AuthUser),
		platform.CSRF,
	)

	addr := getenv("ADDR", ":8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
