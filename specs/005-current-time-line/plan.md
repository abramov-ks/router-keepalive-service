# Implementation Plan: Dashboard Enhancements (004 + 005)

**Features**: 004-last-24h-view + 005-current-time-line
**Date**: 2026-05-18

## Tech Stack

- **Backend**: Go 1.22+, `modernc.org/sqlite`, `github.com/go-chi/chi/v5`
- **Frontend**: Vanilla JS, Chart.js v4 (already bundled), `html/template` server-side rendering
- **Embed**: `//go:embed web/templates`, `//go:embed web/static`

## Files to Modify

| File | Change |
|------|--------|
| `internal/db/queries.go` | Add `Last24hStats()` query |
| `internal/handler/dashboard.go` | Add `Last24hStatsHandler`, inject `TzOffsetMinutes` into template data |
| `main.go` | Wire `/api/stats/last24h` route |
| `web/templates/dashboard.html` | Add "Last 24h" button + canvas, update JS view logic, add current-time line |

## New API Endpoint

`GET /api/stats/last24h?router_id=<id>`

Returns same structure as `/api/stats/daily`: `{"buckets": [{time, count}, ...]}`  
1440 entries covering `[now − 24h, now]` at 1-minute resolution, times in server local timezone.

## Current Time Line Approach

Server injects `window.serverTzOffset` (UTC offset in minutes) into the template.
Chart.js `afterDraw` custom plugin reads this to compute current time position on the X axis
and draws a dashed vertical line using Canvas 2D API — no new JS dependencies required.

## Constitution Check

- Single-binary: no new dependencies ✅
- MikroTik protocol: unchanged ✅
- Data persistence: no schema changes ✅
- Operational correctness: no new failure modes ✅
