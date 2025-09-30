-- 0001_init.sql
-- Minimal schema for thin backend (PostgreSQL 17+)
-- This schema is derived from docs/go_comic_writer_concept.md Phase 7a
-- Focus: projects, documents (for search), cross_refs, assets metadata; FTS via tsvector + GIN

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- for gen_random_uuid()

-- Migration bookkeeping
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     BIGINT PRIMARY KEY,
    name        TEXT NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Organizations are out of scope for now; keep schema project-scoped.

-- Projects: basic metadata with stable UUID and optimistic versioning
CREATE TABLE IF NOT EXISTS projects (
    id           BIGSERIAL PRIMARY KEY,
    stable_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    name         TEXT        NOT NULL,
    slug         TEXT        GENERATED ALWAYS AS (lower(regexp_replace(name, '[^a-zA-Z0-9]+', '-', 'g'))) STORED,
    description  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    version      BIGINT      NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_projects_stable_id ON projects(stable_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_projects_slug ON projects(slug);
CREATE INDEX IF NOT EXISTS ix_projects_updated_at ON projects(updated_at);

-- Documents: flattened textual items for search across script/pages/panels/bible/etc.
-- Keep a simple tokenization config (`simple`) to align with SQLite FTS5 without stemming.
CREATE TABLE IF NOT EXISTS documents (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT      NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stable_id     UUID        NOT NULL DEFAULT gen_random_uuid(),
    doc_type      TEXT        NOT NULL, -- e.g., 'script','page','panel','bible','note','character','scene','sfx','other'
    external_ref  TEXT,                 -- optional path or JSON pointer within comic.json
    title         TEXT,
    raw_text      TEXT,                 -- plain text used for FTS
    tags          TEXT[],
    page_num      INTEGER,              -- for faceting
    panel_index   INTEGER,              -- for faceting
    meta          JSONB       NOT NULL DEFAULT '{}'::jsonb, -- extra metadata
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    version       BIGINT      NOT NULL DEFAULT 1,
    search_vector tsvector    GENERATED ALWAYS AS (to_tsvector('simple', coalesce(raw_text, ''))) STORED
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_documents_stable_id ON documents(stable_id);
CREATE INDEX IF NOT EXISTS ix_documents_project_type ON documents(project_id, doc_type);
CREATE INDEX IF NOT EXISTS ix_documents_facets ON documents(project_id, page_num, panel_index);
CREATE INDEX IF NOT EXISTS ix_documents_updated_at ON documents(updated_at);
CREATE INDEX IF NOT EXISTS gin_documents_search ON documents USING GIN (search_vector);

-- Cross references: generic directed edges between documents (and optionally to assets)
CREATE TABLE IF NOT EXISTS cross_refs (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT   NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    from_doc_id   BIGINT   NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    to_doc_id     BIGINT            REFERENCES documents(id) ON DELETE CASCADE,
    to_asset_id   BIGINT,  -- optional FK to assets (declared after assets table), kept nullable to allow insert order
    ref_type      TEXT     NOT NULL, -- e.g., 'character_appearance','uses_asset','links_to','note_on'
    note          TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_cross_refs_project ON cross_refs(project_id);
CREATE INDEX IF NOT EXISTS ix_cross_refs_from ON cross_refs(from_doc_id);
CREATE INDEX IF NOT EXISTS ix_cross_refs_to ON cross_refs(to_doc_id);

-- Assets metadata: content-addressed where possible
CREATE TABLE IF NOT EXISTS assets (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT      NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stable_id     UUID        NOT NULL DEFAULT gen_random_uuid(),
    external_ref  TEXT,                  -- path under project/assets or external URI
    filename      TEXT,
    content_hash  TEXT,                  -- e.g., sha256 hex
    mime_type     TEXT,
    bytes         BIGINT,
    width         INTEGER,
    height        INTEGER,
    duration_ms   INTEGER,
    metadata      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    version       BIGINT      NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_assets_stable_id ON assets(stable_id);
CREATE INDEX IF NOT EXISTS ix_assets_project ON assets(project_id);
CREATE INDEX IF NOT EXISTS ix_assets_hash ON assets(content_hash);

-- Now that assets exist, add FK from cross_refs.to_asset_id → assets.id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE t.relname = 'cross_refs' AND c.conname = 'fk_cross_refs_to_asset') THEN
        ALTER TABLE cross_refs
            ADD CONSTRAINT fk_cross_refs_to_asset FOREIGN KEY (to_asset_id)
            REFERENCES assets(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Triggers to auto-update updated_at timestamps
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_projects_updated_at'
    ) THEN
        CREATE TRIGGER trg_projects_updated_at
        BEFORE UPDATE ON projects
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_documents_updated_at'
    ) THEN
        CREATE TRIGGER trg_documents_updated_at
        BEFORE UPDATE ON documents
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_assets_updated_at'
    ) THEN
        CREATE TRIGGER trg_assets_updated_at
        BEFORE UPDATE ON assets
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations under the License.
 */

-- Mark migration as applied if not recorded yet
INSERT INTO schema_migrations(version, name)
SELECT 1, '0001_init'
WHERE NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 1);

COMMIT;