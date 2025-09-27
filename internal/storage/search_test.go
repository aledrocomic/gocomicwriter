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
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gocomicwriter/internal/domain"
)

func TestSearchAndWhereUsed(t *testing.T) {
	root := t.TempDir()
	// Initialize project to bootstrap index
	proj := domain.Project{Name: "Search Test"}
	ph, err := InitProject(root, proj)
	if err != nil || ph == nil {
		t.Fatalf("InitProject error: %v", err)
	}
	// Give background initial index build a moment to complete to avoid clobbering our seeds
	time.Sleep(200 * time.Millisecond)
	// Open DB directly
	idx := IndexPath(root)
	dsn := fmt.Sprintf("file:%s?cache=shared&_pragma=busy_timeout(2000)", filepath.ToSlash(idx))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Seed a few documents with distinct patterns
	// Use high doc_ids to avoid collisions
	seed := []struct {
		id      int
		typeStr string
		path    string
		page    any
		char    any
		text    string
	}{
		{1001, "balloon", "issue:1/page:2/panel:P1/balloon:B1", 2, "bob", "Hello there @greet"},
		{1002, "panel_notes", "issue:1/page:5/panel:P2", 5, nil, "Note with @greet tag and BOB: something"},
		{1003, "script", "script:script.txt", nil, nil, "Beach scene with waves"},
	}
	for _, s := range seed {
		_, err := db.ExecContext(ctx, `INSERT INTO documents(doc_id, type, path, page_id, character_id, text) VALUES(?,?,?,?,?,?)`, s.id, s.typeStr, s.path, s.page, s.char, s.text)
		if err != nil {
			t.Fatalf("seed insert: %v", err)
		}
	}
	// Cross-ref: 1002 references 1001
	if _, err := db.ExecContext(ctx, `INSERT INTO cross_refs(from_id, to_id) VALUES(?,?)`, 1002, 1001); err != nil {
		t.Fatalf("insert cross_ref: %v", err)
	}

	// Allow triggers to process
	time.Sleep(50 * time.Millisecond)

	// 1) FTS search for term 'Hello'
	res, err := Search(ctx, root, SearchQuery{Text: "Hello"})
	if err != nil {
		t.Fatalf("search 1: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected results for 'Hello'")
	}
	found := false
	for _, r := range res {
		if r.DocID == 1001 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected doc 1001 in results")
	}

	// 2) Tag filter @greet within page range 2..5
	res, err = Search(ctx, root, SearchQuery{Tags: []string{"greet"}, PageFrom: 2, PageTo: 5})
	if err != nil {
		t.Fatalf("search 2: %v", err)
	}
	// Should include 1001 and 1002
	want := map[int]bool{1001: true, 1002: true}
	for _, r := range res {
		delete(want, int(r.DocID))
	}
	if len(want) != 0 {
		t.Fatalf("missing expected docs after tag+range filter: %v", want)
	}

	// 3) Character filter 'bob' should find 1001 and 1002 (contains 'BOB:')
	res, err = Search(ctx, root, SearchQuery{Character: "bob"})
	if err != nil {
		t.Fatalf("search 3: %v", err)
	}
	want = map[int]bool{1001: true, 1002: true}
	for _, r := range res {
		delete(want, int(r.DocID))
	}
	if len(want) != 0 {
		t.Fatalf("missing expected docs for character filter: %v", want)
	}

	// 4) Where-used from 1001 should return 1002
	wused, err := WhereUsed(ctx, root, 1001, 100, 0)
	if err != nil {
		t.Fatalf("where-used: %v", err)
	}
	if len(wused) == 0 || wused[0].DocID != 1002 {
		t.Fatalf("expected where-used result 1002, got %+v", wused)
	}
}
