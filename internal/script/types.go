/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package script

// Script represents a parsed comic script with scenes and lines.
// This is intentionally minimal for Phase 2: scenes and character/ caption lines.
// Inspired by Fountain/Markdown-like conventions in docs/go_comic_writer_concept.md.

type Script struct {
	Scenes []Scene
}

type Scene struct {
	Title string
	Lines []Line
}

// LineType indicates the kind of a script line.
// Dialogue: CHARACTER: text
// Caption:  CAPTION: text or NARRATION: text
// Note:     lines starting with ";" are author notes and ignored by outline
// Beat:     optional "Panel" or "Beat" markers (not yet mapped to pages)

type LineType int

const (
	LineUnknown LineType = iota
	LineDialogue
	LineCaption
	LineNote
	LineBeat
)

// Line captures a single logical line (possibly with continuations) in a scene.
// For Dialogue, Character holds the speaking character name (upper-cased in parser by default),
// and Text is the spoken content.
// For Caption, Character may contain a label like "CAPTION" or "NARRATION".
// For Beat, Character holds the marker (e.g., "PANEL 1" or "BEAT"), Text the remainder.

type Line struct {
	Type      LineType
	Character string
	Text      string
	Tags      []string
	LineNo    int // 1-based starting line number in the source
}

// Error represents a parse error with position context.

type Error struct {
	Line    int
	Column  int
	Message string
}
