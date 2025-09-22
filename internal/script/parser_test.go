/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package script

import "testing"

func TestParseBasicScenesAndDialogue(t *testing.T) {
	input := `# Opening Scene
ALICE: Hello, world!
  And a continuation line.

; a note that should be captured but not in outline
Beat something
PANEL 1 Introduction

# Second Scene
CAPTION: Meanwhile, elsewhere...
BOB: Hi, Alice.`

	s, errs := Parse(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(s.Scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(s.Scenes))
	}
	if s.Scenes[0].Title != "Opening Scene" {
		t.Fatalf("unexpected scene 1 title: %q", s.Scenes[0].Title)
	}
	// First scene lines: ALICE dialogue (with continuation), note, beat, panel
	if len(s.Scenes[0].Lines) < 1 {
		t.Fatalf("expected at least 1 line in scene 1")
	}
	l0 := s.Scenes[0].Lines[0]
	if l0.Type != LineDialogue || l0.Character != "ALICE" {
		t.Fatalf("expected first line to be ALICE dialogue, got %+v", l0)
	}
	if l0.Text != "Hello, world!\nAnd a continuation line." {
		t.Fatalf("unexpected dialogue text: %q", l0.Text)
	}

	// Second scene checks
	if s.Scenes[1].Title != "Second Scene" {
		t.Fatalf("unexpected scene 2 title: %q", s.Scenes[1].Title)
	}
	if len(s.Scenes[1].Lines) != 2 {
		t.Fatalf("expected 2 lines in scene 2, got %d", len(s.Scenes[1].Lines))
	}
	if s.Scenes[1].Lines[0].Type != LineCaption {
		t.Fatalf("expected first line to be caption, got %+v", s.Scenes[1].Lines[0])
	}
	if s.Scenes[1].Lines[1].Character != "BOB" || s.Scenes[1].Lines[1].Type != LineDialogue {
		t.Fatalf("expected BOB dialogue, got %+v", s.Scenes[1].Lines[1])
	}
}

func TestImplicitSceneAndUnknownLines(t *testing.T) {
	input := `This is a cold open without a scene header.
CAPTION: A caption.
Some freeform line`

	s, errs := Parse(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(s.Scenes) != 1 {
		t.Fatalf("expected 1 scene, got %d", len(s.Scenes))
	}
	if s.Scenes[0].Title != "Untitled" {
		t.Fatalf("expected implicit Untitled scene, got %q", s.Scenes[0].Title)
	}
	if len(s.Scenes[0].Lines) != 3 { // first unknown should be captured, caption second, then unknown third
		t.Fatalf("expected 3 lines in scene, got %d", len(s.Scenes[0].Lines))
	}
}

func TestParseTagsExtraction(t *testing.T) {
	input := `# S
ALICE: Hello @prop
  cont @extra
Beat something @theme-1
CAPTION: Meanwhile @loc1`

	s, errs := Parse(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(s.Scenes) != 1 {
		t.Fatalf("expected 1 scene, got %d", len(s.Scenes))
	}
	lines := s.Scenes[0].Lines
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	// Dialogue with tags across continuation
	dlg := lines[0]
	if dlg.Type != LineDialogue {
		t.Fatalf("expected dialogue line, got %+v", dlg)
	}
	if !containsAll(dlg.Tags, []string{"prop", "extra"}) {
		t.Fatalf("expected tags [prop extra], got %+v", dlg.Tags)
	}
	// Beat line with tag theme-1
	bt := lines[1]
	if bt.Type != LineBeat {
		t.Fatalf("expected beat line, got %+v", bt)
	}
	if !containsAll(bt.Tags, []string{"theme-1"}) {
		t.Fatalf("expected tag [theme-1], got %+v", bt.Tags)
	}
	// Caption with tag
	caption := lines[2]
	if caption.Type != LineCaption {
		t.Fatalf("expected caption line, got %+v", caption)
	}
	if !containsAll(caption.Tags, []string{"loc1"}) {
		t.Fatalf("expected tag [loc1], got %+v", caption.Tags)
	}
}

func containsAll(haystack []string, needles []string) bool {
	m := map[string]struct{}{}
	for _, h := range haystack {
		m[h] = struct{}{}
	}
	for _, n := range needles {
		if _, ok := m[n]; !ok {
			return false
		}
	}
	return true
}
