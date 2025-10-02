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
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_DisabledAndEmptyEventName(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Disabled via OptIn=false
	c := New(Config{OptIn: false, EventsURL: srv.URL + "/events", CrashURL: srv.URL + "/crash", Timeout: time.Second})
	defer c.Close()
	if c.Enabled() {
		t.Fatalf("expected disabled client")
	}
	c.Event("ignored", nil)
	c.UploadCrash([]byte("ignored"))
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("expected no requests when disabled")
	}

	// Enabled but empty event name should be ignored
	c2 := New(Config{OptIn: true, EventsURL: srv.URL + "/events", Timeout: time.Second})
	defer c2.Close()
	c2.Event("", nil)
	c2.Flush(nil)
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("expected no requests for empty event name")
	}
}
