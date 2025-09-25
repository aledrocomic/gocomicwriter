/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

// TextStyle represents a reusable text style preset combining a font spec
// with layout parameters commonly used in comics lettering.
// Tracking and Leading are measured in pixels.
//
// Kerning is applied by default by the text engine (font.Drawer / Face.Kern).
// We keep it always-on for deterministic results at this stage.
//
// The intent is to keep presets simple but practical for the early UI.
// More advanced features like text effects or OpenType features can be added later.

type TextStyle struct {
	Name     string
	Font     FontSpec
	Tracking float32 // px between glyphs (added per inter-glyph gap)
	Leading  float32 // extra px added to line height
}

var builtinStyles = map[string]TextStyle{
	// Reasonable defaults for early lettering.
	// Sizes are in points. Users can override per project.
	"Dialogue": {
		Name:     "Dialogue",
		Font:     FontSpec{Family: "Comic Sans MS", SizePt: 9, Weight: 400, Italic: false},
		Tracking: 0.0,
		Leading:  2.0,
	},
	"Caption": {
		Name:     "Caption",
		Font:     FontSpec{Family: "Comic Sans MS", SizePt: 8.5, Weight: 600, Italic: false},
		Tracking: 0.25,
		Leading:  1.5,
	},
	"SFX": {
		Name:     "SFX",
		Font:     FontSpec{Family: "Impact", SizePt: 24, Weight: 800, Italic: false},
		Tracking: 0.5,
		Leading:  0.0,
	},
}

// GetStyle returns a builtin style preset by name. The second return value is false if
// the style is not found.
func GetStyle(name string) (TextStyle, bool) { s, ok := builtinStyles[name]; return s, ok }

// ListStyles lists the names of the builtin styles in stable order.
func ListStyles() []string {
	// Simple deterministic order
	return []string{"Dialogue", "Caption", "SFX"}
}
