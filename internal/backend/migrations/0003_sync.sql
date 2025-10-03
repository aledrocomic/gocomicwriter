-- Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
-- This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
-- in compliance with the License.  You may obtain a copy of the License at
--   http://www.apache.org/licenses/LICENSE-2.0
-- Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
-- "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
--  specific language governing permissions and limitations under the License.


-- 0003_sync.sql
-- Introduce operation log for basic sync (push/pull) without conflict resolution

BEGIN;

-- Operation log: append-only per project with monotonic version numbers
CREATE TABLE IF NOT EXISTS sync_ops (
    id           BIGSERIAL PRIMARY KEY,
    op_id        UUID        NOT NULL DEFAULT gen_random_uuid(),
    project_id   BIGINT      NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version      BIGINT      NOT NULL, -- per-project, monotonic
    actor        TEXT,
    op_type      TEXT        NOT NULL, -- e.g., 'upsert','delete','comment','meta'
    entity_type  TEXT        NOT NULL, -- e.g., 'page','panel','balloon','style','document','asset'
    entity_id    TEXT        NOT NULL, -- stable identifier (UUID/ULID) from client
    payload      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_sync_ops_op_id ON sync_ops(op_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_sync_ops_project_version ON sync_ops(project_id, version);
CREATE INDEX IF NOT EXISTS ix_sync_ops_project_created ON sync_ops(project_id, created_at);

-- Ensure projects.version exists and defaults; already created in 0001, but keep safe update
ALTER TABLE projects
    ALTER COLUMN version SET DEFAULT 1;

-- Mark migration as applied if not recorded yet
INSERT INTO schema_migrations(version, name)
SELECT 3, '0003_sync'
WHERE NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 3);

COMMIT;
