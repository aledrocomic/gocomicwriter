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

	"gocomicwriter/internal/domain"

	_ "modernc.org/sqlite"
)

func TestIndexInitCreatesWALAndMetaVersion(t *testing.T) {
	root := t.TempDir()
	// Initialize minimal project to trigger index init and background build
	proj := domain.Project{Name: "Index Test"}
	ph, err := InitProject(root, proj)
	if err != nil {
		t.Fatalf("InitProject error: %v", err)
	}
	if ph == nil {
		t.Fatalf("expected project handle")
	}
	idxPath := IndexPath(root)
	if _, err := os.Stat(idxPath); err != nil {
		t.Fatalf("index file missing at %s: %v", idxPath, err)
	}
	// Open DB and verify journal mode and tables
	uriPath := filepath.ToSlash(idxPath)
	dsn := fmt.Sprintf("file:%s?cache=shared&_pragma=busy_timeout(2000)", uriPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var mode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&mode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if mode != "wal" && mode != "WAL" {
		t.Fatalf("expected WAL mode, got %s", mode)
	}
	// Check meta/version tables exist
	var cnt int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('meta','version')").Scan(&cnt); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if cnt != 2 {
		t.Fatalf("expected 2 meta tables, got %d", cnt)
	}
	// Check core schema tables exist (including virtual table)
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('documents','fts_documents','cross_refs','assets','previews','snapshots')").Scan(&cnt); err != nil {
		t.Fatalf("query core tables: %v", err)
	}
	if cnt != 6 {
		t.Fatalf("expected 6 core tables, got %d", cnt)
	}
	// Allow any background index build to settle to avoid clobbering our insert
	time.Sleep(150 * time.Millisecond)
	// Insert a document with a high doc_id to avoid collisions and verify FTS triggers populate index
	if _, err := db.ExecContext(ctx, `INSERT INTO documents(doc_id, type, path, page_id, character_id, text) VALUES(10001,'script','issue1/script:1',NULL,'CHAR_A','hello world');`); err != nil {
		t.Fatalf("insert document: %v", err)
	}
	var ftsCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM fts_documents WHERE fts_documents MATCH 'hello' ").Scan(&ftsCount); err != nil {
		t.Fatalf("fts query: %v", err)
	}
	if ftsCount == 0 {
		t.Fatalf("expected FTS to find inserted document")
	}
}
