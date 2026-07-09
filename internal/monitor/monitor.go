// Package monitor watches per-router ping recency and sends Telegram
// notifications when pings disappear for longer than the configured
// threshold and when they recover after a stability window.
package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/cyrill/mikrotik-keepalive-server/internal/config"
	"github.com/cyrill/mikrotik-keepalive-server/internal/db"
	"github.com/cyrill/mikrotik-keepalive-server/internal/notify"
)

type status int

const (
	statusUp status = iota
	statusDown
	statusRecovering
)

func (s status) String() string {
	switch s {
	case statusUp:
		return "UP"
	case statusDown:
		return "DOWN"
	case statusRecovering:
		return "RECOVERING"
	}
	return "UNKNOWN"
}

type routerState struct {
	lastSeen      time.Time
	status        status
	alertSent     bool
	recoveryStart time.Time
}

// Monitor drives the outage state machine. It polls the database on every
// tick and never touches the ping request path.
type Monitor struct {
	cfg       *config.TelegramConfig
	notifier  notify.Notifier
	threshold time.Duration
	tick      time.Duration

	// injectable for tests
	now      func() time.Time
	lastSeen func() (map[string]time.Time, error)

	watched map[string]struct{}
	states  map[string]*routerState
}

func New(database *sql.DB, notifier notify.Notifier, cfg *config.TelegramConfig) *Monitor {
	watched := make(map[string]struct{}, len(cfg.Routers))
	for _, id := range cfg.Routers {
		watched[id] = struct{}{}
	}
	return &Monitor{
		cfg:       cfg,
		notifier:  notifier,
		threshold: time.Duration(cfg.ThresholdSeconds) * time.Second,
		tick:      10 * time.Second,
		now:       time.Now,
		lastSeen:  func() (map[string]time.Time, error) { return db.LastSeenByRouter(database) },
		watched:   watched,
		states:    make(map[string]*routerState),
	}
}

// Run blocks until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) {
	m.seed()
	routers := "all"
	if len(m.cfg.Routers) > 0 {
		routers = fmt.Sprintf("%v", m.cfg.Routers)
	}
	slog.Info("monitor started",
		"threshold_seconds", m.cfg.ThresholdSeconds,
		"tick_seconds", int(m.tick.Seconds()),
		"routers", routers,
	)
	ticker := time.NewTicker(m.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.evaluate(ctx)
		}
	}
}

// seed initializes state for every known router. Silence is measured from
// service start, not from the stored timestamp: the gap between the last
// stored ping and now may be the server's own downtime, and alerting on it
// would fire for perfectly healthy routers after maintenance. A router that
// is genuinely down across a restart alerts threshold seconds after startup.
func (m *Monitor) seed() {
	seen, err := m.lastSeen()
	if err != nil {
		slog.Error("monitor: seed query failed", "err", err)
		return
	}
	start := m.now()
	for id, dbSeen := range seen {
		if !m.isWatched(id) {
			continue
		}
		eff := dbSeen
		if start.After(eff) {
			eff = start
		}
		m.states[id] = &routerState{lastSeen: eff, status: statusUp}
	}
}

func (m *Monitor) isWatched(id string) bool {
	if len(m.watched) == 0 {
		return true
	}
	_, ok := m.watched[id]
	return ok
}

func (m *Monitor) evaluate(ctx context.Context) {
	seen, err := m.lastSeen()
	if err != nil {
		slog.Error("monitor: last-seen query failed", "err", err)
		return
	}
	now := m.now()

	for id, dbSeen := range seen {
		if !m.isWatched(id) {
			continue
		}
		st, ok := m.states[id]
		if !ok {
			// First sighting of a new router: it starts UP and can only
			// alert after silence observed from this point on (FR-011).
			m.states[id] = &routerState{lastSeen: dbSeen, status: statusUp}
			continue
		}

		advanced := dbSeen.After(st.lastSeen)
		if advanced {
			st.lastSeen = dbSeen
		}
		silence := now.Sub(st.lastSeen)

		switch st.status {
		case statusUp:
			if silence > m.threshold {
				st.alertSent = true
				m.transition(id, st, statusDown, silence)
				m.send(ctx, id, "loss", m.lossText(id))
			}
		case statusDown:
			if advanced {
				st.recoveryStart = st.lastSeen
				m.transition(id, st, statusRecovering, silence)
			}
		case statusRecovering:
			if silence > m.threshold {
				// Gap reopened before the stability window completed:
				// same outage continues, no second loss message (FR-003).
				st.recoveryStart = time.Time{}
				m.transition(id, st, statusDown, silence)
			} else if now.Sub(st.recoveryStart) >= m.threshold && silence < m.threshold && st.alertSent {
				// silence must be strictly inside the threshold when the
				// window completes: a single stray ping followed by nothing
				// is not a recovery.
				st.alertSent = false
				st.recoveryStart = time.Time{}
				m.transition(id, st, statusUp, silence)
				m.send(ctx, id, "recovery", m.recoveryText(id))
			}
		}
	}
}

func (m *Monitor) transition(id string, st *routerState, to status, silence time.Duration) {
	from := st.status
	st.status = to
	slog.Info("monitor: state transition",
		"router_id", id,
		"from", from.String(),
		"to", to.String(),
		"silence_seconds", int(silence.Seconds()),
	)
}

func (m *Monitor) send(ctx context.Context, routerID, kind, text string) {
	if err := m.notifier.Send(ctx, text); err != nil {
		slog.Error("monitor: notification dropped", "router_id", routerID, "kind", kind, "err", err)
		return
	}
	slog.Info("monitor: notification sent", "router_id", routerID, "kind", kind)
}

// lossText resolves the outage message: per-router override → global → built-in.
func (m *Monitor) lossText(id string) string {
	if mp, ok := m.cfg.RouterMessages[id]; ok && mp.Down != "" {
		return mp.Down
	}
	if m.cfg.Messages.Down != "" {
		return m.cfg.Messages.Down
	}
	return fmt.Sprintf("Пинг от роутера %s пропал (нет пингов дольше %d с)", id, m.cfg.ThresholdSeconds)
}

// recoveryText resolves the recovery message: per-router override → global → built-in.
func (m *Monitor) recoveryText(id string) string {
	if mp, ok := m.cfg.RouterMessages[id]; ok && mp.Up != "" {
		return mp.Up
	}
	if m.cfg.Messages.Up != "" {
		return m.cfg.Messages.Up
	}
	return fmt.Sprintf("Пинг от роутера %s восстановился", id)
}
