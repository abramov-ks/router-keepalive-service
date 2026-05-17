package db

import (
	"database/sql"
	"log/slog"
	"time"
)

func DeleteOldEvents(db *sql.DB) error {
	res, err := db.Exec(`DELETE FROM ping_events WHERE received_at < datetime('now', '-7 days')`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		slog.Info("retention cleanup", "deleted_rows", n)
	}
	return nil
}

func StartCleanupJob(db *sql.DB) {
	go func() {
		if err := DeleteOldEvents(db); err != nil {
			slog.Error("cleanup on startup failed", "err", err)
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := DeleteOldEvents(db); err != nil {
				slog.Error("periodic cleanup failed", "err", err)
			}
		}
	}()
}
