// Package notify delivers outbound alert messages. The only implementation
// is a minimal Telegram Bot API client built on the standard library.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Notifier sends one plain-text message to the configured destination.
type Notifier interface {
	Send(ctx context.Context, text string) error
}

const maxAttempts = 3

// Telegram posts messages to a channel via the Bot API sendMessage method.
type Telegram struct {
	token   string
	chatID  string
	baseURL string
	client  *http.Client
	backoff []time.Duration
}

func NewTelegram(token, chatID string) *Telegram {
	return &Telegram{
		token:   token,
		chatID:  chatID,
		baseURL: "https://api.telegram.org",
		client:  &http.Client{Timeout: 10 * time.Second},
		backoff: []time.Duration{5 * time.Second, 25 * time.Second},
	}
}

// Send delivers text with bounded retries. On final failure the error is
// returned; the caller decides how to log it — the message is not queued.
func (t *Telegram) Send(ctx context.Context, text string) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(t.backoff[attempt-2]):
			}
		}
		if err := t.sendOnce(ctx, text); err != nil {
			lastErr = err
			slog.Warn("telegram send failed", "attempt", attempt, "err", err)
			continue
		}
		return nil
	}
	return fmt.Errorf("telegram: giving up after %d attempts: %w", maxAttempts, lastErr)
}

func (t *Telegram) sendOnce(ctx context.Context, text string) error {
	body, err := json.Marshal(map[string]string{
		"chat_id": t.chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}

	url := t.baseURL + "/bot" + t.token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&result); err != nil {
		return fmt.Errorf("status %d: cannot decode response: %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || !result.OK {
		return fmt.Errorf("status %d: %s", resp.StatusCode, result.Description)
	}
	return nil
}
