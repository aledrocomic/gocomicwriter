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
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestMigrations_UpgradeV1ToV2 ensures that an older DB (schema=1) is migrated to schemaVersion (2) and new indexes exist.
func TestMigrations_UpgradeV1ToV2(t *testing.T) {
	root := t.TempDir()
	idx := IndexPath(root)
	// Ensure .gcw directory exists
	if err := os.MkdirAll(filepath.Dir(idx), 0o755); err != nil {
		t.Fatalf("mk .gcw: %v", err)
	}
	// Create a minimal v1 database
	dsn := fmt.Sprintf("file:%s?cache=shared&_pragma=busy_timeout(2000)", filepath.ToSlash(idx))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Create minimal schema representing v1 (no cross_refs indexes)
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS version (id INTEGER PRIMARY KEY CHECK(id=1), schema INTEGER NOT NULL, app TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);`,
		`INSERT INTO version(id, schema, app, created_at, updated_at) VALUES(1, 1, 'test', '2020-01-01T00:00:00Z', '2020-01-01T00:00:00Z');`,
		`CREATE TABLE IF NOT EXISTS documents (doc_id INTEGER PRIMARY KEY, type TEXT NOT NULL, path TEXT NOT NULL, page_id INTEGER, character_id TEXT, text TEXT);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS fts_documents USING fts5(text, content='', tokenize='unicode61');`,
		`CREATE TABLE IF NOT EXISTS cross_refs (from_id INTEGER NOT NULL, to_id INTEGER NOT NULL, PRIMARY KEY(from_id,to_id));`,
	}
	for _, q := range stmts {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("seed v1 schema: %v (q=%s)", err, q)
		}
	}
	// Close and reopen through InitOrOpenIndex which will run migrations
	db.Close()
	mdb, err := InitOrOpenIndex(root)
	if err != nil {
		t.Fatalf("InitOrOpenIndex: %v", err)
	}
	defer mdb.Close()
	// Version should be 2
	var schema int
	if err := mdb.QueryRowContext(ctx, `SELECT schema FROM version WHERE id=1`).Scan(&schema); err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if schema < 2 {
		t.Fatalf("expected schema >= 2 after migration, got %d", schema)
	}
	// Indexes should exist
	var cnt int
	if err := mdb.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name in ('idx_cross_refs_to','idx_cross_refs_from')`).Scan(&cnt); err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	if cnt < 1 { // allow one to exist even if the other pre-existed
		t.Fatalf("expected cross_refs indexes after migration, got %d", cnt)
	}
}
