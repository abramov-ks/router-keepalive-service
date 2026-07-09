# Contract: Outbound Telegram notifications

**Feature**: 006-telegram-alerts

The service's only new external interface toward **Telegram**. No new inbound HTTP endpoints are added; `/ping`, `/health`, `/api/*`, and `/` are unchanged.

## Transport

- `POST https://api.telegram.org/bot<bot_token>/sendMessage`
- `Content-Type: application/json`
- Request body: `{"chat_id": "<chat_id>", "text": "<resolved message text>"}`
- Plain text only — no `parse_mode`, so user-configured messages need no escaping.
- HTTP client timeout: 10 s. Base URL injectable for tests (`httptest`).

## Delivery semantics

| Aspect | Behavior |
|--------|----------|
| Trigger: loss | router silent > N seconds → exactly one message per continuous outage (FR-002/FR-003) |
| Trigger: recovery | pings resumed and stable for N seconds after an alerted outage → exactly one message (FR-004/FR-005) |
| Ordering | per router, messages strictly alternate: loss, recovery, loss, recovery… within one service run |
| Success | Telegram returns HTTP 200 with `"ok": true` |
| Retry | non-2xx response, `"ok": false`, or transport error → retry up to 3 total attempts (backoff 5 s, 25 s) |
| Give-up | after 3 failed attempts: ERROR log `{router_id, kind, error}`, message dropped; state machine unaffected (FR-009) |
| Isolation | sending happens in the monitor goroutine; never blocks or fails ping reception, storage, or dashboard (SC-005) |

## Message texts

Resolution order per router and direction (see data-model.md):
1. `telegram.router_messages.<id>.down|up`
2. `telegram.messages.down|up`
3. Built-in defaults (include router ID and N):
   - down: `Пинг от роутера <id> пропал (нет пингов дольше <N> с)`
   - up: `Пинг от роутера <id> восстановился`

## Structured logging (constitution V)

| Event | Level | Fields |
|-------|-------|--------|
| state transition | INFO | `router_id`, `from`, `to`, `silence_seconds` |
| message sent | INFO | `router_id`, `kind` (`loss`/`recovery`), `attempt` |
| send retry | WARN | `router_id`, `kind`, `attempt`, `err` |
| send gave up | ERROR | `router_id`, `kind`, `attempts`, `err` |
| monitor started | INFO | `threshold_seconds`, `tick_seconds`, `routers` (explicit list or `all`) |
