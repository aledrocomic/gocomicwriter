/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

// Abstractions for cross-platform text shaping and layout.
// The goal is to isolate all text measurement and line breaking behind
// deterministic interfaces that can be implemented with different engines.

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// FontSpec describes a requested font.
type FontSpec struct {
	Family string // logical family name
	SizePt float32
	Weight int // 100..900
	Italic bool
}

// Metrics provides font metrics in pixels for the resolved face.
type Metrics struct {
	Ascent, Descent, LineGap float32
}

// Span is a run of text with the same font/style.
type Span struct {
	Text string
	Font FontSpec
}

// Line is a single laid out line with width and ascent/descent.
type Line struct {
	Spans   []Span
	Width   float32
	Ascent  float32
	Descent float32
}

// TextBox is the result of laying out text into a box width.
type TextBox struct {
	Lines   []Line
	Width   float32
	Height  float32
	Metrics Metrics
}

// Provider maps FontSpec to a concrete font.Face.
type Provider interface {
	Resolve(FontSpec) (font.Face, Metrics)
}

// Layouter performs line-breaking and measurement.
type Layouter interface {
	Layout(spans []Span, maxWidth float32) (TextBox, error)
}

// BasicProvider uses x/image/basicfont Face7x13 for deterministic tests.
type BasicProvider struct{}

func (BasicProvider) Resolve(spec FontSpec) (font.Face, Metrics) {
	f := basicfont.Face7x13
	m := f.Metrics()
	return f, Metrics{
		Ascent:  float32(m.Ascent.Round()),
		Descent: float32(m.Descent.Round()),
		LineGap: float32(m.Height.Round() - m.Ascent.Round() - m.Descent.Round()),
	}
}

// WordWrapLayouter is a simple layouter that breaks on spaces; it does not
// perform shaping or hyphenation. Exact enough for early UI and tests.
type WordWrapLayouter struct{ Provider Provider }

func NewWordWrap(provider Provider) *WordWrapLayouter { return &WordWrapLayouter{Provider: provider} }

func (l *WordWrapLayouter) Layout(spans []Span, maxWidth float32) (TextBox, error) {
	if l.Provider == nil {
		l.Provider = BasicProvider{}
	}
	// For simplicity, assume a single font per box for metrics aggregation.
	face, met := l.Provider.Resolve(FontSpec{})
	drawer := &font.Drawer{Face: face}
	cur := Line{Ascent: met.Ascent, Descent: met.Descent}
	box := TextBox{Metrics: met}
	addLine := func() {
		box.Lines = append(box.Lines, cur)
		if cur.Width > box.Width {
			box.Width = cur.Width
		}
		box.Height += met.Ascent + met.Descent + met.LineGap
		cur = Line{Ascent: met.Ascent, Descent: met.Descent}
	}
	for _, sp := range spans {
		if sp.Text == "" {
			continue
		}
		// naive split by spaces, keep spaces minimal
		start := 0
		for i := 0; i <= len(sp.Text); i++ {
			if i == len(sp.Text) || sp.Text[i] == ' ' || sp.Text[i] == '\n' { // word boundary
				word := sp.Text[start:i]
				space := byte(0)
				if i < len(sp.Text) {
					space = sp.Text[i]
				}
				w := advance(drawer, word)
				// if word alone exceeds maxWidth, force on new line
				if cur.Width > 0 && cur.Width+w > maxWidth && maxWidth > 0 {
					addLine()
				}
				if word != "" {
					cur.Spans = append(cur.Spans, Span{Text: word, Font: sp.Font})
					cur.Width += w
				}
				if space == ' ' {
					ws := advance(drawer, " ")
					cur.Spans = append(cur.Spans, Span{Text: " ", Font: sp.Font})
					cur.Width += ws
				} else if space == '\n' {
					addLine()
				}
				start = i + 1
			}
		}
	}
	// flush last line
	if len(cur.Spans) > 0 || len(box.Lines) == 0 {
		addLine()
	}
	return box, nil
}

func advance(d *font.Drawer, s string) float32 {
	return float32(d.MeasureString(s) >> 6) // fixed.Int26_6 to px
}

// Measure provides a quick way to measure text width/height without line-breaks.
func Measure(provider Provider, spans []Span) (w, h float32) {
	if provider == nil {
		provider = BasicProvider{}
	}
	_, met := provider.Resolve(FontSpec{})
	var width float32
	var face font.Face
	for _, sp := range spans {
		face, _ = provider.Resolve(sp.Font)
		d := &font.Drawer{Face: face}
		width += advance(d, sp.Text)
	}
	lineH := met.Ascent + met.Descent
	return width, lineH
}
