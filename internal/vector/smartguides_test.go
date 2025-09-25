/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import "testing"

func TestComputeSmartGuides_SnapToPanelEdges(t *testing.T) {
	panel := Rect{X: 0, Y: 0, W: 200, H: 100}
	moving := Rect{X: 3, Y: 4, W: 80, H: 40} // near top-left edges
	opts := SnapOptions{Threshold: 6, SnapToEdges: true}

	snapped, guides := ComputeSmartGuides(moving, []Anchor{{Rect: panel, Weight: 1}}, opts)
	if snapped.X != 0 {
		t.Fatalf("expected X snapped to 0, got %v", snapped.X)
	}
	if snapped.Y != 0 {
		t.Fatalf("expected Y snapped to 0, got %v", snapped.Y)
	}
	if len(guides) == 0 {
		t.Fatalf("expected guides for snapping")
	}
	// expect a vertical guide at x=0 and a horizontal at y=0 among guides
	var vOK, hOK bool
	for _, g := range guides {
		if g.Orientation == "vertical" && g.Position == 0 {
			vOK = true
		}
		if g.Orientation == "horizontal" && g.Position == 0 {
			hOK = true
		}
	}
	if !vOK || !hOK {
		t.Fatalf("expected guides at x=0 (%v) and y=0 (%v)", vOK, hOK)
	}
}

func TestComputeSmartGuides_SnapToCenters(t *testing.T) {
	panel := Rect{X: 0, Y: 0, W: 200, H: 100}
	// Place moving so its center is within threshold of panel center
	moving := Rect{X: 200/2 - 50 - 2, Y: 100/2 - 30 - 3, W: 100, H: 60}
	opts := SnapOptions{Threshold: 5, SnapToCenters: true}

	snapped, guides := ComputeSmartGuides(moving, []Anchor{{Rect: panel, Weight: 1}}, opts)
	if snapped.X != (200/2 - 50) {
		t.Fatalf("expected X snapped to center %v, got %v", (200/2 - 50), snapped.X)
	}
	if snapped.Y != (100/2 - 30) {
		t.Fatalf("expected Y snapped to center %v, got %v", (100/2 - 30), snapped.Y)
	}
	if len(guides) == 0 {
		t.Fatalf("expected guides for snapping")
	}
	var vOK, hOK bool
	for _, g := range guides {
		if g.Orientation == "vertical" && g.Kind == "center" && g.Position == float32(panel.X+panel.W/2) {
			vOK = true
		}
		if g.Orientation == "horizontal" && g.Kind == "center" && g.Position == float32(panel.Y+panel.H/2) {
			hOK = true
		}
	}
	if !vOK || !hOK {
		t.Fatalf("expected center guides present")
	}
}

func TestComputeSmartGuides_ThresholdPreventsSnap(t *testing.T) {
	panel := Rect{X: 0, Y: 0, W: 200, H: 100}
	moving := Rect{X: 10, Y: 10, W: 50, H: 20} // 10px away from top-left
	opts := SnapOptions{Threshold: 5, SnapToEdges: true}

	snapped, guides := ComputeSmartGuides(moving, []Anchor{{Rect: panel, Weight: 1}}, opts)
	if snapped.X != moving.X || snapped.Y != moving.Y {
		t.Fatalf("expected no snapping when outside threshold; got %+v", snapped)
	}
	if len(guides) != 0 {
		t.Fatalf("expected no guides when no snap")
	}
}

func TestComputeSmartGuides_PicksClosestAxisIndependently(t *testing.T) {
	anchors := []Anchor{
		{Rect: Rect{X: 0, Y: 0, W: 100, H: 100}, Weight: 1},
		{Rect: Rect{X: 300, Y: 0, W: 100, H: 100}, Weight: 1},
	}
	moving := Rect{X: 2, Y: 298, W: 80, H: 80} // near left edge X=0 and near bottom Y=300+100=400? Actually near top of bottom edge? We'll test right numbers.
	// Adjust to be near Y=100 (bottom of first anchor) and X=0 (left of first anchor)
	moving.Y = 97

	snapped, _ := ComputeSmartGuides(moving, anchors, SnapOptions{Threshold: 5, SnapToEdges: true})
	if snapped.X != 0 {
		t.Fatalf("expected X snapped to 0, got %v", snapped.X)
	}
	if snapped.Y != 100 { // bottom of first anchor
		t.Fatalf("expected Y snapped to 100, got %v", snapped.Y)
	}
}
