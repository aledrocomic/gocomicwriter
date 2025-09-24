/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import "testing"

func TestRoundedRect_Bounds(t *testing.T) {
	n := NewRoundedRect(R(1, 2, 100, 50), 10, Fill{}, Stroke{})
	b := n.Bounds()
	if b.X != 1 || b.Y != 2 || b.W != 100 || b.H != 50 {
		t.Fatalf("unexpected bounds: %+v", b)
	}
}
