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
	"errors"
	"time"
)

// language=SQL
// dialect=SQLite
const insertSnapshotSQL = `INSERT INTO snapshots(page_id, ts, delta_blob) VALUES (?, ?, ?)`

// language=SQL
// dialect=SQLite
const selectLatestSnapshotSQL = `SELECT ts, delta_blob FROM snapshots WHERE page_id = ? ORDER BY ts DESC LIMIT 1`

// language=SQL
// dialect=SQLite
const listSnapshotsSQL = `SELECT ts, delta_blob FROM snapshots WHERE page_id = ? ORDER BY ts DESC LIMIT ?`

// language=SQL
// dialect=SQLite
const pruneOldSnapshotsSQL = `DELETE FROM snapshots WHERE page_id = ? AND id NOT IN (
	SELECT id FROM snapshots WHERE page_id = ? ORDER BY ts DESC LIMIT ?
)`

// SaveSnapshot persists a page snapshot delta blob with a timestamp.
// It opens the project's index database if needed and inserts the record.
func SaveSnapshot(ctx context.Context, ph *ProjectHandle, pageNumber int, delta []byte, ts time.Time) error {
	if ph == nil {
		return errors.New("nil ProjectHandle")
	}
	// Open or init index DB
	db, err := InitOrOpenIndex(ph.Root)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	_, err = db.ExecContext(ctx, insertSnapshotSQL, pageNumber, ts.UTC().Format(time.RFC3339Nano), delta)
	return err
}

// GetLatestSnapshot returns the latest snapshot blob for a page or nil if none.
func GetLatestSnapshot(ctx context.Context, ph *ProjectHandle, pageNumber int) ([]byte, time.Time, error) {
	if ph == nil {
		return nil, time.Time{}, errors.New("nil ProjectHandle")
	}
	db, err := InitOrOpenIndex(ph.Root)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer func() { _ = db.Close() }()
	var tsStr string
	var blob []byte
	err = db.QueryRowContext(ctx, selectLatestSnapshotSQL, pageNumber).Scan(&tsStr, &blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	ts, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return blob, time.Time{}, nil // return blob even if ts parse fails
	}
	return blob, ts, nil
}

// ListSnapshots returns up to limit most recent snapshots for a page.
func ListSnapshots(ctx context.Context, ph *ProjectHandle, pageNumber int, limit int) ([]struct {
	TS   time.Time
	Blob []byte
}, error) {
	if ph == nil {
		return nil, errors.New("nil ProjectHandle")
	}
	if limit <= 0 {
		limit = 50
	}
	db, err := InitOrOpenIndex(ph.Root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	rows, err := db.QueryContext(ctx, listSnapshotsSQL, pageNumber, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []struct {
		TS   time.Time
		Blob []byte
	}
	for rows.Next() {
		var tsStr string
		var blob []byte
		if err := rows.Scan(&tsStr, &blob); err != nil {
			return nil, err
		}
		ts, _ := time.Parse(time.RFC3339Nano, tsStr)
		out = append(out, struct {
			TS   time.Time
			Blob []byte
		}{TS: ts, Blob: blob})
	}
	return out, rows.Err()
}

// PruneOldSnapshots keeps at most keepLast snapshots for the page and deletes older ones.
func PruneOldSnapshots(ctx context.Context, ph *ProjectHandle, pageNumber int, keepLast int) (int64, error) {
	if ph == nil {
		return 0, errors.New("nil ProjectHandle")
	}
	if keepLast <= 0 {
		return 0, nil
	}
	db, err := InitOrOpenIndex(ph.Root)
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()
	// Delete snapshots not in the newest keepLast set
	res, err := db.ExecContext(ctx, pruneOldSnapshotsSQL, pageNumber, pageNumber, keepLast)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
