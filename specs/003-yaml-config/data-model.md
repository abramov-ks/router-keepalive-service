# Data Model: YAML Configuration File

**Feature**: 003-yaml-config
**Date**: 2026-05-17

## Config (in-memory, loaded from config.yaml)

Represents the full set of runtime settings. Loaded once at startup; the
`AllowedRouters` field may be updated in-place via SIGHUP reload.

| Field | Go type | YAML key | Default | Validation |
|-------|---------|----------|---------|------------|
| `Port` | `int` | `port` | `8080` | 1–65535 |
| `Timezone` | `string` | `timezone` | `"UTC"` | Valid IANA name (`time.LoadLocation`) |
| `AllowedRouters` | `[]string` | `allowed_routers` | `[]` (allow all) | Each entry: `^[a-zA-Z0-9_-]{1,64}$` |

## AllowlistStore (runtime, thread-safe)

A wrapper around the `AllowedRouters` list that is safe to read and write
concurrently. Separate from `Config` so the reload path can update only the
allowlist without touching the rest of the config.

| Field | Type | Description |
|-------|------|-------------|
| `mu` | `sync.RWMutex` | Guards `allowed` |
| `allowed` | `map[string]struct{}` | Set of permitted router IDs; nil/empty = allow all |

**Methods**:
- `IsAllowed(id string) bool` — O(1) lookup; returns `true` if `allowed` is empty
- `Set(ids []string)` — atomically replaces the allowlist

## config.yaml file format

```yaml
port: 8080
timezone: Europe/Moscow
allowed_routers:
  - office-router-1
  - branch-router-2
```

All keys are optional. Missing keys retain their defaults. An empty file is valid.

## Validation Rules

| Setting | Rule | Error message pattern |
|---------|------|-----------------------|
| `port` | Integer 1–65535 | `config: port N is out of range (1–65535)` |
| `timezone` | Valid IANA name | `config: timezone "X" is invalid: <time.LoadLocation error>` |
| `allowed_routers[i]` | Matches `^[a-zA-Z0-9_-]{1,64}$` | `config: allowed_routers[i] "X" contains invalid characters` |
| File syntax | Valid YAML | `config: cannot parse config.yaml: <yaml.v3 error with line number>` |
| File missing | Not an error | Service starts with defaults (logged at INFO level) |
| File empty | Not an error | All defaults applied |

## Reload Scope

| Setting | Reloadable via SIGHUP |
|---------|-----------------------|
| `port` | No — logged and ignored |
| `timezone` | No — logged and ignored |
| `allowed_routers` | Yes — applied atomically |
