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
	"golang.org/x/image/font"
)

// SFXStyle describes visual styling for SFX lettering.
// Rendering engines can translate this into concrete paint/effects.
type SFXStyle struct {
	Fill    vector.Color  // glyph fill color
	Outline vector.Stroke // glyph outline stroke
	Shadow  ShadowFX      // optional shadow
}

// ShadowFX describes a simple drop shadow effect.
type ShadowFX struct {
	Enabled bool
	Dx, Dy  float32
	Blur    float32
	Color   vector.Color
}

// DefaultSFXStyle provides practical defaults: white fill, black thick outline.
func DefaultSFXStyle() SFXStyle {
	return SFXStyle{
		Fill:    vector.White,
		Outline: vector.Stroke{Color: vector.Black, Width: 3, Cap: vector.CapRound, Join: vector.JoinRound, MiterLim: 4, Enabled: true},
		Shadow:  ShadowFX{Enabled: false, Dx: 2, Dy: 2, Blur: 2, Color: vector.Black},
	}
}

// SFXSpec bundles content and layout options for SFX, including text-on-path.
type SFXSpec struct {
	Text        string
	Style       SFXStyle
	Font        FontSpec
	Tracking    float32
	OnPath      bool
	Path        *vector.Path
	StartOffset float32
}

// Layout produces glyph poses for the configured SFX.
// If OnPath is true and Path is set, text is laid out along the path; otherwise,
// glyphs are placed on a straight baseline along +X with angle 0, spaced by advances.
func (s SFXSpec) Layout(provider Provider) ([]GlyphPose, float32) {
	if s.OnPath && s.Path != nil {
		return LayoutOnPath(provider, s.Text, s.Font, s.Path, s.Tracking, s.StartOffset)
	}
	if provider == nil {
		provider = BasicProvider{}
	}
	face, _ := provider.Resolve(s.Font)
	d := &font.Drawer{Face: face}
	var poses []GlyphPose
	var x float32 = s.StartOffset
	prev := rune(-1)
	for _, r := range s.Text {
		adv := measureRuneAdvance(d, prev, r)
		pos := vector.Pt{X: x + adv/2, Y: 0}
		poses = append(poses, GlyphPose{Rune: r, Pos: pos, Angle: 0, Advance: adv})
		x += adv + s.Tracking
		prev = r
	}
	return poses, x
}
