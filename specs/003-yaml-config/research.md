# Research: YAML Configuration File

**Feature**: 003-yaml-config
**Date**: 2026-05-17

## Decision 1: YAML Library

**Decision**: `gopkg.in/yaml.v3`

**Rationale**: The de-facto standard Go YAML library. Produces clear, structured
parse errors that include line and column numbers — directly usable in the startup
error message the spec requires (SC-005). Maintained by Canonical, stable API.
Single transitive dependency (none beyond itself for pure parsing).

**Alternatives considered**:
- `github.com/spf13/viper`: Full-featured config framework (env, flags, file, remote).
  Rejected — pulls in ~15 transitive dependencies and far exceeds what this feature
  requires. Constitution Principle IV (Single-Binary Simplicity) prefers minimal deps.
- `encoding/json` with JSON config: Rejected — user explicitly specified YAML format.
- Manual INI/key=value parser: Rejected — harder to represent a list of router IDs
  cleanly without a proper structured format.

---

## Decision 2: Config File Location

**Decision**: `os.Executable()` + `filepath.Dir()` to resolve the binary's own directory.

**Rationale**: The spec says "yaml file in the directory with the service". This is
unambiguous when interpreted as the directory containing the binary. Using the
current working directory (`os.Getwd()`) would break systemd services that set
`WorkingDirectory=/` or Docker containers started from a different path.

**Fallback**: If `os.Executable()` returns an error (rare, e.g. in certain test
environments), fall back to the current working directory with a warning log.

**Alternatives considered**:
- `--config` flag: Out of scope for v1 per spec Assumptions section.
- `$HOME/.config/mikrotik-keepalive/config.yaml`: XDG convention — rejected, the
  spec is explicit about "directory with the service".
- Environment variable `CONFIG_PATH`: Could be a v2 addition; not in scope now.

---

## Decision 3: Allowlist Thread Safety

**Decision**: `sync.RWMutex` wrapping the allowlist slice inside `AllowlistStore`.

**Rationale**: The ping handler runs concurrently (one goroutine per request). The
SIGHUP reload writes a new allowlist. `RLock` during reads (ping handler) allows
unlimited concurrent readers; `Lock` during reload is brief (in-memory slice
replacement). This is the idiomatic Go pattern for shared read-heavy state.

```go
type AllowlistStore struct {
    mu      sync.RWMutex
    allowed map[string]struct{} // set for O(1) lookup
}

func (s *AllowlistStore) IsAllowed(id string) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if len(s.allowed) == 0 { return true } // empty = allow all
    _, ok := s.allowed[id]
    return ok
}

func (s *AllowlistStore) Set(ids []string) {
    m := make(map[string]struct{}, len(ids))
    for _, id := range ids { m[id] = struct{}{} }
    s.mu.Lock()
    s.allowed = m
    s.mu.Unlock()
}
```

**Alternatives considered**:
- `sync/atomic` + `atomic.Value`: Valid but more complex to use correctly with maps.
  RWMutex is simpler and the performance difference is negligible at this scale.
- Channel-based update: Actor pattern — more complex, no benefit here.

---

## Decision 4: Config Precedence

**Decision**: built-in defaults → env vars → config.yaml (highest priority).

**Rationale**: Matches the spec Assumption: "config file takes precedence over env
vars when both are present". Env vars remain useful for Docker deployments where
`config.yaml` is not mounted. Implementation: start with a `defaultConfig()`,
apply env vars, then unmarshal the YAML on top (YAML fields present in the file
override; absent fields keep the env/default value).

**Implementation note**: `yaml.v3` by default only sets fields that are present in
the YAML document, leaving others untouched — this naturally implements the
"override only what's in the file" behaviour.

---

## Decision 5: SIGHUP Handling

**Decision**: `signal.Notify` with a `chan os.Signal`; reload goroutine in `main.go`
calls `config.ReloadAllowlist(path, store)`.

**Rationale**: Standard Unix daemon pattern. Clean, simple, no external dependency.
The reload function re-reads and re-parses `config.yaml`, validates the
`allowed_routers` field, and calls `store.Set(ids)` atomically. If validation
fails, it logs a warning and returns without modifying the store (FR-009).

**What is NOT reloaded**: port and timezone. If these fields differ in the reloaded
file, a warning is logged but they are silently ignored (FR-010). Port cannot be
changed without rebinding the socket; timezone is applied to `time.Local` at
startup and changing it mid-run would cause inconsistent timestamps.
