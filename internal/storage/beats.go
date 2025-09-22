/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"fmt"
	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/script"
)

// BeatIDFor returns a stable identifier for a beat line.
// For now, we key by the absolute source line number to keep things simple and deterministic.
// Format: b:<lineNo> (e.g., b:42)
func BeatIDFor(sceneIdx int, ln script.Line) string { // sceneIdx reserved for future; not used yet
	_ = sceneIdx // explicitly mark unused for now to avoid linter warning
	return fmt.Sprintf("b:%d", ln.LineNo)
}

// MappedBeatSet returns a set of beat IDs that are already linked from any panel in the project.
func MappedBeatSet(p domain.Project) map[string]struct{} {
	s := make(map[string]struct{})
	for _, iss := range p.Issues {
		for _, pg := range iss.Pages {
			for _, pn := range pg.Panels {
				for _, id := range pn.BeatIDs {
					if id == "" {
						continue
					}
					s[id] = struct{}{}
				}
			}
		}
	}
	return s
}

// ComputeUnmappedBeats returns a slice of beat IDs that are present in the parsed script but not
// referenced by any panel's linkedBeats in the given project.
func ComputeUnmappedBeats(sc script.Script, p domain.Project) []string {
	mapped := MappedBeatSet(p)
	var out []string
	for si, scn := range sc.Scenes {
		for _, ln := range scn.Lines {
			if ln.Type == script.LineBeat {
				id := BeatIDFor(si, ln)
				if _, ok := mapped[id]; !ok {
					out = append(out, id)
				}
			}
		}
	}
	return out
}

// MapBeatToPanel adds the given beatID to the specified panel (by page number and panel ID)
// if it's not already included. Returns an error if the page or panel cannot be found.
func MapBeatToPanel(ph *ProjectHandle, pageNumber int, panelID string, beatID string) error {
	if ph == nil {
		return fmt.Errorf("project handle is nil")
	}
	if beatID == "" {
		return fmt.Errorf("beatID is empty")
	}
	for i := range ph.Project.Issues {
		iss := &ph.Project.Issues[i]
		for j := range iss.Pages {
			pg := &iss.Pages[j]
			if pg.Number != pageNumber {
				continue
			}
			for k := range pg.Panels {
				pn := &pg.Panels[k]
				if pn.ID != panelID {
					continue
				}
				// check exists
				for _, id := range pn.BeatIDs {
					if id == beatID {
						return nil // already mapped
					}
				}
				pn.BeatIDs = append(pn.BeatIDs, beatID)
				return nil
			}
			return fmt.Errorf("panel %s not found on page %d", panelID, pageNumber)
		}
	}
	return fmt.Errorf("page %d not found", pageNumber)
}
