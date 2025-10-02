/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestClient_EventAndUploadCrash(t *testing.T) {
	var mu sync.Mutex
	var events [][]byte
	var crashes [][]byte

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		mu.Lock()
		events = append(events, append([]byte(nil), b...))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		mu.Lock()
		crashes = append(crashes, append([]byte(nil), b...))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := Config{OptIn: true, EventsURL: srv.URL + "/events", CrashURL: srv.URL + "/crash", Timeout: 2 * time.Second}
	c := New(cfg)
	defer c.Close()

	if !c.Enabled() {
		t.Fatalf("expected client to be enabled")
	}

	// Send an event and flush
	c.Event("started", map[string]any{"k": "v"})
	c.Flush(context.Background())

	// Wait briefly for loop to send
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	ecount := len(events)
	mu.Unlock()
	if ecount == 0 {
		t.Fatalf("expected at least one event to be sent")
	}

	// Validate event JSON has name and ts
	var m map[string]any
	if err := json.Unmarshal(events[0], &m); err != nil {
		t.Fatalf("bad event json: %v", err)
	}
	if m["name"] != "started" {
		t.Fatalf("event name mismatch: %v", m["name"])
	}
	if _, ok := m["ts"].(string); !ok {
		t.Fatalf("missing ts field")
	}

	// Upload a crash report
	c.UploadCrash([]byte("STACKTRACE"))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	ccount := len(crashes)
	mu.Unlock()
	if ccount == 0 {
		t.Fatalf("expected crash upload to be sent")
	}
}

func TestEnabled_DefaultClientAndFromEnv(t *testing.T) {
	t.Setenv("GCW_TELEMETRY_OPT_IN", "true")
	t.Setenv("GCW_TELEMETRY_URL", "http://127.0.0.1:0") // bogus URL but presence enables
	t.Setenv("GCW_CRASH_UPLOAD_URL", "")
	t.Setenv("GCW_TELEMETRY_TIMEOUT_MS", "100")

	cfg := FromEnv()
	if !cfg.OptIn || cfg.EventsURL == "" || cfg.Timeout <= 0 {
		t.Fatalf("FromEnv did not parse correctly: %+v", cfg)
	}

	NewDefault(cfg)
	if !Enabled() {
		t.Fatalf("default Enabled should be true with env config")
	}
}
