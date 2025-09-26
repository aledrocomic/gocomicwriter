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

func TestExportIssuePDF_CreatesFile(t *testing.T) {
	root := t.TempDir()
	// Minimal project with 1 issue, 1 page, 1 panel, 1 balloon
	proj := domain.Project{
		Name: "Test Project",
		Issues: []domain.Issue{
			{
				TrimWidth:  595, // A4-ish in pt (8.27in*72)
				TrimHeight: 842, // 11.69in*72
				Bleed:      18,  // 0.25in
				DPI:        300,
				Pages: []domain.Page{
					{
						Number: 1,
						Panels: []domain.Panel{
							{
								ID:       "p1",
								Geometry: domain.Rect{X: 36, Y: 36, Width: 523, Height: 770},
								Balloons: []domain.Balloon{
									{
										ID:    "b1",
										Type:  "speech",
										Shape: domain.Shape{Kind: "roundedBox", Rect: domain.Rect{X: 72, Y: 72, Width: 300, Height: 100}, Radius: 12},
										TextRuns: []domain.TextRun{
											{Content: "Hello, PDF!", Font: "Helvetica", Size: 14},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ph, err := storage.InitProject(root, proj)
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	out := filepath.Join(root, "exports", "issue-test.pdf")
	err = ExportIssuePDF(ph, 0, out, PDFOptions{IncludeGuides: true})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() <= 0 {
		t.Fatalf("pdf file empty")
	}
}
