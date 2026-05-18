# Feature Specification: Current Time Indicator Line on Charts

**Feature Branch**: `005-current-time-line`

**Created**: 2026-05-18

**Status**: Draft

**Input**: User description: "Нужно добавить на график вертикальную пунктирную линию которая показывает текущее время"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Current Time Line on Daily and Last 24h Charts (Priority: P1)

A network operator viewing the daily or "Last 24h" chart wants to immediately see where "now" is on the time axis. A vertical dashed line at the current time provides an instant visual reference — the operator can see at a glance how recent the last pings are relative to the present moment.

**Why this priority**: The primary value of the dashboard is real-time awareness. Without a "now" marker, the user has to mentally map the current time to the X-axis, which is error-prone. The line makes the chart self-explanatory.

**Independent Test**: Open the daily chart for today. A vertical dashed line is visible at the position corresponding to the current time. The line is visually distinct from the chart data (dashed style, neutral color). If viewing a past date, no line is shown (the current time is not on that day's chart).

**Acceptance Scenarios**:

1. **Given** the daily chart is showing today's date, **When** the chart renders, **Then** a vertical dashed line appears at the X-axis position corresponding to the current local time.
2. **Given** the "Last 24h" chart is active, **When** the chart renders, **Then** a vertical dashed line appears at the rightmost visible position (representing "now").
3. **Given** the daily chart is showing a past date, **When** the chart renders, **Then** no current-time line is shown (current time is outside the chart's range).
4. **Given** the current time is at 23:59, **When** the chart renders, **Then** the line is positioned near the far right of the X-axis without overflow.

---

### User Story 2 — Current Time Line on Weekly Chart (Priority: P2)

On the weekly chart, a vertical dashed line marks today's column, giving context for which data point is the most recent and which days are in the future.

**Why this priority**: Less critical than the daily view — the weekly chart has day-level granularity so the "today" column is already fairly obvious — but it adds consistency and helps when today is partway through.

**Independent Test**: Open the weekly chart. A vertical dashed line is shown at the position of today's date on the X-axis.

**Acceptance Scenarios**:

1. **Given** the weekly chart is active, **When** the chart renders, **Then** a vertical dashed line is shown at today's date position on the X-axis.

---

### Edge Cases

- What if the current time falls exactly on a chart boundary (midnight)? → Line appears at the far right of today's chart; no line on yesterday's chart.
- What if the chart has no data at all? → The line still renders at the correct time position on the empty chart.
- What if the user's browser timezone differs from the server's configured timezone? → The line should align with the server's configured timezone (same timezone the chart data uses), not the browser's local time.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: On the daily chart, when the selected date is today, a vertical dashed line MUST be displayed at the position of the current time on the X-axis.
- **FR-002**: On the "Last 24h" chart, a vertical dashed line MUST always be displayed at the position representing the current moment (the right edge of the 24-hour window).
- **FR-003**: On the weekly chart, a vertical dashed line MUST be displayed at the position of today's date on the X-axis.
- **FR-004**: The line MUST use a dashed or dotted stroke style to visually distinguish it from chart data.
- **FR-005**: The line MUST NOT obscure or interfere with the readability of chart data.
- **FR-006**: When the selected date is not today (daily chart), no current-time line MUST be shown.
- **FR-007**: The current time used for the line position MUST match the timezone used for chart data (server-configured timezone).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The current-time line is visible on 100% of chart renders where the current time falls within the displayed range.
- **SC-002**: The line position is accurate to within 1 minute of the actual current time on the 1-minute-resolution daily chart.
- **SC-003**: No existing chart functionality (tooltips, click-to-drill-down, zoom) is broken by the addition of the line.

## Assumptions

- The chart already knows the timezone from the data it receives from the server; the current-time line will use the same timezone reference.
- The line is a purely client-side visual element — no new API endpoints are required.
- A neutral color (e.g., dark grey or red) is appropriate for the line; exact styling is a detail left to implementation.
- The line does not need a label or tooltip — its position on the time axis is self-explanatory.
