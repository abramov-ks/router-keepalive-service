package handler

import (
	"database/sql"
	"embed"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/cyrill/mikrotik-keepalive-server/internal/db"
	"github.com/cyrill/mikrotik-keepalive-server/internal/model"
)

type dashboardData struct {
	Router          string
	Date            string
	View            string
	TzOffsetMinutes int
}

func DashboardHandler(database *sql.DB, templateFS embed.FS) http.HandlerFunc {
	tmpl := template.Must(template.ParseFS(templateFS, "web/templates/dashboard.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		_, tzSecs := time.Now().Zone()
		data := dashboardData{
			Router:          q.Get("router"),
			Date:            q.Get("date"),
			View:            q.Get("view"),
			TzOffsetMinutes: tzSecs / 60,
		}
		if data.Date == "" {
			data.Date = time.Now().In(time.Local).Format("2006-01-02")
		}
		if data.View == "" {
			data.View = "24h"
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			slog.Error("template render error", "err", err)
		}
	}
}

func RoutersHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routers, err := db.ListRouters(database)
		if err != nil {
			slog.Error("list routers failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if routers == nil {
			routers = []model.RouterInfo{} // ensure JSON [] not null
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routers)
	}
}

func DailyStatsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routerID := r.URL.Query().Get("router_id")
		date := r.URL.Query().Get("date")
		if date == "" {
			date = time.Now().In(time.Local).Format("2006-01-02")
		}

		if routerID == "" {
			http.Error(w, `{"error":"router_id required"}`, http.StatusBadRequest)
			return
		}

		exists, err := db.RouterExists(database, routerID)
		if err != nil {
			slog.Error("router exists check failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"router not found"}`))
			return
		}

		buckets, err := db.DailyStats(database, routerID, date)
		if err != nil {
			slog.Error("daily stats failed", "router_id", routerID, "date", date, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := map[string]any{
			"router_id":        routerID,
			"date":             date,
			"interval_minutes": 1,
			"buckets":          buckets,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func Last24hStatsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routerID := r.URL.Query().Get("router_id")
		if routerID == "" {
			http.Error(w, `{"error":"router_id required"}`, http.StatusBadRequest)
			return
		}

		exists, err := db.RouterExists(database, routerID)
		if err != nil {
			slog.Error("router exists check failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"router not found"}`))
			return
		}

		buckets, err := db.Last24hStats(database, routerID)
		if err != nil {
			slog.Error("last24h stats failed", "router_id", routerID, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := map[string]any{
			"router_id":        routerID,
			"interval_minutes": 1,
			"buckets":          buckets,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func WeeklyStatsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routerID := r.URL.Query().Get("router_id")
		if routerID == "" {
			http.Error(w, `{"error":"router_id required"}`, http.StatusBadRequest)
			return
		}

		exists, err := db.RouterExists(database, routerID)
		if err != nil {
			slog.Error("router exists check failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"router not found"}`))
			return
		}

		days, err := db.WeeklyStats(database, routerID)
		if err != nil {
			slog.Error("weekly stats failed", "router_id", routerID, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := map[string]any{
			"router_id": routerID,
			"days":      days,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
