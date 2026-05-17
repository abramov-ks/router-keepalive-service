# Research: MikroTik Keepalive Server

**Feature**: 001-ping-handler + 002-stats-dashboard
**Date**: 2026-05-17

## Decision 1: SQLite Driver

**Decision**: `modernc.org/sqlite` (CGO-free pure-Go driver)

**Rationale**: No GCC or CGO toolchain required at build or runtime. Enables
straightforward cross-compilation (`GOOS=linux GOARCH=amd64 go build`) and a lean
Docker image (`FROM scratch` or `FROM alpine` with no extra packages). Performance
is comparable to `mattn/go-sqlite3` for the expected load (~2 writes/min at 30s
interval, <50 routers).

**Alternatives considered**:
- `mattn/go-sqlite3`: Faster in benchmarks, widely used — rejected because it
  requires CGO and GCC at build time, complicating Docker multi-stage builds and
  cross-compilation.
- `zombiezen.com/go/sqlite`: Also CGO-free, newer API — viable fallback if
  `modernc.org/sqlite` proves limiting, but `modernc` is more widely adopted.

---

## Decision 2: HTTP Router

**Decision**: `github.com/go-chi/chi/v5`

**Rationale**: Chi is idiomatic Go, lightweight (~3k LOC core), and zero-overhead
compared to `net/http/ServeMux`. It provides clean route grouping (e.g., `/api/*`
vs `/`), built-in middleware (logger, recoverer), and URL parameter parsing. For a
project of this size the standard library mux is sufficient, but chi reduces
boilerplate with no downside.

**Alternatives considered**:
- `net/http` stdlib mux: No dependency, but pattern matching is limited and
  middleware chaining is verbose. Acceptable but chi offers better ergonomics.
- `gin`: More feature-rich but heavier; not justified for a service with ~6 routes.
- `echo`: Similar to gin; similarly overweight for this scope.

---

## Decision 3: Chart Library

**Decision**: Chart.js v4 (bundled as a single UMD file via `//go:embed`)

**Rationale**: Chart.js provides bar charts and timeline/scatter plots out of the
box with a minimal, well-documented API. The UMD build is a single ~220KB file with
no additional dependencies. It renders client-side via Canvas, which works in all
modern browsers without a build step on the server.

**Daily chart implementation**: A bar chart where the X-axis is time buckets
(1-minute resolution) and each bar is present (green) or absent/zero (grey) based
on whether a ping was received in that minute. This clearly shows gaps.

**Weekly chart implementation**: A bar chart where the X-axis is calendar days and
the Y-axis is the count of pings received per day. Clicking a bar triggers a
`window.location` update with the selected date as a query param.

**Alternatives considered**:
- D3.js: Extremely flexible but ~500KB and requires significant custom code for
  simple bar charts — rejected on complexity and bundle size grounds.
- ApexCharts: Good API, but ~400KB and more complex licensing — not justified.
- `go-echarts` (server-rendered): Would eliminate client-side JS but produces
  complex HTML; less maintainable than a simple Chart.js integration.

---

## Decision 4: Rendering Strategy

**Decision**: Server-side HTML rendering with Go `html/template` + JSON data API
for Chart.js (minimal vanilla JS, no framework)

**Rationale**: The HTML page structure (router selector, date picker, chart
containers) is rendered server-side on each full page load. Chart data is fetched
from `/api/stats/*` JSON endpoints by vanilla JS at load time and on user
interaction (router switch, date change). This eliminates any JS build step,
keeps the binary self-contained, and aligns with the Single-Binary Simplicity
principle.

**Alternatives considered**:
- Full SPA (React/Vue): Would require a Node.js build step and significantly
  increase asset complexity — rejected.
- HTMX: Attractive for eliminating custom JS, but adds another bundled dependency
  and the Chart.js integration would still require vanilla JS — no clear benefit.
- Pure server-side rendering (no client JS): Impossible for interactive charts;
  page reload on every date/router change would be sluggish.

---

## Decision 5: Ping Endpoint Format

**Decision**: `GET /ping?id=<router_id>` — simple HTTP GET with a single query parameter

**Rationale**: MikroTik RouterOS `/tool fetch` can send a GET request to any URL.
The router ID can be embedded in the URL as a query parameter, which the router
administrator sets once in the scheduler script. No custom headers, no POST body,
no authentication — exactly what MikroTik's scheduler can produce natively.

**Example MikroTik scheduler script**:
```
/tool fetch url="http://192.168.1.100:8080/ping?id=office-router-1" keep-result=no
```

**Router ID validation**: Accept `[a-zA-Z0-9_-]{1,64}`. Reject anything else with
HTTP 400. This prevents injection and ensures safe storage as a SQLite column value.

**Alternatives considered**:
- `POST /ping` with JSON body: MikroTik can POST but cannot easily construct a JSON
  body in a scheduler script — rejected.
- Router ID in URL path (`GET /ping/<id>`): Slightly cleaner URL but requires more
  careful path parsing — `?id=` query param is simpler and equally safe.

---

## Decision 6: Data Retention

**Decision**: Soft 7-day retention enforced by a periodic cleanup job (runs on server
startup and then every 24 hours)

**Rationale**: Deleting rows older than 7 days keeps the SQLite file small
(< 10MB for 50 routers). Running cleanup at startup + daily avoids a cron dependency
and is simple to implement with a Go goroutine.

**Schema**: No soft-delete column needed. A simple `DELETE FROM ping_events WHERE
received_at < datetime('now', '-7 days')` is sufficient.

---

## MikroTik Compatibility Notes

- MikroTik RouterOS `/tool fetch` supports HTTP GET and POST.
- Default timeout for `/tool fetch` is 15 seconds; the server MUST respond within
  this window (easily met with SQLite local writes).
- The `keep-result=no` flag prevents RouterOS from storing the response body,
  reducing memory pressure on the router.
- RouterOS scheduler minimum interval is 1 second; 30-second intervals are typical
  for keepalive use cases.
