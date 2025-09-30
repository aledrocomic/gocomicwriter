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

func TestUndoRedoBasic(t *testing.T) {
	m := NewManager(Config{MaxBytes: 1024 * 1024, MaxPerPage: 10, MinInterval: 10 * time.Millisecond})
	pg := 1
	m.PushSnapshot(Snapshot{PageNumber: pg, Blob: []byte("a"), TS: time.Now()})
	m.PushSnapshot(Snapshot{PageNumber: pg, Blob: []byte("b"), TS: time.Now().Add(20 * time.Millisecond)})
	if _, pages, total := m.Stats(); pages != 1 || total != 2 {
		t.Fatalf("expected 1 page and 2 snapshots, got pages=%d total=%d", pages, total)
	}
	s, ok := m.Undo(pg)
	if !ok || string(s.Blob) != "b" {
		t.Fatalf("undo expected 'b', got ok=%v blob=%q", ok, string(s.Blob))
	}
	s, ok = m.Redo(pg)
	if !ok || string(s.Blob) != "b" {
		t.Fatalf("redo expected 'b', got ok=%v blob=%q", ok, string(s.Blob))
	}
}

func TestCoalesce(t *testing.T) {
	m := NewManager(Config{MaxBytes: 1024 * 1024, MaxPerPage: 10, MinInterval: 50 * time.Millisecond})
	pg := 2
	t0 := time.Now()
	m.PushSnapshot(Snapshot{PageNumber: pg, Blob: []byte("1"), TS: t0})
	m.PushSnapshot(Snapshot{PageNumber: pg, Blob: []byte("2"), TS: t0.Add(10 * time.Millisecond)}) // coalesce
	_, _, total := m.Stats()
	if total != 1 {
		t.Fatalf("expected coalesced to 1 snapshot, got %d", total)
	}
	s, ok := m.Undo(pg)
	if !ok || string(s.Blob) != "2" {
		t.Fatalf("expected coalesced snapshot '2', got ok=%v blob=%q", ok, string(s.Blob))
	}
}

func TestCaps(t *testing.T) {
	m := NewManager(Config{MaxBytes: 20, MaxPerPage: 2, MinInterval: 1 * time.Millisecond})
	pg := 3
	for i := 0; i < 10; i++ {
		m.PushSnapshot(Snapshot{PageNumber: pg, Blob: []byte("xxxxx"), TS: time.Now().Add(time.Duration(i) * time.Millisecond)})
	}
	_, _, total := m.Stats()
	if total > 2 {
		t.Fatalf("expected MaxPerPage cap to limit to 2, got %d", total)
	}
}
