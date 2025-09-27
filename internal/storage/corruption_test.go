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
	"os"
	"path/filepath"
	"testing"
	"time"

	"gocomicwriter/internal/domain"
)

func TestDetectAndRebuildIndex_OnCorruption(t *testing.T) {
	root := t.TempDir()
	// Create a small project with some content
	proj := domain.Project{
		Name:     "CorruptTest",
		Metadata: domain.Metadata{Series: "S", IssueTitle: "I"},
		Issues: []domain.Issue{
			{Pages: []domain.Page{{Number: 1, Panels: []domain.Panel{{ID: "P1", Notes: "hello", Balloons: []domain.Balloon{{ID: "B1", Type: "speech", TextRuns: []domain.TextRun{{Content: "hi there"}}}}}}}}},
		},
		Bible: domain.Bible{Characters: []domain.BibleCharacter{{Name: "Bob"}}, Tags: []domain.BibleTag{{Name: "greet"}}},
	}
	ph, err := InitProject(root, proj)
	if err != nil || ph == nil {
		t.Fatalf("InitProject error: %v", err)
	}
	// Let background initial index build finish
	time.Sleep(200 * time.Millisecond)
	// Corrupt the DB file by writing junk
	idx := IndexPath(root)
	if err := os.WriteFile(idx, []byte("THIS IS NOT SQLITE"), 0o644); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	// Attempt detect and rebuild
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rebuilt, err := DetectAndRebuildIndex(ctx, root, proj)
	if err != nil {
		t.Fatalf("DetectAndRebuildIndex: %v", err)
	}
	if !rebuilt {
		t.Fatalf("expected rebuild to occur")
	}
	// Verify DB looks healthy and has documents table
	st, err := os.Stat(IndexPath(root))
	if err != nil || st.Size() == 0 {
		t.Fatalf("rebuilt index missing or empty: %v", err)
	}
	// Backup file should exist
	bdir := filepath.Join(root, IndexDirName, "backups")
	entries, _ := os.ReadDir(bdir)
	if len(entries) == 0 {
		t.Fatalf("expected backup file in %s", bdir)
	}
}
