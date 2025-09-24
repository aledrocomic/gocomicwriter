/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import "testing"

func TestPath_QuadAndCubic_Bounds(t *testing.T) {
	var p Path
	p.MoveTo(0, 0)
	p.QuadTo(10, 10, 20, 0)
	p.CubicTo(30, -10, 40, 10, 50, 0)
	p.Close()

	b := p.Bounds()
	// With our approximation including control points, min/max should reflect extremes
	if b.X != 0 || b.Y != -10 || b.W != 50 || b.H != 20 {
		t.Fatalf("unexpected bounds: %+v", b)
	}
}
