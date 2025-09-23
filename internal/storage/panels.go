/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"fmt"
	"gocomicwriter/internal/domain"
	"sort"
)

// EnsurePage returns a pointer to a page with the given number, creating it if it does not exist yet.
// New pages are appended with empty panel list.
func EnsurePage(ph *ProjectHandle, pageNumber int) (*domain.Page, error) {
	if ph == nil {
		return nil, fmt.Errorf("project handle is nil")
	}
	if pageNumber <= 0 {
		return nil, fmt.Errorf("pageNumber must be >= 1")
	}
	// For now we operate on the first issue only (early milestone scope)
	if len(ph.Project.Issues) == 0 {
		ph.Project.Issues = []domain.Issue{{Pages: []domain.Page{}}}
	}
	iss := &ph.Project.Issues[0]
	for i := range iss.Pages {
		if iss.Pages[i].Number == pageNumber {
			return &iss.Pages[i], nil
		}
	}
	// Create new page
	pg := domain.Page{Number: pageNumber, Panels: []domain.Panel{}}
	iss.Pages = append(iss.Pages, pg)
	// Keep pages sorted by number
	sort.Slice(iss.Pages, func(i, j int) bool { return iss.Pages[i].Number < iss.Pages[j].Number })
	// Return pointer to the (potentially moved) page
	for i := range iss.Pages {
		if iss.Pages[i].Number == pageNumber {
			return &iss.Pages[i], nil
		}
	}
	return nil, fmt.Errorf("failed to create page %d", pageNumber)
}

// NextPanelID returns a unique panel ID like "p1", "p2", ... not used on the given page.
func NextPanelID(pg *domain.Page) string {
	if pg == nil {
		return "p1"
	}
	maxN := 0
	exists := map[string]struct{}{}
	for _, p := range pg.Panels {
		exists[p.ID] = struct{}{}
		var n int
		if _, err := fmt.Sscanf(p.ID, "p%d", &n); err == nil {
			if n > maxN {
				maxN = n
			}
		}
	}
	for n := maxN + 1; n < maxN+10000; n++ {
		id := fmt.Sprintf("p%d", n)
		if _, ok := exists[id]; !ok {
			return id
		}
	}
	return fmt.Sprintf("p%d", maxN+1)
}

// AddPanel creates a new panel on the given page with default geometry if zero and assigns a zOrder after the last.
// If panel.ID is empty, a unique one will be generated. Returns the created panel.
func AddPanel(ph *ProjectHandle, pageNumber int, panel domain.Panel) (domain.Panel, error) {
	pg, err := EnsurePage(ph, pageNumber)
	if err != nil {
		return domain.Panel{}, err
	}
	if panel.ID == "" {
		panel.ID = NextPanelID(pg)
	} else {
		// ensure unique
		for _, p := range pg.Panels {
			if p.ID == panel.ID {
				return domain.Panel{}, fmt.Errorf("panel id %s already exists on page %d", panel.ID, pageNumber)
			}
		}
	}
	// default geometry: center 120x80 inside page trim approximation; we don't know page size here, keep simple
	if panel.Geometry.Width <= 0 || panel.Geometry.Height <= 0 {
		panel.Geometry = domain.Rect{X: 50, Y: 50, Width: 120, Height: 80}
	}
	// zOrder: max+1
	maxZ := -1
	for _, p := range pg.Panels {
		if p.ZOrder > maxZ {
			maxZ = p.ZOrder
		}
	}
	panel.ZOrder = maxZ + 1
	pg.Panels = append(pg.Panels, panel)
	return panel, nil
}

// findPanel returns page pointer, panel index and pointer, or error.
func findPanel(ph *ProjectHandle, pageNumber int, panelID string) (*domain.Page, int, *domain.Panel, error) {
	if ph == nil {
		return nil, -1, nil, fmt.Errorf("project handle is nil")
	}
	for i := range ph.Project.Issues {
		iss := &ph.Project.Issues[i]
		for j := range iss.Pages {
			pg := &iss.Pages[j]
			if pg.Number != pageNumber {
				continue
			}
			for k := range pg.Panels {
				if pg.Panels[k].ID == panelID {
					return pg, k, &pg.Panels[k], nil
				}
			}
			return pg, -1, nil, fmt.Errorf("panel %s not found on page %d", panelID, pageNumber)
		}
	}
	return nil, -1, nil, fmt.Errorf("page %d not found", pageNumber)
}

// MovePanelZ moves the panel up or down in zOrder by delta (+1 moves up/top, -1 moves down/back).
// It adjusts other panels' zOrder to keep a dense sequence starting at 0, then resorts slice by zOrder.
func MovePanelZ(ph *ProjectHandle, pageNumber int, panelID string, delta int) error {
	pg, _, pn, err := findPanel(ph, pageNumber, panelID)
	if err != nil {
		return err
	}
	// Build order list
	order := make([]*domain.Panel, len(pg.Panels))
	for i := range pg.Panels {
		order[i] = &pg.Panels[i]
	}
	// sort by z
	sort.Slice(order, func(i, j int) bool { return order[i].ZOrder < order[j].ZOrder })
	// find index of target in order
	idx := -1
	for i, p := range order {
		if p == pn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("internal: panel not in order list")
	}
	newIdx := idx + delta
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx >= len(order) {
		newIdx = len(order) - 1
	}
	if newIdx == idx {
		return nil
	}
	// move in slice
	p := order[idx]
	if newIdx < idx {
		copy(order[newIdx+1:idx+1], order[newIdx:idx])
		order[newIdx] = p
	} else {
		copy(order[idx:newIdx], order[idx+1:newIdx+1])
		order[newIdx] = p
	}
	// reassign zOrder 0..n-1 in new order
	for i, it := range order {
		it.ZOrder = i
	}
	// also reorder pg.Panels slice to match zOrder for deterministic serialization
	sort.Slice(pg.Panels, func(i, j int) bool { return pg.Panels[i].ZOrder < pg.Panels[j].ZOrder })
	return nil
}

// UpdatePanelMeta updates panel ID (if non-empty and unique) and Notes. BeatIDs and Balloons are preserved.
func UpdatePanelMeta(ph *ProjectHandle, pageNumber int, panelID string, newID string, notes string) error {
	pg, _, pn, err := findPanel(ph, pageNumber, panelID)
	if err != nil {
		return err
	}
	if newID != "" && newID != pn.ID {
		// ensure unique on page
		for _, p := range pg.Panels {
			if p.ID == newID {
				return fmt.Errorf("panel id %s already exists on page %d", newID, pageNumber)
			}
		}
		pn.ID = newID
	}
	pn.Notes = notes
	return nil
}
