# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

<!-- SPECKIT START -->
For additional context about the feature currently in development, read the
current plan at specs/006-telegram-alerts/plan.md
<!-- SPECKIT END -->

## Project

Single-binary Go service that receives keepalive HTTP pings from MikroTik routers (`GET /ping?id=<router_id>`), stores them in SQLite, and serves a web dashboard with uptime charts. See README.md for the full endpoint list and deployment instructions.

## Commands

- Build: `go build -o server .`
- Run: `./server` (dashboard at http://localhost:8080)
- Lint: `go vet ./...`
- Test: `go test ./...` — run a single package with `go test ./internal/db/`, a single test with `go test -run TestName ./internal/db/` (no test files exist yet)
- Linux cross-compile: `GOOS=linux GOARCH=amd64 go build -o server-linux .`

## Configuration

- `config.yaml` is loaded from the **binary's directory** (via `os.Executable()`), not the working directory. All keys optional: `port` (default 8080), `timezone` (IANA name, default UTC), `allowed_routers` (empty = allow all), `telegram` (outage alerts, opt-in — see below). Invalid config exits at startup.
- Env vars: `DB_PATH` (default `keepalive.db`), `BASE_PATH` (reverse-proxy prefix, stripped and injected into the template). Port and timezone are **not** env vars — they live in `config.yaml`.
- `SIGHUP` reloads only the router allowlist (`internal/config/reload.go`); port/timezone/telegram changes log a warning and require restart.
- Optional `telegram:` block (`bot_token`, `chat_id`, `threshold_seconds` default 300, `routers`, `messages`, `router_messages`) enables outage alerting; `Config.Telegram` is nil when absent and the monitor is never constructed. Schema and validation matrix: `specs/006-telegram-alerts/contracts/config-schema.md`.

## Architecture

Request flow: `main.go` (wiring, chi router) → `internal/handler` (HTTP layer) → `internal/db` (SQL queries) → `internal/model` (plain structs). Handlers are constructor functions (`PingHandler(db, store)`) returning `http.HandlerFunc` — dependencies are passed explicitly, no globals.

- **Storage**: one table, `ping_events`, schema created idempotently in `internal/db/db.go` (`Migrate`). SQLite via `modernc.org/sqlite` (pure Go, no CGO). WAL mode plus `SetMaxOpenConns(1)` to serialize writes. Timestamps are stored in **UTC** (`datetime('now')`). Retention: rows older than 7 days deleted at startup and every 24h (`internal/db/cleanup.go`).
- **Timezone handling** (the trickiest part): `main.go` sets process-wide `time.Local` from config. Queries in `internal/db/queries.go` convert stored UTC to local inside SQL using `sqliteTZOffset()`, which builds a `"+3 hours"`-style modifier from the current `time.Local` offset. The dashboard template receives `TzOffsetMinutes` so client-side JS aligns with server-local time. Any new time-bucketed query must apply the same offset conversion.
- **Stats contract**: bucket queries zero-fill and return fixed-length slices — `DailyStats` and `Last24hStats` always 1440 one-minute points, `WeeklyStats` always 7 days. The frontend charts rely on this.
- **Allowlist**: `config.AllowlistStore` (RWMutex-guarded set; empty set = allow everything). Router IDs are validated against `^[a-zA-Z0-9_-]{1,64}$` in both config validation and the ping handler.
- **Frontend**: no build step. Single Go template `web/templates/dashboard.html` with inline JS, vendored Chart.js in `web/static/`. Both directories are embedded with `go:embed` in `main.go`, so the binary is self-contained — new assets must be under `web/` to be picked up.
- **Telegram alerts** (`internal/monitor` + `internal/notify`): a background goroutine polls `LastSeenByRouter` every 10 s and drives a per-router state machine (UP → DOWN on silence > N, one loss message per outage; DOWN → RECOVERING → UP after N seconds of stable pings, then one recovery message; flapping coalesces into a single outage). Startup seeding clamps last-seen to service start so the server's own downtime never causes instant false alerts. `internal/notify` posts to the Telegram Bot API with stdlib HTTP (10 s timeout, 3 attempts, then drop + ERROR log) — delivery failures must never touch the ping path. Both packages take dependencies via constructor injection; the monitor's clock and DB query are injectable fields used by tests.

## Constraints (from .specify/memory/constitution.md)

- The ping endpoint must stay callable by MikroTik's `/tool fetch`: plain GET, no auth headers or cookies. Persistence failures must return 5xx so the router's scheduler can detect them.
- Keep it a single self-contained binary: Go + SQLite only, assets embedded, no external services.

## Spec-driven workflow

Features are developed with Spec Kit: each lives in `specs/NNN-slug/` (spec.md, plan.md, tasks.md), driven by the `speckit-*` skills (specify → plan → tasks → implement).
