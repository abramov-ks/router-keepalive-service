# Implementation Plan: Telegram Alerts on Ping Loss and Recovery

**Branch**: `006-telegram-alerts` | **Date**: 2026-07-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/006-telegram-alerts/spec.md`

## Summary

Add opt-in Telegram notifications: when a monitored router sends no pings for more than N seconds, post a loss message (e.g., «Кажется, пропал пинг с дачи») to the operator's Telegram channel; once pings resume and hold for N seconds (stability window), post a recovery message (e.g., «Кажется, пинг вернулся»). Implemented as a background monitor goroutine that polls the existing `ping_events` table every 10 seconds and drives a per-router state machine, plus a minimal Telegram Bot API client built on the standard library (`net/http`) — no new dependencies, no schema changes, zero impact on the ping reception path.

## Technical Context

**Language/Version**: Go 1.25 (existing)

**Primary Dependencies**: existing only — `go-chi/chi` (HTTP), `gopkg.in/yaml.v3` (config), `modernc.org/sqlite` (storage). Telegram Bot API is called with stdlib `net/http` + `encoding/json`; **no new module dependencies**.

**Storage**: existing SQLite `ping_events` table, read-only for this feature (per-router `MAX(received_at)` polling). No schema changes, no new tables — alert state is in-memory per service run (per spec assumption).

**Testing**: `go test ./...`; monitor state machine unit tests with injected clock; Telegram client tests against `httptest.Server`; config validation table tests; seed-query test on in-memory SQLite (`:memory:`).

**Target Platform**: Linux server / Docker container (existing deployment)

**Project Type**: single web service (existing single-binary layout)

**Performance Goals**: detection granularity ≤ 10 s (SC-001 allows N + 60 s); one lightweight aggregate SQL query per tick; notification sending fully off the request path.

**Constraints**: ping reception must be unaffected by Telegram availability (constitution I / FR-009); single self-contained binary, stdlib-only addition (constitution IV); alerting strictly opt-in — absent `telegram:` config block ⇒ behavior identical to today (FR-007).

**Scale/Scope**: a handful of routers (1–10), one Telegram channel, one background goroutine.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Assessment |
|---|-----------|------------|
| I | Ping Reception Reliability | **PASS** — monitor is a separate goroutine reading the DB; ping handler is untouched. Telegram send failures are logged + retried and never propagate to the request path. Outbound HTTP goes to Telegram (a notification service), not to devices; the service remains a pure receiver toward routers. |
| II | MikroTik Protocol Compatibility | **PASS** — no change to `/ping` URL, method, or response codes. |
| III | Data Persistence & Dashboard | **PASS** — no schema or dashboard changes; feature only reads `ping_events`. |
| IV | Single-Binary Simplicity | **PASS** — no new dependencies (stdlib HTTP client), nothing external beyond the Telegram API the user explicitly requested; binary stays self-contained, zero mandatory flags (feature opt-in via config). |
| V | Operational Correctness | **PASS** — `telegram:` config block validated at startup with descriptive exit errors (FR-008); every state transition and send outcome logged via `slog` (structured); no silent failures. |
| — | Technology Stack | **PASS** — Go + SQLite + existing config.yaml pattern; no stack additions to justify. |
| — | Development Workflow | **PASS** — ping path untouched (no new integration test mandated there); new monitor/config/notify logic gets unit tests; seed query covered on in-memory SQLite per workflow rule. |

**Post-design re-check (after Phase 1)**: PASS — design introduces two internal packages (`internal/monitor`, `internal/notify`), both stdlib-only; no Complexity Tracking entries needed.

## Project Structure

### Documentation (this feature)

```text
specs/006-telegram-alerts/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   ├── config-schema.md
│   └── telegram-notifications.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
main.go                        # wire-up: build notifier + monitor, start monitor goroutine
internal/
├── config/
│   ├── config.go              # extended: Telegram struct, defaults, validation
│   └── config_test.go         # NEW: validation table tests
├── monitor/                   # NEW package: outage detection state machine
│   ├── monitor.go             # per-router state, tick loop, DB polling, alert decisions
│   └── monitor_test.go        # NEW: state-machine tests with injected clock + fake notifier
├── notify/                    # NEW package: Telegram delivery
│   ├── telegram.go            # sendMessage client (stdlib), timeout, bounded retries
│   └── telegram_test.go       # NEW: httptest-based delivery/retry tests
├── db/
│   ├── queries.go             # extended: LastSeenByRouter() aggregate query
│   └── queries_test.go        # NEW: seed-query test on :memory: SQLite
├── handler/                   # unchanged
└── model/                     # unchanged
web/                           # unchanged
```

**Structure Decision**: keep the existing single-project layout and its layering (`main.go` wiring → `internal/*` packages with explicit dependency injection). Two new leaf packages: `internal/monitor` (detection logic, testable with a fake clock and fake notifier interface) and `internal/notify` (Telegram transport). `internal/monitor` depends on `internal/db` (read) and a `Notifier` interface implemented by `internal/notify` — mirroring the handler-constructor pattern already used in the codebase.

## Complexity Tracking

No constitution violations — table intentionally empty.
