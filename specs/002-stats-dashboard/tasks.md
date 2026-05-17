---

description: "Task list for MikroTik Keepalive Server — full service (ping handler + dashboard)"
---

# Tasks: MikroTik Keepalive Server

**Input**: Design documents from `specs/002-stats-dashboard/` (covers both 001-ping-handler and 002-stats-dashboard)

**Prerequisites**: plan.md ✅, spec.md ✅ (both specs), research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Not requested — no test tasks generated.

**Organization**: Tasks grouped by user story to enable independent delivery of each slice.

## Format: `[ID] [P?] [Story?] Description — file path`

- **[P]**: Can run in parallel (different files, no unmet dependencies)
- **[Story]**: User story label — US1/US2/US3/US4
- All file paths are relative to repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize project skeleton — no functional code yet.

- [ ] T001 Initialize Go module: `go mod init github.com/cyrill/mikrotik-keepalive-server` → `go.mod`
- [ ] T002 Add dependencies to `go.mod`: `modernc.org/sqlite`, `github.com/go-chi/chi/v5` — run `go mod tidy`
- [ ] T003 [P] Create directory structure: `internal/db/`, `internal/handler/`, `internal/model/`, `web/templates/`, `web/static/`
- [ ] T004 [P] Download Chart.js v4 UMD bundle and save to `web/static/chart.umd.min.js`
- [ ] T005 [P] Create `web/static/style.css` with minimal dashboard styles (flexbox layout, chart containers, router selector, date picker)
- [ ] T006 [P] Create placeholder `web/templates/dashboard.html` (empty HTML shell — content filled in US2)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that ALL user stories depend on. No user story work begins until this phase is complete.

**⚠️ CRITICAL**: Phases 3–6 cannot start until Phase 2 is complete.

- [ ] T007 Create `internal/model/event.go` — define `PingEvent` struct: `ID int64`, `RouterID string`, `ReceivedAt time.Time`, `SourceIP string`
- [ ] T008 Create `internal/db/db.go` — `Open(path string) (*sql.DB, error)`: open SQLite file, enable WAL mode (`PRAGMA journal_mode=WAL`), enable foreign keys, return connection
- [ ] T009 Create schema migration in `internal/db/db.go` — `Migrate(db *sql.DB) error`: create `ping_events` table and both indexes as defined in `specs/002-stats-dashboard/data-model.md`
- [ ] T010 Create `internal/db/cleanup.go` — `DeleteOldEvents(db *sql.DB) error`: delete rows where `received_at < datetime('now', '-7 days')`; run once at startup then every 24h via goroutine
- [ ] T011 Create `main.go` skeleton — read `PORT` (default `8080`), `DB_PATH` (default `keepalive.db`), `TZ` env vars; call `db.Open`, `db.Migrate`; startup validation: attempt a test write and delete to confirm DB is writable (exit with descriptive error if not); start cleanup goroutine
- [ ] T012 Set up chi router in `main.go` — add `chi.NewRouter()`, attach `middleware.Logger` and `middleware.Recoverer`; mount `/static/` file server from embedded `web/static/` FS
- [ ] T013 Add `//go:embed` declarations in `main.go` (or a dedicated `embed.go`): embed `web/static` as `staticFS` and `web/templates` as `templateFS`
- [ ] T014 Create `internal/handler/health.go` — `HealthHandler(db *sql.DB) http.HandlerFunc`: run `SELECT 1`, return `{"status":"ok","db":"ok"}` on success or `{"status":"degraded","db":"error: ..."}` with HTTP 503 on failure
- [ ] T015 Wire `/health` route in `main.go`; start HTTP server with `http.ListenAndServe`

**Checkpoint**: `go build ./...` succeeds; `./server` starts; `curl localhost:8080/health` returns `{"status":"ok","db":"ok"}`

---

## Phase 3: User Story 1 — Ping Reception (Priority: P1) 🎯 MVP

**Goal**: Accept MikroTik keepalive pings, validate router ID, persist to SQLite.
Supports any number of distinct router IDs on the same endpoint.

**Independent Test**: `curl "localhost:8080/ping?id=router-1"` → `200 OK body:OK`; verify row in DB.
`curl "localhost:8080/ping"` → `400 Bad Request`. Two different router IDs both stored.

### Implementation for User Story 1

- [ ] T016 [US1] Create `StorePing` in `internal/db/queries.go` — `StorePing(db *sql.DB, routerID, sourceIP string) error`: `INSERT INTO ping_events (router_id, source_ip) VALUES (?, ?)` using server-set `received_at` default
- [ ] T017 [US1] Create `internal/handler/ping.go` — `PingHandler(db *sql.DB) http.HandlerFunc`: extract `id` query param; validate against regex `^[a-zA-Z0-9_-]{1,64}$`; return 400 if invalid/missing; call `StorePing`; return `200 OK\nOK`; return 500 on storage error; log each attempt with `slog` (router_id, source_ip, outcome)
- [ ] T018 [US1] Wire `GET /ping` route in `main.go`

**Checkpoint**: Single router pings stored correctly. Multiple router IDs create separate rows. Missing `id` param returns 400. Storage failure returns 500 (not 200).

---

## Phase 4: User Story 2 — Daily Chart Dashboard (Priority: P1)

**Goal**: Browser-accessible dashboard showing today's ping activity chart for the first known
router. Includes date picker to navigate to any of the past 7 days. Chart shows per-minute
ping presence/absence. Empty state displayed when no data exists.

**Independent Test**: With router pings in DB, open `http://localhost:8080/` — daily chart renders
with green bars for received pings. Change date picker to yesterday — chart updates without reload.
Select a date with no data — empty state message shown.

### Implementation for User Story 2

- [ ] T019 [P] [US2] Create `DailyStats` in `internal/db/queries.go` — `DailyStats(db *sql.DB, routerID, date string) ([]BucketPoint, error)`: 1-minute bucket GROUP BY query for the given day; zero-fill all 1440 minutes in Go before returning
- [ ] T020 [P] [US2] Create `ListRouters` in `internal/db/queries.go` — `ListRouters(db *sql.DB) ([]RouterInfo, error)`: SELECT DISTINCT router_id with last_seen, first_seen, total_pings; ORDER BY first_seen ASC
- [ ] T021 [US2] Implement `GET /api/routers` handler in `internal/handler/dashboard.go` — call `ListRouters`, return JSON array; return `[]` if empty
- [ ] T022 [US2] Implement `GET /api/stats/daily` handler in `internal/handler/dashboard.go` — require `router_id` query param; optional `date` (default today in server TZ); call `DailyStats`; return JSON per contract `specs/002-stats-dashboard/contracts/dashboard-api.md`; return 404 if router unknown
- [ ] T023 [US2] Fill `web/templates/dashboard.html` — complete HTML: `<select>` for router, `<input type="date">` limited to today−6d through today, Day/Week toggle buttons, two `<canvas>` elements (daily-chart, weekly-chart), `<div id="last-ping">` for last-ping timestamp display
- [ ] T024 [US2] Add vanilla JS in `web/templates/dashboard.html` — on load: fetch `/api/routers`, populate selector, set default router and date; fetch `/api/stats/daily`, render Chart.js bar chart (green=#4caf50 for count>0, grey=#e0e0e0 for count=0); wire date picker `change` event to refetch and re-render; wire Day/Week toggle buttons; update URL query params on state change (`history.replaceState`)
- [ ] T025 [US2] Implement `GET /` handler in `internal/handler/dashboard.go` — parse `?router=`, `?date=`, `?view=` query params; render `dashboard.html` template passing params as template data for initial page state; wire in `main.go`
- [ ] T026 [US2] Add last-ping timestamp to `GET /api/routers` response and display in dashboard `<div id="last-ping">` via JS

**Checkpoint**: Dashboard loads. Daily chart renders for today. Date picker navigates to past days.
No-data dates show empty state message. URL state preserved on page reload.

---

## Phase 5: User Story 3 — 7-Day Overview (Priority: P2)

**Goal**: Toggle to a weekly summary bar chart spanning the last 7 calendar days. Click a day
bar to drill into that day's daily chart.

**Independent Test**: With data spanning ≥3 days, open dashboard and click "Week" toggle — 7-bar
chart appears covering last 7 days. Days with zero pings show as empty bars. Click a bar with data
— view switches to daily chart for that date.

### Implementation for User Story 3

- [ ] T027 [P] [US3] Create `WeeklyStats` in `internal/db/queries.go` — `WeeklyStats(db *sql.DB, routerID string) ([]DayPoint, error)`: per-day COUNT query for last 7 days; zero-fill all 7 days in Go before returning
- [ ] T028 [US3] Implement `GET /api/stats/weekly` handler in `internal/handler/dashboard.go` — require `router_id`; call `WeeklyStats`; return JSON per contract; return 404 if router unknown; wire in `main.go`
- [ ] T029 [US3] Add weekly chart JS logic in `web/templates/dashboard.html` — on "Week" toggle click: show weekly-chart canvas, hide daily-chart canvas; fetch `/api/stats/weekly` for current router; render Chart.js bar chart (X=day labels, Y=ping count); add `onClick` handler on bars: set date picker value to clicked day, switch to Day view, refetch daily data

**Checkpoint**: Weekly chart renders correctly. Zero-ping days shown as empty bars. Clicking a bar
drills into that day's daily chart.

---

## Phase 6: User Story 4 — Router Selector (Priority: P2)

**Goal**: Router `<select>` lists all known routers. Selecting a different router updates all charts.
Single-router case handled gracefully.

**Independent Test**: With 2 routers in DB, open dashboard — both appear in selector. Switch router
— both daily and weekly charts update. With only 1 router, selector is present but pre-selected.

### Implementation for User Story 4

- [ ] T030 [US4] Update JS in `web/templates/dashboard.html` — wire router `<select>` `change` event: update current router state, refetch and re-render active chart (daily or weekly), update last-ping display, update URL `?router=` param
- [ ] T031 [US4] Handle single-router case in JS — when `/api/routers` returns 1 entry, optionally add `disabled` attribute to `<select>` and show router name as text label
- [ ] T032 [US4] Handle zero-router case in JS — when `/api/routers` returns `[]`, show full-page empty state ("No routers have pinged yet. Configure your MikroTik router to send pings to this server.") and hide chart area

**Checkpoint**: Multi-router switching works. Charts update independently per router. Empty state
shown before any router has pinged.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Operational hardening, deployment, and final validation.

- [ ] T033 [P] Create `Dockerfile` — multi-stage build: `golang:1.22-alpine` build stage → `FROM scratch` (or `alpine`) runtime; copy binary only; expose `PORT`; `ENTRYPOINT ["./server"]`
- [ ] T034 [P] Add `slog` structured logging to all handlers — each request logs: handler name, router_id (where applicable), status code, duration
- [ ] T035 [P] Add `Cache-Control: max-age=86400` header to `/static/` file server for static assets
- [ ] T036 [P] Update `CLAUDE.md` between SPECKIT markers to reflect final project structure and `go build`/`go test` commands
- [ ] T037 Run `quickstart.md` validation — follow all steps in `specs/002-stats-dashboard/quickstart.md`; verify binary builds, pings stored, dashboard renders, Docker image builds

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Phase 2 — no dependency on other user stories
- **US2 (Phase 4)**: Depends on Phase 2 and Phase 3 (reads data written by US1)
- **US3 (Phase 5)**: Depends on Phase 2 and Phase 3 (needs ping data), independent of US2
- **US4 (Phase 6)**: Depends on US2 (extends dashboard router selector behavior)
- **Polish (Phase 7)**: Depends on all user story phases being complete

### User Story Dependencies

- **US1 (P1)**: Foundational only — no story deps
- **US2 (P1)**: Foundational + US1 (dashboard reads persisted pings)
- **US3 (P2)**: Foundational + US1; shares DB layer with US2 but can be developed in parallel
- **US4 (P2)**: US2 must be complete (router selector extends dashboard UI)

### Within Each User Story

- DB query functions before handlers (T016 before T017, T019/T020 before T021/T022)
- Handlers before route wiring (T017 before T018)
- HTML template structure before JS logic (T023 before T024)
- Core chart render before interaction events (T024 chart render before date-picker wiring)

### Parallel Opportunities

- T003, T004, T005, T006 can all run in parallel (Phase 1)
- T007–T015 are mostly sequential (DB layer → server bootstrap → handler → route)
- T019 and T020 can run in parallel (different query functions in same file)
- T021, T022 can run in parallel after T019+T020 (different handler functions)
- T027 (WeeklyStats query) can run in parallel with Phase 4 work
- T033, T034, T035, T036 can all run in parallel (Phase 7)

---

## Parallel Example: Phase 4 (Daily Chart)

```bash
# These two query functions can be built simultaneously:
Task T019: DailyStats query in internal/db/queries.go
Task T020: ListRouters query in internal/db/queries.go

# Then these two handlers can be built simultaneously (after T019+T020):
Task T021: GET /api/routers handler
Task T022: GET /api/stats/daily handler

# Template and JS proceed sequentially (T023 → T024 → T025 → T026)
```

---

## Implementation Strategy

### MVP First (US1 Only — Ping Reception)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL)
3. Complete Phase 3: US1 — Ping Reception
4. **VALIDATE**: `curl "localhost:8080/ping?id=test-router"` → 200 OK; row in DB confirmed
5. Deploy/share if ping-collection alone is useful

### Incremental Delivery

1. Setup + Foundational → skeleton builds and starts
2. US1 complete → pings stored; service is useful for data collection alone
3. US2 complete → daily chart dashboard live; P1 scope fully delivered
4. US3 complete → weekly overview added
5. US4 complete → multi-router support in dashboard; full feature delivered
6. Polish → production-ready Dockerfile and logging

### Notes

- `[P]` tasks = different files, no shared unmet dependencies — safe to parallelize
- `[Story]` label maps each task to its user story for traceability
- Each story phase ends with a **Checkpoint** — validate it before moving to the next phase
- Commit after each phase or checkpoint
