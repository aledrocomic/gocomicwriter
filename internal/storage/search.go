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
	"strings"
)

// SearchQuery describes the in-app search request.
// Text uses SQLite FTS5 syntax (simple terms, phrases in quotes, AND/OR/NOT).
// Filters are optional. Tags should be provided without the leading @.
// Types can restrict to kinds like: balloon, panel_notes, script, character, location, tag, etc.
// PageFrom/To are inclusive; 0 means unset.
// Limit/Offset implement pagination; reasonable defaults applied if zero.
type SearchQuery struct {
	Text      string
	Character string
	Scene     string
	Tags      []string
	Types     []string
	PageFrom  int
	PageTo    int
	Limit     int
	Offset    int
}

// SearchResult represents a single match row.
// Snippet is an optional highlighted excerpt using [ ] markers when FTS text is used.
// PageID is 0 when unknown.
// DocID can be used with WhereUsed to find references.
type SearchResult struct {
	DocID   int64
	Type    string
	Path    string
	PageID  int
	Snippet string
}

// Search performs full-text search with optional filters over the embedded index.
// When q.Text is empty, it falls back to a non-FTS scan over documents with filters applied.
func Search(ctx context.Context, projectRoot string, q SearchQuery) ([]SearchResult, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return nil, errors.New("project root is required")
	}
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return searchDB(ctx, db, q)
}

func searchDB(ctx context.Context, db *sql.DB, q SearchQuery) ([]SearchResult, error) {
	// Build dynamic SQL
	var args []any
	var sb strings.Builder
	useFTS := strings.TrimSpace(q.Text) != ""
	if useFTS {
		sb.WriteString("SELECT d.doc_id, d.type, d.path, COALESCE(d.page_id,0), snippet(fts_documents, 0, '[', ']', '…', 10)\n")
		sb.WriteString("FROM fts_documents JOIN documents d ON fts_documents.rowid = d.doc_id\n")
		sb.WriteString("WHERE fts_documents MATCH ?\n")
		args = append(args, q.Text)
	} else {
		sb.WriteString("SELECT d.doc_id, d.type, d.path, COALESCE(d.page_id,0), ''\n")
		sb.WriteString("FROM documents d\nWHERE 1=1\n")
	}
	// Filters
	// Types filter (IN list)
	if len(q.Types) > 0 {
		sb.WriteString(" AND d.type IN (" + placeholders(len(q.Types)) + ")\n")
		for _, t := range q.Types {
			args = append(args, t)
		}
	}
	// Page range
	if q.PageFrom > 0 && q.PageTo > 0 && q.PageTo >= q.PageFrom {
		sb.WriteString(" AND d.page_id BETWEEN ? AND ?\n")
		args = append(args, q.PageFrom, q.PageTo)
	} else if q.PageFrom > 0 {
		sb.WriteString(" AND d.page_id >= ?\n")
		args = append(args, q.PageFrom)
	} else if q.PageTo > 0 {
		sb.WriteString(" AND d.page_id <= ?\n")
		args = append(args, q.PageTo)
	}
	// Character filter: prefer exact character_id when populated, else fallback to text/path contains
	if s := strings.TrimSpace(q.Character); s != "" {
		ss := strings.ToLower(s)
		sb.WriteString(" AND ( (d.character_id IS NOT NULL AND lower(d.character_id)=?) OR lower(d.text) LIKE ? OR lower(d.path) LIKE ? )\n")
		args = append(args, ss, likeContains(ss+":"), likeContains("character:"+ss))
	}
	// Scene filter: location-related content or text contains the scene token
	if s := strings.TrimSpace(q.Scene); s != "" {
		ss := strings.ToLower(s)
		sb.WriteString(" AND ( lower(d.path) LIKE ? OR lower(d.text) LIKE ? )\n")
		args = append(args, likeContains("location:"+ss), likeContains(ss))
	}
	// Tags: require all tags to appear as @tag tokens in text
	for _, t := range q.Tags {
		tt := strings.ToLower(strings.TrimSpace(t))
		if tt == "" {
			continue
		}
		sb.WriteString(" AND lower(d.text) LIKE ?\n")
		args = append(args, likeContains("@"+tt))
	}
	// Order and pagination
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	sb.WriteString("ORDER BY d.page_id NULLS LAST, d.doc_id\n")
	sb.WriteString("LIMIT ? OFFSET ?")
	args = append(args, limit, q.Offset)

	rows, err := db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		var page sql.NullInt64
		var sn sql.NullString
		if err := rows.Scan(&r.DocID, &r.Type, &r.Path, &page, &sn); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if page.Valid {
			r.PageID = int(page.Int64)
		}
		if sn.Valid {
			r.Snippet = sn.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// WhereUsed returns documents that reference the given target document ID using cross_refs.
func WhereUsed(ctx context.Context, projectRoot string, targetDocID int64, limit, offset int) ([]SearchResult, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return nil, errors.New("project root is required")
	}
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	q := `SELECT d.doc_id, d.type, d.path, COALESCE(d.page_id,0), ''
		FROM cross_refs x
		JOIN documents d ON d.doc_id = x.from_id
		WHERE x.to_id = ?
		ORDER BY d.page_id NULLS LAST, d.doc_id
		LIMIT ? OFFSET ?`
	rows, err := db.QueryContext(ctx, q, targetDocID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("where-used query: %w", err)
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		var page sql.NullInt64
		if err := rows.Scan(&r.DocID, &r.Type, &r.Path, &page, &r.Snippet); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if page.Valid {
			r.PageID = int(page.Int64)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// WhereUsedByPath resolves a document by path then returns references to it.
func WhereUsedByPath(ctx context.Context, projectRoot string, path string, limit, offset int) ([]SearchResult, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("path is required")
	}
	db, err := InitOrOpenIndex(projectRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var id int64
	err = db.QueryRowContext(ctx, "SELECT doc_id FROM documents WHERE path=?", path).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []SearchResult{}, nil
		}
		return nil, err
	}
	return WhereUsed(ctx, projectRoot, id, limit, offset)
}

func likeContains(s string) string { return "%" + s + "%" }

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := strings.Builder{}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
	}
	return b.String()
}
