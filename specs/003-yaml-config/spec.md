# Feature Specification: YAML Configuration File

**Feature Branch**: `003-yaml-config`

**Created**: 2026-05-17

**Status**: Draft

**Input**: User description: "Конфигурация сервиса осуществляется через yaml файл в директории с сервисом. Основные опции: 1. порт на котором работает сервис. 2. часовой пояс 3. список разрешенных id роутеров которые отправляют свои данные"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure Service via YAML File (Priority: P1)

A system administrator deploys the service and wants to control its behaviour
without restarting with different command-line flags or environment variables.
They create a `config.yaml` file in the same directory as the binary, set the
desired port, timezone, and a list of permitted router IDs, then start the
service. The service reads the file at startup and applies the settings.

**Why this priority**: Configuration is foundational — the router allowlist in
particular changes the security model of the service (open to any router → only
named routers). This must work before any other configuration-dependent feature.

**Independent Test**: Create a `config.yaml` with a non-default port and a
timezone, start the service, and verify it listens on the configured port and
timestamps reflect the configured timezone. Fully testable without any router
sending pings.

**Acceptance Scenarios**:

1. **Given** a `config.yaml` exists with `port: 9090`, **When** the service
   starts, **Then** it listens on port 9090, not the default 8080.

2. **Given** a `config.yaml` exists with `timezone: Europe/Moscow`, **When** the
   service starts, **Then** all timestamps in logs and the dashboard reflect the
   Moscow timezone.

3. **Given** no `config.yaml` exists, **When** the service starts, **Then** it
   starts successfully using built-in defaults (port 8080, UTC timezone, no
   router allowlist — all routers accepted).

4. **Given** `config.yaml` exists but contains a syntax error, **When** the
   service starts, **Then** it exits immediately with a descriptive error message
   identifying the file and the nature of the problem.

---

### User Story 2 - Router Allowlist Enforcement (Priority: P1)

A network administrator wants to ensure that only their known MikroTik routers
can submit keepalive pings to the server. They list the permitted router IDs in
`config.yaml`. Any ping arriving from an unlisted router ID is rejected, keeping
the data store clean and preventing unauthorised devices from registering.

**Why this priority**: The allowlist is a security boundary. Without it, any
client that can reach the server's network can create arbitrary router entries
in the database.

**Independent Test**: Configure `config.yaml` with exactly one allowed router ID.
Send a ping from that ID — expect success. Send a ping from a different ID —
expect rejection. Then remove the allowlist entirely and verify both IDs are
accepted.

**Acceptance Scenarios**:

1. **Given** the allowlist contains `["office-router-1"]`, **When** a ping arrives
   with `id=office-router-1`, **Then** the server accepts and stores it (200 OK).

2. **Given** the allowlist contains `["office-router-1"]`, **When** a ping arrives
   with `id=branch-router-2`, **Then** the server rejects it with a 403 response
   and does not store the event.

3. **Given** the allowlist is empty or absent in the config, **When** any ping
   with a valid router ID format arrives, **Then** the server accepts it (existing
   open behaviour is preserved).

4. **Given** the allowlist is configured, **When** the service receives a ping
   from an unlisted router, **Then** a log entry MUST be written indicating the
   rejection and the attempted router ID.

---

### User Story 3 - Reload Config Without Restart (Priority: P2)

A system administrator updates the allowlist in `config.yaml` to add a newly
installed router. They want the change to take effect without restarting the
service (which would briefly interrupt ping reception). They send a reload signal
to the running process and the new allowlist is applied immediately.

**Why this priority**: In production, service restarts cause brief downtime and
lost pings. Live reload avoids this. However, P1 static-at-startup config
already covers the core need; live reload is a convenience improvement.

**Independent Test**: Start the service with an allowlist of one router. Add a
second router ID to `config.yaml`. Send a reload signal. Send a ping from the
second router — expect it to be accepted without having restarted the service.

**Acceptance Scenarios**:

1. **Given** the service is running with an allowlist of `["router-A"]`, **When**
   `config.yaml` is updated to add `"router-B"` and a reload signal is sent,
   **Then** pings from `router-B` are accepted without service restart.

2. **Given** a reload is triggered with a malformed `config.yaml`, **When** the
   service processes the reload, **Then** it logs a warning, keeps the previous
   configuration active, and does not crash.

3. **Given** a reload is triggered, **When** the service re-reads the config,
   **Then** the port and timezone settings are NOT changed (only the allowlist is
   reloadable at runtime; port/timezone require a full restart).

---

### Edge Cases

- What if `config.yaml` specifies a port that is already in use? The service MUST
  exit with a descriptive error; it MUST NOT silently fall back to the default port.
- What if the timezone string in the config is invalid (e.g. `"Europe/Moscov"`)?
  The service MUST exit with a clear error naming the invalid value.
- What if an allowlist entry contains an invalid router ID format? The service
  MUST reject the config at startup with an error naming the offending entry.
- What if the `config.yaml` file exists but is empty? The service MUST treat this
  as all-defaults and start normally.
- What if the config file path is not in the service directory (e.g. a symlink)?
  The service reads the file at the resolved path; symlinks are followed silently.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The service MUST read configuration from a file named `config.yaml`
  located in the same directory as the running binary, if the file exists.
- **FR-002**: The config file MUST support the following settings:
  - `port` — the TCP port the HTTP server listens on (integer, 1–65535)
  - `timezone` — the timezone used for dashboard timestamps and log formatting
    (IANA timezone name, e.g. `Europe/Moscow`)
  - `allowed_routers` — an ordered list of router ID strings that are permitted
    to submit keepalive pings (empty list or absent key = allow all)
- **FR-003**: If `config.yaml` is absent, the service MUST start with built-in
  defaults: port 8080, UTC timezone, no allowlist (all valid router IDs accepted).
- **FR-004**: If `config.yaml` is present but cannot be parsed, the service MUST
  exit at startup with a human-readable error message; it MUST NOT start with
  partial or default configuration silently.
- **FR-005**: If `config.yaml` contains an invalid value for any setting (invalid
  port number, unrecognised timezone, malformed router ID in the allowlist), the
  service MUST exit with an error identifying the setting and the invalid value.
- **FR-006**: When `allowed_routers` is non-empty, pings from router IDs not in
  the list MUST be rejected with HTTP 403 and MUST NOT be stored.
- **FR-007**: Every rejection due to the allowlist MUST produce a structured log
  entry including the attempted router ID and the reason (`not in allowlist`).
- **FR-008**: The service MUST support reloading the `allowed_routers` list from
  `config.yaml` at runtime without restarting, triggered by a SIGHUP signal.
- **FR-009**: If a runtime reload encounters an invalid or unreadable config file,
  the service MUST retain the previously loaded configuration and log a warning;
  it MUST NOT crash or revert to defaults.
- **FR-010**: Port and timezone settings MUST only take effect at startup; they
  are not reloadable at runtime.

### Key Entities

- **Config**: The set of runtime settings loaded from `config.yaml`. Fields:
  port (integer), timezone (string), allowed_routers (list of strings).
- **Allowlist**: The subset of router IDs permitted to submit pings. Derived from
  `config.yaml`. Empty allowlist = no restriction.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The service applies port and timezone settings from `config.yaml`
  within the first startup cycle — no manual intervention required after placing
  the file.
- **SC-002**: 100% of pings from router IDs not in a non-empty allowlist are
  rejected; 0% are stored in the database.
- **SC-003**: A reload triggered by SIGHUP applies the updated `allowed_routers`
  list within 1 second without dropping any in-flight ping requests.
- **SC-004**: A startup with an invalid `config.yaml` produces an error exit
  within 2 seconds with a message sufficient to identify and fix the problem
  without consulting documentation.
- **SC-005**: An operator with no prior knowledge of the service can produce a
  working `config.yaml` by reading only the error message from a failed startup.

## Assumptions

- The config file is always named `config.yaml`; no alternative names or paths
  are supported in v1 (a `--config` flag is out of scope).
- Environment variables set before startup continue to act as fallbacks if a
  setting is absent from the config file (config file takes precedence over env
  vars when both are present).
- The allowlist is case-sensitive; `Office-Router-1` and `office-router-1` are
  treated as different router IDs.
- SIGHUP is the standard Unix reload signal and is available on all target
  platforms (Linux server, Docker container). Windows is out of scope for v1.
- The dashboard does not expose config settings; it remains a read-only ping
  statistics view.
