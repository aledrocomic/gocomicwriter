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
	"encoding/json"
	"testing"
	"time"

	"gocomicwriter/internal/storage"
)

func TestE2E_BackendSchemaAndSearch(t *testing.T) {
	db := openPGForTest(t)
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Insert a project and an index snapshot per concept doc
	var pid int64
	if err := db.QueryRowContext(ctx, `INSERT INTO projects(name, description) VALUES($1,$2) RETURNING id`, "E2E Project", "demo").Scan(&pid); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	// Snapshot payload: small JSON
	snap := map[string]any{"ok": true, "version": 1}
	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO index_snapshots(project_id, version, snapshot) VALUES($1,$2,$3)`, pid, 1, string(b)); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
	// Fetch latest snapshot similar to server route
	var ver int64
	var raw string
	if err := db.QueryRowContext(ctx, `SELECT version, snapshot FROM index_snapshots WHERE project_id=$1 ORDER BY version DESC, id DESC LIMIT 1`, pid).Scan(&ver, &raw); err != nil {
		t.Fatalf("select snapshot: %v", err)
	}
	if ver != 1 || raw == "" {
		t.Fatalf("unexpected snapshot ver=%d raw_empty=%v", ver, raw == "")
	}

	// Seed a document and search it end-to-end through SearchPG
	if _, err := db.ExecContext(ctx, `INSERT INTO documents(id, project_id, doc_type, external_ref, raw_text, page_num) VALUES($1,$2,$3,$4,$5,$6)`, 2001, pid, "script", "script:intro.txt", "Sunrise over the city", 1); err != nil {
		t.Fatalf("seed doc: %v", err)
	}
	res, err := SearchPG(ctx, db, pid, storage.SearchQuery{Text: "Sunrise"})
	if err != nil {
		t.Fatalf("searchpg: %v", err)
	}
	if len(res) == 0 || res[0].DocID != 2001 {
		t.Fatalf("expected result doc 2001, got %+v", res)
	}
}
