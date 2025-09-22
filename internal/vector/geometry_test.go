/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import "testing"

func TestRectContainsAndInset(t *testing.T) {
	r := R(10, 20, 100, 50)
	if !r.Contains(Pt{10, 20}) || !r.Contains(Pt{110, 70}) {
		t.Fatalf("expected edge points to be contained")
	}
	in := r.Inset(5, 5)
	if in.X != 15 || in.Y != 25 || in.W != 90 || in.H != 40 {
		t.Fatalf("unexpected inset: %+v", in)
	}
}

func TestAffineBasic(t *testing.T) {
	m := Translate(10, 5).Mul(Scale(2, 3))
	p := m.Apply(Pt{1, 1})
	if p.X != 12 || p.Y != 8 { // (1*2+10, 1*3+5)
		t.Fatalf("unexpected transform result: %+v", p)
	}
}

func TestRectNode_HitAndBounds(t *testing.T) {
	n := NewRect(R(0, 0, 100, 50), Fill{Enabled: true, Color: White}, Stroke{Enabled: true, Width: 1})
	n.SetTransform(Translate(10, 20))
	if !n.Hit(Pt{50 + 10, 25 + 20}) {
		t.Fatalf("expected hit after translation")
	}
	b := n.Bounds()
	if b.X != 10 || b.Y != 20 || b.W != 100 || b.H != 50 {
		t.Fatalf("unexpected bounds: %+v", b)
	}
}

func TestEllipseNode_Hit(t *testing.T) {
	n := NewEllipse(R(0, 0, 100, 100), Fill{Enabled: true}, Stroke{})
	if !n.Hit(Pt{50, 50}) {
		t.Fatalf("center should hit")
	}
	if n.Hit(Pt{200, 200}) {
		t.Fatalf("far point should not hit")
	}
}

func TestRoundedRectNode_Hit(t *testing.T) {
	n := NewRoundedRect(R(0, 0, 100, 100), 20, Fill{Enabled: true}, Stroke{})
	if !n.Hit(Pt{10, 10}) { // inside top-left corner curve
		t.Fatalf("expected hit near corner")
	}
	if n.Hit(Pt{-5, -5}) {
		t.Fatalf("expected miss outside")
	}
}
