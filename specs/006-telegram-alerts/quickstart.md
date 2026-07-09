# Quickstart: Telegram Alerts on Ping Loss and Recovery

**Feature**: 006-telegram-alerts

## 1. Create a bot and channel (one-time, ~3 minutes)

1. In Telegram, talk to **@BotFather** → `/newbot` → copy the **bot token** (`123456789:AAF...`).
2. Create your channel (or use an existing one) and add the bot as an **administrator** with "Post messages" permission.
3. Get the channel's `chat_id`:
   - Public channel: just use `@channelname`.
   - Private channel: forward any channel post to **@userinfobot** (or similar) and copy the `-100…` ID.

## 2. Configure the service

Append to `config.yaml` next to the binary:

```yaml
telegram:
  bot_token: "123456789:AAF..."
  chat_id: "@my_dacha_channel"
  threshold_seconds: 120                     # N seconds; default 300 if omitted
  messages:
    down: "Кажется, пропал пинг с дачи"
    up: "Кажется, пинг вернулся"
```

Restart the service (`systemctl restart keepalive` or re-run `./server`). Startup log should show `monitor started`. An invalid block (missing token/chat_id, non-positive threshold) makes the service exit with a descriptive error.

## 3. Verify

1. **Loss**: disable the router's scheduler (`/system scheduler disable keepalive-scheduler`) or unplug it. Within `threshold_seconds` + ~10 s the channel receives: «Кажется, пропал пинг с дачи». Exactly one message, regardless of outage length.
2. **Recovery**: re-enable the scheduler. After pings have flowed for `threshold_seconds` (stability window), the channel receives: «Кажется, пинг вернулся».
3. **Quiet operation**: with the router pinging normally, no messages arrive at all.
4. **Isolation**: block outbound access to `api.telegram.org` — pings keep being accepted and the dashboard keeps working; send failures appear in the log as WARN/ERROR.

## 4. Run tests

```bash
go build ./...
go vet ./...
go test ./...          # monitor state machine, notify client, config validation, seed query
```
