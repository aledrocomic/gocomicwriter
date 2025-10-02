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
	"testing"
	"time"
)

// Use an unroutable address to trigger client.Do error path
func TestTelemetry_SendErrorBranches(t *testing.T) {
	cfg := Config{
		OptIn:        true,
		EventsURL:    "http://127.0.0.1:1/events",
		CrashURL:     "http://127.0.0.1:1/crash",
		Timeout:      50 * time.Millisecond,
		DebugLogging: true,
	}
	c := New(cfg)
	defer c.Close()

	// Should not panic when sending fails; just exercise the error/log paths
	c.Event("err", map[string]any{"a": 1})
	c.Flush(context.Background())
	c.UploadCrash([]byte("oops"))
	time.Sleep(50 * time.Millisecond)
}
