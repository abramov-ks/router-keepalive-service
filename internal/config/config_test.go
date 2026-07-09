package config

import (
	"strings"
	"testing"
)

func validTelegram() *TelegramConfig {
	return &TelegramConfig{
		BotToken:         "123:ABC",
		ChatID:           "@chan",
		ThresholdSeconds: 120,
	}
}

func TestValidateTelegram(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // empty = must pass
	}{
		{
			name:   "absent block passes untouched",
			mutate: func(c *Config) { c.Telegram = nil },
		},
		{
			name:   "valid block passes",
			mutate: func(c *Config) {},
		},
		{
			name:    "missing bot_token",
			mutate:  func(c *Config) { c.Telegram.BotToken = "" },
			wantErr: "telegram.bot_token is required",
		},
		{
			name:    "missing chat_id",
			mutate:  func(c *Config) { c.Telegram.ChatID = "" },
			wantErr: "telegram.chat_id is required",
		},
		{
			name:    "negative threshold",
			mutate:  func(c *Config) { c.Telegram.ThresholdSeconds = -5 },
			wantErr: "telegram.threshold_seconds must be positive",
		},
		{
			name:    "invalid router id in list",
			mutate:  func(c *Config) { c.Telegram.Routers = []string{"ok-router", "bad router!"} },
			wantErr: `telegram.routers[1] "bad router!" contains invalid characters`,
		},
		{
			name: "invalid router_messages key",
			mutate: func(c *Config) {
				c.Telegram.RouterMessages = map[string]MessagePair{"bad key!": {Down: "x"}}
			},
			wantErr: `telegram.router_messages key "bad key!" contains invalid characters`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			cfg.Telegram = validTelegram()
			tt.mutate(&cfg)

			err := Validate(cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected pass, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestApplyDefaultsThreshold(t *testing.T) {
	cfg := defaultConfig()
	cfg.Telegram = &TelegramConfig{BotToken: "t", ChatID: "c"} // threshold omitted
	applyDefaults(&cfg)
	if cfg.Telegram.ThresholdSeconds != 300 {
		t.Errorf("threshold default = %d, want 300", cfg.Telegram.ThresholdSeconds)
	}

	cfg2 := defaultConfig()
	applyDefaults(&cfg2) // no telegram block — must not panic
	if cfg2.Telegram != nil {
		t.Error("telegram must stay nil when absent")
	}
}
