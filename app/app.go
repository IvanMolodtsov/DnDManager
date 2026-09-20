// Package app wires the HTTP handler used by cmd/web (Air) and the Vercel api/ entry.
// It is public (not internal/) because Vercel compiles api/index.go outside the
// module tree, where Go forbids importing dndmanager/internal/*.
// Catalog lives in internal/catalog. Future mount points:
//   - internal/items     — inventory
//   - internal/abilities — spells and remaining class options
package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	dndroot "dndmanager"
	"dndmanager/internal/battles"
	"dndmanager/internal/campaigns"
	"dndmanager/internal/catalog"
	"dndmanager/internal/characters"
	"dndmanager/internal/platform"
	"dndmanager/internal/rules"
	"dndmanager/internal/users"
)

// Server is the process (or Fluid Compute instance) HTTP app.
type Server struct {
	Handler http.Handler
	DB      *sql.DB
}

// Close releases the database. Local cmd/web defers this; Vercel keeps the instance.
func (s *Server) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// New opens the DB, runs migrations, and returns the mux wrapped with sessions/CSRF.
func New() (*Server, error) {
	if os.Getenv("VERCEL") == "1" {
		dir, err := platform.UnpackAssets(dndroot.AssetFS)
		if err != nil {
			return nil, fmt.Errorf("assets: %w", err)
		}
		if err := os.Setenv("ASSET_ROOT", dir); err != nil {
			return nil, fmt.Errorf("assets: %w", err)
		}
	}

	db, err := platform.OpenFromEnv()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	migrationsDir := platform.AssetDir("migrations")
	localesDir := platform.AssetDir("locales")
	templatesDir := platform.AssetDir(filepath.Join("web", "templates"))
	staticDir := platform.AssetDir(filepath.Join("web", "static"))
	slog.Info("assets", "migrations", migrationsDir, "locales", localesDir, "templates", templatesDir, "static", staticDir)

	if err := platform.Migrate(db, migrationsDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	bundle, err := platform.LoadBundle(localesDir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("i18n: %w", err)
	}
	renderer, err := platform.NewRenderer(templatesDir, bundle)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("templates: %w", err)
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
	charSvc.OnCompanionChange = battleSvc.SyncCompanions

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

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
	return &Server{Handler: handler, DB: db}, nil
}
