/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import "testing"

func TestPathNode_BoundsAndHit(t *testing.T) {
	var p Path
	p.MoveTo(0, 0)
	p.LineTo(10, 0)
	p.LineTo(0, 10)
	p.Close()

	n := NewPath(p, Fill{Enabled: true, Color: White}, Stroke{Enabled: true, Width: 1})

	// Bounds should cover the triangle extents
	b := n.Bounds()
	if b.X != 0 || b.Y != 0 || b.W != 10 || b.H != 10 {
		t.Fatalf("unexpected bounds: %+v", b)
	}

	// Hit uses bbox for now
	if !n.Hit(Pt{1, 1}) {
		t.Fatalf("expected hit inside bbox")
	}
	if n.Hit(Pt{20, 20}) {
		t.Fatalf("did not expect hit far away")
	}

	// Apply transform and check again
	n.SetTransform(Translate(5, 5))
	if !n.Hit(Pt{6, 6}) {
		t.Fatalf("expected hit after translation")
	}
	bb := n.Bounds()
	if bb.X != 5 || bb.Y != 5 || bb.W != 10 || bb.H != 10 {
		t.Fatalf("unexpected transformed bounds: %+v", bb)
	}
}
