# Research: Telegram Alerts on Ping Loss and Recovery

**Feature**: 006-telegram-alerts | **Date**: 2026-07-09

No NEEDS CLARIFICATION markers remained in the Technical Context; the decisions below resolve the design unknowns.

## R1. Telegram delivery mechanism

**Decision**: Call the Telegram Bot API `sendMessage` method directly with stdlib `net/http`: `POST https://api.telegram.org/bot<token>/sendMessage` with a JSON body `{"chat_id": ..., "text": ...}`. HTTP client with a 10 s timeout.

**Rationale**: One endpoint, one method — a full bot-framework dependency (e.g., `go-telegram-bot-api`) would violate the spirit of constitution IV (single-binary simplicity, minimal deps) for zero benefit. The service only sends; it never receives updates, so no webhook/polling machinery is needed. Channels are addressed by `chat_id` (numeric `-100…` ID or `@channelusername` for public channels) with the bot added as channel administrator.

**Alternatives considered**:
- `github.com/go-telegram-bot-api/telegram-bot-api` — rejected: large dependency for a single POST.
- Notification gateways (ntfy, Apprise) — rejected: extra runtime service, violates single-binary principle; user explicitly asked for Telegram.

## R2. How the monitor learns about pings: DB polling vs in-process signal

**Decision**: Poll SQLite every 10 seconds with one aggregate query (`SELECT router_id, MAX(received_at) … GROUP BY router_id`) instead of having the ping handler push events into the monitor.

**Rationale**: Keeps the ping path byte-for-byte unchanged (constitution I), keeps monitor and handler fully decoupled, and automatically covers pings that arrived while the monitor logic evolves. At this scale (≤10 routers, ≤7 days of rows, indexed by `router_id, received_at`) the query cost is negligible. A 10 s tick gives detection latency ≤ N + 10 s, comfortably inside SC-001's N + 60 s budget.

**Alternatives considered**:
- Channel/callback from `PingHandler` into the monitor — rejected: couples the request path to the alerting subsystem (risk to constitution I), and still needs DB seeding at startup anyway, so polling adds no extra code overall.
- SQLite update hooks — rejected: not portable across drivers, adds complexity.

## R3. Outage/recovery state machine

**Decision**: Per-router in-memory state machine driven on each tick, with three effective states:

```
UP ──(silence > N)──────────────────────► DOWN        [send loss message once]
DOWN ──(new ping seen)──────────────────► RECOVERING  [recoveryStart = last ping time]
RECOVERING ──(gap > N reopens)──────────► DOWN        [no second loss message — same outage]
RECOVERING ──(stable for ≥ N seconds)───► UP          [send recovery message]
```

"Silence" = `now − lastSeen`. "Stable" = `now − recoveryStart ≥ N` with no intra-window gap exceeding N (guaranteed because a gap > N sends the router back to DOWN). A boolean `alertSent` guards the loss message so one continuous outage — including DOWN→RECOVERING→DOWN oscillation — produces exactly one loss and, eventually, one recovery message (FR-003/FR-004, flapping edge case).

**Rationale**: Directly encodes the spec's semantics (loss threshold and stability window both = N). Trivially unit-testable with an injected clock — no timers, just pure transition logic evaluated per tick.

**Alternatives considered**:
- Timer-per-router (`time.AfterFunc`) — rejected: harder to test, more moving parts, no benefit at 10-router scale.
- Debounce via "K consecutive pings" — rejected: spec defines the window in seconds, not ping counts.

## R4. Startup seeding and restart behavior

**Decision**: At startup, seed each router's `lastSeen` from the DB (`MAX(received_at)`), but clamp the effective silence start to service start time: `effectiveLastSeen = max(dbLastSeen, serviceStartTime)`. A router genuinely down across a restart therefore alerts N seconds *after startup*, not instantly.

**Rationale**: Raw DB seeding can't distinguish "router was down" from "the server itself was down and received nothing" — instant alerts on startup would fire for every healthy router after any server maintenance window. Clamping means every alert reflects silence actually observed by the running service. The spec explicitly permits a duplicate loss message after restart, and this design restores the loss→recovery pairing naturally (still-down router re-alerts after N seconds; recovery then follows when pings resume). Routers with no history at all are created in UP state only upon their first ping (FR-011: never alerted before first sighting).

**Alternatives considered**:
- Alert immediately at startup when DB silence > N — rejected: false-positive storm after server downtime.
- Persist notification state in SQLite to survive restarts — rejected: spec explicitly scopes state to a service run; adds a table + migration for marginal benefit.

## R5. Configuration shape and validation

**Decision**: New optional `telegram:` block in the existing `config.yaml` (same file, same load path, same fail-fast validation):

```yaml
telegram:
  bot_token: "123456:ABC-DEF..."
  chat_id: "@my_channel"        # or numeric ID, e.g. "-1001234567890"
  threshold_seconds: 120         # N; default 300 when omitted
  routers: [dacha-router]        # optional; empty/absent = all known routers
  messages:                      # optional global texts
    down: "Кажется, пропал пинг с дачи"
    up: "Кажется, пинг вернулся"
  router_messages:               # optional per-router overrides
    dacha-router:
      down: "Кажется, пропал пинг с дачи"
      up: "Кажется, пинг вернулся"
```

Absent block ⇒ alerting disabled, zero behavior change (FR-007). Present block ⇒ `bot_token` and `chat_id` required, `threshold_seconds` > 0, router IDs must match the existing `^[a-zA-Z0-9_-]{1,64}$` rule. Message fallback chain: per-router override → global `messages` → built-in default that includes the router ID («Пинг от роутера X пропал…» / «…восстановился»). SIGHUP reload does **not** apply to the `telegram:` block (consistent with port/timezone: warn that restart is required if it changed).

**Rationale**: Follows the established feature-003 config pattern exactly — one file next to the binary, validated at startup with descriptive exit errors. Defaulting N to 300 s (10 missed 30-second pings) is a safe out-of-the-box value; the user overrides it to taste.

**Alternatives considered**:
- Environment variables for token/chat — rejected: the project's config precedent (port, timezone, allowlist) is config.yaml; splitting alert config across two mechanisms hurts operability. Constitution lists config file as an accepted option.
- Hot-reload of telegram settings via SIGHUP — rejected for v1: keeps reload semantics identical to existing behavior (allowlist only); can be a follow-up feature.

## R6. Delivery failure handling

**Decision**: Send attempts run in the monitor goroutine with a 10 s HTTP timeout and up to 3 attempts with backoff (5 s, 25 s). On final failure: log at `ERROR` with router, message kind, and cause — then drop the message (state transition still stands; no infinite queue).

**Rationale**: Bounded retries satisfy FR-009 without unbounded memory or a persistent queue. Dropping after retries is acceptable for a home-monitoring alert channel and keeps the design stateless. State transitions are recorded regardless of delivery outcome so the loss/recovery pairing stays consistent even if one message is lost.

**Alternatives considered**:
- Persistent outbox table with redelivery — rejected: over-engineering for the scale; adds schema.
- Retrying forever until delivered — rejected: could pile up goroutines/messages during long Telegram outages.

## R7. Testing approach

**Decision**:
- `internal/monitor`: pure transition logic behind an injected `now func() time.Time` and a `Notifier` interface; table-driven tests simulate tick sequences (outage, flapping, restart seeding, never-seen router).
- `internal/notify`: `httptest.Server` standing in for `api.telegram.org` (base URL injectable); asserts request body, retry-on-500, and give-up-after-3.
- `internal/config`: table tests for the validation matrix (missing token, bad threshold, bad router ID, absent block).
- `internal/db`: `LastSeenByRouter` tested against `:memory:` SQLite per the constitution's development workflow.

**Rationale**: These are the first tests in the repo; the constitution mandates unit tests for DB query logic, and the state machine is exactly the kind of subtle logic (edge: flapping, restart) where tests pay for themselves immediately.
