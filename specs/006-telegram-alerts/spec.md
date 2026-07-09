# Feature Specification: Telegram Alerts on Ping Loss and Recovery

**Feature Branch**: `006-telegram-alerts`

**Created**: 2026-07-09

**Status**: Draft

**Input**: User description: "нужно сделать уведомления в telegram при пропадании пингов, более чем N секунд отправлять сообщение в мой канал «Кажется, пропал пинг с дачи». После этого, при восстановлении пинга в течении N секунд отправлять сообщение в мой канал «Кажется, пинг вернулся»."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Ping Loss Notification (Priority: P1)

The owner of a remote site (e.g., a dacha) has a MikroTik router sending keepalive pings every 30 seconds. When the site loses power or connectivity, pings stop arriving. Instead of noticing the problem hours later by opening the dashboard, the owner receives a Telegram message in their channel — e.g., «Кажется, пропал пинг с дачи» — as soon as the silence exceeds a configured threshold of N seconds.

**Why this priority**: This is the core value of the feature. The dashboard is passive — the user must remember to look at it. A push notification turns the service from a monitoring tool into an alerting tool, which is the whole point of a keepalive server.

**Independent Test**: Configure a threshold (e.g., 120 seconds) and a Telegram channel. Let a router ping normally, then stop the pings (disable the router's scheduler). Within the threshold plus detection interval, the configured loss message appears in the Telegram channel. Exactly one message is sent, no matter how long the outage lasts.

**Acceptance Scenarios**:

1. **Given** a monitored router pinging regularly and a threshold of N seconds, **When** no ping arrives from that router for more than N seconds, **Then** the loss message is sent to the configured Telegram channel exactly once.
2. **Given** a loss notification has already been sent for an ongoing outage, **When** the outage continues, **Then** no further loss messages are sent for that outage.
3. **Given** a router whose pings arrive regularly within N seconds of each other, **When** monitoring runs continuously, **Then** no loss message is ever sent.

---

### User Story 2 — Ping Recovery Notification (Priority: P1)

After receiving a loss notification, the owner wants to know when the site is back online without polling the dashboard — but only once the connection is actually stable, not on a single stray ping. When the router's pings resume and keep arriving for N seconds, the owner receives a follow-up message in the same channel — e.g., «Кажется, пинг вернулся».

**Why this priority**: The loss alert alone leaves the user in suspense; the recovery alert closes the loop. The stability window prevents a flapping connection from producing a false "back online" message. Recovery is only meaningful after a loss alert, so it ships together with User Story 1.

**Independent Test**: Trigger an outage as in User Story 1 and wait for the loss message. Re-enable the router's pings. After pings have been arriving for N seconds without a gap exceeding the threshold, the recovery message appears in the channel. Re-enabling pings for a moment and stopping them again before N seconds elapse produces no recovery message.

**Acceptance Scenarios**:

1. **Given** a loss notification was sent for a router, **When** its pings resume and continue without interruption for N seconds, **Then** the recovery message is sent to the channel.
2. **Given** a loss notification was sent, **When** a ping arrives but pings stop again before the N-second stability window completes, **Then** no recovery message is sent and the outage is treated as still ongoing (no additional loss message either).
3. **Given** a router went silent for less than N seconds (no loss message was sent), **When** its pings resume, **Then** no recovery message is sent.
4. **Given** a recovery message was sent, **When** the router keeps pinging normally, **Then** no further messages are sent until the next outage exceeding N seconds.

---

### User Story 3 — Notification Configuration (Priority: P2)

The operator configures the alerting behavior in the service's existing configuration file: the Telegram channel to post to, the silence threshold N in seconds, and optionally custom loss/recovery message texts (the user's own wording, such as «Кажется, пропал пинг с дачи»). The service validates this configuration at startup and refuses to start with a descriptive error if it is invalid.

**Why this priority**: Without configuration the feature cannot be personalized (channel, threshold, wording), but reasonable defaults for texts mean User Stories 1–2 are demonstrable with minimal setup. Configuration follows the project's existing pattern of a single config file validated at startup.

**Independent Test**: Provide a config with Telegram settings, threshold, and custom messages; start the service; verify alerts use the custom texts. Provide an invalid config (e.g., missing channel with alerts enabled, non-positive threshold); verify the service exits at startup with a clear error message.

**Acceptance Scenarios**:

1. **Given** a config with custom loss and recovery texts, **When** an outage and recovery occur, **Then** the messages sent use exactly the configured texts.
2. **Given** a config with alerting enabled but missing required Telegram settings, **When** the service starts, **Then** it exits with a descriptive validation error.
3. **Given** no alerting configuration at all, **When** the service starts, **Then** it runs exactly as before — no alerting, no errors (feature is opt-in).

---

### Edge Cases

- **Service restart during an outage**: after a restart, the last-seen time of each router is known from stored ping history. If a router is already silent beyond the threshold at startup, a loss notification may be sent even if one was sent before the restart (notification state is not required to survive restarts) — but the loss/recovery pairing must remain consistent: a recovery message follows it when pings resume.
- **Router that has never pinged**: a router that has no ping history is not considered "down" and generates no loss notification until it has been seen at least once.
- **Telegram unreachable**: if sending a message fails, the failure is logged and retried a bounded number of times; ping reception and the dashboard must remain fully functional regardless of Telegram availability (per constitution: ping reception reliability is paramount).
- **Flapping connection** (pings repeatedly stopping and resuming around the threshold): the N-second stability window coalesces intermittent bursts into one ongoing outage — one loss message when the outage starts, one recovery message only after pings hold for N seconds. Distinct outages separated by a stable period each produce their own loss/recovery pair.
- **Multiple routers**: each monitored router is tracked independently; simultaneous outages of two routers produce separate notifications, and messages must make clear which router they concern when custom per-router texts are not configured.
- **Clock/timezone**: threshold comparison is based on elapsed time since the last accepted ping, independent of the configured display timezone.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST track, per router, the time elapsed since its last accepted ping.
- **FR-002**: System MUST send a loss notification to the configured Telegram channel when a monitored router's silence exceeds the configured threshold of N seconds.
- **FR-003**: System MUST send at most one loss notification per continuous outage of a given router (within one service run).
- **FR-004**: System MUST send a recovery notification to the same channel once pings from an alerted router have resumed and continued for N seconds without any gap exceeding the threshold (stability window). If pings stop again before the window completes, no recovery notification is sent and the outage is considered ongoing.
- **FR-005**: System MUST NOT send a recovery notification if no loss notification was sent for the preceding silence.
- **FR-006**: Operator MUST be able to configure: the Telegram channel destination and credentials, the threshold N (seconds), and the loss/recovery message texts. Message texts default to identifying the affected router; the user's own texts (e.g., «Кажется, пропал пинг с дачи» / «Кажется, пинг вернулся») can be set per router or globally.
- **FR-007**: Alerting MUST be opt-in: with no alerting configuration present, the service behaves exactly as it does today.
- **FR-008**: System MUST validate alerting configuration at startup and exit with a descriptive error when it is invalid (e.g., alerting enabled without channel credentials, non-positive threshold).
- **FR-009**: Failure to deliver a Telegram message MUST NOT affect ping reception, storage, or the dashboard; delivery failures MUST be logged and retried a bounded number of times.
- **FR-010**: Monitoring MUST cover each router independently; by default all routers that have pinged at least once are monitored, and the operator MAY restrict monitoring to an explicit list.
- **FR-011**: A router with no ping history MUST NOT trigger a loss notification.

### Key Entities

- **Router monitor state**: per-router record of last accepted ping time, current status (up / down), and whether a loss notification has been sent for the current outage. In-memory; seeded from stored ping history at startup.
- **Alert configuration**: Telegram destination (channel) and credentials, threshold N in seconds, optional monitored-router list, optional message text overrides (global or per router).
- **Notification**: an outbound Telegram message — either a loss or a recovery message — tied to one router and one outage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When a monitored router goes silent, the owner receives the loss message no later than N + 60 seconds after the last ping (threshold plus detection granularity).
- **SC-002**: When pings resume after an alerted outage and remain stable, the owner receives the recovery message no later than N + 60 seconds after the first accepted ping (stability window plus detection granularity).
- **SC-003**: A 24-hour period of normal pinging (every 30 seconds) produces zero notifications.
- **SC-004**: One continuous outage produces exactly two messages: one loss, one recovery — 100% of outages within a single service run, verified over at least 10 simulated outages.
- **SC-005**: With Telegram unreachable, 100% of incoming pings are still accepted and stored, and the dashboard remains available.
- **SC-006**: An operator can enable alerting by editing only the existing configuration file and restarting the service, in under 5 minutes.

## Assumptions

- One Telegram channel receives alerts for all routers; per-channel routing per router is out of scope for this feature.
- The threshold N is a single global value applied to all monitored routers (per-router thresholds are out of scope).
- "Recovery within N seconds" in the user's description means a stability window (confirmed by the user): pings must keep arriving for N seconds after resuming before the recovery message is sent; a single stray ping does not count as recovery. The same value N serves as both the loss threshold and the stability window.
- Notification state (whether a loss message was already sent) lives in memory for the current service run; a restart during an outage may produce one duplicate loss message, which is acceptable.
- The user already has (or will create) a Telegram bot/channel able to receive messages; obtaining credentials is the operator's responsibility and out of scope.
- Message delivery uses Telegram only; other channels (email, SMS) are out of scope.
- Default message texts include the router identifier so multi-router setups remain unambiguous; the user's exact texts are set via configuration.
