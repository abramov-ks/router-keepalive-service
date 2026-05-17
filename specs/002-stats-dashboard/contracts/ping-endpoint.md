# Contract: Ping Endpoint

**Endpoint**: `GET /ping`

**Purpose**: Receive a keepalive signal from a MikroTik router and persist it.

**MikroTik RouterOS example**:
```
/tool fetch url="http://<server-host>:<port>/ping?id=office-router-1" keep-result=no
```

## Request

| Component | Value |
|-----------|-------|
| Method | `GET` |
| Path | `/ping` |
| Auth | None |
| Custom headers | None required |

### Query Parameters

| Parameter | Required | Format | Example |
|-----------|----------|--------|---------|
| `id` | Yes | `[a-zA-Z0-9_-]{1,64}` | `id=office-router-1` |

## Response

### Success (ping stored)

```
HTTP/1.1 200 OK
Content-Type: text/plain

OK
```

### Error: missing or invalid router ID

```
HTTP/1.1 400 Bad Request
Content-Type: text/plain

missing or invalid router id
```

### Error: storage failure

```
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain

internal error
```

## Behaviour Notes

- The server records `received_at` (UTC) and `source_ip` from the request; the
  router does not supply these — they are set server-side.
- A 200 response guarantees the event was written to SQLite before the response
  was sent. A 5xx response means the write failed.
- Concurrent pings from the same router ID within the same second are both stored
  independently (no deduplication).
- The response body is intentionally minimal so MikroTik's `keep-result=no` flag
  can discard it without memory overhead.
