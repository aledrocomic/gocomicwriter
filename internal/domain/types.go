/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany..
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package domain

// This file defines the core data model structures for the Go Comic Writer project.
// The intent is to closely mirror the conceptual model from docs/go_comic_writer_concept.md
// while keeping it lightweight and compile-safe for early development.

// Project represents a comic project and its metadata.
// It is intended to serialize to a human-readable JSON manifest.
type Project struct {
	Name     string   `json:"name"`
	Metadata Metadata `json:"metadata,omitempty"`
	Issues   []Issue  `json:"issues"`
}

// Metadata contains optional descriptive metadata for a project.
type Metadata struct {
	Series     string `json:"series,omitempty"`
	IssueTitle string `json:"issueTitle,omitempty"`
	Creators   string `json:"creators,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// Issue captures configuration that applies to the whole comic issue.
type Issue struct {
	TrimWidth        float64 `json:"trimWidth"` // in points or mm (unit TBD)
	TrimHeight       float64 `json:"trimHeight"`
	Bleed            float64 `json:"bleed"`
	DPI              int     `json:"dpi"`
	ReadingDirection string  `json:"readingDirection"` // ltr or rtl
	Pages            []Page  `json:"pages"`
}

// Page represents a single page in an issue.
type Page struct {
	Number int     `json:"number"`
	Grid   string  `json:"grid,omitempty"` // e.g., "3x3", "2x3", or custom reference
	Panels []Panel `json:"panels"`
	Layers []Layer `json:"layers,omitempty"`
	Styles []Style `json:"styles,omitempty"`
}

// Layer can be used in later phases for ordering elements or grouping.
type Layer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Z     int    `json:"z"`
	Notes string `json:"notes,omitempty"`
}

// Panel defines a panel region and associated metadata.
type Panel struct {
	ID       string    `json:"id"`
	Geometry Rect      `json:"geometry"`
	ZOrder   int       `json:"zOrder"`
	BeatIDs  []string  `json:"linkedBeats,omitempty"`
	Balloons []Balloon `json:"balloons,omitempty"`
	Notes    string    `json:"notes,omitempty"`
}

// Balloon is a lettering element (speech, caption, SFX, etc.).
type Balloon struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"` // speech, whisper, thought, caption, sfx
	TextRuns []TextRun `json:"textRuns"`
	Shape    Shape     `json:"shape"`
	Tail     Tail      `json:"tail,omitempty"`
	StyleRef string    `json:"styleRef,omitempty"`
}

// TextRun represents a run of text with typography settings.
type TextRun struct {
	Content  string  `json:"content"`
	Font     string  `json:"font"`
	Size     float64 `json:"size"`
	Tracking float64 `json:"tracking,omitempty"`
	Leading  float64 `json:"leading,omitempty"`
	Kerning  float64 `json:"kerning,omitempty"`
	StyleRef string  `json:"styleRef,omitempty"`
}

// Style defines visual styling attributes used by balloons and SFX.
type Style struct {
	Name   string `json:"name"`
	Font   string `json:"font,omitempty"`
	Fill   Color  `json:"fill,omitempty"`
	Stroke Stroke `json:"stroke,omitempty"`
	FX     FX     `json:"fx,omitempty"`
}

// Asset describes external resources like fonts and images.
type Asset struct {
	Type    string `json:"type"` // font, image, ref
	Path    string `json:"path"`
	License string `json:"license,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

// Geometry and rendering primitives (simplified placeholders for now).

type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Shape struct {
	Kind   string  `json:"kind"` // ellipse, roundedBox, rect, path
	Rect   Rect    `json:"rect"`
	Radius float64 `json:"radius,omitempty"`
	// For future: path data, control points, etc.
}

type Tail struct {
	AnchorX float64 `json:"anchorX"`
	AnchorY float64 `json:"anchorY"`
	Angle   float64 `json:"angle,omitempty"`
}

type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

type Stroke struct {
	Color Color   `json:"color"`
	Width float64 `json:"width"`
}

type FX struct {
	Outline bool    `json:"outline,omitempty"`
	Shadow  bool    `json:"shadow,omitempty"`
	Blur    float64 `json:"blur,omitempty"`
}
