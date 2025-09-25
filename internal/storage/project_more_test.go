/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gocomicwriter/internal/domain"
)

func TestSaveAsAndScriptIO(t *testing.T) {
	root := t.TempDir()
	ph, err := InitProject(root, domain.Project{Name: "Orig"})
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Change project and SaveAs to new root
	ph.Project.Name = "Renamed"
	newRoot := filepath.Join(root, "newproj")
	if err := SaveAs(ph, newRoot); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	if ph.Root != newRoot || ph.ManifestPath != filepath.Join(newRoot, ManifestFileName) {
		t.Fatalf("ProjectHandle paths not updated: %+v", ph)
	}

	// Manifest at new location should reflect updated name
	b, err := os.ReadFile(ph.ManifestPath)
	if err != nil {
		t.Fatalf("read new manifest: %v", err)
	}
	var got domain.Project
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal new manifest: %v", err)
	}
	if got.Name != "Renamed" {
		t.Fatalf("unexpected project name in new manifest: %q", got.Name)
	}

	// ScriptFilePath should point under script folder
	sp := ScriptFilePath(ph)
	if filepath.Dir(sp) != filepath.Join(newRoot, "script") {
		t.Fatalf("script path dir mismatch: %q", sp)
	}

	// ReadScript should be empty when missing
	txt, err := ReadScript(ph)
	if err != nil || txt != "" {
		t.Fatalf("expected empty script, got %q err=%v", txt, err)
	}

	// WriteScript then read back
	content := "PAGE 1\nPanel 1: ..."
	if err := WriteScript(ph, content); err != nil {
		t.Fatalf("WriteScript: %v", err)
	}
	txt, err = ReadScript(ph)
	if err != nil || txt != content {
		t.Fatalf("ReadScript mismatch: %q err=%v", txt, err)
	}
}
