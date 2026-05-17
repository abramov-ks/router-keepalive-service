package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cyrill/mikrotik-keepalive-server/internal/config"
	"github.com/cyrill/mikrotik-keepalive-server/internal/db"
	"github.com/cyrill/mikrotik-keepalive-server/internal/handler"
)

//go:embed web/static
var staticFS embed.FS

//go:embed web/templates
var templateFS embed.FS

func main() {
	// ── Config ────────────────────────────────────────────────────────────────
	exe, _ := os.Executable()
	binDir := filepath.Dir(exe)

	cfg, err := config.Load(binDir)
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Error("cannot load timezone", "timezone", cfg.Timezone, "err", err)
		os.Exit(1)
	}
	time.Local = loc

	port   := strconv.Itoa(cfg.Port)
	dbPath := envOr("DB_PATH", "keepalive.db")

	// ── Database ──────────────────────────────────────────────────────────────
	database, err := db.Open(dbPath)
	if err != nil {
		slog.Error("cannot open database", "path", dbPath, "err", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}

	// Startup validation: confirm DB is writable
	if err := validateDB(database); err != nil {
		slog.Error("database write check failed", "err", err)
		os.Exit(1)
	}

	// ── Allowlist & reload ────────────────────────────────────────────────────
	store := config.NewAllowlistStore(cfg.AllowedRouters)
	configPath := filepath.Join(binDir, "config.yaml")
	config.StartReloadHandler(configPath, store, cfg.Port, cfg.Timezone)

	// ── Background jobs ───────────────────────────────────────────────────────
	db.StartCleanupJob(database)

	// ── Static files ──────────────────────────────────────────────────────────
	staticSub, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		slog.Error("cannot sub static FS", "err", err)
		os.Exit(1)
	}

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Handle("/static/*", http.StripPrefix("/static/", cacheHandler(http.FileServer(http.FS(staticSub)))))
	r.Get("/health", handler.HealthHandler(database))
	r.Get("/ping", handler.PingHandler(database, store))
	r.Get("/api/routers", handler.RoutersHandler(database))
	r.Get("/api/stats/daily", handler.DailyStatsHandler(database))
	r.Get("/api/stats/weekly", handler.WeeklyStatsHandler(database))
	r.Get("/", handler.DashboardHandler(database, templateFS))

	slog.Info("starting server", "port", port, "db", dbPath)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func validateDB(database *sql.DB) error {
	_, err := database.Exec(
		`INSERT INTO ping_events (router_id, source_ip) VALUES ('__health__', ''); ` +
		`DELETE FROM ping_events WHERE router_id='__health__'`,
	)
	return err
}

func cacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=86400")
		h.ServeHTTP(w, r)
	})
}
