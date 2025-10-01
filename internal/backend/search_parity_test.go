/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package backend

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

func openPGForTest(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GCW_PG_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/gocomicwriter?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("cannot open postgres: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skipf("postgres not available: %v", err)
	}
	if err := applyMigrations(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func seedSQLiteProject(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	proj := domain.Project{Name: "Search Test"}
	ph, err := storage.InitProject(root, proj)
	if err != nil || ph == nil {
		t.Fatalf("InitProject error: %v", err)
	}
	// Wait briefly to avoid clobber by background index
	time.Sleep(150 * time.Millisecond)
	// Open DB directly
	idx := storage.IndexPath(root)
	dsn := fmt.Sprintf("file:%s?cache=shared&_pragma=busy_timeout(2000)", filepath.ToSlash(idx))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Seed
	seeds := []struct {
		id        int
		typ, path string
		page      any
		char      any
		text      string
	}{
		{1001, "balloon", "issue:1/page:2/panel:P1/balloon:B1", 2, "bob", "Hello there @greet"},
		{1002, "panel_notes", "issue:1/page:5/panel:P2", 5, nil, "Note with @greet tag and BOB: something"},
		{1003, "script", "script:script.txt", nil, nil, "Beach scene with waves"},
	}
	for _, s := range seeds {
		if _, err := db.ExecContext(ctx, `INSERT INTO documents(doc_id, type, path, page_id, character_id, text) VALUES(?,?,?,?,?,?)`, s.id, s.typ, s.path, s.page, s.char, s.text); err != nil {
			t.Fatalf("sqlite seed: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO cross_refs(from_id, to_id) VALUES(?,?)`, 1002, 1001); err != nil {
		t.Fatalf("sqlite cross_ref: %v", err)
	}
	// small delay for any triggers
	time.Sleep(50 * time.Millisecond)
	return root
}

func seedPGProject(t *testing.T, db *sql.DB) (projectID int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Create project
	if err := db.QueryRowContext(ctx, `INSERT INTO projects(name) VALUES($1) RETURNING id`, "Search Test").Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	// Seed documents with matching IDs
	type doc struct {
		id              int
		typ, path, text string
		page            any
	}
	seeds := []doc{
		{1001, "balloon", "issue:1/page:2/panel:P1/balloon:B1", "Hello there @greet", 2},
		{1002, "panel_notes", "issue:1/page:5/panel:P2", "Note with @greet tag and BOB: something", 5},
		{1003, "script", "script:script.txt", "Beach scene with waves", nil},
	}
	for _, s := range seeds {
		if _, err := db.ExecContext(ctx, `INSERT INTO documents(id, project_id, doc_type, external_ref, raw_text, page_num) VALUES($1,$2,$3,$4,$5,$6)`, s.id, projectID, s.typ, s.path, s.text, s.page); err != nil {
			t.Fatalf("pg seed: %v", err)
		}
	}
	return projectID
}

func idsSet(list []storage.SearchResult) map[int64]bool {
	m := map[int64]bool{}
	for _, r := range list {
		m[r.DocID] = true
	}
	return m
}

func TestSearchParity_SQLite_vs_Postgres(t *testing.T) {
	// SQLite side
	root := seedSQLiteProject(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Postgres side
	db := openPGForTest(t)
	defer func() { _ = db.Close() }()
	pid := seedPGProject(t, db)

	cases := []struct {
		name string
		q    storage.SearchQuery
		want map[int64]bool
	}{
		{"fts_hello", storage.SearchQuery{Text: "Hello"}, map[int64]bool{1001: true}},
		{"tags_range", storage.SearchQuery{Tags: []string{"greet"}, PageFrom: 2, PageTo: 5}, map[int64]bool{1001: true, 1002: true}},
		{"character_bob", storage.SearchQuery{Character: "bob"}, map[int64]bool{1001: true, 1002: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// SQLite
			sres, err := storage.Search(ctx, root, tc.q)
			if err != nil {
				t.Fatalf("sqlite search: %v", err)
			}
			// PG
			pres, err := SearchPG(ctx, db, pid, tc.q)
			if err != nil {
				t.Fatalf("pg search: %v", err)
			}
			// Compare sets against expected and between each other
			sset := idsSet(sres)
			pset := idsSet(pres)
			if len(sset) != len(pset) || len(sset) != len(tc.want) {
				t.Fatalf("mismatch sizes: sqlite=%d pg=%d want=%d", len(sset), len(pset), len(tc.want))
			}
			for id := range tc.want {
				if !sset[id] || !pset[id] {
					t.Fatalf("missing id %d in sqlite=%v pg=%v", id, sset[id], pset[id])
				}
			}
		})
	}
}
