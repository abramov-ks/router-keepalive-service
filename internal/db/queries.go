package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cyrill/mikrotik-keepalive-server/internal/model"
)

// StorePing persists one keepalive event. received_at is set by the server.
func StorePing(database *sql.DB, routerID, sourceIP string) error {
	_, err := database.Exec(
		`INSERT INTO ping_events (router_id, source_ip) VALUES (?, ?)`,
		routerID, sourceIP,
	)
	return err
}

// ListRouters returns all known routers ordered by first_seen ASC.
func ListRouters(database *sql.DB) ([]model.RouterInfo, error) {
	rows, err := database.Query(`
		SELECT router_id,
		       MAX(received_at) AS last_seen,
		       MIN(received_at) AS first_seen,
		       COUNT(*)         AS total_pings
		FROM ping_events
		GROUP BY router_id
		ORDER BY first_seen ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.RouterInfo
	for rows.Next() {
		var r model.RouterInfo
		var lastSeen, firstSeen string
		if err := rows.Scan(&r.RouterID, &lastSeen, &firstSeen, &r.TotalPings); err != nil {
			return nil, err
		}
		r.LastSeen, _ = time.Parse("2006-01-02 15:04:05", lastSeen)
		r.FirstSeen, _ = time.Parse("2006-01-02 15:04:05", firstSeen)
		result = append(result, r)
	}
	return result, rows.Err()
}

// sqliteTZOffset returns an offset string like "+3 hours" or "-5 hours" for use
// in SQLite's datetime() function, derived from the current time.Local zone.
func sqliteTZOffset() string {
	_, secs := time.Now().Zone()
	hours := secs / 3600
	if hours >= 0 {
		return fmt.Sprintf("+%d hours", hours)
	}
	return fmt.Sprintf("%d hours", hours)
}

// DailyStats returns 1-minute bucket ping counts for a router on a given date (YYYY-MM-DD).
// The returned slice always has exactly 1440 entries (one per minute of the day).
// Times are expressed in time.Local so the dashboard shows the configured timezone.
func DailyStats(database *sql.DB, routerID, date string) ([]model.BucketPoint, error) {
	tz := sqliteTZOffset()
	rows, err := database.Query(`
		SELECT strftime('%Y-%m-%dT%H:%M:00', datetime(received_at, ?)) AS bucket,
		       COUNT(*) AS cnt
		FROM ping_events
		WHERE router_id = ? AND date(datetime(received_at, ?)) = ?
		GROUP BY bucket
		ORDER BY bucket
	`, tz, routerID, tz, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build a map of bucket → count from DB
	counts := make(map[string]int, 1440)
	for rows.Next() {
		var bucket string
		var cnt int
		if err := rows.Scan(&bucket, &cnt); err != nil {
			return nil, err
		}
		counts[bucket] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Zero-fill all 1440 minutes in local time
	result := make([]model.BucketPoint, 1440)
	base, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", date, err)
	}
	for i := 0; i < 1440; i++ {
		t := base.Add(time.Duration(i) * time.Minute)
		key := t.Format("2006-01-02T15:04:05")
		result[i] = model.BucketPoint{Time: key, Count: counts[key]}
	}
	return result, nil
}

// Last24hStats returns 1-minute bucket ping counts for the rolling 24-hour window
// ending at the current moment. The returned slice always has exactly 1440 entries.
// Times are expressed in time.Local.
func Last24hStats(database *sql.DB, routerID string) ([]model.BucketPoint, error) {
	tz := sqliteTZOffset()
	now := time.Now().In(time.Local)
	start := now.Add(-24 * time.Hour)

	// Round start down to the minute
	start = start.Truncate(time.Minute)

	startUTC := start.UTC().Format("2006-01-02 15:04:05")
	endUTC := now.UTC().Format("2006-01-02 15:04:05")

	rows, err := database.Query(`
		SELECT strftime('%Y-%m-%dT%H:%M:00', datetime(received_at, ?)) AS bucket,
		       COUNT(*) AS cnt
		FROM ping_events
		WHERE router_id = ?
		  AND received_at >= ?
		  AND received_at <= ?
		GROUP BY bucket
		ORDER BY bucket
	`, tz, routerID, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int, 1440)
	for rows.Next() {
		var bucket string
		var cnt int
		if err := rows.Scan(&bucket, &cnt); err != nil {
			return nil, err
		}
		counts[bucket] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]model.BucketPoint, 1440)
	for i := 0; i < 1440; i++ {
		t := start.Add(time.Duration(i) * time.Minute)
		key := t.Format("2006-01-02T15:04:05")
		result[i] = model.BucketPoint{Time: key, Count: counts[key]}
	}
	return result, nil
}

// WeeklyStats returns per-day ping counts for the last 7 days for a router.
// The returned slice always has exactly 7 entries (today−6 through today).
// Dates are in time.Local so day boundaries match the configured timezone.
func WeeklyStats(database *sql.DB, routerID string) ([]model.DayPoint, error) {
	tz := sqliteTZOffset()
	rows, err := database.Query(`
		SELECT date(datetime(received_at, ?)) AS day, COUNT(*) AS cnt
		FROM ping_events
		WHERE router_id = ? AND received_at >= datetime('now', '-7 days')
		GROUP BY day
		ORDER BY day
	`, tz, routerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int, 7)
	for rows.Next() {
		var day string
		var cnt int
		if err := rows.Scan(&day, &cnt); err != nil {
			return nil, err
		}
		counts[day] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Zero-fill 7 days: today−6 through today
	today := time.Now().In(time.Local).Format("2006-01-02")
	base, _ := time.Parse("2006-01-02", today)
	result := make([]model.DayPoint, 7)
	for i := 0; i < 7; i++ {
		d := base.AddDate(0, 0, i-6).Format("2006-01-02")
		result[i] = model.DayPoint{Date: d, Count: counts[d]}
	}
	return result, nil
}

// RouterExists returns true when the given router ID has at least one ping stored.
func RouterExists(database *sql.DB, routerID string) (bool, error) {
	var count int
	err := database.QueryRow(
		`SELECT COUNT(*) FROM ping_events WHERE router_id = ? LIMIT 1`, routerID,
	).Scan(&count)
	return count > 0, err
}
