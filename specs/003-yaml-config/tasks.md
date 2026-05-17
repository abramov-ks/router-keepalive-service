---

description: "Task list for YAML configuration file feature"
---

# Tasks: YAML Configuration File

**Input**: Design documents from `specs/003-yaml-config/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Not requested — no test tasks generated.

**Organization**: Tasks grouped by user story. US1 and US2 are both P1 and can be
developed sequentially; US3 (SIGHUP reload) depends on US2's AllowlistStore.

## Format: `[ID] [P?] [Story?] Description — file path`

- **[P]**: Parallelizable (different files, no unmet dependencies)
- **[Story]**: US1 / US2 / US3
- All file paths relative to repository root

---

## Phase 1: Setup

**Purpose**: Add the one new external dependency.

- [x] T001 Add `gopkg.in/yaml.v3` dependency — run `go get gopkg.in/yaml.v3` and verify `go.mod`/`go.sum` updated

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config package skeleton that both US1 and US2 build on.

**⚠️ CRITICAL**: US1 and US2 cannot start until this phase is complete.

- [x] T002 Create `internal/config/` package — create `internal/config/config.go` with `Config` struct (`Port int \`yaml:"port"\``, `Timezone string \`yaml:"timezone"\``, `AllowedRouters []string \`yaml:"allowed_routers"\``) and `defaultConfig() Config` returning `Config{Port: 8080, Timezone: "UTC"}`
- [x] T003 Implement `Validate(c Config) error` in `internal/config/config.go` — check port in range 1–65535 (`config: port N is out of range (1–65535)`); validate timezone via `time.LoadLocation` (`config: timezone "X" is invalid: <err>`); validate each `AllowedRouters` entry against `^[a-zA-Z0-9_-]{1,64}$` (`config: allowed_routers[i] "X" contains invalid characters`)
- [x] T004 Implement `Load(binDir string) (Config, error)` in `internal/config/config.go` — build path `filepath.Join(binDir, "config.yaml")`; if file does not exist log `slog.Info("no config.yaml found, using defaults")` and return `defaultConfig()`; read file; `yaml.Unmarshal` into a `Config` initialised from `defaultConfig()` so absent keys keep defaults; call `Validate`; return error with prefix `config: cannot parse config.yaml: <yaml error>` on parse failure

**Checkpoint**: `go build ./...` succeeds after Phase 2.

---

## Phase 3: User Story 1 — YAML File Loading at Startup (Priority: P1) 🎯 MVP

**Goal**: Service reads `config.yaml` on startup, applies port and timezone, exits
with a descriptive error on invalid config. No config file → defaults, starts normally.

**Independent Test**: Create `config.yaml` with `port: 9090` and `timezone: Europe/Moscow`
in the binary's directory. Run `./server` — it MUST listen on 9090 and log timestamps in
Moscow time. Delete the file — run again and it MUST use port 8080 and UTC. Create a file
with `port: 99999` — run and it MUST exit with an error message naming the bad value.

### Implementation for User Story 1

- [x] T005 [US1] Wire config loading in `main.go` — find binary dir: `exe, _ := os.Executable(); binDir := filepath.Dir(exe)`; call `config.Load(binDir)`; on error `slog.Error` + `os.Exit(1)`; replace hardcoded default port `"8080"` with `strconv.Itoa(cfg.Port)`; replace `TZ` env-var timezone block with config-based timezone: `time.LoadLocation(cfg.Timezone)` + `time.Local = loc` (this supersedes the previous env-var only approach; env var `TZ` remains a fallback when `timezone` absent from config)

**Checkpoint**: Binary reads config.yaml; wrong port/timezone exit with clear error; missing file uses defaults.

---

## Phase 4: User Story 2 — Router Allowlist Enforcement (Priority: P1)

**Goal**: When `allowed_routers` is non-empty in config, pings from unlisted router IDs
receive HTTP 403 and are never written to the database. Empty allowlist = allow all (existing behaviour).

**Independent Test**: Start server with `config.yaml` containing `allowed_routers: [router-A]`.
`curl localhost:<port>/ping?id=router-A` → 200 OK. `curl localhost:<port>/ping?id=router-B` → 403.
Check DB — only router-A row exists. Remove `allowed_routers` from config, restart — both IDs accepted.

### Implementation for User Story 2

- [x] T006 [US2] Create `AllowlistStore` in `internal/config/reload.go` — struct with `mu sync.RWMutex` and `allowed map[string]struct{}`; method `IsAllowed(id string) bool` (if `len(allowed)==0` return `true`; else RLock, lookup, RUnlock); method `Set(ids []string)` (build new map, Lock, replace, Unlock); constructor `NewAllowlistStore(ids []string) *AllowlistStore`
- [x] T007 [US2] Update `PingHandler` signature in `internal/handler/ping.go` — add `store *config.AllowlistStore` parameter; after format validation (existing 400 check) and before `StorePing`, call `store.IsAllowed(routerID)`; if false: `slog.Warn("ping rejected: router not in allowlist", "router_id", routerID, "remote", sourceIP)` + `http.Error(w, "router not allowed", http.StatusForbidden)`; return without storing
- [x] T008 [US2] Wire allowlist in `main.go` — after `config.Load`: `store := config.NewAllowlistStore(cfg.AllowedRouters)`; pass `store` to `handler.PingHandler(database, store)`; update the route wiring accordingly

**Checkpoint**: 403 returned for unlisted routers; 0 DB rows for blocked IDs; unlisted ID logged. Empty allowlist accepts all.

---

## Phase 5: User Story 3 — SIGHUP Live Reload (Priority: P2)

**Goal**: Sending SIGHUP to the running process re-reads `allowed_routers` from
`config.yaml` and applies it atomically without restarting or dropping in-flight requests.
A bad config file during reload keeps the previous allowlist. Port/timezone changes during
reload are ignored with a warning log.

**Independent Test**: Start server with `allowed_routers: [router-A]`. Confirm router-B
is blocked (403). Add `router-B` to `config.yaml`. Send `kill -HUP <pid>`. Confirm
router-B now returns 200 — without restarting the server.

### Implementation for User Story 3

- [x] T009 [US3] Implement `StartReloadHandler(configPath string, store *AllowlistStore)` in `internal/config/reload.go` — `sigCh := make(chan os.Signal, 1); signal.Notify(sigCh, syscall.SIGHUP)`; start goroutine: on each signal read the file at `configPath`, yaml.Unmarshal into a temp Config (using `defaultConfig()` as base), validate only the `allowed_routers` field (log WARN + return on error without touching `store`), call `store.Set(newCfg.AllowedRouters)`, log `slog.Info("allowlist reloaded", "count", len(newCfg.AllowedRouters))`; if `newCfg.Port != originalPort` log `slog.Warn("port change ignored on reload — restart required")`; same for timezone
- [x] T010 [US3] Wire `StartReloadHandler` in `main.go` — after `store` is initialised: `configPath := filepath.Join(binDir, "config.yaml"); config.StartReloadHandler(configPath, store)` — pass original port and timezone values so the reload handler can detect and log ignored changes

**Checkpoint**: `kill -HUP $(pgrep server)` reloads allowlist. Bad config on reload keeps old allowlist. Port/timezone changes logged and ignored.

---

## Phase 6: Polish & Validation

- [x] T011 [P] Run `go build ./...` and `go vet ./...` — must produce no errors or warnings
- [x] T012 [P] Run `go mod tidy` — confirm `go.sum` is up to date and no unused deps remain
- [x] T013 Smoke test per `specs/003-yaml-config/contracts/config-file.md` — create each scenario from the Behaviour Matrix table and verify the described outcome

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Phase 1 (yaml.v3 must be available)
- **US1 (Phase 3)**: Depends on Phase 2 (needs `config.Load`)
- **US2 (Phase 4)**: Depends on Phase 2 (needs `AllowlistStore`) and US1 (config already wired in main.go)
- **US3 (Phase 5)**: Depends on US2 (`AllowlistStore` must exist and be wired)
- **Polish (Phase 6)**: Depends on all story phases

### Within Each User Story

- T002 before T003+T004 (struct before validation and loading)
- T003 and T004 can run in parallel after T002
- T006 before T007+T008 (AllowlistStore before handler update and wiring)
- T007 and T008 can be developed together (T008 wires what T007 requires)
- T009 before T010 (implement before wiring)

### Parallel Opportunities

- T003 and T004 can run in parallel after T002 (both in `config.go`, but distinct functions)
- T011 and T012 can run in parallel (Phase 6)

---

## Implementation Strategy

### MVP First (US1 + US2 — Config loading + Allowlist)

1. Phase 1: Add yaml.v3
2. Phase 2: Config package skeleton
3. Phase 3: Wire config at startup (port, timezone)
4. Phase 4: Allowlist enforcement
5. **VALIDATE**: Config file controls port; unlisted routers get 403

### Full Feature

1. MVP above
2. Phase 5: SIGHUP reload
3. Phase 6: Polish

### Notes

- `[P]` = different files or fully independent functions — safe to work simultaneously
- Each phase ends with a **Checkpoint** — verify before moving on
- The existing env var `TZ` is superseded by `config.yaml`'s `timezone` field (config > env > default)
- No database schema changes — this feature is entirely in-memory config + HTTP layer
