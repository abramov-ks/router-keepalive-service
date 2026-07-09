# Data Model: Telegram Alerts on Ping Loss and Recovery

**Feature**: 006-telegram-alerts | **Date**: 2026-07-09

No persistent schema changes. All new entities are in-memory (config structs and monitor state); the feature reads the existing `ping_events` table only.

## TelegramConfig (config.yaml → `telegram:` block)

| Field | YAML key | Type | Required | Default | Validation |
|-------|----------|------|----------|---------|------------|
| BotToken | `bot_token` | string | yes (when block present) | — | non-empty |
| ChatID | `chat_id` | string | yes (when block present) | — | non-empty (numeric ID or `@channelname`) |
| ThresholdSeconds | `threshold_seconds` | int | no | 300 | > 0 (FR-008) |
| Routers | `routers` | []string | no | empty = monitor all known routers | each matches `^[a-zA-Z0-9_-]{1,64}$` |
| Messages | `messages` | MessagePair | no | built-in defaults | — |
| RouterMessages | `router_messages` | map[routerID]MessagePair | no | empty | keys match router ID regex |

**MessagePair**: `{ down: string, up: string }` — either field may be empty (falls through the resolution chain).

**Enablement rule**: the `telegram:` block absent from config.yaml ⇒ `Config.Telegram == nil` ⇒ monitor and notifier are never constructed (FR-007). Block present ⇒ full validation at startup, exit on error (FR-008).

**Message resolution** (per router, per direction):
1. `router_messages[<id>].down|up` if non-empty
2. `messages.down|up` if non-empty
3. Built-in default including the router ID: `Пинг от роутера <id> пропал (нет пингов дольше <N> с)` / `Пинг от роутера <id> восстановился`

## RouterMonitorState (in-memory, one per monitored router)

| Field | Type | Meaning |
|-------|------|---------|
| RouterID | string | router being tracked |
| LastSeen | time.Time | most recent accepted ping (from polling query), clamped to ≥ service start at seed time |
| Status | enum: `UP`, `DOWN`, `RECOVERING` | current state-machine state |
| AlertSent | bool | loss message sent for the current outage (guards FR-003) |
| RecoveryStart | time.Time (zero when not RECOVERING) | first ping after outage; stability window anchor |

**Lifecycle**:
- Created at startup for every router returned by `LastSeenByRouter()` (seeded UP with `LastSeen = max(dbLastSeen, serviceStart)`), or on first appearance of a new router in a later poll (FR-011: a router with no history cannot alert before its first ping).
- When `Routers` list in config is non-empty, states are created only for listed routers; polling results for other routers are ignored (FR-010).
- Never persisted; discarded on shutdown (spec assumption: duplicate loss message after restart is acceptable).

**State transitions** (evaluated every tick, N = ThresholdSeconds):

| From | Condition | To | Side effect |
|------|-----------|----|-------------|
| UP | `now − LastSeen > N` | DOWN | send **loss** message once; `AlertSent = true` |
| DOWN | `LastSeen` advanced (new ping) | RECOVERING | `RecoveryStart = LastSeen` |
| RECOVERING | `now − LastSeen > N` (gap reopens) | DOWN | none — same outage, `AlertSent` still true (FR-003) |
| RECOVERING | `now − RecoveryStart ≥ N` | UP | send **recovery** message; `AlertSent = false`, `RecoveryStart = 0` (FR-004) |
| UP | silence < N and pings arriving | UP | none (SC-003: quiet operation) |

Invariant: a **recovery** message is only ever sent from RECOVERING, which is only reachable from DOWN with `AlertSent == true` — FR-005 holds by construction.

## Notification (transient value object)

| Field | Type | Notes |
|-------|------|-------|
| RouterID | string | which router the message concerns |
| Kind | enum: `loss`, `recovery` | direction |
| Text | string | fully resolved message text |

Handed to the `Notifier` interface (`Send(ctx, text) error`); delivered with ≤3 attempts, then dropped with an ERROR log (FR-009). Not stored.

## Existing entities touched (read-only)

- **ping_events** (SQLite): new aggregate read `SELECT router_id, MAX(received_at) FROM ping_events GROUP BY router_id` (`LastSeenByRouter()` in `internal/db`). Uses the existing `idx_ping_events_router_time` index. No writes, no schema change.
