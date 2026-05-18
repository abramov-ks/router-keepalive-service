# Feature Specification: Last 24 Hours Dashboard View

**Feature Branch**: `004-last-24h-view`

**Created**: 2026-05-18

**Status**: Draft

**Input**: User description: "На дашборде нужно к пикеру day, week добавить кнопку last 24 hours для того чтобы видеть график последних 24 часов и сделать его по умолчанию"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Last 24 Hours as Default View (Priority: P1)

When a user opens the dashboard, they immediately see a rolling 24-hour view of ping activity — the most recent and actionable data. The view starts at the current moment and goes back exactly 24 hours, regardless of day boundaries.

**Why this priority**: This is the primary use case — a network operator wants to know "is my router alive right now and for the past day?" without having to navigate. Making it the default removes friction.

**Independent Test**: Open the dashboard with no URL parameters. The "Last 24h" button is active, and the chart shows data from the last 24 hours ending at the current time. Data from 25 hours ago is not shown.

**Acceptance Scenarios**:

1. **Given** the dashboard is opened with no view parameter, **When** the page loads, **Then** the "Last 24h" view is active by default and the chart shows a rolling 24-hour window ending now.
2. **Given** the "Last 24h" view is active, **When** the page is refreshed, **Then** the chart re-renders with an updated 24-hour window ending at the new current time.
3. **Given** the "Last 24h" view is active, **When** a router is selected, **Then** the chart updates to show that router's last 24 hours of activity.

---

### User Story 2 — Navigation Between Views (Priority: P1)

Users can switch between "Last 24h", "Day", and "Week" views using the toggle buttons. The date picker for the "Day" view is disabled when "Last 24h" or "Week" is active.

**Why this priority**: The existing Day and Week views remain useful; users need a clear way to switch between all three modes.

**Independent Test**: Click "Day" button — date picker becomes enabled and shows today's daily chart. Click "Week" — shows weekly chart. Click "Last 24h" — returns to rolling 24-hour chart, date picker disabled.

**Acceptance Scenarios**:

1. **Given** the "Last 24h" view is active, **When** the user clicks "Day", **Then** the daily chart for today is shown and the date picker becomes enabled.
2. **Given** the "Day" view is active, **When** the user clicks "Last 24h", **Then** the rolling 24-hour chart is shown and the date picker becomes disabled.
3. **Given** the user navigates to a specific day, **When** the user clicks "Last 24h", **Then** the view resets to the current rolling 24-hour window.

---

### Edge Cases

- What happens when there are no pings in the last 24 hours? → Chart renders as empty/flat (consistent with existing empty state behavior).
- What happens at midnight? → The 24-hour window is always relative to the current time, not a calendar day, so midnight causes no special behavior.
- What if the user has the dashboard open for a long time? → Each time they switch to "Last 24h" or refresh, the window recalculates from the current time.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The dashboard MUST display a "Last 24h" toggle button alongside the existing "Day" and "Week" buttons.
- **FR-002**: The "Last 24h" view MUST be the default view when the dashboard is opened with no view parameter.
- **FR-003**: The "Last 24h" chart MUST show ping activity for the rolling window from exactly 24 hours ago until the current moment, at 1-minute resolution.
- **FR-004**: The date picker MUST be disabled when the "Last 24h" view is active.
- **FR-005**: The selected view MUST be reflected in the page URL so the state can be shared or bookmarked (e.g., `?view=24h`).
- **FR-006**: Switching to the "Day" view from "Last 24h" MUST default to today's date.
- **FR-007**: The "Last 24h" view MUST refresh its time window each time it is activated or the page is reloaded.

### Key Entities

- **View mode**: One of `24h`, `day`, `week`. Replaces the previous two-state toggle. `24h` is the default.
- **24-hour window**: A time range defined as `[now − 24h, now]`, recalculated on each render.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The dashboard opens in "Last 24h" mode by default — 100% of cold loads without a `?view=` parameter show the 24h chart.
- **SC-002**: The 24-hour chart covers exactly the last 1440 minutes with no gaps in the time axis.
- **SC-003**: Switching between all three views takes less than 1 second on a normal connection.
- **SC-004**: The URL correctly encodes the active view so that sharing a link opens the same view.

## Assumptions

- The existing 1-minute bucket resolution used for the daily chart is sufficient for the 24-hour view.
- The 24-hour window is always rolling (relative to now), not aligned to a calendar day.
- The "Last 24h" view uses the same data endpoint as the daily view but with a dynamic date/time range spanning two calendar days if necessary.
- The date picker visibility/disabled state follows the same rules as the existing "Week" view (disabled when not in "Day" mode).
- No server-side pagination is needed — 1440 data points is within the current response budget.
