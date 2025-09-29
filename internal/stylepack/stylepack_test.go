/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package stylepack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExportAndInstallPack(t *testing.T) {
	// Create temp project with styles
	projDir := t.TempDir()
	stylesDir := filepath.Join(projDir, "styles")
	if err := os.MkdirAll(stylesDir, 0o755); err != nil {
		t.Fatalf("mkdir styles: %v", err)
	}
	// Create some files and subdirs
	if err := os.WriteFile(filepath.Join(stylesDir, "dialogue.json"), []byte("{\n  \"name\": \"Dialogue\"\n}"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	sub := filepath.Join(stylesDir, "templates")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "grid-3x3.txt"), []byte("3x3"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	// Export pack
	zipPath := filepath.Join(projDir, "out.zip")
	if err := ExportProjectStyles(projDir, zipPath); err != nil {
		t.Fatalf("export pack: %v", err)
	}
	// Basic check: zip exists and has entries
	st, err := os.Stat(zipPath)
	if err != nil || st.Size() == 0 {
		t.Fatalf("zip not created or empty: %v", err)
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	_ = r.Close()

	// Install into a new project
	proj2 := t.TempDir()
	installed, err := InstallPack(proj2, zipPath)
	if err != nil {
		t.Fatalf("install pack: %v", err)
	}
	if installed == 0 {
		t.Fatalf("expected installed > 0")
	}
	// Files should exist under proj2/styles
	if _, err := os.Stat(filepath.Join(proj2, "styles", "dialogue.json")); err != nil {
		t.Fatalf("expected dialogue.json installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(proj2, "styles", "templates", "grid-3x3.txt")); err != nil {
		t.Fatalf("expected template installed: %v", err)
	}
}
