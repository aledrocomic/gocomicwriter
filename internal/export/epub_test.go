/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package export

import (
	"archive/zip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

func TestExportIssueEPUB_Structure(t *testing.T) {
	root := t.TempDir()
	proj := domain.Project{
		Name:     "Test Project",
		Metadata: domain.Metadata{Series: "Series X", IssueTitle: "Pilot", Creators: "A. Writer"},
		Issues: []domain.Issue{{
			TrimWidth:        360,
			TrimHeight:       540,
			Bleed:            18,
			DPI:              96,
			ReadingDirection: "ltr",
			Pages: []domain.Page{{
				Number: 1,
				Panels: []domain.Panel{{
					ID:       "p1",
					Geometry: domain.Rect{X: 18, Y: 18, Width: 324, Height: 504},
					Balloons: []domain.Balloon{{
						ID:       "b1",
						Type:     "speech",
						Shape:    domain.Shape{Kind: "rect", Rect: domain.Rect{X: 40, Y: 40, Width: 220, Height: 80}},
						TextRuns: []domain.TextRun{{Content: "Hello, EPUB!", Font: "Helvetica", Size: 12}},
					}},
				}},
			}},
		}},
	}
	ph, err := storage.InitProject(root, proj)
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	out := filepath.Join(root, "exports", "issue-1.epub")
	if err := ExportIssueEPUB(ph, 0, out, EPUBOptions{IncludeGuides: true, Language: "en"}); err != nil {
		t.Fatalf("export epub: %v", err)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() <= 0 {
		t.Fatalf("epub empty")
	}

	rd, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = rd.Close() }()

	if len(rd.File) == 0 {
		t.Fatalf("zip has no entries")
	}
	if rd.File[0].Name != "mimetype" {
		t.Fatalf("first entry is not mimetype, got %s", rd.File[0].Name)
	}
	if rd.File[0].Method != zip.Store {
		t.Fatalf("mimetype is not stored (uncompressed)")
	}

	// presence checks
	want := map[string]bool{
		"META-INF/container.xml":  false,
		"OEBPS/content.opf":       false,
		"OEBPS/nav.xhtml":         false,
		"OEBPS/styles/epub.css":   false,
		"OEBPS/images/page-1.png": false,
		"OEBPS/page-1.xhtml":      false,
	}
	for _, f := range rd.File {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	}
	for name, ok := range want {
		if !ok {
			t.Fatalf("missing entry: %s", name)
		}
	}

	// Quick metadata checks in content.opf
	for _, f := range rd.File {
		if f.Name == "OEBPS/content.opf" {
			r, err := f.Open()
			if err != nil {
				t.Fatalf("open opf: %v", err)
			}
			data, err := io.ReadAll(r)
			_ = r.Close()
			if err != nil {
				t.Fatalf("read opf: %v", err)
			}
			txt := string(data)
			if !stringsContains(txt, "rendition:layout\">pre-paginated") {
				t.Fatalf("opf missing fixed-layout metadata: %s", txt)
			}
			if !stringsContains(txt, "page-progression-direction\"\u003e") && !stringsContains(txt, "page-progression-direction=\"ltr\"") {
				// The attribute should be present, simplest check is explicit ltr
				t.Fatalf("opf missing page-progression-direction: %s", txt)
			}
		}
	}
}

// Optional: run epubcheck if EPUBCHECK_JAR is set (path to epubcheck.jar) and Java is available.
func TestExportIssueEPUB_WithEpubCheck(t *testing.T) {
	jar := os.Getenv("EPUBCHECK_JAR")
	if jar == "" {
		t.Skip("EPUBCHECK_JAR not set; skipping epubcheck integration test")
	}
	if _, err := os.Stat(jar); err != nil {
		t.Skip("epubcheck jar missing; skipping")
	}
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("java not found; skipping")
	}
	root := t.TempDir()
	proj := domain.Project{Name: "Test", Issues: []domain.Issue{{TrimWidth: 360, TrimHeight: 540, Bleed: 18, DPI: 96, ReadingDirection: "ltr", Pages: []domain.Page{{Number: 1}}}}}
	ph, err := storage.InitProject(root, proj)
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	out := filepath.Join(root, "exports", "issue-1.epub")
	if err := ExportIssueEPUB(ph, 0, out, EPUBOptions{}); err != nil {
		t.Fatalf("export epub: %v", err)
	}
	cmd := exec.Command("java", "-jar", jar, out)
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("epubcheck failed: %v\nOutput:\n%s", err, string(outb))
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (func() bool { return (string([]byte(s)[:len(sub)]) == sub) || stringsContains(s[1:], sub) })()))
}
