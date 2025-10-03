/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/script"
	"testing"
)

func TestComputeUnmappedBeats(t *testing.T) {
	// Prepare a script with two beats and one dialogue
	txt := `# Scene One
Beat Establishing shot @intro
ALICE: Hello there
Beat Close-up @drama`
	sc, errs := script.Parse(txt)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %+v", errs)
	}

	// Build a project where only the first beat is mapped
	p := domain.Project{
		Name: "Test",
		Issues: []domain.Issue{{
			Pages: []domain.Page{{
				Number: 1,
				Panels: []domain.Panel{{ID: "p1"}},
			}},
		}},
	}

	// Map first beat to panel p1
	// Beat IDs are based on source line number: find first beat
	var firstBeatID string
	for _, scn := range sc.Scenes {
		for _, ln := range scn.Lines {
			if ln.Type == script.LineBeat {
				firstBeatID = BeatIDFor(ln)
				break
			}
		}
	}
	if firstBeatID == "" {
		t.Fatalf("failed to locate first beat id")
	}

	// mutate project by mapping
	ph := &ProjectHandle{Project: p}
	if err := MapBeatToPanel(ph, 1, "p1", firstBeatID); err != nil {
		t.Fatalf("MapBeatToPanel error: %v", err)
	}

	// Now compute unmapped: should be exactly 1 (the second beat)
	unmapped := ComputeUnmappedBeats(sc, ph.Project)
	if len(unmapped) != 1 {
		t.Fatalf("expected 1 unmapped beat, got %d (ids=%v)", len(unmapped), unmapped)
	}
}

func TestMapBeatToPanelAddsUnique(t *testing.T) {
	ph := &ProjectHandle{Project: domain.Project{
		Name: "Test",
		Issues: []domain.Issue{{
			Pages: []domain.Page{{
				Number: 2,
				Panels: []domain.Panel{{ID: "pA"}},
			}},
		}},
	}}

	// Add mapping
	if err := MapBeatToPanel(ph, 2, "pA", "b:100"); err != nil {
		t.Fatalf("MapBeatToPanel error: %v", err)
	}
	// Add again should be no-op
	if err := MapBeatToPanel(ph, 2, "pA", "b:100"); err != nil {
		t.Fatalf("MapBeatToPanel duplicate error: %v", err)
	}

	// Verify only one entry exists
	var got []string
	for _, iss := range ph.Project.Issues {
		for _, pg := range iss.Pages {
			if pg.Number != 2 {
				continue
			}
			for _, pn := range pg.Panels {
				if pn.ID == "pA" {
					got = pn.BeatIDs
				}
			}
		}
	}
	if len(got) != 1 || got[0] != "b:100" {
		t.Fatalf("unexpected mapping content: %+v", got)
	}
}
