/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gocomicwriter/internal/domain"
	applog "gocomicwriter/internal/log"
	"gocomicwriter/internal/version"
	"log/slog"

	// Pure-Go SQLite driver (CGO-free)
	_ "modernc.org/sqlite"
)

const (
	// IndexDirName stores all per-project ephemeral/index data under the project root.
	IndexDirName  = ".gcw"
	IndexFileName = "index.sqlite"

	// schemaVersion tracks the local SQLite schema for the embedded index.
	// Bump this when you perform breaking schema changes and add migrations.
	schemaVersion = 2
)

// IndexPath returns the full path to the project's embedded index database file.
func IndexPath(projectRoot string) string {
	return filepath.Join(projectRoot, IndexDirName, IndexFileName)
}

// InitOrOpenIndex ensures that the per-project SQLite index exists at .gcw/index.sqlite,
// opens the database, enables WAL mode, and ensures the meta/version tables exist.
// The returned *sql.DB is ready for use. Callers may close it when no longer needed.
func InitOrOpenIndex(projectRoot string) (*sql.DB, error) {
	l := applog.WithOperation(applog.WithComponent("storage"), "index_init").With(
		slog.String("root", projectRoot),
	)
	if stringsTrim(projectRoot) == "" {
		return nil, errors.New("project root is required")
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Join(projectRoot, IndexDirName), 0o755); err != nil {
		l.Error("create .gcw dir failed", slog.Any("err", err))
		return nil, fmt.Errorf("create .gcw dir: %w", err)
	}

	path := IndexPath(projectRoot)
	// Use a URI with shared cache and set busy timeout. Convert to forward slashes for SQLite URI.
	uriPath := filepath.ToSlash(path)
	dsn := fmt.Sprintf("file:%s?cache=shared&_pragma=busy_timeout(5000)", uriPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		l.Error("sqlite open failed", slog.Any("err", err))
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Set reasonable connection pool limits for embedded usage.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure WAL mode is active.
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		l.Error("enable WAL failed", slog.Any("err", err))
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	// Enforce foreign keys just in case future schema uses them.
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON;"); err != nil {
		l.Warn("enable foreign_keys failed", slog.Any("err", err))
	}

	if err := ensureMetaAndVersion(ctx, db); err != nil {
		_ = db.Close()
		l.Error("ensure meta/version failed", slog.Any("err", err))
		return nil, err
	}
	// Ensure core index schema exists (documents, FTS, cross-refs, assets, previews, snapshots)
	if err := ensureIndexSchema(ctx, db); err != nil {
		_ = db.Close()
		l.Error("ensure index schema failed", slog.Any("err", err))
		return nil, err
	}
	// Run migrations to bring DB schema up to date
	if err := runMigrations(ctx, db); err != nil {
		_ = db.Close()
		l.Error("run migrations failed", slog.Any("err", err))
		return nil, err
	}

	l.Info("index ready", slog.String("path", path))
	return db, nil
}

func ensureMetaAndVersion(ctx context.Context, db *sql.DB) error {
	// Create tables if not exist
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS version (
			id          INTEGER PRIMARY KEY CHECK(id=1),
			schema      INTEGER NOT NULL,
			app         TEXT,
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		);`,
	}
	for _, q := range ddl {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	// Seed or update single-row version info
	now := time.Now().UTC().Format(time.RFC3339)
	appv := version.String()
	// Check if a version row exists
	var curSchema int
	err := db.QueryRowContext(ctx, `SELECT schema FROM version WHERE id=1`).Scan(&curSchema)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Insert new row with current schemaVersion for a fresh DB
		if _, err := db.ExecContext(ctx, `INSERT INTO version (id, schema, app, created_at, updated_at) VALUES(1, ?, ?, ?, ?)`, schemaVersion, appv, now, now); err != nil {
			return fmt.Errorf("insert version: %w", err)
		}
	case err != nil:
		return fmt.Errorf("read version: %w", err)
	default:
		// Update app and timestamp only; keep existing schema for migrations
		if _, err := db.ExecContext(ctx, `UPDATE version SET app=?, updated_at=? WHERE id=1`, appv, now); err != nil {
			return fmt.Errorf("update version: %w", err)
		}
	}
	return nil
}

// runMigrations applies incremental schema migrations up to schemaVersion.
func runMigrations(ctx context.Context, db *sql.DB) error {
	var cur int
	if err := db.QueryRowContext(ctx, `SELECT schema FROM version WHERE id=1`).Scan(&cur); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if cur > schemaVersion {
		// Do not downgrade; just log and continue
		return nil
	}
	for cur < schemaVersion {
		next := cur + 1
		switch next {
		case 2:
			// Add helpful indexes for cross-refs and optimize FTS
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("begin migration %d: %w", next, err)
			}
			stmts := []string{
				`CREATE INDEX IF NOT EXISTS idx_cross_refs_to ON cross_refs(to_id);`,
				`CREATE INDEX IF NOT EXISTS idx_cross_refs_from ON cross_refs(from_id);`,
			}
			for _, q := range stmts {
				if _, err := tx.ExecContext(ctx, q); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("migration %d stmt failed: %w", next, err)
				}
			}
			if _, err := tx.ExecContext(ctx, `UPDATE version SET schema=?, updated_at=? WHERE id=1`, next, time.Now().UTC().Format(time.RFC3339)); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("migration %d update version: %w", next, err)
			}
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("migration %d commit: %w", next, err)
			}
			// Best-effort FTS optimize (outside the tx)
			if _, err := db.ExecContext(ctx, `INSERT INTO fts_documents(fts_documents) VALUES('optimize')`); err != nil {
				// best-effort optimize; ignore errors
			}
		default:
			// Unknown future step; break
		}
		cur = next
	}
	return nil
}

// ensureIndexSchema creates core index tables and FTS structures if they do not exist.
func ensureIndexSchema(ctx context.Context, db *sql.DB) error {
	ddl := []string{
		// Core documents table: represents script lines, page text, captions, SFX etc.
		`CREATE TABLE IF NOT EXISTS documents (
			doc_id       INTEGER PRIMARY KEY,
			type         TEXT    NOT NULL,
			path         TEXT    NOT NULL,
			page_id      INTEGER,
			character_id TEXT,
			text         TEXT
		);`,
		// Helpful indices for lookup
		`CREATE INDEX IF NOT EXISTS idx_documents_path ON documents(path);`,
		`CREATE INDEX IF NOT EXISTS idx_documents_page ON documents(page_id);`,

		// Contentless FTS5 index fed from documents via triggers.
		`CREATE VIRTUAL TABLE IF NOT EXISTS fts_documents USING fts5(
			text,
			content='',
			tokenize = 'unicode61'
		);`,

		// Cross references between documents (where-used, links)
		`CREATE TABLE IF NOT EXISTS cross_refs (
			from_id INTEGER NOT NULL,
			to_id   INTEGER NOT NULL,
			PRIMARY KEY(from_id, to_id),
			FOREIGN KEY(from_id) REFERENCES documents(doc_id) ON DELETE CASCADE,
			FOREIGN KEY(to_id)   REFERENCES documents(doc_id) ON DELETE CASCADE
		);`,

		// Assets catalog (images/fonts/etc.)
		`CREATE TABLE IF NOT EXISTS assets (
			hash TEXT PRIMARY KEY,
			path TEXT NOT NULL,
			type TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_assets_path ON assets(path);`,

		// Previews cache (page/panel thumbnails)
		`CREATE TABLE IF NOT EXISTS previews (
			id         INTEGER PRIMARY KEY,
			page_id    INTEGER NOT NULL,
			panel_id   INTEGER,
			thumb_blob BLOB    NOT NULL,
			updated_at TEXT    NOT NULL
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_previews_page_panel ON previews(page_id, panel_id);`,

		// Snapshots (history of page changes)
		`CREATE TABLE IF NOT EXISTS snapshots (
			id         INTEGER PRIMARY KEY,
			page_id    INTEGER NOT NULL,
			ts         TEXT    NOT NULL,
			delta_blob BLOB    NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_page_ts ON snapshots(page_id, ts);`,

		// Script snapshots (history of script text for change tracking)
		`CREATE TABLE IF NOT EXISTS script_snapshots (
			id    INTEGER PRIMARY KEY,
			ts    TEXT    NOT NULL,
			text  TEXT    NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_script_snapshots_ts ON script_snapshots(ts);`,
	}
	for _, q := range ddl {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("ensure index schema: %w", err)
		}
	}
	// Triggers for contentless FTS synchronization with documents.text
	triggers := []string{
		`CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
			INSERT INTO fts_documents(rowid, text) VALUES (new.doc_id, new.text);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
			INSERT INTO fts_documents(fts_documents, rowid, text) VALUES ('delete', old.doc_id, old.text);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE OF text ON documents BEGIN
			INSERT INTO fts_documents(fts_documents, rowid, text) VALUES ('delete', old.doc_id, old.text);
			INSERT INTO fts_documents(rowid, text) VALUES (new.doc_id, new.text);
		END;`,
	}
	for _, q := range triggers {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("ensure fts triggers: %w", err)
		}
	}
	// Perform previews schema migration/additional indexes for caching variants and LRU
	if err := EnsurePreviewsMigrated(ctx, db); err != nil {
		return err
	}
	return nil
}

// DetectAndRebuildIndex checks for corruption or missing schema and rebuilds the index if needed.
// It returns true when a rebuild was performed.
func DetectAndRebuildIndex(ctx context.Context, projectRoot string, proj domain.Project) (bool, error) {
	path := IndexPath(projectRoot)
	// Try to open DB; if fails, attempt backup+delete+rebuild
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		backupIndexFile(path)
		_ = os.Remove(path)
		if rbErr := RebuildIndex(ctx, projectRoot, proj); rbErr != nil {
			return false, fmt.Errorf("rebuild after open failure: %w (open err: %v)", rbErr, err)
		}
		return true, nil
	}
	defer db.Close()
	needs := false
	// quick_check for corruption
	var chk string
	if err := db.QueryRowContext(ctx, `PRAGMA quick_check;`).Scan(&chk); err != nil || !strings.Contains(strings.ToLower(chk), "ok") {
		needs = true
	}
	// Probe core table
	if !needs {
		if _, err := db.ExecContext(ctx, `SELECT 1 FROM documents LIMIT 1;`); err != nil {
			needs = true
		}
	}
	if !needs {
		return false, nil
	}
	// Backup and remove existing DB file
	backupIndexFile(path)
	_ = os.Remove(path)
	// Rebuild
	if err := RebuildIndex(ctx, projectRoot, proj); err != nil {
		return false, err
	}
	return true, nil
}

// backupIndexFile copies the current index file into a timestamped backup in .gcw/backups.
func backupIndexFile(indexPath string) {
	bdir := filepath.Join(filepath.Dir(indexPath), "backups")
	_ = os.MkdirAll(bdir, 0o755)
	stamp := time.Now().Format("20060102-150405")
	bak := filepath.Join(bdir, fmt.Sprintf("%s.%s.bak", filepath.Base(indexPath), stamp))
	if data, err := os.ReadFile(indexPath); err == nil {
		_ = os.WriteFile(bak, data, 0o644)
	}
}

// stringsTrim is a tiny helper to avoid importing strings here just for TrimSpace.
func stringsTrim(s string) string {
	// manual trim of spaces and tabs
	i := 0
	j := len(s)
	for i < j {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		break
	}
	for i < j {
		c := s[j-1]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			j--
			continue
		}
		break
	}
	return s[i:j]
}

// BuildIndexIfEmpty performs a minimal background index build if the index has no user tables.
// For now, this is a no-op placeholder that ensures the database exists; future work will populate FTS and caches.
// BuildIndexIfEmpty performs a minimal background index build if the index has no user content.
// It ensures the DB exists and, if the documents table is empty, populates it from the given manifest and script text.
func BuildIndexIfEmpty(ctx context.Context, projectRoot string, proj domain.Project) error {
	// Ensure the DB exists and is initialized
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	// Check if documents has any rows
	var cnt int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents;").Scan(&cnt); err != nil {
		return fmt.Errorf("check documents count: %w", err)
	}
	if cnt > 0 {
		return nil // already built
	}
	return rebuildDocumentsFromProject(ctx, db, projectRoot, proj)
}

// UpdateIndex updates the embedded index with changes from the project manifest.
// Minimal safe implementation: replace the documents content from the provided manifest.
func UpdateIndex(ctx context.Context, projectRoot string, proj domain.Project) error {
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	return rebuildDocumentsFromProject(ctx, db, projectRoot, proj)
}

// RebuildIndex drops and recreates core index tables and rebuilds content from the manifest.
// It preserves meta/version tables. This is a safe operation; the index is derived from comic.json and assets.
func RebuildIndex(ctx context.Context, projectRoot string, proj domain.Project) error {
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	// Drop core tables inside a transaction and recreate schema
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	drops := []string{
		"DROP TABLE IF EXISTS cross_refs;",
		"DROP TABLE IF EXISTS assets;",
		"DROP TABLE IF EXISTS previews;",
		"DROP TABLE IF EXISTS snapshots;",
		"DROP TRIGGER IF EXISTS documents_ai;",
		"DROP TRIGGER IF EXISTS documents_ad;",
		"DROP TRIGGER IF EXISTS documents_au;",
		"DROP TABLE IF EXISTS documents;",
		"DROP TABLE IF EXISTS fts_documents;",
	}
	for _, q := range drops {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("drop schema: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("drop commit: %w", err)
	}
	// Recreate schema and populate
	if err := ensureIndexSchema(ctx, db); err != nil {
		return err
	}
	return rebuildDocumentsFromProject(ctx, db, projectRoot, proj)
}

// rebuildDocumentsFromProject replaces the documents table content from the given project manifest and script text.
func rebuildDocumentsFromProject(ctx context.Context, db *sql.DB, projectRoot string, proj domain.Project) error {
	// Build list of rows
	type row struct {
		typeStr     string
		path        string
		pageID      sql.NullInt64
		characterID sql.NullString
		text        string
	}
	rows := make([]row, 0, 256)
	// Project-level metadata
	if s := stringsTrim(proj.Name); s != "" {
		rows = append(rows, row{typeStr: "project_name", path: "project:name", text: s})
	}
	if s := stringsTrim(proj.Metadata.Series); s != "" {
		rows = append(rows, row{typeStr: "project_series", path: "project:series", text: s})
	}
	if s := stringsTrim(proj.Metadata.IssueTitle); s != "" {
		rows = append(rows, row{typeStr: "issue_title", path: "project:issue_title", text: s})
	}
	if s := stringsTrim(proj.Metadata.Creators); s != "" {
		rows = append(rows, row{typeStr: "creators", path: "project:creators", text: s})
	}
	if s := stringsTrim(proj.Metadata.Notes); s != "" {
		rows = append(rows, row{typeStr: "project_notes", path: "project:notes", text: s})
	}
	// Bible entries
	for _, bc := range proj.Bible.Characters {
		if s := stringsTrim(bc.Name); s != "" {
			rows = append(rows, row{typeStr: "character", path: "bible:character:" + s, text: s})
		}
		if s := stringsTrim(strings.Join(bc.Aliases, ", ")); s != "" {
			rows = append(rows, row{typeStr: "character_aliases", path: "bible:character_aliases:" + bc.Name, text: s})
		}
		if s := stringsTrim(bc.Notes); s != "" {
			rows = append(rows, row{typeStr: "character_notes", path: "bible:character_notes:" + bc.Name, text: s})
		}
	}
	for _, bl := range proj.Bible.Locations {
		if s := stringsTrim(bl.Name); s != "" {
			rows = append(rows, row{typeStr: "location", path: "bible:location:" + s, text: s})
		}
		if s := stringsTrim(strings.Join(bl.Aliases, ", ")); s != "" {
			rows = append(rows, row{typeStr: "location_aliases", path: "bible:location_aliases:" + bl.Name, text: s})
		}
		if s := stringsTrim(bl.Notes); s != "" {
			rows = append(rows, row{typeStr: "location_notes", path: "bible:location_notes:" + bl.Name, text: s})
		}
	}
	for _, bt := range proj.Bible.Tags {
		if s := stringsTrim(bt.Name); s != "" {
			rows = append(rows, row{typeStr: "tag", path: "bible:tag:" + s, text: s})
		}
		if s := stringsTrim(bt.Notes); s != "" {
			rows = append(rows, row{typeStr: "tag_notes", path: "bible:tag_notes:" + bt.Name, text: s})
		}
	}
	// Issues/pages/panels/balloons
	for issIdx, iss := range proj.Issues {
		_ = issIdx // reserved for multi-issue future
		for _, pg := range iss.Pages {
			pageID := int64(pg.Number)
			// Panel notes and balloon texts
			for _, pnl := range pg.Panels {
				if s := stringsTrim(pnl.Notes); s != "" {
					rows = append(rows, row{typeStr: "panel_notes", path: fmt.Sprintf("issue:1/page:%d/panel:%s", pg.Number, pnl.ID), pageID: sql.NullInt64{Int64: pageID, Valid: true}, text: s})
				}
				for _, bln := range pnl.Balloons {
					// Aggregate text runs
					buf := make([]byte, 0, 64)
					for _, tr := range bln.TextRuns {
						ct := stringsTrim(tr.Content)
						if ct == "" {
							continue
						}
						if len(buf) > 0 {
							buf = append(buf, ' ')
						}
						buf = append(buf, ct...)
					}
					if len(buf) > 0 {
						rows = append(rows, row{typeStr: "balloon", path: fmt.Sprintf("issue:1/page:%d/panel:%s/balloon:%s", pg.Number, pnl.ID, bln.ID), pageID: sql.NullInt64{Int64: pageID, Valid: true}, text: string(buf)})
					}
				}
			}
		}
	}
	// Script text (if present)
	scriptPath := filepath.Join(projectRoot, "script", "script.txt")
	if b, err := os.ReadFile(scriptPath); err == nil {
		if s := stringsTrim(string(b)); s != "" {
			rows = append(rows, row{typeStr: "script", path: "script:script.txt", text: s})
		}
	}
	// Write in a transaction: clear documents and insert new rows.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM documents;"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("clear documents: %w", err)
	}
	ins, err := tx.PrepareContext(ctx, "INSERT INTO documents(type, path, page_id, character_id, text) VALUES(?,?,?,?,?);")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer ins.Close()
	for _, r := range rows {
		if _, err := ins.ExecContext(ctx, r.typeStr, r.path, r.pageID, r.characterID, r.text); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert document: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
