# Implementation Plan: MikroTik Keepalive Server (Full Service)

**Branch**: `002-stats-dashboard` | **Date**: 2026-05-17 | **Specs**: [001-ping-handler](../001-ping-handler/spec.md), [002-stats-dashboard](spec.md)

**Input**: Feature specifications from `specs/001-ping-handler/spec.md` and `specs/002-stats-dashboard/spec.md`

## Summary

A single Go binary that receives scheduled HTTP keepalive pings from MikroTik routers,
persists them in a local SQLite database keyed by router ID, and serves a server-rendered
web dashboard with Chart.js visualizations showing per-router daily and weekly ping
activity. No external dependencies at runtime — everything embeds into one binary.

## Technical Context

**Language/Version**: Go 1.22+

**Primary Dependencies**:
- `modernc.org/sqlite` — CGO-free pure-Go SQLite driver (no GCC, no CGO)
- `github.com/go-chi/chi/v5` — lightweight HTTP router
- Chart.js v4 (bundled via `//go:embed`) — client-side chart rendering
- Go standard library `html/template`, `net/http`, `embed`, `log/slog`

**Storage**: SQLite (single local file, auto-migrated on startup)

**Testing**: `go test ./...` with `net/http/httptest` for handler integration tests;
in-memory SQLite DSN (`file::memory:?cache=shared`) for DB unit tests

**Target Platform**: Linux server or Docker container; single self-contained binary

**Project Type**: Web service (single binary, server-rendered + JSON data API)

**Performance Goals**: Ping reception p95 < 200ms; dashboard initial load < 3s on LAN;
chart update on router/date switch < 1s

**Constraints**: No CGO; no external runtime dependencies; binary MUST start cleanly in
< 5s; zero manual DB setup

**Scale/Scope**: 1–50 routers; ~30-second ping interval per router; 7-day retention
(~10 080 pings/router/week; < 5MB SQLite per router/week)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| I. Ping Reception Reliability | Ping write is atomic within request cycle; 5xx on storage failure | ✅ Pass |
| II. MikroTik Protocol Compatibility | Endpoint is `GET /ping?id=<id>` — no custom headers, no auth | ✅ Pass |
| III. Data Persistence & Dashboard | SQLite with auto-migration; Chart.js bundled; 7-day retention enforced | ✅ Pass |
| IV. Single-Binary Simplicity | CGO-free SQLite, `//go:embed` for all web assets; startable with `./server` | ✅ Pass |
| V. Operational Correctness | Startup validates port + DB write; auto-schema-migration; `/health` endpoint; `slog` structured logs | ✅ Pass |

No violations. Complexity Tracking section not required.

## Project Structure

### Documentation (this feature)

```text
specs/001-ping-handler/
├── spec.md
├── checklists/requirements.md
specs/002-stats-dashboard/
├── spec.md
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── ping-endpoint.md
│   ├── health-endpoint.md
│   └── dashboard-api.md
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
mikrotik-keepalive-server/
├── main.go                        # Config, startup validation, server launch
├── internal/
│   ├── db/
│   │   ├── db.go                  # Open connection, run migrations, startup check
│   │   └── queries.go             # StorePing, ListRouters, DailyStats, WeeklyStats
│   ├── handler/
│   │   ├── ping.go                # GET /ping?id=<router_id>
│   │   ├── health.go              # GET /health
│   │   └── dashboard.go           # GET /, GET /api/routers, GET /api/stats/*
│   └── model/
│       └── event.go               # PingEvent struct
├── web/
│   ├── templates/
│   │   └── dashboard.html         # Server-side template (embedded)
│   └── static/
│       ├── chart.umd.min.js       # Chart.js bundle (embedded)
│       └── style.css              # Dashboard styles (embedded)
├── go.mod
├── go.sum
└── Dockerfile
```

**Structure Decision**: Single project layout. No frontend build step — all web assets are
static files embedded with `//go:embed`. Go standard library template engine renders the
HTML shell; Chart.js fetches data from JSON endpoints and renders charts client-side.

## Complexity Tracking

> No violations found. Section intentionally empty.
