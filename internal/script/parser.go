/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package script

import (
	"bufio"
	"regexp"
	"strings"
)

// Parse parses a script text into a structured Script.
// Supported syntax (minimal):
// - Scene headings:
//   - Lines starting with "#" or "Scene:" introduce a new scene. The rest of the line is the title.
//
// - Dialogue: NAME: text  (NAME is captured as Character; converted to upper-case trim)
//   - Continuation lines indented by 2+ spaces are appended to the previous Dialogue/Caption.
//
// - Caption: CAPTION: text or NARRATION: text
// - Beat markers: lines starting with "Panel"/"PANEL" or "Beat"/"BEAT" are classified as LineBeat.
// - Notes: lines starting with ';' are LineNote.
// Blank lines are preserved as separators but not represented as lines.
func Parse(input string) (Script, []Error) {
	s := Script{Scenes: []Scene{}}
	var errs []Error

	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNo := 0
	currentScene := Scene{}
	var lastLine *Line

	// Patterns
	reScene := regexp.MustCompile(`^(#+)\s*(.*)$`)
	reSceneAlt := regexp.MustCompile(`^(?i)\s*Scene:\s*(.+)$`)
	reName := regexp.MustCompile(`^([A-Za-z0-9_\- ]{1,64})\s*:\s*(.*)$`)
	reBeat := regexp.MustCompile(`^(?i)\s*(Panel\s*\d+|Beat)\b\s*(.*)$`)
	reTag := regexp.MustCompile(`(?i)@([a-z0-9_\-]+)`) // tags like @tag-name

	extractTags := func(s string) []string {
		found := reTag.FindAllStringSubmatch(s, -1)
		if len(found) == 0 {
			return nil
		}
		m := map[string]struct{}{}
		for _, f := range found {
			if len(f) > 1 {
				t := strings.ToLower(strings.TrimSpace(f[1]))
				if t != "" {
					m[t] = struct{}{}
				}
			}
		}
		if len(m) == 0 {
			return nil
		}
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out
	}

	flushScene := func() {
		if strings.TrimSpace(currentScene.Title) != "" || len(currentScene.Lines) > 0 {
			s.Scenes = append(s.Scenes, currentScene)
		}
	}

	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		line := strings.TrimRight(raw, "\r\n")

		// Continuation line (indented) -> append to last dialogue/caption
		if strings.HasPrefix(line, "  ") && lastLine != nil && (lastLine.Type == LineDialogue || lastLine.Type == LineCaption) {
			cont := strings.TrimSpace(line)
			if cont != "" {
				lastLine.Text += "\n" + cont
				// Extract any tags from continuation and merge
				if tags := extractTags(cont); len(tags) > 0 {
					m := map[string]struct{}{}
					for _, t := range lastLine.Tags {
						m[t] = struct{}{}
					}
					for _, t := range tags {
						m[t] = struct{}{}
					}
					merged := make([]string, 0, len(m))
					for k := range m {
						merged = append(merged, k)
					}
					lastLine.Tags = merged
				}
			}
			continue
		}

		trim := strings.TrimSpace(line)
		if trim == "" {
			lastLine = nil
			continue
		}

		// Scene heading
		if m := reScene.FindStringSubmatch(trim); m != nil {
			// Flush previous scene
			flushScene()
			currentScene = Scene{Title: strings.TrimSpace(m[2])}
			lastLine = nil
			continue
		}
		if m := reSceneAlt.FindStringSubmatch(trim); m != nil {
			flushScene()
			currentScene = Scene{Title: strings.TrimSpace(m[1])}
			lastLine = nil
			continue
		}

		// Note line
		if strings.HasPrefix(trim, ";") {
			currentScene.Lines = append(currentScene.Lines, Line{Type: LineNote, Text: strings.TrimSpace(strings.TrimPrefix(trim, ";")), LineNo: lineNo})
			lastLine = nil
			continue
		}

		// Beat
		if m := reBeat.FindStringSubmatch(trim); m != nil {
			text := strings.TrimSpace(m[2])
			tags := extractTags(text)
			currentScene.Lines = append(currentScene.Lines, Line{Type: LineBeat, Character: strings.ToUpper(strings.TrimSpace(m[1])), Text: text, Tags: tags, LineNo: lineNo})
			lastLine = nil
			continue
		}

		// NAME: text or CAPTION/NARRATION
		if m := reName.FindStringSubmatch(trim); m != nil {
			name := strings.TrimSpace(m[1])
			text := strings.TrimSpace(m[2])
			lt := LineDialogue
			upper := strings.ToUpper(name)
			if upper == "CAPTION" || upper == "NARRATION" {
				lt = LineCaption
			}
			tags := extractTags(text)
			ln := Line{Type: lt, Character: upper, Text: text, Tags: tags, LineNo: lineNo}
			currentScene.Lines = append(currentScene.Lines, ln)
			lastLine = &currentScene.Lines[len(currentScene.Lines)-1]
			continue
		}

		// If we reach here and we have no scene yet, start an implicit scene
		if len(s.Scenes) == 0 && strings.TrimSpace(currentScene.Title) == "" && len(currentScene.Lines) == 0 {
			currentScene.Title = "Untitled"
		}
		// Otherwise treat as unknown, accumulate as note to avoid data loss
		currentScene.Lines = append(currentScene.Lines, Line{Type: LineUnknown, Text: trim, LineNo: lineNo})
		lastLine = &currentScene.Lines[len(currentScene.Lines)-1]
	}
	// Append last scene
	flushScene()

	if err := scanner.Err(); err != nil {
		errs = append(errs, Error{Line: lineNo, Column: 1, Message: err.Error()})
	}
	return s, errs
}
