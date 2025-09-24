/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package log

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFromEnvAndGetenv(t *testing.T) {
	t.Setenv("GCW_LOG_LEVEL", "warn")
	t.Setenv("GCW_LOG_FORMAT", "json")
	t.Setenv("GCW_LOG_SOURCE", "true")
	// GCW_LOG_FILE intentionally unset

	opts := FromEnv()
	if opts.Level != "warn" || opts.Format != "json" || !opts.AddSource || opts.File != "" {
		t.Fatalf("FromEnv mismatch: %+v", opts)
	}

	// Also verify getenv default fallback when var missing
	if err := os.Unsetenv("SOME_UNSET_VAR"); err != nil {
		t.Fatalf("Unsetenv error: %v", err)
	}
	if v := getenv("SOME_UNSET_VAR", "fallback"); v != "fallback" {
		t.Fatalf("getenv fallback failed: %q", v)
	}
}

func TestPrettyTextHandler_Behavior(t *testing.T) {
	// Capture output into a buffer
	var buf bytes.Buffer
	h := &prettyTextHandler{opts: prettyOpts{Level: slog.LevelWarn, AddSource: true}, w: &buf}

	// Enabled should filter below WARN
	if h.Enabled(nil, slog.LevelInfo) {
		t.Fatalf("info should not be enabled at warn level")
	}
	if !h.Enabled(nil, slog.LevelError) {
		t.Fatalf("error should be enabled at warn level")
	}

	// WithAttrs and WithGroup should accumulate
	h2 := h.WithAttrs([]slog.Attr{slog.String("k", "v")})
	h2 = h2.WithGroup("grp")

	// Build a record and handle it
	r := slog.Record{Time: time.Now(), Level: slog.LevelError, Message: "boom"}
	r.AddAttrs(slog.Int("n", 42), slog.Float64("pi", 3.14), slog.Bool("ok", true))
	if err := h2.Handle(nil, r); err != nil {
		t.Fatalf("handle error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "boom") || !strings.Contains(out, "k=v") {
		t.Fatalf("output missing expected content: %q", out)
	}
	// Grouped key should appear as prefix
	if !strings.Contains(out, "grp.n=42") {
		t.Fatalf("grouped attr missing or malformed: %q", out)
	}

	// Spot check level and value stringers
	if !strings.Contains(out, "ERR") { // levelString
		t.Fatalf("expected ERR level tag in output: %q", out)
	}
	if !strings.Contains(out, "pi=3.14") { // attrValueString float trim
		t.Fatalf("expected trimmed float: %q", out)
	}
}
