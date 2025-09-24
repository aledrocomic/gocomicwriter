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

func TestBaseNode_GettersSetters(t *testing.T) {
	n := NewRect(R(1, 2, 3, 4), Fill{}, Stroke{})
	if got := n.Transform(); got != Identity {
		t.Fatalf("initial transform not identity: %+v", got)
	}
	if n.Fill().Enabled || n.Stroke().Enabled {
		t.Fatalf("initial styles should be disabled")
	}
	// Use promoted setters
	n.setFill(Fill{Enabled: true})
	n.setStroke(Stroke{Enabled: true, Width: 2})
	if !n.Fill().Enabled || !n.Stroke().Enabled || n.Stroke().Width != 2 {
		t.Fatalf("setters not applied")
	}
}

func TestEllipse_RectBoundsAndBoundsWithTransform(t *testing.T) {
	n := NewEllipse(R(0, 0, 10, 20), Fill{}, Stroke{})
	// Without transform, bounds should equal rect
	b := n.Bounds()
	if b.X != 0 || b.Y != 0 || b.W != 10 || b.H != 20 {
		t.Fatalf("unexpected bounds: %+v", b)
	}
	// Apply transform and verify bounds moved
	n.SetTransform(Translate(5, -3))
	tb := n.Bounds()
	if tb.X != 5 || tb.Y != -3 || tb.W != 10 || tb.H != 20 {
		t.Fatalf("unexpected transformed bounds: %+v", tb)
	}
}

func TestGroup_BoundsAndHit(t *testing.T) {
	r1 := NewRect(R(0, 0, 10, 10), Fill{}, Stroke{})
	r2 := NewRect(R(20, 0, 10, 10), Fill{}, Stroke{})
	g := NewGroup(r1, r2)

	b := g.Bounds()
	if b.X != 0 || b.Y != 0 || b.W != 30 || b.H != 10 {
		t.Fatalf("unexpected group bounds: %+v", b)
	}
	// Hit on first rect
	if !g.Hit(Pt{5, 5}) {
		t.Fatalf("expected hit on first child")
	}
	// Hit on second rect
	if !g.Hit(Pt{25, 5}) {
		t.Fatalf("expected hit on second child")
	}
	// Miss
	if g.Hit(Pt{-5, -5}) {
		t.Fatalf("did not expect hit outside")
	}

	// Apply a rotation to group and verify hit still works near origin (rotate 90deg around origin)
	g.SetTransform(Rotate(float32(math.Pi / 2)))
	if !g.Hit(Pt{-5, 5}) { // Former (5,5) rotated lands at approximately (-5,5)
		t.Fatalf("expected hit after rotation")
	}
}

func TestGeometry_Union_MinMax_Rotate_FloatRound(t *testing.T) {
	a := R(0, 0, 10, 10)
	b := R(5, -5, 5, 10)
	u := a.Union(b)
	if u.X != 0 || u.Y != -5 || u.W != 10 || u.H != 15 {
		t.Fatalf("unexpected union: %+v", u)
	}

	// Min/Max helpers indirectly via Rect.Min/Max
	if m := a.Min(); m.X != 0 || m.Y != 0 {
		t.Fatalf("min wrong: %+v", m)
	}
	if m := a.Max(); m.X != 10 || m.Y != 10 {
		t.Fatalf("max wrong: %+v", m)
	}

	// Rotate unit vector (1,0) by 180 degrees should go to (-1,0)
	m := Rotate(float32(math.Pi))
	p := m.Apply(Pt{1, 0})
	if math.Abs(float64(p.X-(-1))) > 1e-5 || math.Abs(float64(p.Y-0)) > 1e-5 {
		t.Fatalf("unexpected rotate result: %+v", p)
	}

	if FloatRound(1.23456, 2) != 1.23 {
		t.Fatalf("float round fail")
	}
	if FloatRound(1.23456, -1) != 1.23456 {
		t.Fatalf("negative places should be no-op")
	}
}
