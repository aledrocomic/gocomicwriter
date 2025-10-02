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

func TestExportProjectStyles_ErrorArgsAndEmptyDir(t *testing.T) {
	if err := ExportProjectStyles("", ""); err == nil {
		t.Fatalf("expected error on empty args")
	}
	proj := t.TempDir()
	zipPath := filepath.Join(proj, "only_manifest.zip")
	// styles dir does not exist; function should create it and still produce a zip with manifest
	if err := ExportProjectStyles(proj, zipPath); err != nil {
		t.Fatalf("export empty styles: %v", err)
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()
	foundManifest := false
	for _, f := range r.File {
		if f.Name == "stylepack.manifest.txt" {
			foundManifest = true
			break
		}
	}
	if !foundManifest {
		t.Fatalf("manifest not found in zip")
	}
}

func TestInstallPack_ZipSlipAndSkipExisting(t *testing.T) {
	// Build a zip with a malicious entry and a good entry
	proj := t.TempDir()
	zpath := filepath.Join(proj, "pack.zip")
	f, err := os.Create(zpath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	// Malicious entry
	w, err := zw.Create("../evil.txt")
	if err != nil {
		t.Fatalf("create malicious zip entry: %v", err)
	}
	if _, err := w.Write([]byte("nope")); err != nil {
		t.Fatalf("write malicious entry: %v", err)
	}
	// Good entry under styles/
	w2, err := zw.Create("styles/good.txt")
	if err != nil {
		t.Fatalf("create good zip entry: %v", err)
	}
	if _, err := w2.Write([]byte("ok")); err != nil {
		t.Fatalf("write good entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	// Pre-create an existing file to test skip-existing
	target := filepath.Join(proj, "styles", "good.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir styles dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("precreate file: %v", err)
	}

	installed, err := InstallPack(proj, zpath)
	if err != nil {
		t.Fatalf("install pack: %v", err)
	}
	// Should skip existing file, and malicious should be ignored => nothing installed
	if installed != 0 {
		t.Fatalf("expected 0 installed due to skip+malicious, got %d", installed)
	}
	// Ensure no evil file was written outside styles
	if _, err := os.Stat(filepath.Join(proj, "evil.txt")); err == nil {
		t.Fatalf("evil.txt should not exist")
	}
}
