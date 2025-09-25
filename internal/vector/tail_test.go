/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import (
	"math"
	"testing"
)

func almostEq(a, b, eps float32) bool { return float32(math.Abs(float64(a-b))) <= eps }

func TestComputeBalloonTailEllipse_LeftAnchor(t *testing.T) {
	balloon := R(100, 100, 160, 120) // center (180,160), rx=80, ry=60
	anchor := Pt{X: 50, Y: 160}
	geo := ComputeBalloonTailEllipse(balloon, anchor, TailOptions{BaseWidth: 20, Length: 80})

	if geo.Side != "left" {
		t.Fatalf("expected side left, got %s", geo.Side)
	}
	cx, cy := balloon.X+balloon.W/2, balloon.Y+balloon.H/2
	rx, ry := balloon.W/2, balloon.H/2
	// base center must lie on ellipse boundary
	lhs := ((geo.BaseCenter.X-cx)/(rx))*((geo.BaseCenter.X-cx)/(rx)) + ((geo.BaseCenter.Y-cy)/(ry))*((geo.BaseCenter.Y-cy)/(ry))
	if !almostEq(lhs, 1, 0.01) {
		t.Fatalf("base center not on ellipse; got %v", lhs)
	}
	// For leftmost anchor, base center Y should be ~cy
	if !almostEq(geo.BaseCenter.Y, cy, 0.001) {
		t.Fatalf("expected base center Y ≈ %v, got %v", cy, geo.BaseCenter.Y)
	}
	// tip should be the anchor (within rounding)
	if !almostEq(geo.Tip.X, anchor.X, 0.001) || !almostEq(geo.Tip.Y, anchor.Y, 0.001) {
		t.Fatalf("expected tip at anchor, got %+v", geo.Tip)
	}
	// path should be closed triangle
	if len(geo.Path.Cmds) == 0 || geo.Path.Cmds[len(geo.Path.Cmds)-1].Op != Close {
		t.Fatalf("expected closed path")
	}
}

func TestComputeBalloonTailEllipse_QuadrantTopRight(t *testing.T) {
	balloon := R(100, 100, 160, 120) // center (180,160), rx=80, ry=60
	anchor := Pt{X: 260, Y: 110}     // top-right quadrant
	geo := ComputeBalloonTailEllipse(balloon, anchor, TailOptions{BaseWidth: 24, Length: 100})

	if geo.Side != "right" { // dominant axis is X here
		t.Fatalf("expected side right, got %s", geo.Side)
	}
	cx, cy := balloon.X+balloon.W/2, balloon.Y+balloon.H/2
	rx, ry := balloon.W/2, balloon.H/2
	lhs := ((geo.BaseCenter.X-cx)/(rx))*((geo.BaseCenter.X-cx)/(rx)) + ((geo.BaseCenter.Y-cy)/(ry))*((geo.BaseCenter.Y-cy)/(ry))
	if !almostEq(lhs, 1, 0.01) {
		t.Fatalf("base center not on ellipse; got %v", lhs)
	}
	// tip should be at the anchor (length is long enough)
	if !almostEq(geo.Tip.X, anchor.X, 0.001) || !almostEq(geo.Tip.Y, anchor.Y, 0.001) {
		t.Fatalf("expected tip at anchor, got %+v", geo.Tip)
	}
}

func TestComputeBalloonTailEllipse_AnchorInsideUsesOutwardLength(t *testing.T) {
	balloon := R(100, 100, 160, 120)
	cx, cy := balloon.X+balloon.W/2, balloon.Y+balloon.H/2
	anchor := Pt{X: cx, Y: cy} // inside center
	geo := ComputeBalloonTailEllipse(balloon, anchor, TailOptions{BaseWidth: 20, Length: 40})

	// Expect the tip to be outside the ellipse above the top (default orientation up)
	// Check that tip is farther from center than base center (outward)
	bcDist := float32(math.Hypot(float64(geo.BaseCenter.X-cx), float64(geo.BaseCenter.Y-cy)))
	tipDist := float32(math.Hypot(float64(geo.Tip.X-cx), float64(geo.Tip.Y-cy)))
	if !(tipDist > bcDist) {
		t.Fatalf("expected tip farther from center than base center (outward)")
	}
}
