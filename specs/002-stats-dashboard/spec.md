# Feature Specification: Ping Statistics Dashboard

**Feature Branch**: `002-stats-dashboard`

**Created**: 2026-05-17

**Status**: Draft

**Input**: User description: "Вторая основная функция: web интерфейс для просмотра статистики пингов с роутера. должен быть график за выбранный день (по умолчанию: сегодня) и возможность просмотра статистики за последние 7 дней. Так же должна быть возможность выбора роутера по id который шлет события"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Today's Ping Activity (Priority: P1)

A network administrator opens the dashboard in a browser. By default, they see a
chart of keepalive activity for the current day for the router selected by default
(or the first known router). The chart clearly shows which 30-second intervals had
a successful ping and which were missed, giving an immediate picture of uptime today.

**Why this priority**: This is the primary daily-use scenario. Administrators most
often want to check "is my router up right now and was it up today?" — the default
view must answer this without any interaction.

**Independent Test**: With a router sending pings for at least one hour, open the
dashboard URL. Verify: a chart appears for today, it reflects the correct number
of received pings, and missed intervals are visually distinct from received ones.

**Acceptance Scenarios**:

1. **Given** the dashboard is opened with no query parameters, **When** the page
   loads, **Then** a daily chart is shown for today with each time slot indicating
   whether a ping was received in that interval.

2. **Given** a router has missed several consecutive pings, **When** the daily chart
   is viewed, **Then** those time slots are visually marked as gaps/failures, distinct
   from successful intervals.

3. **Given** no pings have been received today, **When** the dashboard is opened,
   **Then** the chart is shown as empty (all intervals missed) rather than showing
   an error page.

---

### User Story 2 - Select a Specific Day (Priority: P1)

A network administrator needs to investigate an outage that was reported yesterday
evening. They use the dashboard to navigate to a past date and view the ping activity
chart for that specific day, identifying exactly when connectivity was lost and restored.

**Why this priority**: Historical day-level investigation is essential for
post-incident analysis and is the core navigation feature of the dashboard.

**Independent Test**: With data covering at least 2 days, use the date selector to
switch to yesterday. Verify the chart updates to show only that day's data and the
selected date is reflected in the page title or header.

**Acceptance Scenarios**:

1. **Given** the dashboard is showing today's chart, **When** the administrator
   selects a past date using the date picker, **Then** the chart updates to show
   ping activity for the selected day without reloading the entire page.

2. **Given** the administrator selects a date for which no data exists, **When** the
   chart updates, **Then** the chart shows an empty state (no pings recorded) with a
   clear message rather than an error.

3. **Given** the administrator selects a date more than 7 days in the past, **When**
   the system responds, **Then** the chart shows an empty state (data older than
   7 days is not retained) with an explanatory message.

---

### User Story 3 - View 7-Day Overview (Priority: P2)

A network administrator wants a quick weekly summary of a router's reliability.
They switch to the 7-day view which shows a summary chart spanning the last 7 days,
making it easy to spot days with high failure rates or full outages at a glance.

**Why this priority**: The weekly overview provides trend visibility beyond today,
complementing the daily detailed view. It is a secondary but important analytical tool.

**Independent Test**: With data covering at least 3 days, switch to the 7-day view.
Verify: the chart spans 7 days, each day's summary is visible, and the relative
density of pings per day is distinguishable.

**Acceptance Scenarios**:

1. **Given** the dashboard is in daily view, **When** the administrator switches to
   the 7-day view, **Then** a summary chart spanning the last 7 days is shown, with
   each day represented as a unit.

2. **Given** some days have no ping data, **When** the 7-day chart is displayed,
   **Then** those days are shown as empty/zero activity, not omitted from the chart.

3. **Given** the 7-day view is active, **When** the administrator clicks a day bar
   or segment on the 7-day chart, **Then** the view switches to the daily chart for
   that specific day.

---

### User Story 4 - Switch Between Routers (Priority: P2)

A network administrator manages three routers across different office locations.
From the dashboard, they can pick any router by its ID from a selector and
immediately see that router's ping statistics without navigating to a separate page.

**Why this priority**: Multi-router support is defined in the constitution. Without
a router selector, the dashboard is unusable when more than one router is registered.

**Independent Test**: With two routers sending pings, open the dashboard. Use the
router selector to switch between the two routers. Verify the chart data updates to
reflect each router's independent ping history.

**Acceptance Scenarios**:

1. **Given** multiple routers have sent pings, **When** the dashboard is opened,
   **Then** a router selector shows all known routers by their ID.

2. **Given** the administrator selects a different router from the selector, **When**
   the selection is confirmed, **Then** the chart updates to show the selected
   router's ping data for the currently active date or date range.

3. **Given** only one router is known to the system, **When** the dashboard loads,
   **Then** that router is pre-selected and the router selector may be hidden or
   shown as a single non-interactive label.

---

### Edge Cases

- What happens when the dashboard is opened before any router has sent a ping?
  The dashboard MUST load and display an empty state with a message indicating
  no data is available yet.
- What if a router's ID contains long strings or many characters?
  The router selector MUST display IDs without breaking the layout (truncation
  with tooltip is acceptable).
- What happens when the user navigates to a day at the boundary of the retention
  period (exactly 7 days ago)? The data for that day MUST be shown if available.
- What if the server's clock differs significantly from the user's browser clock?
  Timestamps MUST be displayed in UTC or a clearly labeled timezone to avoid confusion.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a browser-accessible dashboard page at a fixed URL
  that requires no authentication to access.
- **FR-002**: The dashboard MUST display a chart of ping activity for a selected day,
  defaulting to the current day on first load.
- **FR-003**: The dashboard MUST provide a date picker or navigation control allowing
  the administrator to select any day within the last 7 days.
- **FR-004**: The daily chart MUST clearly distinguish between time intervals where
  a ping was received and intervals where no ping was received (gaps/outages).
- **FR-005**: The dashboard MUST provide a toggle or tab to switch between the
  daily chart view and a 7-day summary chart view.
- **FR-006**: The 7-day chart MUST display one aggregate data point per day covering
  the last 7 calendar days.
- **FR-007**: Clicking or tapping a day on the 7-day chart MUST navigate to the
  daily chart for that specific day.
- **FR-008**: The dashboard MUST provide a router selector listing all router IDs
  that have sent at least one ping to the server.
- **FR-009**: Selecting a router from the selector MUST update all charts to show
  data for the selected router only.
- **FR-010**: The router selector MUST default to the first known router (by
  earliest recorded ping) when no router has been explicitly selected.
- **FR-011**: When no ping data exists (no routers known, or selected date has no
  data), the dashboard MUST display a clear empty-state message rather than an error.
- **FR-012**: All chart assets MUST be bundled with the service; the dashboard MUST
  be fully functional without any external network requests.
- **FR-013**: The dashboard MUST show the timestamp of the most recent ping received
  from the currently selected router, displayed prominently alongside the chart.

### Key Entities

- **Router**: A source of keepalive pings, identified by a unique string ID. The
  dashboard discovers routers from stored ping events (no manual registration).
- **Keepalive Event**: A single recorded ping. Attributes: receipt timestamp,
  router ID. Used to populate all charts and the last-ping indicator.
- **Daily Chart**: A time-series visualization of one router's pings for a single
  calendar day, showing presence or absence of pings per interval.
- **Weekly Chart**: A summary visualization of one router's ping activity aggregated
  by day across the last 7 calendar days.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The dashboard loads and renders all charts within 3 seconds of page
  open on a local network connection.
- **SC-002**: Switching between routers or changing the selected date updates the
  chart within 1 second without a full page reload.
- **SC-003**: The daily chart correctly reflects 100% of stored pings for the
  selected router and day — no pings are omitted or duplicated in the visualization.
- **SC-004**: The 7-day chart correctly summarizes activity for each of the last
  7 days, with zero days omitted even if they have no data.
- **SC-005**: An administrator with no prior training can identify a connectivity
  gap in the daily chart within 30 seconds of opening the dashboard.
- **SC-006**: The dashboard remains functional and displays a meaningful empty state
  when zero ping records exist in the system.

## Assumptions

- The dashboard is accessed on a local network; no authentication is required.
  Network-level access control is assumed to be managed externally.
- "Last 7 days" means the current day plus the 6 preceding calendar days; the system
  does not need to support arbitrary historical date ranges beyond this window.
- The expected ping interval is approximately 30 seconds; the daily chart resolution
  is designed around this cadence (e.g., 1-minute or 5-minute buckets are acceptable
  aggregation units for the daily view).
- Timezone display: timestamps are shown in the server's local timezone (configurable
  via environment variable); UTC is used as the default if not configured.
- The dashboard is a read-only interface; administrators cannot delete or edit ping
  records from the UI.
- There is no pagination on the router selector; it is assumed the number of routers
  is small (under 50) and all can be listed in a single dropdown.
