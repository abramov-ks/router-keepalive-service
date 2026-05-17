<!--
SYNC IMPACT REPORT
==================
Version change: 1.0.0 → 1.1.0 (MINOR)

Bump rationale: User supplied concrete project definition. Principle I reframed from
"sending keepalives" to "receiving pings from MikroTik" — a materially different role.
Technology stack (Go, SQLite) and dashboard requirements added as binding constraints.

Modified principles:
  - I. Reliability First → I. Ping Reception Reliability (reframed: receiver, not sender)
  - III. Observability → III. Data Persistence & Dashboard (the service IS the observability tool)
  - IV. Simplicity → IV. Single-Binary Simplicity (made concrete: Go binary, SQLite only)
  - V. Operational Correctness (unchanged in name, updated in scope)

Added sections:
  - Technology Stack (Go, SQLite, embedded assets — now binding)

Removed sections: None

Templates reviewed:
  - .specify/templates/plan-template.md     ✅ Constitution Check gate is generic; aligns with updated principles
  - .specify/templates/spec-template.md     ✅ No changes required
  - .specify/templates/tasks-template.md    ✅ Phase structure aligns with principle-driven task types
  - .specify/templates/checklist-template.md ✅ No principle-specific references found

Follow-up TODOs: None — all fields resolved.
-->

# MikroTik Keepalive Server Constitution

## Core Principles

### I. Ping Reception Reliability

This service is a **receiver**: MikroTik routers push scheduled HTTP requests to it; the
service does not initiate outbound connections to devices. Every incoming ping request MUST
be acknowledged and persisted within a single request-response cycle. The server MUST NOT
drop pings under normal load (single router, 30-second intervals). If persistence fails,
the error MUST be logged and a 5xx response returned so the MikroTik scheduler can detect
and retry the failure.

### II. MikroTik Protocol Compatibility

The ping endpoint MUST be reachable via MikroTik's built-in `/tool fetch` HTTP client,
which supports only basic HTTP GET and POST with limited header control. The endpoint MUST
require no authentication headers, cookies, or custom request bodies that MikroTik cannot
produce natively. The URL scheme MUST be simple (e.g., `GET /ping` or `GET /ping?device=X`)
so it can be entered directly into a MikroTik scheduler script.

### III. Data Persistence & Dashboard

All received pings MUST be stored in the local SQLite database with at minimum: timestamp,
source identifier, and HTTP status of the reception. The database MUST retain at least
7 days of ping records to support the weekly chart. The web dashboard MUST display:

- The timestamp of the most recent ping received.
- A chart of keepalive events over the last 24 hours.
- A chart of keepalive events over the last 7 days.

Charts MUST be rendered without requiring external CDN resources at runtime (assets MUST be
bundled). The dashboard MUST be accessible from a standard browser with no login required
(local network deployment assumption).

### IV. Single-Binary Simplicity

The service MUST ship as a single self-contained Go binary. SQLite (via an embedded driver)
is the only required runtime dependency — no external database, message broker, or cache.
All web assets (HTML, CSS, JS, chart libraries) MUST be embedded in the binary using Go's
`embed` package. The binary MUST be startable with a single command and zero mandatory flags
(sensible defaults for port and database path MUST be provided).

### V. Operational Correctness

The service MUST validate its configuration (port availability, database writability) at
startup and exit with a descriptive error if validation fails. The database schema MUST be
auto-migrated on first start. The HTTP server MUST respond to a `/health` endpoint with
current status. Structured log output (JSON or logfmt) MUST include: event type, timestamp,
source IP, and outcome for every ping received. Silent failures are not acceptable.

## Technology Stack

- **Language**: Go (latest stable)
- **Database**: SQLite via embedded CGO-free driver (e.g., `modernc.org/sqlite` or `mattn/go-sqlite3`)
- **Web assets**: Embedded via `//go:embed`; chart rendering via a bundled library (e.g., Chart.js)
- **HTTP server**: Go standard library `net/http` or a lightweight router (e.g., `chi`)
- **Deployment target**: Linux server or container (Docker); single binary with no external runtime deps
- **Configuration**: Environment variables with sensible defaults; optional config file as secondary option

Additions to this stack MUST be justified in the feature's plan.md under Complexity Tracking.

## Development Workflow

All changes to the ping reception path MUST be validated with an integration test that
exercises the full HTTP request → SQLite write → read-back cycle. Dashboard chart data
queries MUST be covered by unit tests using an in-memory SQLite instance. Changes that
alter the ping endpoint URL or request format MUST include a note in the spec confirming
MikroTik `/tool fetch` compatibility. The binary MUST build with `go build ./...` and all
tests MUST pass with `go test ./...` before merge. The CLAUDE.md file is the authoritative
runtime development guide for AI-assisted work on this project.

## Governance

This constitution supersedes all other development practices and conventions. Amendments
MUST include: a description of what changed and why, a version bump per the semantic
versioning policy below, and updates to any affected templates or documentation.

**Versioning policy**:
- MAJOR: Removal or redefinition of an existing principle, or backward-incompatible governance change.
- MINOR: New principle or section added, or material expansion of existing guidance.
- PATCH: Clarifications, wording improvements, typo fixes.

All pull requests MUST verify compliance with the principles above before merge. Any
deviation from a principle MUST be documented in plan.md under Complexity Tracking with
explicit justification.

**Version**: 1.1.0 | **Ratified**: 2026-05-17 | **Last Amended**: 2026-05-17
