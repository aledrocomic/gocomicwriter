/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package storage

import (
	"context"
	"testing"
	"time"

	"gocomicwriter/internal/domain"
)

// Validates FTS5 and cross-ref queries using index built from a domain.Project, reflecting the concept doc fields.
func TestIndexBuildFromProjectFTSAndCrossRef(t *testing.T) {
	root := t.TempDir()
	proj := domain.Project{
		Name:     "Concept Case",
		Metadata: domain.Metadata{Series: "Series X", IssueTitle: "Ep 1", Creators: "A Drost"},
		Bible: domain.Bible{
			Characters: []domain.BibleCharacter{{Name: "Alice", Aliases: []string{"Al"}}},
			Locations:  []domain.BibleLocation{{Name: "Beach"}},
			Tags:       []domain.BibleTag{{Name: "greet"}},
		},
		Issues: []domain.Issue{{
			Pages: []domain.Page{{
				Number: 1,
				Panels: []domain.Panel{{
					ID:    "P1",
					Notes: "Intro panel",
					Balloons: []domain.Balloon{{
						ID:       "B1",
						Type:     "speech",
						TextRuns: []domain.TextRun{{Content: "Hello from Alice @greet at the beach"}},
					}},
				}},
			}},
		}},
	}
	ph, err := InitProject(root, proj)
	if err != nil || ph == nil {
		t.Fatalf("InitProject: %v", err)
	}
	// Wait for background first build to complete to avoid locking
	time.Sleep(300 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := RebuildIndex(ctx, root, proj); err != nil {
		t.Fatalf("RebuildIndex: %v", err)
	}
	// FTS: search phrase Hello
	res, err := Search(ctx, root, SearchQuery{Text: "Hello"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected FTS results for 'Hello'")
	}
	// Tag filter
	res, err = Search(ctx, root, SearchQuery{Tags: []string{"greet"}})
	if err != nil || len(res) == 0 {
		t.Fatalf("Search tags: %v len=%d", err, len(res))
	}
	// Character filter should find balloon and possibly notes
	res, err = Search(ctx, root, SearchQuery{Character: "alice"})
	if err != nil || len(res) == 0 {
		t.Fatalf("Search character: %v len=%d", err, len(res))
	}
}
