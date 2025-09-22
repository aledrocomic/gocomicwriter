//go:build fyne

/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

// These tests validate the Fyne-based UI components. They are gated behind the
// "fyne" build tag so CI (which is headless) does not need Fyne or a display.
// To run locally:
//
//	go test -tags fyne ./internal/ui
//
// Ensure you have the Fyne dependencies installed and a working OS driver.
package ui

import (
	"testing"

	"fyne.io/fyne/v2"
)

func almostEqual(a, b, eps float32) bool {
	if a > b {
		return a-b <= eps
	}
	return b-a <= eps
}

func TestPageCanvas_Defaults(t *testing.T) {
	pc := NewPageCanvas()
	if pc.zoom != 0.5 {
		t.Fatalf("expected default zoom 0.5, got %v", pc.zoom)
	}
	sz := pc.PreferredSize()
	if sz.Width != 800 || sz.Height != 600 {
		t.Fatalf("unexpected PreferredSize: %v", sz)
	}
}

func TestPageCanvas_LayoutGeometry(t *testing.T) {
	pc := NewPageCanvas()
	r, ok := pc.CreateRenderer().(*pageCanvasRenderer)
	if !ok {
		t.Fatalf("expected pageCanvasRenderer, got %T", pc.CreateRenderer())
	}

	// Layout within a known container size
	containerSize := fyne.NewSize(1000, 800)
	r.Layout(containerSize)

	page := r.page
	trim := r.trim
	bleed := r.bleed

	// Expected sizes with default zoom 0.5 and A4 logic from implementation
	expectedPageW := float32(595) * 0.5
	expectedPageH := float32(842) * 0.5
	if !almostEqual(page.Size().Width, expectedPageW, 0.2) || !almostEqual(page.Size().Height, expectedPageH, 0.2) {
		t.Fatalf("unexpected page size: got %v, want approx (%v x %v)", page.Size(), expectedPageW, expectedPageH)
	}

	expectedTrimW := (float32(595) - 2*9) * 0.5
	expectedTrimH := (float32(842) - 2*9) * 0.5
	if !almostEqual(trim.Size().Width, expectedTrimW, 0.2) || !almostEqual(trim.Size().Height, expectedTrimH, 0.2) {
		t.Fatalf("unexpected trim size: got %v, want approx (%v x %v)", trim.Size(), expectedTrimW, expectedTrimH)
	}

	expectedBleedW := (float32(595) + 2*18) * 0.5
	expectedBleedH := (float32(842) + 2*18) * 0.5
	if !almostEqual(bleed.Size().Width, expectedBleedW, 0.2) || !almostEqual(bleed.Size().Height, expectedBleedH, 0.2) {
		t.Fatalf("unexpected bleed size: got %v, want approx (%v x %v)", bleed.Size(), expectedBleedW, expectedBleedH)
	}

	// Positional relationships
	pPos := page.Position()
	tPos := trim.Position()
	bPos := bleed.Position()
	if tPos.X < pPos.X-0.1 || tPos.Y < pPos.Y-0.1 {
		t.Fatalf("trim should be inside page: trim %v vs page %v", tPos, pPos)
	}
	if bPos.X > pPos.X+0.1 || bPos.Y > pPos.Y+0.1 {
		t.Fatalf("bleed should start outside (above/left) of page: bleed %v vs page %v", bPos, pPos)
	}

	// Now apply a pan offset and ensure the page moves accordingly
	oldX := page.Position().X
	oldY := page.Position().Y
	pc.offsetX += 100
	pc.offsetY += 50
	r.Layout(containerSize)
	newX := page.Position().X
	newY := page.Position().Y
	if newX <= oldX+80 || newY <= oldY+30 { // allow for minor rounding
		t.Fatalf("expected page to move with offsets; before (%v,%v), after (%v,%v)", oldX, oldY, newX, newY)
	}
}
