package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTestTelegram(baseURL string) *Telegram {
	tg := NewTelegram("TESTTOKEN", "@testchan")
	tg.baseURL = baseURL
	tg.backoff = []time.Duration{time.Millisecond, time.Millisecond}
	return tg
}

func TestSendSuccessBody(t *testing.T) {
	var gotPath string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := newTestTelegram(srv.URL)
	if err := tg.Send(context.Background(), "Кажется, пропал пинг с дачи"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotPath != "/botTESTTOKEN/sendMessage" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["chat_id"] != "@testchan" {
		t.Errorf("chat_id = %q", gotBody["chat_id"])
	}
	if gotBody["text"] != "Кажется, пропал пинг с дачи" {
		t.Errorf("text = %q", gotBody["text"])
	}
}

func TestSendRetriesOn500ThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"ok":false,"description":"boom"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := newTestTelegram(srv.URL)
	if err := tg.Send(context.Background(), "hi"); err != nil {
		t.Fatalf("Send should succeed on third attempt: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestSendRetriesOnOKFalse(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 2 {
			// HTTP 200 but logical failure
			w.Write([]byte(`{"ok":false,"description":"chat not found"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := newTestTelegram(srv.URL)
	if err := tg.Send(context.Background(), "hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", calls.Load())
	}
}

func TestSendGivesUpAfterThreeAttempts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"ok":false,"description":"down"}`))
	}))
	defer srv.Close()

	tg := newTestTelegram(srv.URL)
	err := tg.Send(context.Background(), "hi")
	if err == nil {
		t.Fatal("Send should fail after exhausting retries")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}
