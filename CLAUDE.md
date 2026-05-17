<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan at
specs/002-stats-dashboard/plan.md

## Quick reference

- Build: `go build -o server .`
- Run: `./server` (defaults: PORT=8080, DB_PATH=keepalive.db)
- Test: `go test ./...`
- Ping endpoint: `GET /ping?id=<router_id>`
- Dashboard: `GET /` (browser)
- Health: `GET /health`
<!-- SPECKIT END -->
