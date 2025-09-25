/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

import (
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// FontLibrary stores loaded OpenType fonts mapped by family/weight/italic.
// Note: this is a minimal in-memory library for early usage; it does not
// support named instances/variations beyond weight and italic flags.

type FontLibrary struct {
	fonts map[fontKey]*opentype.Font
}

type fontKey struct {
	family string
	weight int
	italic bool
}

func NewFontLibrary() *FontLibrary { return &FontLibrary{fonts: make(map[fontKey]*opentype.Font)} }

// LoadTTF loads a font file into the library under the given family/weight/italic.
func (fl *FontLibrary) LoadTTF(family string, weight int, italic bool, path string) error {
	if fl.fonts == nil {
		fl.fonts = make(map[fontKey]*opentype.Font)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read font %s: %w", path, err)
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return fmt.Errorf("parse font %s: %w", path, err)
	}
	fl.fonts[fontKey{family: family, weight: weight, italic: italic}] = f
	return nil
}

func (fl *FontLibrary) find(spec FontSpec) *opentype.Font {
	if fl == nil || fl.fonts == nil {
		return nil
	}
	// Exact match first
	if f, ok := fl.fonts[fontKey{family: spec.Family, weight: spec.Weight, italic: spec.Italic}]; ok {
		return f
	}
	// Fallbacks: try by weight buckets and italic variations.
	// Simplistic approach: try same family any weight/italic
	for k, f := range fl.fonts {
		if k.family == spec.Family {
			return f
		}
	}
	return nil
}

// OTProvider resolves FontSpec using a FontLibrary and falls back to another Provider.
// It uses kerning as provided by opentype.Face and font.Drawer.

type OTProvider struct {
	Lib      *FontLibrary
	DPI      float64 // default 72 if zero
	Fallback Provider
}

func (p OTProvider) Resolve(spec FontSpec) (font.Face, Metrics) {
	// Defaults
	if spec.SizePt <= 0 {
		spec.SizePt = 12
	}
	dpi := p.DPI
	if dpi <= 0 {
		dpi = 72
	}

	if p.Lib != nil {
		if f := p.Lib.find(spec); f != nil {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(spec.SizePt), DPI: dpi, Hinting: font.HintingFull})
			if err == nil {
				m := face.Metrics()
				return face, Metrics{
					Ascent:  float32(m.Ascent.Round()),
					Descent: float32(m.Descent.Round()),
					LineGap: float32(m.Height.Round() - m.Ascent.Round() - m.Descent.Round()),
				}
			}
		}
	}
	fb := p.Fallback
	if fb == nil {
		fb = BasicProvider{}
	}
	return fb.Resolve(spec)
}
