/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package log

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestInitAndStructuredLoggingToFile verifies that Init with a file handler writes JSON logs
// and that static and contextual attributes are present.
func TestInitAndStructuredLoggingToFile(t *testing.T) {
	// Use a file in the system temp dir to avoid Windows deleting a still-open handle
	fpath := filepath.Join(os.TempDir(), fmt.Sprintf("gcw_log_%d.json", time.Now().UnixNano()))

	// Init logger with JSON format and file handler
	Init(Options{Level: "debug", Format: "json", File: fpath})

	l := WithComponent("testcomp")
	l = WithOperation(l, "op1")
	l.Info("hello world", slog.String("k", "v"))

	// Give a brief moment for the async filesystem to settle (Windows)
	time.Sleep(50 * time.Millisecond)

	// Ensure file exists and has content.
	b, err := os.ReadFile(fpath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("log file is empty")
	}

	// Parse last non-empty line as JSON and assert fields
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	var last string
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		if s != "" {
			last = s
		}
	}
	if last == "" {
		t.Fatalf("no log lines found")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(last), &m); err != nil {
		t.Fatalf("unmarshal json log: %v", err)
	}

	// Assert static attrs
	if m["app"] != "gocomicwriter" {
		t.Fatalf("missing app attr: %v", m["app"])
	}
	if _, ok := m["ver"].(string); !ok {
		t.Fatalf("missing ver attr")
	}
	// Assert context attrs
	if m["component"] != "testcomp" {
		t.Fatalf("component attr mismatch: %v", m["component"])
	}
	if m["op"] != "op1" {
		t.Fatalf("op attr mismatch: %v", m["op"])
	}
	if m["msg"] != "hello world" {
		t.Fatalf("msg mismatch: %v", m["msg"])
	}
}
