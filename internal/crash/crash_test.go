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
