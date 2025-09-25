/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import "testing"

func TestSuggestBalloonLayout_BasicLTRTopLeft(t *testing.T) {
	panel := R(0, 0, 300, 200)
	content := Size{W: 100, H: 60}
	pos, attempts := SuggestBalloonLayout(panel, content, nil, SuggestOptions{})
	if attempts == 0 {
		t.Fatalf("expected attempts > 0")
	}
	// Defaults: Margin=8, Padding=8 => top-left at (8,8)
	if pos.X != 8 || pos.Y != 8 {
		t.Fatalf("expected position at (8,8), got (%.1f,%.1f)", pos.X, pos.Y)
	}
	if pos.W != 116 || pos.H != 76 {
		t.Fatalf("expected size 116x76, got %.1fx%.1f", pos.W, pos.H)
	}
}

func TestSuggestBalloonLayout_AvoidsObstacleInTopLeft(t *testing.T) {
	panel := R(0, 0, 300, 200)
	content := Size{W: 100, H: 60}
	// Obstacle covering top-left band
	obstacles := []Rect{R(8, 8, 150, 100)}
	pos, _ := SuggestBalloonLayout(panel, content, obstacles, SuggestOptions{})
	// Should be on same row (y=8) but moved to x >= 158
	if pos.Y != 8 {
		t.Fatalf("expected Y=8, got %.1f", pos.Y)
	}
	if pos.X != 160 {
		t.Fatalf("expected X=160 to clear obstacle, got %.1f", pos.X)
	}
	// Ensure no intersection
	if pos.Intersects(obstacles[0]) {
		t.Fatalf("expected no intersection with obstacle; got overlap")
	}
}

func TestSuggestBalloonLayout_RTLPrefersTopRight(t *testing.T) {
	panel := R(0, 0, 300, 200)
	content := Size{W: 100, H: 60}
	pos, _ := SuggestBalloonLayout(panel, content, nil, SuggestOptions{ReadingDirection: "rtl"})
	// inner.X=8, inner.W=284, bw=116 => x=8+284-116=176
	if pos.X != 176 || pos.Y != 8 {
		t.Fatalf("expected top-right at (176,8), got (%.1f,%.1f)", pos.X, pos.Y)
	}
}

func TestSuggestBalloonLayout_AnchorBiasToBottomCenter(t *testing.T) {
	panel := R(0, 0, 300, 200)
	content := Size{W: 100, H: 60}
	anchor := Pt{150, 180}
	pos, _ := SuggestBalloonLayout(panel, content, nil, SuggestOptions{HasAnchor: true, Anchor: anchor})
	// GridStep=8, bw=116 -> ideal x≈92 -> nearest grid is 88 (tie with 96, smaller x wins),
	// y1=108 (bottom-most y) expected for closest to anchor.
	if pos.Y != 116 {
		t.Fatalf("expected Y=116 close to bottom, got %.1f", pos.Y)
	}
	if pos.X != 88 {
		t.Fatalf("expected X=88 nearest to anchor center, got %.1f", pos.X)
	}
}

func TestSuggestBalloonLayout_FallbackWithinBounds(t *testing.T) {
	panel := R(0, 0, 200, 120)
	content := Size{W: 180, H: 100}
	// Fill inner area with overlapping obstacles so no perfect fit
	inner := panel.Inset(8, 8)
	obstacles := []Rect{
		R(inner.X, inner.Y, inner.W, inner.H/2),
		R(inner.X, inner.Y+inner.H/2-4, inner.W, inner.H/2+4),
	}
	pos, attempts := SuggestBalloonLayout(panel, content, obstacles, SuggestOptions{})
	if attempts == 0 {
		t.Fatalf("expected attempts > 0")
	}
	// Should be clamped within inner bounds
	if pos.X < inner.X-1e-3 || pos.Y < inner.Y-1e-3 || pos.X+pos.W > inner.X+inner.W+1e-3 || pos.Y+pos.H > inner.Y+inner.H+1e-3 {
		t.Fatalf("expected result within inner bounds; got %+v vs inner %+v", pos, inner)
	}
}
