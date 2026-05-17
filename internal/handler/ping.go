package handler

import (
	"database/sql"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/cyrill/mikrotik-keepalive-server/internal/db"
)

var routerIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func PingHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routerID := r.URL.Query().Get("id")
		sourceIP := r.RemoteAddr

		if !routerIDRegex.MatchString(routerID) {
			slog.Warn("ping rejected: invalid router id", "id", routerID, "remote", sourceIP)
			http.Error(w, "missing or invalid router id", http.StatusBadRequest)
			return
		}

		if err := db.StorePing(database, routerID, sourceIP); err != nil {
			slog.Error("ping storage failed", "router_id", routerID, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		slog.Info("ping received", "router_id", routerID, "remote", sourceIP)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
