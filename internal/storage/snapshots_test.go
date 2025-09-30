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
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotsCRUD(t *testing.T) {
	root := t.TempDir()
	ph := &ProjectHandle{Root: root, ManifestPath: filepath.Join(root, ManifestFileName)}
	ctx := context.Background()
	// Ensure DB exists
	db, err := InitOrOpenIndex(root)
	if err != nil {
		t.Fatalf("InitOrOpenIndex error: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close error: %v", err)
	}
	delta1 := []byte("hello")
	if err := SaveSnapshot(ctx, ph, 1, delta1, time.Now()); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}
	blob, _, err := GetLatestSnapshot(ctx, ph, 1)
	if err != nil || string(blob) != "hello" {
		t.Fatalf("GetLatestSnapshot got %q err %v", string(blob), err)
	}
	// Add more snapshots
	for i := 0; i < 5; i++ {
		b := []byte{byte('a' + i)}
		if err := SaveSnapshot(ctx, ph, 1, b, time.Now().Add(time.Duration(i+1)*time.Millisecond)); err != nil {
			t.Fatalf("SaveSnapshot %d: %v", i, err)
		}
	}
	list, err := ListSnapshots(ctx, ph, 1, 10)
	if err != nil || len(list) != 6 {
		t.Fatalf("ListSnapshots got %d err %v", len(list), err)
	}
	// Prune keep last 3
	n, err := PruneOldSnapshots(ctx, ph, 1, 3)
	if err != nil {
		t.Fatalf("PruneOldSnapshots: %v", err)
	}
	if n <= 0 {
		t.Fatalf("expected deletions > 0, got %d", n)
	}
	list, err = ListSnapshots(ctx, ph, 1, 10)
	if err != nil || len(list) != 3 {
		t.Fatalf("ListSnapshots after prune got %d err %v", len(list), err)
	}
	// Clean up DB file
	_ = os.Remove(IndexPath(root))
}
