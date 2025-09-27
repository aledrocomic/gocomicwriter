/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * Licensed under the Apache License, Version 2.0.
 */

package storage

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"gocomicwriter/internal/domain"
)

func TestPreviewsPutGetAndEvict(t *testing.T) {
	root := t.TempDir()
	ph, err := InitProject(root, domain.Project{Name: "Prev Test"})
	if err != nil || ph == nil {
		t.Fatalf("InitProject: %v", err)
	}
	// Give background index init a moment to settle to avoid lock contention
	time.Sleep(200 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set a tiny cap to force eviction quickly
	os.Setenv("GCW_PREVIEWS_MAX_BYTES", "64")
	defer os.Unsetenv("GCW_PREVIEWS_MAX_BYTES")

	pnull := sql.NullInt64{Valid: false}
	// Insert 3 previews of 40 bytes each
	blobA := make([]byte, 40)
	blobB := make([]byte, 40)
	blobC := make([]byte, 40)
	if err := PutPreview(ctx, ph.Root, 1, pnull, PreviewKindThumb, 100, 100, blobA); err != nil {
		t.Fatalf("put A: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // different access times
	if err := PutPreview(ctx, ph.Root, 1, pnull, PreviewKindThumb, 200, 200, blobB); err != nil {
		t.Fatalf("put B: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := PutPreview(ctx, ph.Root, 1, pnull, PreviewKindThumb, 300, 300, blobC); err != nil {
		t.Fatalf("put C: %v", err)
	}

	// Cap is 64 bytes; after inserts total 120 -> eviction should have occurred, leaving last inserted(s)
	total, err := TotalPreviewBytes(ctx, ph.Root)
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	if total > 64 {
		t.Fatalf("expected eviction to <=64 bytes, got %d", total)
	}

	// Access the 200x200 one (if present)
	_, _ = GetPreview(ctx, ph.Root, 1, pnull, PreviewKindThumb, 200, 200)
	// Insert another 40-byte; should evict oldest by last_access
	if err := PutPreview(ctx, ph.Root, 1, pnull, PreviewKindThumb, 400, 400, make([]byte, 40)); err != nil {
		t.Fatalf("put D: %v", err)
	}
	if total2, err := TotalPreviewBytes(ctx, ph.Root); err != nil || total2 > 64 {
		t.Fatalf("post total: %v / %d", err, total2)
	}
}

func TestGetOrCreatePreview(t *testing.T) {
	root := t.TempDir()
	ph, err := InitProject(root, domain.Project{Name: "Prev Create"})
	if err != nil || ph == nil {
		t.Fatalf("InitProject: %v", err)
	}
	// Allow background indexer to settle
	time.Sleep(200 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pnull := sql.NullInt64{Valid: false}
	calls := 0
	gen := func(context.Context) ([]byte, error) { calls++; return []byte("abcd"), nil }
	b, err := GetOrCreatePreview(ctx, ph.Root, 2, pnull, PreviewKindGeom, 0, 0, gen)
	if err != nil {
		t.Fatalf("getOrCreate: %v", err)
	}
	if string(b) != "abcd" {
		t.Fatalf("unexpected data: %q", string(b))
	}
	// Second call should hit cache and not call generator
	b, err = GetOrCreatePreview(ctx, ph.Root, 2, pnull, PreviewKindGeom, 0, 0, gen)
	if err != nil {
		t.Fatalf("getOrCreate 2: %v", err)
	}
	if calls != 1 {
		t.Fatalf("generator should be called once, got %d", calls)
	}
}
