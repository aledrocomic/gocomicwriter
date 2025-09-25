/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

import (
	"math"
	"testing"

	"golang.org/x/image/font"

	"gocomicwriter/internal/vector"
)

func TestLayoutOnPath_StraightLine(t *testing.T) {
	// Path: line from (0,0) to (200,0)
	var p vector.Path
	p.MoveTo(0, 0)
	p.LineTo(200, 0)

	fontSpec := FontSpec{Family: "", SizePt: 0}
	poses, total := LayoutOnPath(BasicProvider{}, "ABC", fontSpec, &p, 0, 0)
	if total <= 0 {
		t.Fatalf("expected positive path length, got %v", total)
	}
	if len(poses) != 3 {
		t.Fatalf("expected 3 glyph poses, got %d", len(poses))
	}
	// Verify monotonic X, Y=0, angle=0
	prevX := float32(-1)
	for i, gp := range poses {
		if gp.Pos.Y != 0 {
			t.Fatalf("glyph %d Y expected 0, got %v", i, gp.Pos.Y)
		}
		if math.Abs(float64(gp.Angle)) > 1e-6 {
			t.Fatalf("glyph %d angle expected 0, got %v", i, gp.Angle)
		}
		if gp.Pos.X <= prevX {
			t.Fatalf("glyph %d X not increasing: %v after %v", i, gp.Pos.X, prevX)
		}
		prevX = gp.Pos.X
	}

	// Check expected centers using the same font measurement
	face, _ := BasicProvider{}.Resolve(fontSpec)
	d := &font.Drawer{Face: face}
	wA := float32(d.MeasureString("A") >> 6)
	wB := float32(d.MeasureString("B") >> 6)
	wC := float32(d.MeasureString("C") >> 6)
	exp := []float32{wA / 2, wA + wB/2, wA + wB + wC/2}
	for i := range poses {
		if math.Abs(float64(poses[i].Pos.X-exp[i])) > 0.01 {
			t.Fatalf("glyph %d center X expected %.2f got %.2f", i, exp[i], poses[i].Pos.X)
		}
	}
}
