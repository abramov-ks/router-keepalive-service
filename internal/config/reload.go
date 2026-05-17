package config

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"
)

type AllowlistStore struct {
	mu      sync.RWMutex
	allowed map[string]struct{}
}

func NewAllowlistStore(ids []string) *AllowlistStore {
	s := &AllowlistStore{}
	s.Set(ids)
	return s
}

func (s *AllowlistStore) IsAllowed(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.allowed) == 0 {
		return true
	}
	_, ok := s.allowed[id]
	return ok
}

func (s *AllowlistStore) Set(ids []string) {
	m := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	s.mu.Lock()
	s.allowed = m
	s.mu.Unlock()
}

func StartReloadHandler(configPath string, store *AllowlistStore, originalPort int, originalTimezone string) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	go func() {
		for range sigCh {
			data, err := os.ReadFile(configPath)
			if err != nil {
				slog.Warn("SIGHUP: cannot read config, keeping previous allowlist", "err", err)
				continue
			}
			base := defaultConfig()
			if err := yaml.Unmarshal(data, &base); err != nil {
				slog.Warn("SIGHUP: cannot parse config, keeping previous allowlist", "err", err)
				continue
			}
			for i, id := range base.AllowedRouters {
				if !routerIDRegex.MatchString(id) {
					slog.Warn("SIGHUP: invalid router id in allowlist, keeping previous", "index", i, "id", id)
					continue
				}
			}
			store.Set(base.AllowedRouters)
			slog.Info("allowlist reloaded", "count", len(base.AllowedRouters))
			if base.Port != originalPort {
				slog.Warn("port change ignored on reload — restart required", "configured", base.Port, "active", originalPort)
			}
			if base.Timezone != originalTimezone {
				slog.Warn("timezone change ignored on reload — restart required", "configured", base.Timezone, "active", originalTimezone)
			}
		}
	}()
}
