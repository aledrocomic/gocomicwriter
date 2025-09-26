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
	"path/filepath"
	"testing"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

func TestExportIssueCBZ(t *testing.T) {
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
						TextRuns: []domain.TextRun{{Content: "Hello, CBZ!", Font: "Helvetica", Size: 12}},
					}},
				}},
			}},
		}},
	}
	ph, err := storage.InitProject(root, proj)
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	out := filepath.Join(root, "exports", "issue-1.cbz")
	if err := ExportIssueCBZ(ph, 0, out, CBZOptions{IncludeGuides: true, DPI: 96}); err != nil {
		t.Fatalf("export cbz: %v", err)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() <= 0 {
		t.Fatalf("cbz empty")
	}

	// Open zip and check entries
	rd, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = rd.Close() }()
	var foundPNG, foundXML bool
	for _, f := range rd.File {
		if f.Name == "1.png" || f.Name == "01.png" || f.Name == "001.png" {
			foundPNG = true
		}
		if f.Name == "ComicInfo.xml" {
			foundXML = true
		}
	}
	if !foundPNG {
		t.Fatalf("png page not found in zip")
	}
	if !foundXML {
		t.Fatalf("ComicInfo.xml not found in zip")
	}

	// Read manifest to ensure fields exist
	for _, f := range rd.File {
		if f.Name == "ComicInfo.xml" {
			r, err := f.Open()
			if err != nil {
				t.Fatalf("open manifest: %v", err)
			}
			data, err := io.ReadAll(r)
			if err != nil {
				_ = r.Close()
				t.Fatalf("read manifest: %v", err)
			}
			if err := r.Close(); err != nil {
				t.Fatalf("close manifest: %v", err)
			}
			text := string(data)
			if !(contains(text, "<Series>Series X</Series>") && contains(text, "<Title>Pilot</Title>") && contains(text, "<Number>1</Number>")) {
				t.Fatalf("manifest missing expected fields: %s", text)
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (func() bool { return (string([]byte(s)[:len(sub)]) == sub) || contains(s[1:], sub) })()))
}
