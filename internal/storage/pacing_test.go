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
	"testing"
)

func TestComputeBeatCoverageAndTurns(t *testing.T) {
	proj := domain.Project{
		Name: "Test",
		Issues: []domain.Issue{{
			ReadingDirection: "ltr",
			Pages: []domain.Page{
				{Number: 1, Panels: []domain.Panel{
					{ID: "p1", ZOrder: 0, BeatIDs: []string{"b:1", "b:2"}},
					{ID: "p2", ZOrder: 1},
				}},
				{Number: 2, Panels: []domain.Panel{
					{ID: "p3", ZOrder: 0},
					{ID: "p4", ZOrder: 1, BeatIDs: []string{"b:3"}},
				}},
			},
		}},
	}

	cov := ComputeBeatCoverage(proj)
	if len(cov) != 2 {
		t.Fatalf("expected 2 coverage entries, got %d", len(cov))
	}
	if cov[0].PageNumber != 1 || cov[0].TotalBeats != 2 {
		t.Fatalf("unexpected page 1 coverage: %+v", cov[0])
	}
	if cov[1].PageNumber != 2 || cov[1].TotalBeats != 1 {
		t.Fatalf("unexpected page 2 coverage: %+v", cov[1])
	}
	if cov[0].PanelBeatCounts["p1"] != 2 || cov[0].PanelBeatCounts["p2"] != 0 {
		t.Fatalf("unexpected per-panel counts page1: %+v", cov[0].PanelBeatCounts)
	}
	if cov[1].PanelBeatCounts["p3"] != 0 || cov[1].PanelBeatCounts["p4"] != 1 {
		t.Fatalf("unexpected per-panel counts page2: %+v", cov[1].PanelBeatCounts)
	}

	turns := ComputePageTurnIndicators(proj.Issues[0])
	if len(turns) != 2 {
		t.Fatalf("expected 2 turn entries, got %d", len(turns))
	}
	if !turns[0].IsTurn {
		t.Fatalf("expected page 1 to be a turn in LTR")
	}
	if turns[0].LastPanelHasBeats {
		t.Fatalf("expected page 1 last panel to have no beats")
	}
	if !turns[1].HasBeats || !turns[1].LastPanelHasBeats {
		t.Fatalf("expected page 2 to have beats on last panel")
	}
}
