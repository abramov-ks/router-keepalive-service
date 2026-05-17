# Feature Specification: MikroTik Keepalive Ping Handler

**Feature Branch**: `001-ping-handler`

**Created**: 2026-05-17

**Status**: Draft

**Input**: User description: "Основная функция: хендлер для приема запросов от mikrotik роутера и сохранение их в базу, так же будет передаваться уникальный id роутера, чтобы можно было от разных роутеров принимать на один сервис"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Single Router Keepalive (Priority: P1)

A network administrator configures a MikroTik router to send a scheduled keepalive
signal to the server every 30 seconds. The server receives each signal, records it,
and confirms receipt. The administrator can later verify that the router was online
and reachable during a given time window.

**Why this priority**: This is the core function of the service. Without reliable
ping reception and storage, all other features are meaningless.

**Independent Test**: Configure a MikroTik router (or simulate its request format)
to send a ping to the server endpoint. Verify the server responds with success and
the event is recorded. Delivers value as a standalone uptime logging solution.

**Acceptance Scenarios**:

1. **Given** the server is running, **When** a MikroTik router sends a scheduled
   keepalive request with its router ID, **Then** the server responds with a success
   status and the event is stored with timestamp and router ID.

2. **Given** the server is running, **When** a keepalive request arrives without a
   router ID, **Then** the server responds with an error status and no event is stored.

3. **Given** the server experiences a storage failure, **When** a keepalive request
   arrives, **Then** the server responds with an error status (not a silent success)
   so the router's scheduler can detect and log the failure.

---

### User Story 2 - Multiple Routers on One Server (Priority: P1)

A network administrator manages several MikroTik routers across multiple locations.
All routers are configured to send keepalives to the same server endpoint, each using
its own unique identifier. The administrator can distinguish between routers and track
each one's uptime independently.

**Why this priority**: Multi-router support is listed as a primary requirement and
defines the data model for the entire system.

**Independent Test**: Send keepalive requests from two different router identifiers
to the same endpoint. Verify both are stored separately and appear as distinct routers
in any listing or query.

**Acceptance Scenarios**:

1. **Given** two routers with different IDs both sending keepalives, **When** events
   are stored, **Then** each event is attributed to its correct router ID.

2. **Given** router A has been active and router B has never pinged, **When** the
   administrator queries the system, **Then** router A shows a recent last-ping time
   and router B does not appear or shows no activity.

---

### User Story 3 - Uptime Dashboard (Priority: P2)

A network administrator opens the web dashboard in a browser to quickly assess the
health of all monitored routers. They can see the most recent ping time for each
router, as well as a chart showing keepalive activity over the last 24 hours and the
last 7 days.

**Why this priority**: The dashboard is the primary observability surface, but the
system delivers value for data collection (P1) even without it.

**Independent Test**: With at least one router sending regular keepalives, open the
dashboard URL in a browser. Verify the last-ping timestamp updates in near-real-time
and the 24-hour chart reflects recent activity.

**Acceptance Scenarios**:

1. **Given** a router has been sending keepalives, **When** the administrator opens
   the dashboard, **Then** they see the router's ID, the time of its most recent ping,
   and charts of its activity for the last 24 hours and last 7 days.

2. **Given** multiple routers are active, **When** the administrator opens the
   dashboard, **Then** all routers are listed with their respective last-ping times
   and status.

3. **Given** a router has stopped sending keepalives, **When** the administrator
   views the dashboard, **Then** the router's last-ping time reflects when it last
   checked in (not a false "online" status).

---

### Edge Cases

- What happens when the same router ID sends two pings within the same second?
  Both MUST be recorded independently.
- How does the system handle a router ID containing special characters (spaces,
  slashes, Unicode)? The system MUST reject IDs that cannot be safely stored;
  alphanumeric characters, hyphens, and underscores MUST be accepted.
- What happens when the server restarts? All previously stored pings MUST be
  retained and visible on the dashboard after restart.
- What happens when storage is full or the database is corrupted? The server MUST
  return an error response and log the failure; it MUST NOT silently drop pings.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose a keepalive endpoint that MikroTik routers can
  call on a schedule (e.g., every 30 seconds) using the router's built-in HTTP
  request capabilities with no custom headers or authentication.
- **FR-002**: Each keepalive request MUST include a router identifier supplied by
  the router; requests without an identifier MUST be rejected with an error response.
- **FR-003**: System MUST persistently store each accepted keepalive event with:
  receipt timestamp, router identifier, and source network address.
- **FR-004**: System MUST support an unlimited number of distinct router identifiers
  sending keepalives to the same endpoint simultaneously.
- **FR-005**: System MUST return a distinct error response when a keepalive cannot
  be stored, so the sending router can detect and log the failure.
- **FR-006**: System MUST expose a web dashboard (browser-accessible, no login
  required) showing for each known router:
  - Timestamp of the most recent keepalive received.
  - A chart of keepalive activity over the last 24 hours.
  - A chart of keepalive activity over the last 7 days.
- **FR-007**: Dashboard charts MUST be rendered using assets bundled with the service;
  no external network requests MUST be required at runtime.
- **FR-008**: System MUST expose a health-check endpoint confirming the service is
  running and storage is accessible.
- **FR-009**: System MUST retain at least 7 days of keepalive history to support the
  weekly chart.

### Key Entities

- **Router**: A monitored MikroTik device identified by a unique string ID. Known
  routers are discovered automatically from incoming keepalive events (no manual
  registration required).
- **Keepalive Event**: A single ping received from a router. Attributes: receipt
  timestamp, router ID, source IP address, reception outcome (success/failure).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every keepalive request received under normal operating conditions is
  recorded and retrievable within 1 second of receipt.
- **SC-002**: The server correctly attributes keepalive events from at least 10
  simultaneously active routers without data loss or cross-contamination.
- **SC-003**: The dashboard loads and displays current data for all known routers
  within 3 seconds of page open on a local network.
- **SC-004**: 7 days of keepalive history (at 30-second intervals per router) is
  retained and visible on the weekly chart without data loss.
- **SC-005**: A failed storage attempt always produces an error response — zero
  silent failures under any tested failure scenario.
- **SC-006**: The service starts and is ready to accept pings within 5 seconds of
  launch with no manual database setup required.

## Assumptions

- MikroTik routers use their built-in scheduler and HTTP fetch tool to send keepalive
  requests; no custom firmware or scripts beyond standard MikroTik RouterOS are assumed.
- Router IDs are alphanumeric strings (with hyphens/underscores allowed) assigned by
  the administrator when configuring the router's scheduler script.
- The server runs on a local network or private server; the dashboard requires no
  authentication (network-level access control is assumed to be handled externally).
- A single server instance serves all monitored routers; no clustering or high-availability
  deployment is required for v1.
- Keepalive interval is approximately 30 seconds per router; the system is not designed
  for sub-second ping rates.
- Data older than 7 days MAY be pruned; administrators needing longer history will
  configure retention separately (out of scope for v1).
