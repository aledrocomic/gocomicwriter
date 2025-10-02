/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

// Package crash /*
package crash

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	applog "gocomicwriter/internal/log"
	"gocomicwriter/internal/storage"
	"gocomicwriter/internal/telemetry"
	"gocomicwriter/internal/version"
)

// exitFn is used to allow testing of Recover without terminating the test process.
var exitFn = os.Exit

// Recover captures a panic, logs an error with stacktrace,
// writes an error report file, and attempts a crash-safe autosave
// of the project manifest (if provided).
//
// Usage: defer func(){ crash.Recover(ph) }()
func Recover(ph *storage.ProjectHandle) {
	if r := recover(); r != nil {
		l := applog.WithComponent("crash")
		stack := debug.Stack()
		l.Error("panic recovered", slog.Any("panic", r), slog.String("stack", string(stack)))

		reportPath, _ := writeReport(ph, r, stack)
		if ph != nil {
			if path, err := storage.AutosaveCrashSnapshot(ph); err != nil {
				l.Error("autosave crash snapshot failed", slog.Any("err", err))
			} else {
				l.Info("autosave crash snapshot written", slog.String("path", path))
			}
		}

		if _, err := fmt.Fprintf(os.Stderr, "A fatal error occurred. A crash report was saved to: %s\n", reportPath); err != nil {
			l.Error("failed to write crash message to stderr", slog.Any("err", err))
		}
		if _, err := fmt.Fprintf(os.Stderr, "Version: %s\nOS/Arch: %s/%s\n", version.String(), runtime.GOOS, runtime.GOARCH); err != nil {
			l.Error("failed to write version info to stderr", slog.Any("err", err))
		}
		// Exit with a non-zero code to indicate failure in CLI context.
		exitFn(2)
	}
}

func writeReport(ph *storage.ProjectHandle, panicVal any, stack []byte) (string, error) {
	dir := os.TempDir()
	if ph != nil && ph.Root != "" {
		dir = filepath.Join(ph.Root, storage.BackupsDirName)
		_ = os.MkdirAll(dir, 0o755)
	}
	stamp := time.Now().Format("20060102-150405")
	fname := fmt.Sprintf("crash-%s.log", stamp)
	path := filepath.Join(dir, fname)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return path, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			applog.WithComponent("crash").Error("failed to close crash report file", slog.Any("err", err), slog.String("path", path))
		}
	}()

	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "Go Comic Writer Crash Report\n")
	_, _ = fmt.Fprintf(&buf, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	_, _ = fmt.Fprintf(&buf, "Version: %s\n", version.String())
	_, _ = fmt.Fprintf(&buf, "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	if ph != nil {
		_, _ = fmt.Fprintf(&buf, "ProjectRoot: %s\n", ph.Root)
		_, _ = fmt.Fprintf(&buf, "Manifest: %s\n", ph.ManifestPath)
	}
	_, _ = fmt.Fprintf(&buf, "\nPanic: %v\n\n", panicVal)
	_, _ = fmt.Fprintf(&buf, "Stack:\n%s\n", string(stack))

	// write to file
	if _, err := f.Write(buf.Bytes()); err != nil {
		return path, err
	}
	_ = f.Sync()

	// optionally upload anonymized crash report (opt-in via env)
	telemetry.UploadCrash(buf.Bytes())
	return path, nil
}
