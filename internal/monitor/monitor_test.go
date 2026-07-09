package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/cyrill/mikrotik-keepalive-server/internal/config"
)

type fakeNotifier struct {
	texts []string
}

func (f *fakeNotifier) Send(_ context.Context, text string) error {
	f.texts = append(f.texts, text)
	return nil
}

// harness drives the monitor with a fake clock and a mutable last-seen map.
type harness struct {
	m        *Monitor
	notifier *fakeNotifier
	clock    time.Time
	seen     map[string]time.Time
}

func newHarness(t *testing.T, cfg *config.TelegramConfig) *harness {
	t.Helper()
	h := &harness{
		notifier: &fakeNotifier{},
		clock:    time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC),
		seen:     map[string]time.Time{},
	}
	watched := make(map[string]struct{}, len(cfg.Routers))
	for _, id := range cfg.Routers {
		watched[id] = struct{}{}
	}
	h.m = &Monitor{
		cfg:       cfg,
		notifier:  h.notifier,
		threshold: time.Duration(cfg.ThresholdSeconds) * time.Second,
		tick:      10 * time.Second,
		now:       func() time.Time { return h.clock },
		lastSeen: func() (map[string]time.Time, error) {
			copied := make(map[string]time.Time, len(h.seen))
			for k, v := range h.seen {
				copied[k] = v
			}
			return copied, nil
		},
		watched: watched,
		states:  make(map[string]*routerState),
	}
	return h
}

// tickUntil advances the fake clock in 10 s steps up to deadline, evaluating each step.
func (h *harness) tickUntil(deadline time.Time) {
	for h.clock.Before(deadline) {
		h.clock = h.clock.Add(10 * time.Second)
		h.m.evaluate(context.Background())
	}
}

// ping records a router ping at the current fake time.
func (h *harness) ping(id string) {
	h.seen[id] = h.clock
}

func baseCfg() *config.TelegramConfig {
	return &config.TelegramConfig{
		BotToken:         "token",
		ChatID:           "@chan",
		ThresholdSeconds: 120,
	}
}

// ── US1: loss detection ──────────────────────────────────────────────────────

func TestLossFiresExactlyOnce(t *testing.T) {
	h := newHarness(t, baseCfg())
	h.ping("dacha")
	h.m.seed()

	h.tickUntil(h.clock.Add(10 * time.Minute)) // silence far beyond N=120s

	if len(h.notifier.texts) != 1 {
		t.Fatalf("expected exactly 1 loss message, got %d: %v", len(h.notifier.texts), h.notifier.texts)
	}
	if h.m.states["dacha"].status != statusDown {
		t.Errorf("status = %v, want DOWN", h.m.states["dacha"].status)
	}
}

func TestRegularPingsNeverAlert(t *testing.T) {
	h := newHarness(t, baseCfg())
	h.ping("dacha")
	h.m.seed()

	for range 240 { // 40 minutes of health, ping every 30 s
		h.clock = h.clock.Add(10 * time.Second)
		if h.clock.Second()%30 == 0 {
			h.ping("dacha")
		}
		h.m.evaluate(context.Background())
	}

	if len(h.notifier.texts) != 0 {
		t.Fatalf("expected no messages, got %v", h.notifier.texts)
	}
}

func TestNewRouterMidRunDoesNotAlertRetroactively(t *testing.T) {
	h := newHarness(t, baseCfg())
	h.m.seed() // empty DB at startup

	h.tickUntil(h.clock.Add(1 * time.Minute))
	h.ping("newbie") // first ping ever
	h.tickUntil(h.clock.Add(1 * time.Minute))

	if len(h.notifier.texts) != 0 {
		t.Fatalf("expected no messages for a healthy new router, got %v", h.notifier.texts)
	}

	h.tickUntil(h.clock.Add(10 * time.Minute)) // now it goes silent
	if len(h.notifier.texts) != 1 {
		t.Fatalf("expected 1 loss for new router after silence, got %v", h.notifier.texts)
	}
}

func TestStartupClampPreventsInstantAlert(t *testing.T) {
	h := newHarness(t, baseCfg())
	// Router last pinged an hour before startup (server itself may have been down).
	h.seen["dacha"] = h.clock.Add(-1 * time.Hour)
	h.m.seed()

	// Shortly after startup: no alert yet — silence is clamped to service start.
	h.tickUntil(h.clock.Add(60 * time.Second))
	if len(h.notifier.texts) != 0 {
		t.Fatalf("expected no instant alert at startup, got %v", h.notifier.texts)
	}

	// Continued silence past the threshold measured from startup → alert.
	h.tickUntil(h.clock.Add(2 * time.Minute))
	if len(h.notifier.texts) != 1 {
		t.Fatalf("expected 1 loss after threshold from startup, got %v", h.notifier.texts)
	}
}

// ── US2: recovery and flapping ───────────────────────────────────────────────

func TestFullOutageCycleSendsExactlyTwoMessages(t *testing.T) {
	cfg := baseCfg()
	cfg.Messages = config.MessagePair{Down: "Кажется, пропал пинг с дачи", Up: "Кажется, пинг вернулся"}
	h := newHarness(t, cfg)
	h.ping("dacha")
	h.m.seed()

	h.tickUntil(h.clock.Add(5 * time.Minute)) // outage
	// Recovery: pings resume every 30 s and stay stable.
	for range 30 {
		h.clock = h.clock.Add(10 * time.Second)
		if h.clock.Second()%30 == 0 {
			h.ping("dacha")
		}
		h.m.evaluate(context.Background())
	}

	want := []string{"Кажется, пропал пинг с дачи", "Кажется, пинг вернулся"}
	if len(h.notifier.texts) != 2 || h.notifier.texts[0] != want[0] || h.notifier.texts[1] != want[1] {
		t.Fatalf("expected %v, got %v", want, h.notifier.texts)
	}
	if h.m.states["dacha"].status != statusUp {
		t.Errorf("status = %v, want UP", h.m.states["dacha"].status)
	}
}

func TestFlappingCoalescesIntoOneOutage(t *testing.T) {
	h := newHarness(t, baseCfg())
	h.ping("dacha")
	h.m.seed()

	h.tickUntil(h.clock.Add(5 * time.Minute)) // outage → 1 loss message
	h.ping("dacha")                           // single ping, then silence again
	h.m.evaluate(context.Background())
	if h.m.states["dacha"].status != statusRecovering {
		t.Fatalf("status = %v, want RECOVERING", h.m.states["dacha"].status)
	}
	h.tickUntil(h.clock.Add(5 * time.Minute)) // gap reopens before window completes

	if len(h.notifier.texts) != 1 {
		t.Fatalf("flapping must not add messages: got %v", h.notifier.texts)
	}
	if h.m.states["dacha"].status != statusDown {
		t.Errorf("status = %v, want DOWN after failed recovery", h.m.states["dacha"].status)
	}

	// Now a real stable recovery → exactly one recovery message.
	for range 30 {
		h.clock = h.clock.Add(10 * time.Second)
		if h.clock.Second()%30 == 0 {
			h.ping("dacha")
		}
		h.m.evaluate(context.Background())
	}
	if len(h.notifier.texts) != 2 {
		t.Fatalf("expected loss+recovery total, got %v", h.notifier.texts)
	}
}

func TestNoRecoveryWithoutLoss(t *testing.T) {
	h := newHarness(t, baseCfg())
	h.ping("dacha")
	h.m.seed()

	// Silence shorter than threshold, then pings resume.
	h.tickUntil(h.clock.Add(90 * time.Second)) // N=120, not exceeded
	for range 30 {
		h.clock = h.clock.Add(10 * time.Second)
		if h.clock.Second()%30 == 0 {
			h.ping("dacha")
		}
		h.m.evaluate(context.Background())
	}

	if len(h.notifier.texts) != 0 {
		t.Fatalf("expected no messages (no loss ⇒ no recovery), got %v", h.notifier.texts)
	}
}

// ── US3: message resolution and router filter ────────────────────────────────

func TestMessageResolutionChain(t *testing.T) {
	cfg := baseCfg()
	cfg.Messages = config.MessagePair{Down: "global down", Up: "global up"}
	cfg.RouterMessages = map[string]config.MessagePair{
		"dacha": {Down: "дача пропала"}, // Up empty → falls back to global
	}
	h := newHarness(t, cfg)

	if got := h.m.lossText("dacha"); got != "дача пропала" {
		t.Errorf("per-router override: got %q", got)
	}
	if got := h.m.recoveryText("dacha"); got != "global up" {
		t.Errorf("fallback to global: got %q", got)
	}
	if got := h.m.lossText("office"); got != "global down" {
		t.Errorf("global for other router: got %q", got)
	}

	cfg.Messages = config.MessagePair{}
	if got := h.m.lossText("office"); got != "Пинг от роутера office пропал (нет пингов дольше 120 с)" {
		t.Errorf("built-in default: got %q", got)
	}
	if got := h.m.recoveryText("office"); got != "Пинг от роутера office восстановился" {
		t.Errorf("built-in default: got %q", got)
	}
}

func TestRouterFilterIgnoresUnlisted(t *testing.T) {
	cfg := baseCfg()
	cfg.Routers = []string{"dacha"}
	h := newHarness(t, cfg)
	h.ping("dacha")
	h.ping("office")
	h.m.seed()

	if _, tracked := h.m.states["office"]; tracked {
		t.Fatal("unlisted router must not be tracked")
	}

	h.tickUntil(h.clock.Add(10 * time.Minute)) // both silent

	if len(h.notifier.texts) != 1 {
		t.Fatalf("expected 1 message (dacha only), got %v", h.notifier.texts)
	}
}
