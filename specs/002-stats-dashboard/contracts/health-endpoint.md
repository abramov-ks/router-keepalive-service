# Contract: Health Endpoint

**Endpoint**: `GET /health`

**Purpose**: Confirm the service is running and the database is writable.
Used by monitoring tools, load balancers, or uptime checkers.

## Request

| Component | Value |
|-----------|-------|
| Method | `GET` |
| Path | `/health` |
| Auth | None |
| Custom headers | None |

## Response

### Healthy

```
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok","db":"ok"}
```

### Unhealthy (DB not writable)

```
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{"status":"degraded","db":"error: <message>"}
```

## Behaviour Notes

- The health check performs a lightweight DB probe (e.g., `SELECT 1`) on each
  request; it does not write to the database.
- HTTP 200 means the service is ready to accept pings.
- HTTP 503 means the DB is unavailable; pings will return 500 until resolved.
