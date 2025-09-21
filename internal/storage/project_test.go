package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gocomicwriter/internal/domain"
)

func TestInitProjectCreatesStructureAndManifest(t *testing.T) {
	root := t.TempDir()
	proj := domain.Project{Name: "Test Project", Issues: []domain.Issue{}}

	ph, err := InitProject(root, proj)
	if err != nil {
		t.Fatalf("InitProject error: %v", err)
	}
	if ph == nil {
		t.Fatalf("InitProject returned nil handle")
	}

	// Check manifest exists
	if ph.ManifestPath == "" {
		t.Fatalf("ManifestPath not set")
	}
	// Load manifest and compare
	b, err := os.ReadFile(ph.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var got domain.Project
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if got.Name != proj.Name {
		t.Fatalf("manifest name mismatch: got %q want %q", got.Name, proj.Name)
	}

	// Standard subdirs should exist
	wantDirs := []string{"script", "pages", "assets", "styles", "exports", BackupsDirName}
	for _, d := range wantDirs {
		p := filepath.Join(root, d)
		if fi, err := os.Stat(p); err != nil || !fi.IsDir() {
			t.Fatalf("expected directory %s to exist", p)
		}
	}
}

func TestSaveCreatesTimestampedBackup(t *testing.T) {
	root := t.TempDir()
	proj := domain.Project{Name: "Backup Test", Issues: []domain.Issue{}}
	ph, err := InitProject(root, proj)
	if err != nil {
		t.Fatalf("InitProject error: %v", err)
	}

	// Change something and save again to force a backup
	ph.Project.Metadata.Notes = "changed"
	if err := Save(ph); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Expect at least one .bak file under backups
	ents, err := os.ReadDir(filepath.Join(root, BackupsDirName))
	if err != nil {
		t.Fatalf("read backups dir: %v", err)
	}
	var bakCount int
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ManifestFileName+".") && strings.HasSuffix(name, ".bak") {
			bakCount++
		}
	}
	if bakCount == 0 {
		t.Fatalf("expected at least one backup file, found 0")
	}
}

func TestOpenFallsBackToLatestBackupOnCorruption(t *testing.T) {
	root := t.TempDir()
	proj := domain.Project{Name: "Open From Backup", Issues: []domain.Issue{}}
	ph, err := InitProject(root, proj)
	if err != nil {
		t.Fatalf("InitProject error: %v", err)
	}

	// Force a backup to exist by saving
	ph.Project.Metadata.Notes = "touch"
	if err := Save(ph); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Corrupt the manifest
	if err := os.WriteFile(ph.ManifestPath, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatalf("corrupt manifest: %v", err)
	}

	// Now opening should succeed via latest backup
	opened, err := Open(root)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	if opened.Project.Name != proj.Name {
		t.Fatalf("opened project name mismatch: got %q want %q", opened.Project.Name, proj.Name)
	}
}

func TestAutosaveCrashSnapshotWritesFile(t *testing.T) {
	root := t.TempDir()
	proj := domain.Project{Name: "Crash Snapshot", Issues: []domain.Issue{}}
	ph, err := InitProject(root, proj)
	if err != nil {
		t.Fatalf("InitProject error: %v", err)
	}

	path, err := AutosaveCrashSnapshot(ph)
	if err != nil {
		t.Fatalf("AutosaveCrashSnapshot error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("snapshot file does not exist: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var got domain.Project
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if got.Name != proj.Name {
		t.Fatalf("snapshot content mismatch: got %q want %q", got.Name, proj.Name)
	}
}
