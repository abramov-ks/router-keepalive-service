package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
	"log/slog"
)

var routerIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type Config struct {
	Port           int             `yaml:"port"`
	Timezone       string          `yaml:"timezone"`
	AllowedRouters []string        `yaml:"allowed_routers"`
	Telegram       *TelegramConfig `yaml:"telegram"`
}

// MessagePair holds notification texts for one direction pair.
// Empty fields fall through the resolution chain (per-router → global → built-in).
type MessagePair struct {
	Down string `yaml:"down"`
	Up   string `yaml:"up"`
}

// TelegramConfig enables outage alerting when present in config.yaml.
// A nil TelegramConfig means alerting is disabled entirely.
type TelegramConfig struct {
	BotToken         string                 `yaml:"bot_token"`
	ChatID           string                 `yaml:"chat_id"`
	ThresholdSeconds int                    `yaml:"threshold_seconds"`
	Routers          []string               `yaml:"routers"`
	Messages         MessagePair            `yaml:"messages"`
	RouterMessages   map[string]MessagePair `yaml:"router_messages"`
}

const defaultThresholdSeconds = 300

func defaultConfig() Config {
	return Config{
		Port:     8080,
		Timezone: "UTC",
	}
}

// applyDefaults fills in defaults that depend on which optional blocks are present.
func applyDefaults(cfg *Config) {
	if cfg.Telegram != nil && cfg.Telegram.ThresholdSeconds == 0 {
		cfg.Telegram.ThresholdSeconds = defaultThresholdSeconds
	}
}

// Load reads config.yaml from binDir if it exists and returns a validated Config.
// Missing file → returns defaults. Parse/validation error → returns error.
func Load(binDir string) (Config, error) {
	path := filepath.Join(binDir, "config.yaml")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		slog.Info("no config.yaml found, using defaults", "looked_in", path)
		return defaultConfig(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("config: cannot read config.yaml: %w", err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: cannot parse config.yaml: %w", err)
	}
	applyDefaults(&cfg)

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks all fields and returns the first error found.
func Validate(cfg Config) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("config: port %d is out of range (1–65535)", cfg.Port)
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return fmt.Errorf("config: timezone %q is invalid: %w", cfg.Timezone, err)
	}
	for i, id := range cfg.AllowedRouters {
		if !routerIDRegex.MatchString(id) {
			return fmt.Errorf("config: allowed_routers[%d] %q contains invalid characters", i, id)
		}
	}
	if err := validateTelegram(cfg.Telegram); err != nil {
		return err
	}
	return nil
}

func validateTelegram(tg *TelegramConfig) error {
	if tg == nil {
		return nil
	}
	if tg.BotToken == "" {
		return fmt.Errorf("config: telegram.bot_token is required")
	}
	if tg.ChatID == "" {
		return fmt.Errorf("config: telegram.chat_id is required")
	}
	if tg.ThresholdSeconds <= 0 {
		return fmt.Errorf("config: telegram.threshold_seconds must be positive")
	}
	for i, id := range tg.Routers {
		if !routerIDRegex.MatchString(id) {
			return fmt.Errorf("config: telegram.routers[%d] %q contains invalid characters", i, id)
		}
	}
	for id := range tg.RouterMessages {
		if !routerIDRegex.MatchString(id) {
			return fmt.Errorf("config: telegram.router_messages key %q contains invalid characters", id)
		}
	}
	return nil
}
