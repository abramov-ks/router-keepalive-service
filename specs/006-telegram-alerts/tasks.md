# Tasks: Telegram Alerts on Ping Loss and Recovery

**Input**: Design documents from `/specs/006-telegram-alerts/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included — the project constitution (Development Workflow) mandates unit tests for DB query logic, and plan.md/research.md (R7) commit to tests for the state machine, notify client, and config validation. These are the first tests in the repository.

**Organization**: Tasks are grouped by user story. US1 (loss alert) is the MVP; US2 (recovery alert) builds on the same state machine; US3 (configuration) completes validation and message customization.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

Single Go project at repository root: `main.go`, `internal/*`, per plan.md structure.

---

## Phase 1: Setup

**Purpose**: Package skeletons so subsequent tasks compile independently

- [X] T001 Create package skeletons `internal/monitor/monitor.go` and `internal/notify/telegram.go` with package declarations and doc comments; verify `go build ./...` still passes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config surface, DB query, and Telegram transport that every user story depends on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Extend `Config` in `internal/config/config.go` with optional `Telegram *TelegramConfig` block (`bot_token`, `chat_id`, `threshold_seconds` default 300, `routers`, `messages`, `router_messages` per data-model.md); nil when block absent; minimal presence-only validation for now (full matrix is US3)
- [X] T003 [P] Implement `LastSeenByRouter(database *sql.DB) (map[string]time.Time, error)` in `internal/db/queries.go` — `SELECT router_id, MAX(received_at) FROM ping_events GROUP BY router_id`, parsed as UTC like existing queries
- [X] T004 [P] Implement Telegram client in `internal/notify/telegram.go`: `Notifier` interface (`Send(ctx context.Context, text string) error`), `sendMessage` POST with JSON body per contracts/telegram-notifications.md, 10 s HTTP timeout, ≤3 attempts with 5 s/25 s backoff, injectable base URL, WARN/ERROR logs on retry/give-up
- [X] T005 [P] Unit test `LastSeenByRouter` against `:memory:` SQLite in `internal/db/queries_test.go` (empty table, one router, multiple routers with interleaved pings)
- [X] T006 [P] Unit test Telegram client in `internal/notify/telegram_test.go` using `httptest.Server`: asserts request body (chat_id, text), retry on 500 and on `"ok": false`, give-up after 3 attempts returns error

**Checkpoint**: Foundation ready — `go test ./...` green on db + notify packages

---

## Phase 3: User Story 1 — Ping Loss Notification (Priority: P1) 🎯 MVP

**Goal**: One Telegram loss message per continuous outage when a router is silent > N seconds; feature fully opt-in

**Independent Test**: Configure threshold + channel, stop a router's pings; the loss message arrives once within N + ~10 s (quickstart.md § 3.1); no config block ⇒ behavior identical to today

### Implementation for User Story 1

- [X] T007 [US1] Create monitor core in `internal/monitor/monitor.go`: `Monitor` struct holding per-router `RouterMonitorState` (data-model.md), constructor taking DB, `notify.Notifier`, `TelegramConfig`, injectable `now func() time.Time` and tick interval (default 10 s); `Run(ctx)` loop polling `LastSeenByRouter` each tick
- [X] T008 [US1] Implement seeding and loss transition in `internal/monitor/monitor.go`: seed states at startup with `LastSeen = max(dbLastSeen, serviceStart)` (research.md R4), create new routers on first sighting in UP (FR-011), transition UP→DOWN when `now − LastSeen > N` with `AlertSent` guard so one outage sends exactly one message (FR-002/FR-003); INFO log for every transition with `router_id`, `from`, `to`, `silence_seconds`
- [X] T009 [US1] Implement loss message resolution in `internal/monitor/monitor.go`: `messages.down` from config, else built-in default `Пинг от роутера <id> пропал (нет пингов дольше <N> с)` (per-router overrides come in US3)
- [X] T010 [US1] Wire up in `main.go`: when `cfg.Telegram != nil`, construct `notify` client and `monitor.Monitor`, start `Run` goroutine after `db.StartCleanupJob`; log `monitor started` with `threshold_seconds`, `tick_seconds`, `routers`; when nil, construct nothing (FR-007)
- [X] T011 [P] [US1] Unit tests for loss scenarios in `internal/monitor/monitor_test.go` with fake clock + fake notifier: silence > N fires once; continued silence fires nothing more; regular pings fire nothing (SC-003); never-seen router never alerts (FR-011); startup clamp prevents instant alert after restart (R4)

**Checkpoint**: MVP — build, configure a real bot, verify loss message per quickstart.md § 3.1

---

## Phase 4: User Story 2 — Ping Recovery Notification (Priority: P1)

**Goal**: One recovery message after pings resume and hold for the N-second stability window; flapping coalesces into a single outage

**Independent Test**: After an alerted outage, resume pings; recovery message arrives after N seconds of stability (quickstart.md § 3.2); a brief ping burst that dies before N seconds produces no message

### Implementation for User Story 2

- [X] T012 [US2] Add RECOVERING state transitions in `internal/monitor/monitor.go` per data-model.md: DOWN→RECOVERING when `LastSeen` advances (`RecoveryStart = LastSeen`); RECOVERING→DOWN when gap > N reopens (no second loss message — same outage); RECOVERING→UP when `now − RecoveryStart ≥ N`, sending the recovery message and resetting `AlertSent`/`RecoveryStart` (FR-004/FR-005)
- [X] T013 [US2] Implement recovery message resolution in `internal/monitor/monitor.go`: `messages.up` from config, else built-in default `Пинг от роутера <id> восстановился`
- [X] T014 [P] [US2] Unit tests for recovery scenarios in `internal/monitor/monitor_test.go`: stable recovery sends exactly one message; flapping (ping → gap > N before window completes) sends nothing and returns to DOWN without a second loss message; silence < N then resume sends nothing (no loss ⇒ no recovery, FR-005); full outage cycle produces exactly two messages loss→recovery (SC-004)

**Checkpoint**: Loss + recovery pair works end-to-end; flapping produces one pair, not spam

---

## Phase 5: User Story 3 — Notification Configuration (Priority: P2)

**Goal**: Full startup validation with descriptive exit errors, per-router message overrides, explicit monitored-router list, SIGHUP restart warning

**Independent Test**: Invalid configs (missing token/chat_id, threshold ≤ 0, bad router ID) exit at startup with clear errors (quickstart.md § 2); custom per-router texts are used verbatim; `routers:` list restricts monitoring

### Implementation for User Story 3

- [X] T015 [US3] Implement full validation matrix V2–V6 from contracts/config-schema.md in `internal/config/config.go` `Validate()`: required `bot_token`/`chat_id`, `threshold_seconds > 0`, router-ID regex on `routers` entries and `router_messages` keys; error strings match the contract wording (FR-008)
- [X] T016 [US3] Implement full message resolution chain in `internal/monitor/monitor.go`: `router_messages[<id>]` override → global `messages` → built-in default (FR-006, data-model.md resolution order)
- [X] T017 [US3] Implement monitored-router filter in `internal/monitor/monitor.go`: when `cfg.Telegram.Routers` non-empty, seed/track only listed routers and ignore others in poll results; empty list = all known routers (FR-010)
- [X] T018 [US3] Add SIGHUP warning in `internal/config/reload.go`: when the reloaded file's `telegram` block differs from the active one, log `telegram settings change ignored on reload — restart required` (same pattern as port/timezone)
- [X] T019 [P] [US3] Table tests for the validation matrix in `internal/config/config_test.go`: absent block passes untouched (V1), each of V2–V6 rejects with the expected message, valid block with defaults applied (`threshold_seconds` → 300) passes
- [X] T020 [P] [US3] Unit tests for resolution chain and router filter in `internal/monitor/monitor_test.go`: per-router override wins over global, global wins over default, unlisted router never tracked when `routers:` set

**Checkpoint**: All three stories independently functional; invalid config cannot start the service

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, end-to-end verification

- [X] T021 [P] Update `README.md`: new "Telegram alerts" section with config example from contracts/config-schema.md, bot/channel setup pointer, note that SIGHUP does not reload telegram settings
- [X] T022 [P] Update `CLAUDE.md` Architecture/Configuration sections: `internal/monitor` + `internal/notify` packages, `telegram:` config block, monitor polling design
- [X] T023 Run full quickstart.md validation end-to-end with a real bot and router (or curl-simulated pings): loss, recovery, quiet operation, Telegram-unreachable isolation (SC-005)
- [X] T024 Final gate: `go build ./...`, `go vet ./...`, `go test ./...` all pass (constitution Development Workflow)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none — start immediately
- **Foundational (Phase 2)**: after T001; T002/T003/T004 are mutually parallel; T005 needs T003; T006 needs T004
- **US1 (Phase 3)**: after Phase 2 complete (T007 needs config struct, DB query, notifier interface)
- **US2 (Phase 4)**: after US1 (extends the same state machine in `internal/monitor/monitor.go`)
- **US3 (Phase 5)**: T015/T018/T019 only need Phase 2 and can run parallel to US1/US2; T016/T017/T020 touch `internal/monitor/monitor.go` and must follow US2
- **Polish (Phase 6)**: after all user stories

### User Story Dependencies

- **US1 (P1)**: Foundational only — MVP
- **US2 (P1)**: builds on US1's state machine (same file); sequential after US1
- **US3 (P2)**: config-side tasks (T015, T018, T019) independent of US1/US2; monitor-side tasks (T016, T017, T020) after US2

### Within Each User Story

- Implementation before its unit-test task only where the test targets the same file being modified; test tasks marked [P] can be written alongside by a second contributor
- `internal/monitor/monitor.go` is intentionally serialized across T007→T008→T009→T012→T013→T016→T017 (same file — no [P])

### Parallel Opportunities

- Phase 2: T002 ∥ T003 ∥ T004, then T005 ∥ T006
- T011 (US1 tests) ∥ T012 (US2 impl) if staffed separately — different concerns, same package but different test/impl files
- US3 config-side (T015, T018, T019) ∥ all of US1/US2
- Phase 6: T021 ∥ T022

---

## Parallel Example: Foundational Phase

```bash
# After T001, launch together:
Task: "Extend Config with Telegram block in internal/config/config.go"        # T002
Task: "Implement LastSeenByRouter in internal/db/queries.go"                  # T003
Task: "Implement Telegram client in internal/notify/telegram.go"              # T004

# Then together:
Task: "Test LastSeenByRouter in internal/db/queries_test.go"                  # T005
Task: "Test Telegram client in internal/notify/telegram_test.go"              # T006
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 → Phase 2 (foundation)
2. Phase 3 (US1): loss alerts, opt-in wiring
3. **STOP and VALIDATE**: quickstart.md § 3.1 with a real bot — the dacha owner already gets the critical "пропал пинг" message
4. Phase 4 (US2) closes the loop with recovery; Phase 5 (US3) completes configuration hardening

### Incremental Delivery

Each checkpoint leaves the binary shippable: after US1 — loss alerts only; after US2 — full loss/recovery pairs; after US3 — validated, customizable configuration. No phase breaks existing dashboard/ping behavior (constitution I is protected by design — monitor never touches the request path).

---

## Notes

- Total: 24 tasks (Setup 1, Foundational 5, US1 5, US2 3, US3 6, Polish 4)
- `internal/monitor/monitor.go` accumulates the state machine across US1→US2→US3 by design; do not parallelize edits to it
- Commit after each task or logical group; every checkpoint must keep `go build ./...` and `go test ./...` green
