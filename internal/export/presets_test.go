/*
 * Copyright (c) 2025
 */
package export

import (
	"os"
	"path/filepath"
	"testing"

	"gocomicwriter/internal/storage"
)

func TestBatchExport_WebPreset(t *testing.T) {
	root := t.TempDir()
	ph, err := storage.InitProject(root, sampleProject())
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	if err := BatchExport(ph, BatchOptions{Preset: PresetWeb}); err != nil {
		t.Fatalf("batch export web: %v", err)
	}
	checks := []string{
		filepath.Join(root, "exports", "web", "png", "issue-1-page-1.png"),
		filepath.Join(root, "exports", "web", "svg", "issue-1-page-1.svg"),
		filepath.Join(root, "exports", "web", "cbz", "issue-1.cbz"),
	}
	for _, p := range checks {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
		if st.Size() <= 0 {
			t.Fatalf("empty file: %s", p)
		}
	}
}

func TestBatchExport_PrintPreset(t *testing.T) {
	root := t.TempDir()
	ph, err := storage.InitProject(root, sampleProject())
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	if err := BatchExport(ph, BatchOptions{Preset: PresetPrint}); err != nil {
		t.Fatalf("batch export print: %v", err)
	}
	checks := []string{
		filepath.Join(root, "exports", "print", "pdf", "issue-1.pdf"),
		filepath.Join(root, "exports", "print", "png", "issue-1-page-1.png"),
	}
	for _, p := range checks {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
		if st.Size() <= 0 {
			t.Fatalf("empty file: %s", p)
		}
	}
}
