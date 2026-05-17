package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func HealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var status struct {
			Status string `json:"status"`
			DB     string `json:"db"`
		}
		if _, err := db.ExecContext(r.Context(), "SELECT 1"); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status.Status = "degraded"
			status.DB = fmt.Sprintf("error: %v", err)
		} else {
			status.Status = "ok"
			status.DB = "ok"
		}
		json.NewEncoder(w).Encode(status)
	}
}
