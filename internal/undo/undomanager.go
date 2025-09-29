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
	"sync"
	"time"
)

// Snapshot represents a reversible state blob for a page.
// Blob content is opaque to the manager; size is estimated as len(Blob).
// TS is when the snapshot was captured.
type Snapshot struct {
	PageNumber int
	Blob       []byte
	TS         time.Time
}

// Config controls memory and depth caps and coalescing behavior.
type Config struct {
	// MaxBytes is a soft cap; older entries are pruned when exceeded.
	MaxBytes int
	// MaxPerPage limits number of snapshots per page kept in memory (0 means unlimited).
	MaxPerPage int
	// MinInterval coalesces snapshots captured within the interval for the same page,
	// replacing the previous one instead of pushing a new entry.
	MinInterval time.Duration
}

// Manager provides an in-memory undo/redo stack per page with performance safeguards.
// It is safe for concurrent use.
type Manager struct {
	cfg Config
	mu  sync.Mutex
	// per-page stacks
	undo map[int][]Snapshot
	redo map[int][]Snapshot
	// accounting
	totalBytes int
}

func NewManager(cfg Config) *Manager {
	// Set conservative defaults if not provided
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 16 * 1024 * 1024 // 16 MiB
	}
	if cfg.MinInterval <= 0 {
		cfg.MinInterval = 250 * time.Millisecond
	}
	return &Manager{cfg: cfg, undo: make(map[int][]Snapshot), redo: make(map[int][]Snapshot)}
}

// PushSnapshot records a snapshot for a page. If within MinInterval from the last
// snapshot on the same page, it replaces the last one. Clears redo stack for that page.
func (m *Manager) PushSnapshot(s Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stack := m.undo[s.PageNumber]
	if n := len(stack); n > 0 {
		last := stack[n-1]
		if s.TS.Sub(last.TS) < m.cfg.MinInterval {
			// Coalesce: adjust accounting and replace
			m.totalBytes -= len(last.Blob)
			m.totalBytes += len(s.Blob)
			stack[n-1] = s
			m.undo[s.PageNumber] = stack
			m.redo[s.PageNumber] = nil
			m.enforceCapsLocked(s.PageNumber)
			return
		}
	}
	// Push new
	stack = append(stack, s)
	m.undo[s.PageNumber] = stack
	m.totalBytes += len(s.Blob)
	// Any new change invalidates redo for the page
	m.redo[s.PageNumber] = nil
	m.enforceCapsLocked(s.PageNumber)
}

// Undo pops from the page undo stack and pushes to redo stack, returning the snapshot.
func (m *Manager) Undo(pageNumber int) (Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stack := m.undo[pageNumber]
	if len(stack) == 0 {
		return Snapshot{}, false
	}
	s := stack[len(stack)-1]
	m.undo[pageNumber] = stack[:len(stack)-1]
	m.totalBytes -= len(s.Blob)
	m.redo[pageNumber] = append(m.redo[pageNumber], s)
	return s, true
}

// Redo pops from redo and pushes back to undo.
func (m *Manager) Redo(pageNumber int) (Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.redo[pageNumber]
	if len(r) == 0 {
		return Snapshot{}, false
	}
	s := r[len(r)-1]
	m.redo[pageNumber] = r[:len(r)-1]
	m.undo[pageNumber] = append(m.undo[pageNumber], s)
	m.totalBytes += len(s.Blob)
	m.enforceCapsLocked(pageNumber)
	return s, true
}

// ClearPage clears undo/redo stacks for a page to free memory.
func (m *Manager) ClearPage(pageNumber int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.undo[pageNumber] {
		m.totalBytes -= len(s.Blob)
	}
	delete(m.undo, pageNumber)
	delete(m.redo, pageNumber)
	if m.totalBytes < 0 {
		m.totalBytes = 0
	}
}

// Stats returns current sizes for diagnostics.
func (m *Manager) Stats() (totalBytes int, pages int, totalSnapshots int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pages = len(m.undo)
	for _, v := range m.undo {
		totalSnapshots += len(v)
	}
	return m.totalBytes, pages, totalSnapshots
}

func (m *Manager) enforceCapsLocked(pageNumber int) {
	// Per-page depth cap
	if m.cfg.MaxPerPage > 0 {
		stack := m.undo[pageNumber]
		if len(stack) > m.cfg.MaxPerPage {
			// drop the oldest extras
			toDrop := len(stack) - m.cfg.MaxPerPage
			for i := 0; i < toDrop; i++ {
				m.totalBytes -= len(stack[i].Blob)
			}
			m.undo[pageNumber] = append([]Snapshot{}, stack[toDrop:]...)
		}
	}
	// Global memory cap: prune oldest across all pages
	for m.cfg.MaxBytes > 0 && m.totalBytes > m.cfg.MaxBytes {
		oldestPage := 0
		oldestIdx := -1
		var oldestTS time.Time
		for page, stack := range m.undo {
			if len(stack) == 0 {
				continue
			}
			if oldestIdx == -1 || stack[0].TS.Before(oldestTS) {
				oldestPage = page
				oldestIdx = 0
				oldestTS = stack[0].TS
			}
		}
		if oldestIdx == -1 {
			break
		}
		stack := m.undo[oldestPage]
		m.totalBytes -= len(stack[0].Blob)
		m.undo[oldestPage] = stack[1:]
		if len(m.undo[oldestPage]) == 0 {
			delete(m.undo, oldestPage)
		}
	}
}
