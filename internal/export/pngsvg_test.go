/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package export

import (
	"os"
	"path/filepath"
	"testing"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

func sampleProject() domain.Project {
	return domain.Project{
		Name: "Test Project",
		Issues: []domain.Issue{{
			TrimWidth:  360,
			TrimHeight: 540,
			Bleed:      18,
			DPI:        150,
			Pages: []domain.Page{{
				Number: 1,
				Panels: []domain.Panel{{
					ID:       "p1",
					Geometry: domain.Rect{X: 18, Y: 18, Width: 324, Height: 504},
					Balloons: []domain.Balloon{{
						ID:       "b1",
						Type:     "speech",
						Shape:    domain.Shape{Kind: "rect", Rect: domain.Rect{X: 40, Y: 40, Width: 220, Height: 80}},
						TextRuns: []domain.TextRun{{Content: "Hello, raster!", Font: "Helvetica", Size: 12}},
					}},
				}},
			}},
		}},
	}
}

func TestExportIssuePNGPages(t *testing.T) {
	root := t.TempDir()
	proj := sampleProject()
	ph, err := storage.InitProject(root, proj)
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	outDir := filepath.Join(root, "exports", "pngtest")
	if err := ExportIssuePNGPages(ph, 0, outDir, PNGOptions{IncludeGuides: true, DPI: 96}); err != nil {
		t.Fatalf("export png: %v", err)
	}
	path := filepath.Join(outDir, "issue-1-page-1.png")
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() <= 0 {
		t.Fatalf("png empty")
	}
}

func TestExportIssueSVGPages(t *testing.T) {
	root := t.TempDir()
	proj := sampleProject()
	ph, err := storage.InitProject(root, proj)
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	outDir := filepath.Join(root, "exports", "svgtest")
	if err := ExportIssueSVGPages(ph, 0, outDir, SVGOptions{IncludeGuides: true, DPI: 144}); err != nil {
		t.Fatalf("export svg: %v", err)
	}
	path := filepath.Join(outDir, "issue-1-page-1.svg")
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() <= 0 {
		t.Fatalf("svg empty")
	}
}
