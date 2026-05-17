# Contract: Ping Rejection — 403 Forbidden

**Endpoint**: `GET /ping`

**Condition**: The `allowed_routers` list in `config.yaml` is non-empty AND the
`id` query parameter value is not in that list.

## Response

```
HTTP/1.1 403 Forbidden
Content-Type: text/plain

router not allowed
```

## Behaviour Notes

- The 403 response is returned AFTER the router ID format is validated (400 is
  returned first if the ID itself is malformed or missing).
- No ping event is written to the database on a 403 response.
- A structured log entry is written for every 403 rejection:
  ```
  level=WARN msg="ping rejected: router not in allowlist" router_id=<id> remote=<ip>
  ```
- When `allowed_routers` is empty or absent from config, this response is never
  returned (all validly-formatted router IDs are accepted).

## Decision Flow for GET /ping

```
1. Parse ?id= query param
   → missing or invalid format → 400 Bad Request
2. Check allowlist (if non-empty)
   → id not in allowlist → 403 Forbidden  ← this contract
3. Write to database
   → write failure → 500 Internal Server Error
4. → 200 OK "OK"
```
