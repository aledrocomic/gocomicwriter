/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

// Package storage implements project persistence and indexing.
// It handles create/open/save for the canonical JSON manifest (comic.json) with transactional writes and timestamped backups.
// It also manages the per‑project embedded SQLite index at <project>/.gcw/index.sqlite used for search and caches.
// The embedded index is derived from comic.json and assets and is rebuildable/disposable by design.
package storage
