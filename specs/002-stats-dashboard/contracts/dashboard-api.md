# Contract: Dashboard API

The dashboard is served as a server-rendered HTML page. Chart data is loaded
asynchronously from JSON endpoints called by client-side JavaScript.

---

## Dashboard Page

**Endpoint**: `GET /`

**Purpose**: Serve the dashboard HTML page.

### Request

| Component | Value |
|-----------|-------|
| Method | `GET` |
| Path | `/` |
| Auth | None |

### Query Parameters (optional, for state persistence via URL)

| Parameter | Default | Description |
|-----------|---------|-------------|
| `router` | first known router by first_seen | Selected router ID |
| `date` | today (server local date) | Selected date `YYYY-MM-DD` |
| `view` | `day` | `day` or `week` |

### Response

```
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8

<full HTML page>
```

The page includes:
- Router selector `<select>` populated with all known router IDs
- Date picker `<input type="date">` limited to today − 6 days through today
- Day / Week toggle
- Two `<canvas>` elements for Chart.js (daily chart + weekly chart)
- Embedded or inline JS that calls `/api/routers`, `/api/stats/daily`, and
  `/api/stats/weekly` to populate the charts

---

## List Routers

**Endpoint**: `GET /api/routers`

**Purpose**: Return all router IDs known to the system.

### Response

```json
[
  {
    "router_id": "office-router-1",
    "last_seen": "2026-05-17T14:32:10Z",
    "first_seen": "2026-05-10T08:00:01Z",
    "total_pings": 20160
  },
  {
    "router_id": "branch-router-2",
    "last_seen": "2026-05-17T14:31:58Z",
    "first_seen": "2026-05-12T09:15:00Z",
    "total_pings": 14400
  }
]
```

Empty array `[]` if no routers have pinged yet.

---

## Daily Stats

**Endpoint**: `GET /api/stats/daily`

**Purpose**: Return per-minute ping counts for a router on a given calendar day.
Used to render the daily chart.

### Query Parameters

| Parameter | Required | Format | Example |
|-----------|----------|--------|---------|
| `router_id` | Yes | router ID string | `router_id=office-router-1` |
| `date` | No | `YYYY-MM-DD` | `date=2026-05-17` (default: today) |

### Response

```json
{
  "router_id": "office-router-1",
  "date": "2026-05-17",
  "interval_minutes": 1,
  "buckets": [
    { "time": "2026-05-17T00:00:00Z", "count": 2 },
    { "time": "2026-05-17T00:01:00Z", "count": 0 },
    { "time": "2026-05-17T00:02:00Z", "count": 2 },
    "..."
  ]
}
```

- `buckets` always contains exactly 1440 entries (one per minute of the day),
  even if most have `count: 0`.
- All timestamps are UTC ISO-8601.
- `count: 0` buckets represent missed ping intervals (rendered as gaps/grey bars).

### Error: unknown router

```
HTTP/1.1 404 Not Found
{"error":"router not found"}
```

---

## Weekly Stats

**Endpoint**: `GET /api/stats/weekly`

**Purpose**: Return per-day ping counts for a router over the last 7 calendar days.
Used to render the weekly chart.

### Query Parameters

| Parameter | Required | Format | Example |
|-----------|----------|--------|---------|
| `router_id` | Yes | router ID string | `router_id=office-router-1` |

### Response

```json
{
  "router_id": "office-router-1",
  "days": [
    { "date": "2026-05-11", "count": 2880 },
    { "date": "2026-05-12", "count": 2880 },
    { "date": "2026-05-13", "count": 1440 },
    { "date": "2026-05-14", "count": 0 },
    { "date": "2026-05-15", "count": 2880 },
    { "date": "2026-05-16", "count": 2880 },
    { "date": "2026-05-17", "count": 960 }
  ]
}
```

- `days` always contains exactly 7 entries (today − 6 through today), even if some
  days have `count: 0`.
- Days with `count: 0` may indicate the router was offline or not yet registered.

### Error: unknown router

```
HTTP/1.1 404 Not Found
{"error":"router not found"}
```

---

## Static Assets

**Endpoint**: `GET /static/<filename>`

Serves embedded static files: `chart.umd.min.js`, `style.css`.

```
HTTP/1.1 200 OK
Content-Type: <appropriate MIME type>
Cache-Control: max-age=86400
```
