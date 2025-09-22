/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

import "testing"

func TestWordWrap_Naive(t *testing.T) {
	l := NewWordWrap(BasicProvider{})
	box, err := l.Layout([]Span{{Text: "Hello world from Go", Font: FontSpec{}}}, 50)
	if err != nil {
		t.Fatalf("layout error: %v", err)
	}
	if len(box.Lines) < 2 {
		t.Fatalf("expected wrapping into multiple lines, got %d", len(box.Lines))
	}
	if box.Width <= 0 || box.Height <= 0 {
		t.Fatalf("expected positive box size: %+v", box)
	}
}

func TestMeasure_Deterministic(t *testing.T) {
	w1, h1 := Measure(BasicProvider{}, []Span{{Text: "ABC"}})
	w2, h2 := Measure(BasicProvider{}, []Span{{Text: "A"}, {Text: "BC"}})
	if w1 != w2 || h1 != h2 {
		t.Fatalf("expected same measure, got w1=%v h1=%v vs w2=%v h2=%v", w1, h1, w2, h2)
	}
}
