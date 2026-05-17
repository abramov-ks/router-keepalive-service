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
	Port           int      `yaml:"port"`
	Timezone       string   `yaml:"timezone"`
	AllowedRouters []string `yaml:"allowed_routers"`
}

func defaultConfig() Config {
	return Config{
		Port:     8080,
		Timezone: "UTC",
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
	return nil
}
