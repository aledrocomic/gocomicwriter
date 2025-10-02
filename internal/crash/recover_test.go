/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package crash

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gocomicwriter/internal/storage"
)

// TestRecover_PanickingGoroutine ensures Recover handles a panic, writes a report,
// attempts autosave, and does not terminate the test process due to injected exitFn.
func TestRecover_PanickingGoroutine(t *testing.T) {
	// Capture stderr temporarily to avoid noisy test logs
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		_ = w.Close()
		os.Stderr = oldStderr
		_, _ = io.Copy(os.Stderr, r) // drain pipe
	}()

	// Override exitFn to avoid os.Exit during test and to assert it was called
	called := 0
	oldExit := exitFn
	exitFn = func(code int) { called = code }
	defer func() { exitFn = oldExit }()

	root := t.TempDir()
	ph := &storage.ProjectHandle{Root: root, ManifestPath: filepath.Join(root, storage.ManifestFileName)}

	// Trigger a panic that Recover will catch
	func() {
		defer Recover(ph)
		panic("boom")
	}()

	// Allow time for filesystem writes
	time.Sleep(50 * time.Millisecond)

	// Verify a crash report exists either under temp or backups
	var found string
	entries, _ := os.ReadDir(root)
	_ = entries
	// crash report is in backups dir for project handle
	bdir := filepath.Join(root, storage.BackupsDirName)
	files, _ := os.ReadDir(bdir)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "crash-") && strings.HasSuffix(f.Name(), ".log") {
			found = filepath.Join(bdir, f.Name())
			break
		}
	}
	if found == "" {
		t.Fatalf("expected crash report file under backups dir")
	}
	b, err := os.ReadFile(found)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !bytes.Contains(b, []byte("Panic: boom")) {
		t.Fatalf("report does not contain panic: %s", string(b))
	}

	// Ensure exit was attempted with code 2 (but intercepted)
	if called != 2 {
		t.Fatalf("expected exit code 2, got %d", called)
	}
}
