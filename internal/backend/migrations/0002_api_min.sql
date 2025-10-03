
-- Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
-- This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
-- in compliance with the License.  You may obtain a copy of the License at
--   http://www.apache.org/licenses/LICENSE-2.0
-- Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
-- "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
--  specific language governing permissions and limitations under the License.


-- 0002_api_min.sql
-- Adds minimal tables for API endpoints: users (optional), index_snapshots

BEGIN;

-- Users are optional for early dev. Create a minimal table to attach ownership later.
CREATE TABLE IF NOT EXISTS users (
    id           BIGSERIAL PRIMARY KEY,
    stable_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    email        TEXT        UNIQUE,
    display_name TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Optional mapping of projects to owners (many-to-many ready)
CREATE TABLE IF NOT EXISTS project_members (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role       TEXT   NOT NULL DEFAULT 'owner',
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, project_id)
);

-- Index snapshots for pull endpoint
CREATE TABLE IF NOT EXISTS index_snapshots (
    id          BIGSERIAL PRIMARY KEY,
    project_id  BIGINT      NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version     BIGINT      NOT NULL DEFAULT 1,
    snapshot    JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_index_snapshots_project_version ON index_snapshots(project_id, version DESC);

-- Mark migration as applied if not recorded yet
INSERT INTO schema_migrations(version, name)
SELECT 2, '0002_api_min'
WHERE NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 2);

COMMIT;
