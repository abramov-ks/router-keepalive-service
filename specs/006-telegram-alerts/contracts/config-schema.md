# Contract: config.yaml `telegram:` block

**Feature**: 006-telegram-alerts

The service's only new external interface toward the **operator**. Extends the existing `config.yaml` (loaded from the binary's directory, validated at startup, exit code 1 with a descriptive error on violation).

## Schema

```yaml
telegram:                        # optional block; absent ⇒ alerting disabled entirely
  bot_token: string              # REQUIRED. Telegram bot token from @BotFather
  chat_id: string                # REQUIRED. Target channel: numeric ID ("-1001234567890") or "@channelname"
  threshold_seconds: int         # optional, default 300. N: loss threshold AND recovery stability window. Must be > 0
  routers:                       # optional, default [] = monitor every router that has ever pinged
    - string                     # each must match ^[a-zA-Z0-9_-]{1,64}$
  messages:                      # optional global message texts
    down: string                 # optional; sent when pings disappear
    up: string                   # optional; sent when pings recover
  router_messages:               # optional per-router overrides (win over `messages`)
    <router_id>:                 # key must match ^[a-zA-Z0-9_-]{1,64}$
      down: string
      up: string
```

## Validation rules (startup, fail-fast)

| # | Rule | Error behavior |
|---|------|----------------|
| V1 | Block absent → feature disabled, no validation | service starts normally, no alerting (FR-007) |
| V2 | Block present, `bot_token` empty | exit: `config: telegram.bot_token is required` |
| V3 | Block present, `chat_id` empty | exit: `config: telegram.chat_id is required` |
| V4 | `threshold_seconds` ≤ 0 | exit: `config: telegram.threshold_seconds must be positive` |
| V5 | `routers[i]` fails ID regex | exit: `config: telegram.routers[i] "<id>" contains invalid characters` |
| V6 | `router_messages` key fails ID regex | exit: `config: telegram.router_messages key "<id>" contains invalid characters` |

## Reload semantics

`SIGHUP` continues to reload **only** `allowed_routers`. If the `telegram:` block changed on disk at SIGHUP time, log a warning that a restart is required (same pattern as port/timezone today). No hot-reload of alerting settings in this feature.

## Example (user's scenario)

```yaml
port: 8080
timezone: Europe/Moscow
allowed_routers:
  - dacha-router

telegram:
  bot_token: "123456789:AAF..."
  chat_id: "@my_dacha_channel"
  threshold_seconds: 120
  messages:
    down: "Кажется, пропал пинг с дачи"
    up: "Кажется, пинг вернулся"
```
