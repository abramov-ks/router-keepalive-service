# Contract: config.yaml File Format

**Location**: Same directory as the running binary (`config.yaml`)

**Format**: YAML (UTF-8, any line endings)

## Schema

```yaml
# All keys are optional. Absent keys use built-in defaults.

port: 8080                  # integer, 1–65535 (default: 8080)
timezone: UTC               # IANA timezone name (default: UTC)
allowed_routers:            # list of permitted router IDs (default: [] = allow all)
  - office-router-1
  - branch-router-2
```

## Field Definitions

### `port`
- Type: integer
- Default: `8080`
- Valid range: 1–65535
- Effect: TCP port the HTTP server binds to. Startup-only; not reloadable.

### `timezone`
- Type: string (IANA timezone database name)
- Default: `"UTC"`
- Examples: `"Europe/Moscow"`, `"America/New_York"`, `"Asia/Tokyo"`
- Effect: Applied to `time.Local` at startup. Controls dashboard timestamps and
  log date formatting. Startup-only; not reloadable.
- Reference: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones

### `allowed_routers`
- Type: list of strings
- Default: `[]` (empty list = no restriction, all valid router IDs accepted)
- Each entry: must match `^[a-zA-Z0-9_-]{1,64}$`
- Effect: When non-empty, only router IDs in this list may submit pings. Reloadable
  at runtime via SIGHUP.

## Behaviour Matrix

| File state | Outcome |
|------------|---------|
| File absent | Service starts with built-in defaults; logs INFO "no config.yaml found, using defaults" |
| File present, empty | Service starts with built-in defaults |
| File present, valid | Settings from file applied; absent keys use defaults |
| File present, YAML syntax error | Service exits with error: `config: cannot parse config.yaml: <detail>` |
| File present, invalid field value | Service exits with error identifying the field and value |

## Reload Behaviour (SIGHUP)

On receiving SIGHUP, the service re-reads `config.yaml` and applies only the
`allowed_routers` field. Port and timezone changes are detected and logged as
warnings but not applied.

If the file cannot be read or parsed during reload, the previous `allowed_routers`
list remains active and a warning is logged. The service does NOT crash.

## Minimal Example

```yaml
allowed_routers:
  - my-router-1
```

This is a valid config that only sets the allowlist and leaves port/timezone at defaults.
