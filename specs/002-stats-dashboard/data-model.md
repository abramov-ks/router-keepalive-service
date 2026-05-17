# Data Model: MikroTik Keepalive Server

**Feature**: 001-ping-handler + 002-stats-dashboard
**Date**: 2026-05-17

## Entities

### PingEvent

Represents a single keepalive ping received from a MikroTik router.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | Surrogate key |
| `router_id` | TEXT | NOT NULL, CHECK regex `[a-zA-Z0-9_-]{1,64}` | Router-supplied identifier |
| `received_at` | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | UTC receipt timestamp |
| `source_ip` | TEXT | NOT NULL | Client IP address as received by the server |

**Notes**:
- `received_at` is stored as UTC ISO-8601 string in SQLite (`YYYY-MM-DD HH:MM:SS`).
- `router_id` is validated on ingest; invalid IDs are rejected with HTTP 400.
- No foreign key to a Routers table — router identity is derived from `ping_events`.

### Router (derived, no dedicated table)

Routers are not pre-registered. A router is "known" when it has at least one row in
`ping_events`. All router-level queries are derived views over `ping_events`.

**Derived fields available via query**:
- `router_id` — from DISTINCT on `ping_events.router_id`
- `last_seen` — MAX(`received_at`) GROUP BY `router_id`
- `first_seen` — MIN(`received_at`) GROUP BY `router_id`

## SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS ping_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    router_id   TEXT    NOT NULL
                        CHECK(length(router_id) BETWEEN 1 AND 64
                              AND router_id GLOB '[A-Za-z0-9_-]*'),
    received_at DATETIME NOT NULL DEFAULT (datetime('now')),
    source_ip   TEXT    NOT NULL DEFAULT ''
);

-- Supports time-range queries per router (dashboard charts, cleanup)
CREATE INDEX IF NOT EXISTS idx_ping_events_router_time
    ON ping_events (router_id, received_at);

-- Supports listing known routers with last-seen time
CREATE INDEX IF NOT EXISTS idx_ping_events_received_at
    ON ping_events (received_at);
```

## Key Queries

### Store a ping
```sql
INSERT INTO ping_events (router_id, source_ip)
VALUES (?, ?);
```

### List known routers (with last-seen)
```sql
SELECT router_id,
       MAX(received_at) AS last_seen,
       MIN(received_at) AS first_seen,
       COUNT(*)         AS total_pings
FROM ping_events
GROUP BY router_id
ORDER BY first_seen ASC;
```

### Daily stats (1-minute buckets for a given router and calendar day)
```sql
SELECT strftime('%Y-%m-%d %H:%M', received_at, 'start of minute') AS bucket,
       COUNT(*) AS ping_count
FROM ping_events
WHERE router_id = ?
  AND date(received_at) = ?      -- e.g. '2026-05-17'
GROUP BY bucket
ORDER BY bucket;
```
The server fills in zero-count buckets for all 1440 minutes of the day before
returning the response, so the chart always has a full 24-hour dataset.

### Weekly stats (per-day summary for last 7 days)
```sql
SELECT date(received_at) AS day,
       COUNT(*)           AS ping_count
FROM ping_events
WHERE router_id = ?
  AND received_at >= datetime('now', '-7 days')
GROUP BY day
ORDER BY day;
```
The server fills in zero-count days for any of the 7 days with no data.

### Cleanup (retention enforcement — runs at startup + every 24h)
```sql
DELETE FROM ping_events
WHERE received_at < datetime('now', '-7 days');
```

## Validation Rules

| Rule | Where enforced |
|------|---------------|
| `router_id` matches `[a-zA-Z0-9_-]{1,64}` | HTTP handler (before DB write) |
| `router_id` is not empty | HTTP handler |
| `received_at` is always UTC | Set by server at write time; never from client |
| `source_ip` is the request's remote address | Set by server; never from client |

## Storage Estimates

| Scenario | Rows/day | Row size (~bytes) | DB size/week |
|----------|----------|-------------------|--------------|
| 1 router, 30s interval | 2,880 | ~80 | ~1.6 MB |
| 10 routers, 30s interval | 28,800 | ~80 | ~16 MB |
| 50 routers, 30s interval | 144,000 | ~80 | ~80 MB |

SQLite handles all of these comfortably. WAL mode is enabled for concurrent
read/write access (dashboard reads while ping writes occur).
