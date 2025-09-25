/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

import (
	"gocomicwriter/internal/vector"
	"testing"
)

func TestDefaultSFXStyle(t *testing.T) {
	st := DefaultSFXStyle()
	if st.Fill != vector.White {
		t.Fatalf("expected white fill, got %+v", st.Fill)
	}
	if !st.Outline.Enabled || st.Outline.Width <= 0 {
		t.Fatalf("expected enabled outline with positive width, got %+v", st.Outline)
	}
}

func TestSFXSpec_LayoutBaseline(t *testing.T) {
	spec := SFXSpec{Text: "BOOM", Style: DefaultSFXStyle(), Font: FontSpec{}, Tracking: 0, OnPath: false}
	poses, total := spec.Layout(BasicProvider{})
	if len(poses) != 4 {
		t.Fatalf("expected 4 glyphs, got %d", len(poses))
	}
	if total <= 0 {
		t.Fatalf("expected positive total advance, got %v", total)
	}
	// Baseline angle should be zero
	for i, gp := range poses {
		if gp.Angle != 0 {
			t.Fatalf("glyph %d angle expected 0, got %v", i, gp.Angle)
		}
	}
}
