---
description: "Task list for Last 24h View + Current Time Line features (004 + 005)"
---

# Tasks: Dashboard Enhancements (004 + 005)

**Input**: specs/004-last-24h-view/spec.md + specs/005-current-time-line/spec.md

**Prerequisites**: plan.md ✅, spec.md ✅

**Tests**: Not requested — no test tasks generated.

**Note**: Features 004 and 005 share the same modified files and are implemented together.

## Format: `[ID] [P?] [Story?] Description — file path`

---

## Phase 1: Setup

_No new dependencies or infrastructure required._

---

## Phase 2: Foundational — Last 24h API Endpoint

**Purpose**: New backend endpoint that serves the rolling 24-hour window. All frontend tasks depend on it.

- [x] T001 Add `Last24hStats(database *sql.DB, routerID string) ([]model.BucketPoint, error)` to `internal/db/queries.go` — compute window as `[time.Now().Add(-24*time.Hour), time.Now()]` in local timezone; apply `sqliteTZOffset()` to SQLite datetime(); return 1440 BucketPoints (zero-filled) with keys formatted as `"2006-01-02T15:04:05"` in local time; reuse same pattern as `DailyStats`
- [x] T002 Add `Last24hStatsHandler(database *sql.DB) http.HandlerFunc` to `internal/handler/dashboard.go` — read `router_id` query param; return 404 if router unknown (call `db.RouterExists`); call `db.Last24hStats`; return JSON `{"buckets": [...]}` with same structure as daily stats
- [x] T003 Wire route `r.Get("/api/stats/last24h", handler.Last24hStatsHandler(database))` in `main.go` after the existing `/api/stats/weekly` route

**Checkpoint**: `curl 'localhost:8080/api/stats/last24h?router_id=X'` returns 1440 bucket objects.

---

## Phase 3: User Story 1 (004) — Last 24h View UI

**Goal**: "Last 24h" button appears in toggle group, is the default view, renders a line chart of the rolling 24h window, disables the date picker.

**Independent Test**: Open dashboard with no `?view=` param → "Last 24h" button is active, chart shows 1440 minute buckets, date picker is disabled.

- [x] T004 [US1] Add server timezone offset injection to dashboard template data in `internal/handler/dashboard.go` — add `TzOffsetMinutes int` field to the anonymous template data struct; compute with `_, secs := time.Now().Zone(); data.TzOffsetMinutes = secs / 60`
- [x] T005 [US1] Update `web/templates/dashboard.html` — add "Last 24h" button to the toggle group before "Day": `<button id="btn-24h" class="toggle-btn" onclick="switchView('24h')">Last 24h</button>`; add `window.serverTzOffset = {{.TzOffsetMinutes}};` in the existing `<script>` block before the IIFE; add a third `<canvas id="last24h-chart">` wrapped in a card div (initially hidden)

- [x] T006 [US1] Update JS in `web/templates/dashboard.html` — extend `switchView(view)` to handle `'24h'` case: show `last24h-chart` canvas, hide daily and weekly canvases, disable date picker, set `btn-24h` active; update `updateURL()` to include `view=24h` param; change default view from `'day'` to `'24h'` in init (when no `?view=` param); add `renderLast24h()` async function that fetches `/api/stats/last24h?router_id=...`, renders a line chart on `last24h-chart` canvas with same style as `renderDaily()` (green line, cubic interpolation, fill); wire it in `switchView('24h')` and on initial router load

**Checkpoint**: Default view is "Last 24h", chart loads data, switching to Day/Week and back works, URL reflects `?view=24h`.

---

## Phase 4: User Story 2 (004) — Navigation Between Three Views

**Goal**: All three toggle buttons work correctly; date picker state is consistent.

**Independent Test**: Click Day → date picker enabled; click Week → disabled; click Last 24h → disabled.

- [x] T007 [US2] Update `switchView` and date picker logic in `web/templates/dashboard.html` — ensure date picker is enabled only in `'day'` mode and disabled in both `'24h'` and `'week'` modes; verify `?view=day`, `?view=week`, `?view=24h` URL params are correctly read on page load and route to the right view; verify that switching from `'24h'` to `'day'` defaults to today's date

**Checkpoint**: All three toggle buttons render correct charts; date picker enabled only in Day view; URL round-trips correctly.

---

## Phase 5: User Story 1 (005) — Current Time Line on Daily and Last 24h Charts

**Goal**: Vertical dashed line at current time on daily chart (today only) and last-24h chart (always).

**Independent Test**: Open today's daily chart → dashed vertical line at current HH:MM position. Open past date → no line. Open Last 24h → line at rightmost position.

- [x] T008 [US1] Add `nowMinutes()` helper and daily chart current-time line in `web/templates/dashboard.html` — add helper function `function nowMinutes() { const now = new Date(); return (now.getUTCHours() * 60 + now.getUTCMinutes() + window.serverTzOffset) % 1440; }` in the IIFE; in `renderDaily()`, after chart creation, call a `drawNowLine(chart, index)` function only when `currentDate === todayStr()`; `drawNowLine` uses `chart.ctx`, `chart.chartArea`, and `chart.scales.x.getPixelForIndex(index)` to draw a vertical dashed line (strokeStyle: `'rgba(220,50,50,0.8)'`, lineWidth: 1.5, setLineDash: `[4,4]`)

- [x] T009 [US1] Add current-time line to Last 24h chart in `web/templates/dashboard.html` — in `renderLast24h()`, after chart creation, call `drawNowLine(chart, 1439)` since "now" is always the last bucket (index 1439) in the 24h window

**Checkpoint**: Dashed red line visible at current minute on today's daily chart; at far right on last-24h chart; absent on past dates.

---

## Phase 6: User Story 2 (005) — Current Time Line on Weekly Chart

**Goal**: Vertical dashed line at today's date column on the weekly chart.

**Independent Test**: Open weekly chart → dashed line at today's date position.

- [x] T010 [US2] Add current-day line to weekly chart in `web/templates/dashboard.html` — in `renderWeekly()`, after chart creation, find the index of today's date string in `data.days` array; if found, call `drawNowLine(chart, idx)`

**Checkpoint**: Dashed line visible at today's column on the weekly chart.

---

## Phase 7: Polish & Validation

- [x] T011 [P] Run `go build ./...` and `go vet ./...` — must produce no errors
- [x] T012 [P] Smoke test: verify `/api/stats/last24h` returns 1440 buckets; verify all three views load and switch correctly; verify current-time lines appear and disappear correctly for past dates

---

## Dependencies & Execution Order

- T001 → T002 → T003 (backend endpoint chain)
- T003 must complete before T006 (JS needs working `/api/stats/last24h`)
- T004 → T005 → T006 → T007 (dashboard template/JS chain)
- T008 depends on T005 (needs `window.serverTzOffset` injected)
- T009 depends on T006 (needs `renderLast24h` to exist)
- T010 depends on T007 (needs `renderWeekly` to be finalized)
- T011, T012 after all implementation tasks

## Parallel Opportunities

- T001, T004, T005 can start in parallel (different files or independent additions)
- T011 and T012 can run in parallel (Phase 7)

## Implementation Strategy

### MVP (Phase 2 + Phase 3)
1. Add `Last24hStats` query and handler
2. Wire route
3. Add toggle button and JS render function
4. **Validate**: Default view is Last 24h, chart shows correct data

### Full Feature
1. MVP above
2. Phase 4: Navigation polish
3. Phase 5–6: Current time lines
4. Phase 7: Polish
