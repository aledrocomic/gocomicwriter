/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScriptFilePath_NilHandle(t *testing.T) {
	if p := ScriptFilePath(nil); p != "" {
		t.Fatalf("expected empty path for nil handle, got %q", p)
	}
}

func TestReadScript_MissingReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	ph := &ProjectHandle{Root: root, ManifestPath: filepath.Join(root, ManifestFileName)}
	s, err := ReadScript(ph)
	if err != nil {
		t.Fatalf("ReadScript unexpected error for missing file: %v", err)
	}
	if s != "" {
		t.Fatalf("expected empty string for missing script, got %q", s)
	}
}

func TestWriteScript_AndReadBack(t *testing.T) {
	root := t.TempDir()
	ph := &ProjectHandle{Root: root, ManifestPath: filepath.Join(root, ManifestFileName)}

	text := "INT. LAB - DAY\nThe experiment begins."
	if err := WriteScript(ph, text); err != nil {
		t.Fatalf("WriteScript error: %v", err)
	}
	// Verify file exists at expected location
	p := ScriptFilePath(ph)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected script file to exist at %s: %v", p, err)
	}
	// Read back and compare
	got, err := ReadScript(ph)
	if err != nil {
		t.Fatalf("ReadScript error: %v", err)
	}
	if got != text {
		t.Fatalf("roundtrip mismatch: %q vs %q", got, text)
	}
}
