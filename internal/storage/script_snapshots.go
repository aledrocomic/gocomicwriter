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
const insertScriptSnapshotSQL = `INSERT INTO script_snapshots(ts, text) VALUES (?, ?)`

// language=SQL
// dialect=SQLite
const selectLatestScriptSnapshotSQL = `SELECT ts, text FROM script_snapshots ORDER BY ts DESC LIMIT 1`

// language=SQL
// dialect=SQLite
const listScriptSnapshotsSQL = `SELECT ts, text FROM script_snapshots ORDER BY ts DESC LIMIT ?`

// language=SQL
// dialect=SQLite
const pruneOldScriptSnapshotsSQL = `DELETE FROM script_snapshots WHERE id NOT IN (
	SELECT id FROM script_snapshots ORDER BY ts DESC LIMIT ?
)`

// SaveScriptSnapshot persists a script snapshot full text with a timestamp.
// The index database is ephemeral and derived; this history is meant for editor change tracking, not canonical storage.
func SaveScriptSnapshot(ctx context.Context, ph *ProjectHandle, text string, ts time.Time) error {
	if ph == nil {
		return errors.New("nil ProjectHandle")
	}
	// Open or init index DB
	db, err := InitOrOpenIndex(ph.Root)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	_, err = db.ExecContext(ctx, insertScriptSnapshotSQL, ts.UTC().Format(time.RFC3339Nano), text)
	return err
}

// GetLatestScriptSnapshot returns the latest script snapshot text and timestamp, or empty if none.
func GetLatestScriptSnapshot(ctx context.Context, ph *ProjectHandle) (string, time.Time, error) {
	if ph == nil {
		return "", time.Time{}, errors.New("nil ProjectHandle")
	}
	db, err := InitOrOpenIndex(ph.Root)
	if err != nil {
		return "", time.Time{}, err
	}
	defer func() { _ = db.Close() }()
	var tsStr string
	var txt string
	err = db.QueryRowContext(ctx, selectLatestScriptSnapshotSQL).Scan(&tsStr, &txt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, nil
	}
	if err != nil {
		return "", time.Time{}, err
	}
	ts, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return txt, time.Time{}, nil
	}
	return txt, ts, nil
}

// ListScriptSnapshots returns up to limit most recent script snapshots.
func ListScriptSnapshots(ctx context.Context, ph *ProjectHandle, limit int) ([]struct {
	TS   time.Time
	Text string
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
	rows, err := db.QueryContext(ctx, listScriptSnapshotsSQL, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []struct {
		TS   time.Time
		Text string
	}
	for rows.Next() {
		var tsStr string
		var txt string
		if err := rows.Scan(&tsStr, &txt); err != nil {
			return nil, err
		}
		ts, _ := time.Parse(time.RFC3339Nano, tsStr)
		out = append(out, struct {
			TS   time.Time
			Text string
		}{TS: ts, Text: txt})
	}
	return out, rows.Err()
}

// PruneOldScriptSnapshots keeps at most keepLast snapshots and deletes older ones.
func PruneOldScriptSnapshots(ctx context.Context, ph *ProjectHandle, keepLast int) (int64, error) {
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
	res, err := db.ExecContext(ctx, pruneOldScriptSnapshotsSQL, keepLast)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
