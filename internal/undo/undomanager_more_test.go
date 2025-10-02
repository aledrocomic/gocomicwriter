/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

package undo

import (
	"testing"
	"time"
)

func TestClearPageAndStats(t *testing.T) {
	m := NewManager(Config{MaxBytes: 1024, MaxPerPage: 10, MinInterval: time.Millisecond})
	pg := 7
	m.PushSnapshot(Snapshot{PageNumber: pg, Blob: []byte("abcdef"), TS: time.Now()})
	tb, pages, total := m.Stats()
	if tb == 0 || pages != 1 || total != 1 {
		t.Fatalf("unexpected stats before clear: tb=%d pages=%d total=%d", tb, pages, total)
	}
	m.ClearPage(pg)
	tb2, pages2, total2 := m.Stats()
	if tb2 != 0 || pages2 != 0 || total2 != 0 {
		t.Fatalf("expected cleared stats to be zero, got tb=%d pages=%d total=%d", tb2, pages2, total2)
	}
}

func TestGlobalPruneAcrossPages(t *testing.T) {
	// Very small MaxBytes so pruning triggers across pages
	m := NewManager(Config{MaxBytes: 8, MaxPerPage: 0, MinInterval: time.Millisecond})
	t0 := time.Now()
	// Page 1 older snapshot
	m.PushSnapshot(Snapshot{PageNumber: 1, Blob: []byte("xxxx"), TS: t0})
	// Page 2 newer snapshot
	m.PushSnapshot(Snapshot{PageNumber: 2, Blob: []byte("yyyy"), TS: t0.Add(time.Second)})

	// Add another snapshot to exceed cap and force prune of oldest page snapshot
	m.PushSnapshot(Snapshot{PageNumber: 2, Blob: []byte("zzzz"), TS: t0.Add(2 * time.Second)})

	// After pruning, oldest (page 1) should be removed
	_, pages, total := m.Stats()
	if pages == 0 || total == 0 {
		t.Fatalf("expected some snapshots to remain")
	}
	// Undo page 1 should now be empty
	if _, ok := m.Undo(1); ok {
		t.Fatalf("expected page 1 to have been pruned")
	}
	// Undo page 2 should still work
	if _, ok := m.Undo(2); !ok {
		t.Fatalf("expected page 2 to have snapshots")
	}
}
