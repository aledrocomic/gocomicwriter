/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"os"
	"path/filepath"
	"testing"

	gojsonschema "github.com/xeipuuv/gojsonschema"
	"gocomicwriter/internal/domain"
)

func TestManifestConformsToSchema(t *testing.T) {
	root := t.TempDir()
	ph, err := InitProject(root, defaultMinimalProject())
	if err != nil {
		t.Fatalf("InitProject error: %v", err)
	}

	// Load manifest bytes
	data, err := os.ReadFile(ph.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	// Load schema bytes via relative path to repository docs
	schemaPath := filepath.Join("..", "..", "docs", "comic.schema.json")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	docLoader := gojsonschema.NewBytesLoader(data)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		t.Fatalf("schema validate error: %v", err)
	}
	if !result.Valid() {
		for _, e := range result.Errors() {
			t.Logf("schema error: %s", e)
		}
		t.Fatalf("manifest does not conform to schema")
	}
}

// defaultMinimalProject returns a minimal project for schema compliance
func defaultMinimalProject() domain.Project {
	return domain.Project{Name: "Schema Test", Issues: []domain.Issue{}}
}
