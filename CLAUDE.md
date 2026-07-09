# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

<!-- SPECKIT START -->
For additional context about the feature currently in development, read the
newest plan under specs/ (highest-numbered directory).
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

- `config.yaml` is loaded from the **binary's directory** (via `os.Executable()`), not the working directory. All keys optional: `port` (default 8080), `timezone` (IANA name, default UTC), `allowed_routers` (empty = allow all). Invalid config exits at startup.
- Env vars: `DB_PATH` (default `keepalive.db`), `BASE_PATH` (reverse-proxy prefix, stripped and injected into the template). Port and timezone are **not** env vars — they live in `config.yaml`.
- `SIGHUP` reloads only the router allowlist (`internal/config/reload.go`); port/timezone changes log a warning and require restart.

## Architecture

Request flow: `main.go` (wiring, chi router) → `internal/handler` (HTTP layer) → `internal/db` (SQL queries) → `internal/model` (plain structs). Handlers are constructor functions (`PingHandler(db, store)`) returning `http.HandlerFunc` — dependencies are passed explicitly, no globals.

- **Storage**: one table, `ping_events`, schema created idempotently in `internal/db/db.go` (`Migrate`). SQLite via `modernc.org/sqlite` (pure Go, no CGO). WAL mode plus `SetMaxOpenConns(1)` to serialize writes. Timestamps are stored in **UTC** (`datetime('now')`). Retention: rows older than 7 days deleted at startup and every 24h (`internal/db/cleanup.go`).
- **Timezone handling** (the trickiest part): `main.go` sets process-wide `time.Local` from config. Queries in `internal/db/queries.go` convert stored UTC to local inside SQL using `sqliteTZOffset()`, which builds a `"+3 hours"`-style modifier from the current `time.Local` offset. The dashboard template receives `TzOffsetMinutes` so client-side JS aligns with server-local time. Any new time-bucketed query must apply the same offset conversion.
- **Stats contract**: bucket queries zero-fill and return fixed-length slices — `DailyStats` and `Last24hStats` always 1440 one-minute points, `WeeklyStats` always 7 days. The frontend charts rely on this.
- **Allowlist**: `config.AllowlistStore` (RWMutex-guarded set; empty set = allow everything). Router IDs are validated against `^[a-zA-Z0-9_-]{1,64}$` in both config validation and the ping handler.
- **Frontend**: no build step. Single Go template `web/templates/dashboard.html` with inline JS, vendored Chart.js in `web/static/`. Both directories are embedded with `go:embed` in `main.go`, so the binary is self-contained — new assets must be under `web/` to be picked up.

## Constraints (from .specify/memory/constitution.md)

- The ping endpoint must stay callable by MikroTik's `/tool fetch`: plain GET, no auth headers or cookies. Persistence failures must return 5xx so the router's scheduler can detect them.
- Keep it a single self-contained binary: Go + SQLite only, assets embedded, no external services.

## Spec-driven workflow

Features are developed with Spec Kit: each lives in `specs/NNN-slug/` (spec.md, plan.md, tasks.md), driven by the `speckit-*` skills (specify → plan → tasks → implement).
