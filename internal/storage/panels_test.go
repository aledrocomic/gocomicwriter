/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"testing"

	"gocomicwriter/internal/domain"
)

func TestAddPanelAndOrdering(t *testing.T) {
	ph := &ProjectHandle{Project: domain.Project{Name: "Test"}}
	// Ensure page 1 exists
	pg, err := EnsurePage(ph, 1)
	if err != nil {
		t.Fatalf("EnsurePage error: %v", err)
	}
	if pg.Number != 1 {
		t.Fatalf("expected page 1, got %d", pg.Number)
	}

	// Add three panels
	p1, err := AddPanel(ph, 1, domain.Panel{})
	if err != nil {
		t.Fatalf("AddPanel p1: %v", err)
	}
	p2, err := AddPanel(ph, 1, domain.Panel{})
	if err != nil {
		t.Fatalf("AddPanel p2: %v", err)
	}
	p3, err := AddPanel(ph, 1, domain.Panel{ID: "custom"})
	if err != nil {
		t.Fatalf("AddPanel p3: %v", err)
	}
	if p1.ZOrder != 0 || p2.ZOrder != 1 || p3.ZOrder != 2 {
		t.Fatalf("unexpected zOrders: p1=%d p2=%d p3=%d", p1.ZOrder, p2.ZOrder, p3.ZOrder)
	}

	// Try duplicate ID
	if _, err := AddPanel(ph, 1, domain.Panel{ID: p1.ID}); err == nil {
		t.Fatalf("expected duplicate id error")
	}

	// Move middle (p2) up to top
	if err := MovePanelZ(ph, 1, p2.ID, +1); err != nil {
		t.Fatalf("MovePanelZ up: %v", err)
	}
	// After move, re-check ordering
	pg2, err := EnsurePage(ph, 1)
	if err != nil {
		t.Fatalf("EnsurePage after move: %v", err)
	}
	if len(pg2.Panels) != 3 {
		t.Fatalf("expected 3 panels, got %d", len(pg2.Panels))
	}
	if !(pg2.Panels[2].ID == p2.ID && pg2.Panels[2].ZOrder == 2) {
		t.Fatalf("expected %s to be top (z=2), got %+v", p2.ID, pg2.Panels[2])
	}

	// Move top down beyond bottom (no change expected)
	if err := MovePanelZ(ph, 1, p2.ID, +10); err != nil {
		t.Fatalf("MovePanelZ out-of-range: %v", err)
	}
	pg3, _ := EnsurePage(ph, 1)
	if pg3.Panels[2].ID != p2.ID {
		t.Fatalf("expected still top: %+v", pg3.Panels)
	}
}

func TestUpdatePanelMeta(t *testing.T) {
	ph := &ProjectHandle{Project: domain.Project{Name: "Test", Issues: []domain.Issue{{Pages: []domain.Page{{
		Number: 1,
		Panels: []domain.Panel{{ID: "p1", ZOrder: 0}, {ID: "p2", ZOrder: 1}},
	}}}}}}
	// Rename p1 to pA and set notes
	if err := UpdatePanelMeta(ph, 1, "p1", "pA", "first panel"); err != nil {
		t.Fatalf("UpdatePanelMeta: %v", err)
	}
	pg := ph.Project.Issues[0].Pages[0]
	if pg.Panels[0].ID != "pA" || pg.Panels[0].Notes != "first panel" {
		t.Fatalf("unexpected panel meta: %+v", pg.Panels[0])
	}
	// Renaming to duplicate should error
	if err := UpdatePanelMeta(ph, 1, "pA", "p2", ""); err == nil {
		t.Fatalf("expected duplicate rename error")
	}
}
