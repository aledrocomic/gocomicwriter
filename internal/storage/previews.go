/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * Licensed under the Apache License, Version 2.0.
 */

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// PreviewKind is a type discriminator for previews table rows.
// - thumb: raster thumbnail image (PNG) for whole page or a panel
// - geom: geometry cache blob (implementation-defined; JSON or binary)
const (
	PreviewKindThumb = "thumb"
	PreviewKindGeom  = "geom"
)

// EnsurePreviewsMigrated guarantees the previews table has columns needed for
// caching variants and LRU tracking. It is safe to call multiple times.
func EnsurePreviewsMigrated(ctx context.Context, db *sql.DB) error {
	// Ensure table exists (older ensureIndexSchema will have created a minimal version)
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS previews (
		id           INTEGER PRIMARY KEY,
		page_id      INTEGER NOT NULL,
		panel_id     INTEGER,
		thumb_blob   BLOB,
		updated_at   TEXT NOT NULL
	);`); err != nil {
		return fmt.Errorf("ensure previews table: %w", err)
	}
	// If the table definition still enforces NOT NULL on thumb_blob, rebuild table with new schema
	var tblSQL string
	_ = db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='previews'`).Scan(&tblSQL)
	if tblSQL != "" && (containsIgnoreCase(tblSQL, "thumb_blob BLOB NOT NULL") || containsIgnoreCase(tblSQL, "thumb_blob BLOB    NOT NULL")) {
		// Rebuild into previews_new
		rebuild := []string{
			`CREATE TABLE IF NOT EXISTS previews_new (
				id           INTEGER PRIMARY KEY,
				page_id      INTEGER NOT NULL,
				panel_id     INTEGER,
				kind         TEXT    NOT NULL DEFAULT 'thumb',
				w            INTEGER NOT NULL DEFAULT 0,
				h            INTEGER NOT NULL DEFAULT 0,
				thumb_blob   BLOB,
				geom_blob    BLOB,
				size         INTEGER NOT NULL DEFAULT 0,
				updated_at   TEXT    NOT NULL,
				last_access  TEXT
			);`,
			// Copy existing rows as thumb variants, size from blob length
			`INSERT INTO previews_new(id,page_id,panel_id,kind,w,h,thumb_blob,size,updated_at,last_access)
				SELECT id,page_id,panel_id,'thumb',0,0,thumb_blob,COALESCE(length(thumb_blob),0),updated_at,NULL FROM previews;`,
			`DROP TABLE previews;`,
			`ALTER TABLE previews_new RENAME TO previews;`,
		}
		for _, q := range rebuild {
			if _, err := db.ExecContext(ctx, q); err != nil {
				return fmt.Errorf("rebuild previews: %w", err)
			}
		}
	}
	// Inspect current columns
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(previews);`)
	if err != nil {
		return fmt.Errorf("table_info previews: %w", err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		cols[name] = true
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	// Add missing columns for new schema
	alter := func(sqlStmt string) error {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return err
		}
		return nil
	}
	if !cols["kind"] {
		if err := alter(`ALTER TABLE previews ADD COLUMN kind TEXT DEFAULT 'thumb'`); err != nil {
			return fmt.Errorf("add kind: %w", err)
		}
	}
	if !cols["w"] {
		if err := alter(`ALTER TABLE previews ADD COLUMN w INTEGER DEFAULT 0`); err != nil {
			return fmt.Errorf("add w: %w", err)
		}
	}
	if !cols["h"] {
		if err := alter(`ALTER TABLE previews ADD COLUMN h INTEGER DEFAULT 0`); err != nil {
			return fmt.Errorf("add h: %w", err)
		}
	}
	if !cols["size"] {
		if err := alter(`ALTER TABLE previews ADD COLUMN size INTEGER DEFAULT 0`); err != nil {
			return fmt.Errorf("add size: %w", err)
		}
	}
	if !cols["last_access"] {
		if err := alter(`ALTER TABLE previews ADD COLUMN last_access TEXT`); err != nil {
			return fmt.Errorf("add last_access: %w", err)
		}
	}
	if !cols["geom_blob"] {
		if err := alter(`ALTER TABLE previews ADD COLUMN geom_blob BLOB`); err != nil {
			return fmt.Errorf("add geom_blob: %w", err)
		}
	}
	// Drop old unique index to allow multiple variants, if it exists
	_, _ = db.ExecContext(ctx, `DROP INDEX IF EXISTS ux_previews_page_panel`)
	// Create new unique index covering variant and size
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS ux_previews_variant ON previews(page_id, panel_id, kind, w, h)`); err != nil {
		return fmt.Errorf("create variant index: %w", err)
	}
	// Also helpful index for LRU eviction by access time
	_, _ = db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_previews_access ON previews(last_access)`)
	return nil
}

func containsIgnoreCase(s, sub string) bool {
	ls := strings.ToLower(s)
	return strings.Contains(ls, strings.ToLower(sub))
}

// GetPreview returns the blob bytes for a preview of given key and updates last_access.
// For kind==thumb, returns the thumb blob; for kind==geom, returns the geometry blob.
func GetPreview(ctx context.Context, projectRoot string, pageID int, panelID sql.NullInt64, kind string, w int, h int) ([]byte, error) {
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := EnsurePreviewsMigrated(ctx, db); err != nil {
		return nil, err
	}
	col := "thumb_blob"
	if kind == PreviewKindGeom {
		col = "geom_blob"
	}
	q := fmt.Sprintf("SELECT %s FROM previews WHERE page_id=? AND panel_id IS ? AND kind=? AND w=? AND h=?", col)
	var blob []byte
	// Note: panelID may be NULL; use IS ? which compares NULLs in SQLite when arg is nil
	var pn any
	if panelID.Valid {
		pn = panelID.Int64
	} else {
		pn = nil
	}
	err = db.QueryRowContext(ctx, q, pageID, pn, kind, w, h).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query preview: %w", err)
	}
	// touch
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.ExecContext(ctx, `UPDATE previews SET last_access=? WHERE page_id=? AND panel_id IS ? AND kind=? AND w=? AND h=?`, now, pageID, pn, kind, w, h)
	return blob, nil
}

// PutPreview upserts a preview blob and enforces the cache size cap via LRU eviction.
// For kind==thumb, blob bytes should be PNG or JPEG; for kind==geom, arbitrary.
func PutPreview(ctx context.Context, projectRoot string, pageID int, panelID sql.NullInt64, kind string, w int, h int, blob []byte) error {
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := EnsurePreviewsMigrated(ctx, db); err != nil {
		return err
	}
	if kind != PreviewKindThumb && kind != PreviewKindGeom {
		return fmt.Errorf("invalid kind: %s", kind)
	}
	pn := any(nil)
	if panelID.Valid {
		pn = panelID.Int64
	}
	now := time.Now().UTC().Format(time.RFC3339)
	size := len(blob)
	// Upsert
	if kind == PreviewKindThumb {
		_, err = db.ExecContext(ctx, `INSERT INTO previews(page_id,panel_id,kind,w,h,thumb_blob,size,updated_at,last_access)
			VALUES(?,?,?,?,?,?,?,?,?)
			ON CONFLICT(page_id,panel_id,kind,w,h) DO UPDATE SET thumb_blob=excluded.thumb_blob, size=excluded.size, updated_at=excluded.updated_at, last_access=excluded.last_access`,
			pageID, pn, kind, w, h, blob, size, now, now)
	} else {
		_, err = db.ExecContext(ctx, `INSERT INTO previews(page_id,panel_id,kind,w,h,geom_blob,size,updated_at,last_access)
			VALUES(?,?,?,?,?,?,?,?,?)
			ON CONFLICT(page_id,panel_id,kind,w,h) DO UPDATE SET geom_blob=excluded.geom_blob, size=excluded.size, updated_at=excluded.updated_at, last_access=excluded.last_access`,
			pageID, pn, kind, w, h, blob, size, now, now)
	}
	if err != nil {
		return fmt.Errorf("upsert preview: %w", err)
	}
	// Enforce cap
	capBytes := MaxPreviewsBytesFromEnv()
	if capBytes > 0 {
		if err := EvictPreviewsToFit(ctx, db, capBytes); err != nil {
			return err
		}
	}
	return nil
}

// GetOrCreatePreview fetches a preview or generates and stores it using the provided generator.
func GetOrCreatePreview(ctx context.Context, projectRoot string, pageID int, panelID sql.NullInt64, kind string, w int, h int, gen func(context.Context) ([]byte, error)) ([]byte, error) {
	// Try to get existing first
	if b, err := GetPreview(ctx, projectRoot, pageID, panelID, kind, w, h); err != nil {
		return nil, err
	} else if b != nil {
		return b, nil
	}
	if gen == nil {
		return nil, nil
	}
	data, err := gen(ctx)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	if err := PutPreview(ctx, projectRoot, pageID, panelID, kind, w, h, data); err != nil {
		return nil, err
	}
	return data, nil
}

// EvictPreviewsToFit deletes least-recently-used rows until total size <= capBytes.
func EvictPreviewsToFit(ctx context.Context, db *sql.DB, capBytes int64) error {
	var total int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size),0) FROM previews`).Scan(&total); err != nil {
		return fmt.Errorf("sum previews size: %w", err)
	}
	if total <= capBytes {
		return nil
	}
	// Select victim ids ordered by last_access asc (oldest first), NULLs first
	rows, err := db.QueryContext(ctx, `SELECT id, size FROM previews ORDER BY 
		CASE WHEN last_access IS NULL THEN 0 ELSE 1 END ASC, last_access ASC`)
	if err != nil {
		return fmt.Errorf("select victims: %w", err)
	}
	toDelete := make([]int64, 0, 32)
	var cur = total
	for rows.Next() {
		var id int64
		var sz int64
		if err := rows.Scan(&id, &sz); err != nil {
			_ = rows.Close()
			return err
		}
		toDelete = append(toDelete, id)
		cur -= sz
		if cur <= capBytes {
			break
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	// Important: close the rows cursor before attempting to write
	if err := rows.Close(); err != nil {
		return err
	}
	if len(toDelete) == 0 {
		return nil
	}
	// Delete selected ids
	// Build placeholders
	sqlBase := `DELETE FROM previews WHERE id IN (`
	for i := range toDelete {
		if i > 0 {
			sqlBase += ","
		}
		sqlBase += "?"
	}
	sqlBase += ")"
	args := make([]any, len(toDelete))
	for i, v := range toDelete {
		args[i] = v
	}
	if _, err := db.ExecContext(ctx, sqlBase, args...); err != nil {
		return fmt.Errorf("evict delete: %w", err)
	}
	return nil
}

// TotalPreviewBytes returns total bytes tracked by previews.size
func TotalPreviewBytes(ctx context.Context, projectRoot string) (int64, error) {
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var total int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size),0) FROM previews`).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// MaxPreviewsBytesFromEnv reads GCW_PREVIEWS_MAX_BYTES, defaulting to 256MB if unset.
func MaxPreviewsBytesFromEnv() int64 {
	v := os.Getenv("GCW_PREVIEWS_MAX_BYTES")
	if v == "" {
		return 256 * 1024 * 1024 // 256MB
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return 256 * 1024 * 1024
	}
	return n
}
