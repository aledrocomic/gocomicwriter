/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * Licensed under the Apache License, Version 2.0.
 */
package backend

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gocomicwriter/internal/storage"
)

// SearchPG executes a search over the Postgres documents table using tsvector and filters
// and returns results mapped to storage.SearchResult to ease parity checks.
func SearchPG(ctx context.Context, db *sql.DB, projectID int64, q storage.SearchQuery) ([]storage.SearchResult, error) {
	var (
		args []any
		b    strings.Builder
	)
	useFTS := strings.TrimSpace(q.Text) != ""
	if useFTS {
		b.WriteString("SELECT d.id AS doc_id, d.doc_type AS type, COALESCE(d.external_ref,'') AS path, COALESCE(d.page_num,0) AS page_id, ")
		b.WriteString("COALESCE(ts_headline('simple', COALESCE(d.raw_text,''), plainto_tsquery('simple', $1), 'StartSel=[, StopSel=], MaxFragments=1, MaxWords=12'), '') AS snippet ")
		b.WriteString("FROM documents d WHERE d.project_id = $2 AND d.search_vector @@ plainto_tsquery('simple', $1) ")
		args = append(args, q.Text, projectID)
	} else {
		b.WriteString("SELECT d.id AS doc_id, d.doc_type AS type, COALESCE(d.external_ref,'') AS path, COALESCE(d.page_num,0) AS page_id, '' AS snippet ")
		b.WriteString("FROM documents d WHERE d.project_id = $1 ")
		args = append(args, projectID)
	}

	// Helper to add parameter and return placeholder like $n
	place := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	// Types filter
	if len(q.Types) > 0 {
		b.WriteString(" AND d.doc_type = ANY (" + place(q.Types) + ") ")
	}
	// Page range
	if q.PageFrom > 0 && q.PageTo > 0 && q.PageTo >= q.PageFrom {
		b.WriteString(" AND d.page_num BETWEEN " + place(q.PageFrom) + " AND " + place(q.PageTo) + " ")
	} else if q.PageFrom > 0 {
		b.WriteString(" AND d.page_num >= " + place(q.PageFrom) + " ")
	} else if q.PageTo > 0 {
		b.WriteString(" AND d.page_num <= " + place(q.PageTo) + " ")
	}
	// Character filter
	if s := strings.TrimSpace(q.Character); s != "" {
		ss := strings.ToLower(s)
		b.WriteString(" AND ( lower(COALESCE(d.raw_text,'')) LIKE " + place("%"+ss+":%") + " OR lower(COALESCE(d.external_ref,'')) LIKE " + place("%character:"+ss+"%") + " ) ")
	}
	// Scene filter
	if s := strings.TrimSpace(q.Scene); s != "" {
		ss := strings.ToLower(s)
		b.WriteString(" AND ( lower(COALESCE(d.external_ref,'')) LIKE " + place("%location:"+ss+"%") + " OR lower(COALESCE(d.raw_text,'')) LIKE " + place("%"+ss+"%") + " ) ")
	}
	// Tags: require all tags to appear as @tag tokens in raw_text
	for _, t := range q.Tags {
		tt := strings.ToLower(strings.TrimSpace(t))
		if tt == "" {
			continue
		}
		b.WriteString(" AND lower(COALESCE(d.raw_text,'')) LIKE " + place("%@"+tt+"%") + " ")
	}
	// Order and pagination
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	b.WriteString(" ORDER BY d.page_num NULLS LAST, d.id ")
	b.WriteString(" LIMIT " + place(limit) + " OFFSET " + place(offset))

	rows, err := db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search pg query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []storage.SearchResult
	for rows.Next() {
		var r storage.SearchResult
		if err := rows.Scan(&r.DocID, &r.Type, &r.Path, &r.PageID, &r.Snippet); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
