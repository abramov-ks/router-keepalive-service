# Quickstart: MikroTik Keepalive Server

## Prerequisites

- Go 1.22+ (`go version`)
- A MikroTik router with RouterOS scheduler access (or `curl` for testing)

## Build & Run

```bash
# Clone and enter the project
cd mikrotik-keepalive-server

# Download dependencies
go mod tidy

# Build the binary
go build -o server .

# Run with defaults (port 8080, database: keepalive.db in current dir)
./server
```

## Configuration (environment variables)

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `keepalive.db` | SQLite database file path |
| `TZ` | system timezone | Timezone for dashboard display |

```bash
# Custom port and database location
PORT=9000 DB_PATH=/data/keepalive.db ./server
```

## Test the Ping Endpoint

```bash
# Simulate a MikroTik ping (replace with your server address)
curl "http://localhost:8080/ping?id=my-router-1"
# Expected: 200 OK, body: "OK"

# Missing router ID — should return 400
curl "http://localhost:8080/ping"
# Expected: 400 Bad Request

# Health check
curl "http://localhost:8080/health"
# Expected: 200 OK, body: {"status":"ok","db":"ok"}
```

## View the Dashboard

Open `http://localhost:8080/` in a browser.

- The router selector auto-populates with all known router IDs.
- Default view: today's ping activity chart for the first known router.
- Use the date picker to navigate to past days (up to 7 days ago).
- Use the Day / Week toggle to switch to the 7-day summary view.

## Configure MikroTik Router

In RouterOS, add a scheduler entry:

```
/system scheduler add \
  name=keepalive \
  interval=30s \
  on-event="/tool fetch url=\"http://<server-ip>:8080/ping?id=<router-name>\" keep-result=no"
```

Replace `<server-ip>` with the server's IP and `<router-name>` with a unique
alphanumeric identifier for this router (e.g., `office-router-1`).

## Docker

```bash
# Build image
docker build -t mikrotik-keepalive .

# Run with persistent storage
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  -e DB_PATH=/data/keepalive.db \
  mikrotik-keepalive
```

## Validate Installation

After running the server and sending at least a few test pings:

1. `curl localhost:8080/api/routers` — should list your test router
2. `curl "localhost:8080/api/stats/daily?router_id=my-router-1"` — should show
   1440 minute-buckets with non-zero counts around the current time
3. Open `http://localhost:8080/` — should display the daily chart with green bars
   for the minutes when pings were received
