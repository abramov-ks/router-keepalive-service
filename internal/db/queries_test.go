package db

import (
	"database/sql"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func insertPing(t *testing.T, database *sql.DB, routerID, receivedAt string) {
	t.Helper()
	_, err := database.Exec(
		`INSERT INTO ping_events (router_id, received_at, source_ip) VALUES (?, ?, '')`,
		routerID, receivedAt,
	)
	if err != nil {
		t.Fatalf("insert ping: %v", err)
	}
}

func TestLastSeenByRouterEmpty(t *testing.T) {
	database := openTestDB(t)

	seen, err := LastSeenByRouter(database)
	if err != nil {
		t.Fatalf("LastSeenByRouter: %v", err)
	}
	if len(seen) != 0 {
		t.Fatalf("expected empty map, got %v", seen)
	}
}

func TestLastSeenByRouterSingle(t *testing.T) {
	database := openTestDB(t)
	insertPing(t, database, "dacha", "2026-07-09 10:00:00")
	insertPing(t, database, "dacha", "2026-07-09 10:00:30")

	seen, err := LastSeenByRouter(database)
	if err != nil {
		t.Fatalf("LastSeenByRouter: %v", err)
	}
	want := time.Date(2026, 7, 9, 10, 0, 30, 0, time.UTC)
	if got := seen["dacha"]; !got.Equal(want) {
		t.Errorf("dacha last seen = %v, want %v", got, want)
	}
}

func TestLastSeenByRouterMultipleInterleaved(t *testing.T) {
	database := openTestDB(t)
	insertPing(t, database, "office", "2026-07-09 09:00:00")
	insertPing(t, database, "dacha", "2026-07-09 09:00:10")
	insertPing(t, database, "office", "2026-07-09 09:01:00")
	insertPing(t, database, "dacha", "2026-07-09 08:59:00")

	seen, err := LastSeenByRouter(database)
	if err != nil {
		t.Fatalf("LastSeenByRouter: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("expected 2 routers, got %d: %v", len(seen), seen)
	}
	wantOffice := time.Date(2026, 7, 9, 9, 1, 0, 0, time.UTC)
	if got := seen["office"]; !got.Equal(wantOffice) {
		t.Errorf("office last seen = %v, want %v", got, wantOffice)
	}
	wantDacha := time.Date(2026, 7, 9, 9, 0, 10, 0, time.UTC)
	if got := seen["dacha"]; !got.Equal(wantDacha) {
		t.Errorf("dacha last seen = %v, want %v", got, wantDacha)
	}
}
