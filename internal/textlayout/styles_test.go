/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

import "testing"

func TestBuiltinStyles(t *testing.T) {
	names := ListStyles()
	if len(names) < 3 {
		t.Fatalf("expected at least 3 builtin styles, got %v", names)
	}
	if _, ok := GetStyle("Dialogue"); !ok {
		t.Fatalf("Dialogue style missing")
	}
	if _, ok := GetStyle("Caption"); !ok {
		t.Fatalf("Caption style missing")
	}
	if _, ok := GetStyle("SFX"); !ok {
		t.Fatalf("SFX style missing")
	}
}

func TestTrackingIncreasesWidth(t *testing.T) {
	base := []Span{{Text: "ABCD", Font: FontSpec{}}}
	withTrack := []Span{{Text: "ABCD", Font: FontSpec{}, Tracking: 1}}
	w0, _ := Measure(BasicProvider{}, base)
	w1, _ := Measure(BasicProvider{}, withTrack)
	if !(w1 > w0) {
		t.Fatalf("expected tracking to increase width: w0=%v w1=%v", w0, w1)
	}
}

func TestLeadingIncreasesHeight(t *testing.T) {
	l := NewWordWrap(BasicProvider{})
	// Force two lines using maxWidth
	spans0 := []Span{{Text: "Hello world from Go", Font: FontSpec{}, Leading: 0}}
	spans1 := []Span{{Text: "Hello world from Go", Font: FontSpec{}, Leading: 4}}
	b0, err := l.Layout(spans0, 50)
	if err != nil {
		t.Fatalf("layout0: %v", err)
	}
	b1, err := l.Layout(spans1, 50)
	if err != nil {
		t.Fatalf("layout1: %v", err)
	}
	if !(b1.Height > b0.Height) {
		t.Fatalf("expected leading to increase height: h0=%v h1=%v", b0.Height, b1.Height)
	}
}

func TestOTProvider_Fallback(t *testing.T) {
	// No fonts loaded but resolve should work via fallback
	otp := OTProvider{Lib: NewFontLibrary()}
	w, h := Measure(otp, []Span{{Text: "Hello", Font: FontSpec{Family: "Nonexistent", SizePt: 12}}})
	if w <= 0 || h <= 0 {
		t.Fatalf("expected positive measure with fallback: w=%v h=%v", w, h)
	}
}
