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
	"testing"
	"time"

	"gocomicwriter/internal/domain"
)

func BenchmarkSearchFTS(b *testing.B) {
	root := b.TempDir()
	proj := domain.Project{
		Name: "Bench",
		Issues: []domain.Issue{{
			Pages: []domain.Page{{Number: 1, Panels: []domain.Panel{{ID: "P1", Balloons: []domain.Balloon{{ID: "B1", Type: "speech", TextRuns: []domain.TextRun{{Content: "Hello world benchmark"}}}}}}}},
		}},
	}
	ph, err := InitProject(root, proj)
	if err != nil || ph == nil {
		b.Fatalf("InitProject: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := RebuildIndex(ctx, root, proj); err != nil {
		b.Fatalf("RebuildIndex: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Search(ctx, root, SearchQuery{Text: "Hello"})
		if err != nil {
			b.Fatalf("Search: %v", err)
		}
	}
}

func BenchmarkRebuildIndex(b *testing.B) {
	root := b.TempDir()
	proj := domain.Project{
		Name: "Bench",
		Issues: []domain.Issue{{
			Pages: []domain.Page{{Number: 1, Panels: []domain.Panel{{ID: "P1", Balloons: []domain.Balloon{{ID: "B1", Type: "speech", TextRuns: []domain.TextRun{{Content: "Hello world benchmark"}}}}}}}},
		}},
	}
	ph, err := InitProject(root, proj)
	if err != nil || ph == nil {
		b.Fatalf("InitProject: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = RebuildIndex(ctx, root, proj)
		cancel()
	}
}
