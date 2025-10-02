# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog (https://keepachangelog.com/en/1.1.0/),
and this project adheres to Semantic Versioning. This is a pre-release
(0.x) and APIs may change at any time.

## [2025.12-Beta2] - 2025-10-dd

### Added
- Changed look of the bible dialog 
-  Reorganized AWS things within the project
- Added a settings dialog to gocomicwriter to enable/disable environment variables.
- Auto-create database in gcwserver if missing.
- Enable multi-user access to shared database (Desktop/Ui and SQL only option).

## [2025.12-Beta1] - 2025-10-02

### Added
- Cross-platform builds and installers.
- Crash reporting and opt-in telemetry (anonymous usage metrics).
- Documentation PDF with tutorials and templates (see docs/tutorials_and_templates.md; build via scripts/build_docs.ps1 → docs/gocomicwriter_tutorials_templates.pdf).
- Added additional unit tests to achieve a higher test coverage

## [0.10.0-dev] - 2025-10-01

### Added
- Define minimal backend service (Go) using PostgreSQL Version 17+: schema for projects, `documents`, 
  full-text via `tsvector`+GIN, `cross_refs`, assets metadata; migrations.
- API endpoints: auth (token), list projects, pull index snapshot; later: push deltas and comments.
- Desktop integration (behind feature flag): manual "Connect to Server"; read-only listing and search first; 
  file-based `comic.json` remains the source of truth.
- Sync prototype: append-only op-log with stable IDs; created_at/updated_at/version columns; basic push/pull over HTTPS.
- Security/ops: optional TLS for server (GCW_TLS_ENABLE), HMAC tokens bound to per-user subject, static admin mode with X-API-Key and user upsert.
- Health checks: /healthz (liveness) and /readyz (DB ping; optional object storage check via GCW_OBJECT_HEALTH_URL or GCW_MINIO_ENDPOINT).
- Docker dev stack: docker-compose with PostgreSQL 17 and optional MinIO; .env.example for configuration.
- Search API endpoint: GET /api/projects/{id}/search with Postgres tsvector + filters (types, pages, tags, character, scene).
- Client: added Client.Search(ctx, projectID, storage.SearchQuery) returning []storage.SearchResult.
- Tests: SQLite ↔ Postgres search parity checks and end-to-end integration tests aligned with go_comic_writer_concept.md.

### Fixed
- Workflow does not contain permissions
- Arbitrary file access during archive extraction ("Zip Slip")
- aws-codepipeline.yml build uses `golang: latest` and env variable `CGO_ENABLED=1`

## [0.9.0-dev] - 2025-10-01

### Added
- Commenting and review mode on script and pages (minimal; behind feature flag).
- Change tracking in script editor.
- Merge-friendly project format guidance and diff tips (see “Merge-friendly Project Format & Diff Tips” in go_comic_writer_concept.md)

## [0.8.0-dev] - 2025-09-30 

### Added
- Added delete option for pages and inserted items
- Added sidebar navigation for pages
- internal package features are available in the UI
- Project dashboard with recent files and templates.
- Asset library with previews and drag-and-drop into pages.
- Style pack manager (install/export styles and templates).
- Undo/redo stack with snapshots and performance safeguards.

## [0.7.0-dev] - 2025-09-27

### Added
- Updated version for the used gofpdf module to v1.16.2
- Set minimum Version for PostgreSQL
- Updated Go version in aws-codepipeline.yml
- Added a developer guide to project docs
- Added an export option to .epub
- Establish per-project index store at `project/.gcw/index.sqlite`; enable WAL; add `meta/version` tables.
- Define schema: `documents` (doc_id, type, path, page_id, character_id, text), `fts_documents` (FTS5, contentless with external content), `cross_refs` (from_id → to_id), `assets` (hash, path, type), `previews` (page_id/panel_id, thumb_blob, updated_at), `snapshots` (page_id, ts, delta_blob).
- Implement background indexer: initial full build from `comic.json` and incremental updates on save; safe rebuild command ("Rebuild Index").
- Add search service in-app: full-text with filters (character, scene, page range, tags) and "where-used" via `cross_refs`.
- Wire UI: search panel/omnibox; navigate results to issue/page/panel; highlight hits.
- Add caching pipeline: generate/stash thumbnails and geometry caches in `previews`; LRU eviction and max-size cap.
- Add schema migrations, corruption handling, index rebuild logic, and tests for SQLite-based project indexing. Document maintenance and recovery steps.
- Vision for Version 2.x


## [0.6.0-dev] - 2025-09-26

### Added
- Minimal PDF exporter: multi-page export with trim and bleed guides; panels/balloons as vector shapes; vector text via built-in Helvetica when possible.
- PNG and SVG per-page exporters with DPI control and guide/bleed support.
- CBZ packaging with metadata manifest.
- Export presets (web, print) and batch export coordinator.
- Made Exporters UI only
- Optimized ergonomics of the UI
- File menu: Close Project (Ctrl+W) closes the current project without closing the window; OS Quit/Beenden remains distinct (Ctrl+Q). Close Project is disabled when no project is open.


## [0.5.0-dev] - 2025-09-25

### Added
- Fixed AWS CodePipline with a role policy for S3
- Deleted release.yml as Github action
-Added copyright pop-up window in about menu
- Disabled CLI for UI-Only User interaction
- Enhanced User interaction and event info logging
- Added additional tests to achieve a minimum of 80% code test coverage
- Fixed character addition issue in open project
- Balloon and caption tools with snapping and smart guides.
- Tail drawing with speaker anchor and auto-orient.
- Typography engine: font loading, style presets, kerning/leading/tracking.
- SFX tool with outline/fill/effects and text-on-path.
- Auto layout suggestion for balloons with collision avoidance.
- Style sheets (global, per-issue, per-page)
-Added databases for search and thin Postgesql Backend to concept

## [0.4.0-dev] - 2025-09-23

### Added
- Issue setup dialog (trim, bleed, dpi, reading direction).
- Grid templates and custom grids; apply per page.
- Panel creation, ordering, and metadata editing.
- Beat coverage overlay; page-turn pacing indicators.
- CI/CD with AWS CloudFormation
- UI standalone Project initialization
- About Menu with environment info popup

## [0.3.0-dev] - 2025-09-22

### Added
#### Script Editor
- Implement a structured script editor with scene/character syntax support.
- Character/location bible with auto-complete and tagging.
- Beat tagging and a sidebar outline; search/filter.
- Map beats to pages/panels; show unmapped beat warnings.

## [0.2.1-dev] - 2025-09-22

### Added
- Fix undefined canvas.Polygon errors in Go build

## [0.2.0-dev] - 2025-09-22

### Added
- Build vector primitives and text layout abstraction.
- Implement page canvas with trim/bleed/gutter guides.
- Implement shapes: rectangles, ellipses, rounded boxes, paths.
- [Implement hit testing and selection; transform handles (move/scale/rotate).

## [0.1.2-dev] - 2025-09-21

### Added
- Fixed binary build process with UI support

## [0.1.1-dev] - 2025-09-21

### Added
- Fix for fyne dependencies and cgo dependiencies.
- Expanded README.md to explain the cgo dependency

## [0.1.0-dev] - 2025-09-21

### Added
- Initial project skeleton and CLI application (gocomicwriter).
- CLI commands:
  - `version` (prints application version)
  - `init` (initialize a new project)
  - `open` / `save` (open and save project manifests)
  - `ui` (launch basic desktop UI when built with the `fyne` tag)
- Transactional project storage with a human‑readable manifest `comic.json` and
  timestamped backups under `backups/`.
- Crash safety:
  - On panic, write a crash report and autosave snapshot.
  - On open, fall back to the latest valid backup if the manifest is unreadable.
- Structured logging built on Go's `log/slog` with simple environment configuration
  and optional rotating file via `GCW_LOG_FILE`.
- Core domain model (internal/domain) and a public JSON schema at `docs/comic.schema.json`.
- Basic desktop UI shell (build tag `fyne`) with a placeholder canvas editor that shows
  page/trim/bleed guides, simple pan/zoom, and File → Open/Save.
- Sample project manifest at `tmp_proj/comic.json` (with backups under `tmp_proj/backups/`).
- Unit tests for core packages (storage, logging, crash, version, schema, UI stubs).
- Continuous integration workflow for Go builds/tests under `.github/workflows/go.yml`.
- Community and licensing docs: `CODE_OF_CONDUCT.md`, `LICENSE`.

### Known limitations
- Not production‑ready; functionality and file formats may change without notice.

