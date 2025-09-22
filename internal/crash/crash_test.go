/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package crash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gocomicwriter/internal/storage"
)

func TestWriteReportCreatesFileInTemp(t *testing.T) {
	path, err := writeReport(nil, "boom", []byte("stacktrace"))
	if err != nil {
		t.Fatalf("writeReport error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("report file missing: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "Go Comic Writer Crash Report") {
		t.Fatalf("report header missing")
	}
	if !strings.Contains(s, "Panic: boom") {
		t.Fatalf("panic content missing: %s", s)
	}
}

func TestWriteReportCreatesFileInProjectBackups(t *testing.T) {
	root := t.TempDir()
	ph := &storage.ProjectHandle{Root: root, ManifestPath: filepath.Join(root, storage.ManifestFileName)}

	path, err := writeReport(ph, "kaboom", []byte("stack"))
	if err != nil {
		t.Fatalf("writeReport error: %v", err)
	}
	if !strings.Contains(path, filepath.Join(root, storage.BackupsDirName)) {
		t.Fatalf("expected crash report under backups dir, got %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("report file missing: %v", err)
	}
}
