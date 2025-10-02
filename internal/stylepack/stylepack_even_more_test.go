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

func TestInstallPack_InstallsNonStylesPrefixAndDirectoryEntries(t *testing.T) {
	proj := t.TempDir()
	zpath := filepath.Join(proj, "pack2.zip")
	f, err := os.Create(zpath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)

	// Directory entry
	dh := &zip.FileHeader{Name: "styles/subdir/"}
	dh.SetMode(os.ModeDir | 0o755)
	if _, err := zw.CreateHeader(dh); err != nil {
		t.Fatalf("create dir header: %v", err)
	}

	// Non-styles entry should be prefixed by installer under styles/
	w, _ := zw.Create("top/inner.txt")
	_, _ = w.Write([]byte("content"))

	_ = zw.Close()
	_ = f.Close()

	installed, err := InstallPack(proj, zpath)
	if err != nil {
		t.Fatalf("install pack: %v", err)
	}
	if installed != 1 { // only the file counts, directory entry doesn't
		t.Fatalf("expected installed=1, got %d", installed)
	}
	// Verify installed under styles/top/inner.txt
	if _, err := os.Stat(filepath.Join(proj, "styles", "top", "inner.txt")); err != nil {
		t.Fatalf("expected installed file under styles/top: %v", err)
	}
}
