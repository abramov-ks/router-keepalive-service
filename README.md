# MikroTik Keepalive Server

Lightweight Go service that receives periodic HTTP pings from MikroTik routers and displays uptime statistics on a web dashboard.

## Features

- Accepts `GET /ping?id=<router_id>` keepalive requests from one or more MikroTik routers
- Stores ping events in a local SQLite database (no external dependencies)
- Web dashboard with three views:
  - **Last 24h** (default) — rolling 24-hour line chart at 1-minute resolution
  - **Day** — any specific day with a date picker
  - **Week** — last 7 days, click a day to drill into it
- Current-time indicator: red dashed vertical line on all charts
- Router allowlist: restrict which router IDs are accepted (403 for others)
- Telegram alerts: message to your channel when a router's pings disappear for more than N seconds, and again once they are back and stable
- SIGHUP live reload of the allowlist without restarting
- Single self-contained binary — all assets embedded via `go:embed`

## Quick Start

### Build and run

```bash
go build -o server .
./server
```

Dashboard available at `http://localhost:8080`.

### Docker

```bash
docker build -t keepalive .
docker run -p 8080:8080 -v $(pwd)/data:/data keepalive
```

## Configuration

Place `config.yaml` in the same directory as the binary:

```yaml
port: 8080                  # default: 8080
timezone: Europe/Moscow     # IANA timezone, default: UTC
allowed_routers:            # empty = allow all
  - office-router
  - branch-router
```

All keys are optional. Absent keys use built-in defaults. An invalid config causes the service to exit with a descriptive error.

### Telegram alerts

Add an optional `telegram` block to get notified when a router stops pinging:

```yaml
telegram:
  bot_token: "123456789:AAF..."   # from @BotFather
  chat_id: "@my_channel"          # or numeric ID, e.g. "-1001234567890"
  threshold_seconds: 120          # N: silence threshold AND recovery stability window (default 300)
  routers: [dacha-router]         # optional; empty = monitor all known routers
  messages:                       # optional; defaults include the router ID
    down: "Кажется, пропал пинг с дачи"
    up: "Кажется, пинг вернулся"
  router_messages:                # optional per-router overrides
    dacha-router:
      down: "Дача offline"
      up: "Дача online"
```

Create a bot via **@BotFather** and add it to your channel as an administrator with the "Post messages" permission.

Behavior: one "down" message per continuous outage once silence exceeds `threshold_seconds`; one "up" message after pings have resumed and stayed stable for the same window (an unstable, flapping connection is coalesced into a single outage). Without the `telegram` block the feature is fully disabled. Telegram being unreachable never affects ping reception or the dashboard — delivery is retried up to 3 times, then dropped with an error log. The `telegram` block is **not** reloaded on SIGHUP; restart the service after changing it.

### Reload allowlist without restart

```bash
kill -HUP $(pgrep -f ./server)
# or via systemd:
systemctl kill -s HUP keepalive
```

Port and timezone changes require a full restart.

## MikroTik Setup

Add a script and scheduler in the MikroTik terminal:

```routeros
/system script add name="keepalive-ping" source={
    /tool fetch url="http://<server-ip>:<port>/ping?id=<router-id>" mode=http keep-result=no
}

/system scheduler add name="keepalive-scheduler" interval=30s on-event="keepalive-ping" start-time=startup
```

> **Tip**: Enter the script source via Winbox (System → Scripts) to avoid the `?` character being interpreted as terminal help.

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /ping?id=<router_id>` | Record a keepalive ping. Returns `200 OK` or `403 Forbidden` if not in allowlist. |
| `GET /health` | Health check. Returns `{"status":"ok","db":"ok"}`. |
| `GET /` | Web dashboard. |
| `GET /api/routers` | List all known routers with first/last seen and total ping count. |
| `GET /api/stats/daily?router_id=X&date=YYYY-MM-DD` | 1-minute buckets for a specific day. |
| `GET /api/stats/last24h?router_id=X` | 1-minute buckets for the rolling last 24 hours. |
| `GET /api/stats/weekly?router_id=X` | Per-day ping counts for the last 7 days. |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `keepalive.db` | Path to the SQLite database file |

Port and timezone are configured via `config.yaml`, not environment variables.

## systemd

```ini
[Unit]
Description=MikroTik Keepalive Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/keepalive
ExecStart=/opt/keepalive/server
Environment=DB_PATH=/var/lib/keepalive/keepalive.db
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Development

```bash
go build ./...       # build
go vet ./...         # lint
go test ./...        # tests
```

Data is retained for 7 days; older records are purged automatically every 24 hours.
