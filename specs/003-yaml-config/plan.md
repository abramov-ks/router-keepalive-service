# Implementation Plan: YAML Configuration File

**Branch**: `003-yaml-config` | **Date**: 2026-05-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-yaml-config/spec.md`

## Summary

Add a `config.yaml` file that controls service port, timezone, and an optional
allowlist of permitted router IDs. The allowlist is enforced in the ping handler
(403 for unlisted routers). The allowlist can be reloaded at runtime via SIGHUP
without restarting the service. Port and timezone are startup-only settings.

## Technical Context

**Language/Version**: Go 1.22+

**Primary Dependencies**:
- `gopkg.in/yaml.v3` — YAML parsing (one new dependency)
- `os/signal`, `syscall` — SIGHUP handling (stdlib)
- `sync` — `sync.RWMutex` for thread-safe allowlist access (stdlib)

**Storage**: No change — SQLite as before

**Testing**: `go test ./...` — unit tests for config loading and validation

**Target Platform**: Linux server / Docker container (SIGHUP available; Windows out of scope)

**Project Type**: Enhancement to existing web service

**Performance Goals**: Allowlist check adds < 1µs per ping request (RLock on a small slice)

**Constraints**: Config file found relative to binary location via `os.Executable()`,
not CWD; single new external dependency only (`yaml.v3`)

**Scale/Scope**: Config struct with 3 fields; allowlist up to ~100 entries

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| I. Ping Reception Reliability | RWMutex RLock during ping — no blocking; SIGHUP reload holds Lock < 1ms | ✅ Pass |
| II. MikroTik Protocol Compatibility | Ping endpoint URL/format unchanged; only new 403 for unlisted routers | ✅ Pass |
| III. Data Persistence & Dashboard | No storage changes; dashboard unaffected | ✅ Pass |
| IV. Single-Binary Simplicity | One new dep (`yaml.v3`); config file optional; binary still starts with zero config | ✅ Pass |
| V. Operational Correctness | Full validation at startup, descriptive errors, graceful reload failure | ✅ Pass |

## Project Structure

### Documentation (this feature)

```text
specs/003-yaml-config/
├── spec.md
├── plan.md              # This file
├── research.md
├── data-model.md
├── contracts/
│   ├── config-file.md
│   └── ping-403.md
└── tasks.md             # /speckit-tasks output
```

### Source Code changes

```text
internal/config/
├── config.go      # Config struct, Load(), Validate(), defaults
└── reload.go      # AllowlistStore (RWMutex wrapper), StartReloadHandler()

main.go            # Wire config loading; pass AllowlistStore to PingHandler
internal/handler/
└── ping.go        # Add allowlist check before StorePing (403 if blocked)
```

No new top-level directories. All other existing files unchanged.

**Structure Decision**: Single `internal/config` package owns all config concerns.
`AllowlistStore` is a thread-safe value passed by pointer to the ping handler,
keeping the handler stateless with respect to config.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| New dep: `gopkg.in/yaml.v3` | User explicitly requires YAML format | stdlib has no YAML parser; hand-rolling one is far more complex |
