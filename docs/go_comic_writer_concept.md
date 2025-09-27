# Go Comic Writer — Product Concept and Implementation Plan

A cross‑platform desktop application written in Go that helps comic creators write scripts, plan pages and panels, add dialogue balloons and sound effects, and export professional‑quality issues to print- and screen-ready formats.

## Vision
Empower writers and comic teams to go from script to lettered pages in one streamlined tool—fast, reliable, and offline‑first—with an ergonomic writing experience and precise layout control.

## Product Pillars
- Writing-first: frictionless script writing with structure, character/tag management, and fast formatting.
- Page-aware: page and panel planning that stays in sync with the script beats.
- Precise lettering: pro typography, balloons, tails, and SFX with predictable export.
- Asset organization: characters, locations, props, and visual references, all searchable and reusable.
- Reliable exports: CBZ, PDF, PNG/SVG, EPUB with consistent rendering, bleeds, and trim boxes.
- Fast search and cross-references: instant full-text search across script, captions, SFX, and Bible, with where-used lookups via cross-references.
- Connected by design: offline-first with a local embedded index; thin backend for organization-wide search and future sync.
- Offline-first and cross-platform: Windows, macOS, Linux.

## Primary Personas
- Comic Writer: focuses on story, scenes, dialogue, and pacing; minimal layout needs.
- Writer–Letterer: writes and letters in one tool; requires robust text and balloon controls.
- Small Team (Writer + Artist + Letterer): needs shared assets, comments, and predictable exports.

## Core Use Cases
- Draft a script with scenes, beats, and dialogue.
- Plan issues: define page count, grid templates, panel breakdowns.
- Auto-link script beats to panels; track coverage and page turns.
- Letter pages: add balloons, tails, captions, SFX; adjust typography and styles.
- Export to CBZ/PDF/PNG/SVG/EPUB with per-page and whole-issue options.
- Reuse style packs and templates across projects.
- Search across the project: global full-text with filters (character, scene, page range, tags) and fast "where-used" for characters/assets/scenes.
- Connect to the thin backend: organization-wide search and project listing; lays groundwork for future sync.

---

## High-Level Feature Set

1. Script Editor
- Syntax for scenes, panels, characters, dialogue, captions, notes (e.g., Fountain-like or Markdown conventions).
- Character and location bible with auto-complete.
- Beat tagging and linking to pages/panels.
- Outline and scene navigator; real-time word count and reading-time estimator.
- Comments and change tracking.

2. Page & Panel Planner
- Issue definition (trim, bleed, DPI, page count, reading direction).
- Grid templates (3×3, 2×3, custom), gutters, margins; per-page overrides.
- Panel objects with order, geometry, and metadata.
- Beat-to-panel mapping with coverage warnings.

3. Lettering Tools
- Balloons (speech, whisper, thought), captions, SFX, narration.
- Balloon shapes (ellipse, rounded box), tails with snapping to speakers.
- Text on path; SFX outlines and fills; stroke/blur effects.
- Typography: font selection, size, tracking, leading, kerning; stylesheets.
- Automatic balloon layout suggestions with collision avoidance (user-editable).

4. Asset Library
- Characters, locations, props with notes and image refs.
- Style packs (fonts, colors, balloon presets).
- Templates (page grids, balloon presets, title pages, credits).

5. Export
- Single page and full issue export: PDF (with bleeds/trim), PNG/SVG per page, CBZ, and EPUB (fixed-layout).
- Optional metadata embedding (title, issue, creators).
- Export profiles (web, print, social).

6. Search & Indexing
- Full-text search across script, captions, SFX, and the story Bible with filters (character, scene, page range, tags).
- Fast cross-references and "where-used" queries (characters, assets, scenes) via maintained backlink tables.
- Embedded per-project index (SQLite FTS5) with thumbnail and geometry caches under .gcw; rebuildable and disposable.
- Parity with backend search semantics where applicable.

7. Collaboration (Phase 2+)
- Commenting and review mode.
- Project files designed for merge friendliness via clear JSON manifests and separate assets.

---

## Non-Functional Requirements
- Cross-platform: Windows, macOS, Linux.
- Offline-first: local projects with an embedded per-project index (SQLite FTS5) and a thin backend for organization-wide search and future sync.
- Deterministic rendering and search: identical exports and identical search results across platforms/installs (controlled SQLite build and pinned tokenizer config).
- Robustness: index is rebuildable and disposable; tolerate corruption with automatic rebuild paths; crash-safe autosave and backups.
- Stable performance on mid-range machines; render and search large issues without UI freeze.
- Accessible UI and keyboard-centric workflows.
- Backend SLOs: responsive search APIs, secure TLS, and predictable pagination for large result sets.

---

## Architecture Overview

- Core: Go modules for domain models (Project, Issue, Page, Panel, Beat, Balloon, TextRun, Style, Asset).
- Rendering: Vector-first drawing pipeline with resolution-independent primitives; rasterization for PNG exports.
- UI: Desktop application shell with a modern, responsive interface and a canvas editor for pages.
- Storage: Local project directory with a human-readable manifest and organized assets.
- Indexing: Embedded SQLite FTS5 per project at .gcw/index.sqlite with thumbnails/geometry caches; rebuildable and disposable.
- Backend: Thin Go service with PostgreSQL for organization-wide search and future sync; offline-first integration; JSON manifest remains the source of truth.
- Exporters: PDF, PNG/SVG, CBZ, EPUB pipeline honoring trim/bleed, color profiles, and fonts.
- Extension points: Style packs, templates, and future scripting hooks.

### Project Structure (Conceptual)
- /project-root
  - comic.json (manifest: issues, pages, mapping, styles, fonts used)
  - /script (raw script files and notes)
  - /pages (page state: panels, balloons, geometry)
  - /assets (fonts, images, refs)
  - /styles (style packs, templates)
  - /exports (generated outputs)

### Data Model Essentials
- Project: name, metadata, issues[].
- Issue: trim, bleed, dpi, reading direction, pages[].
- Page: number, grid, panels[], layers, styles.
- Panel: id, geometry, z-order, linked beats[], balloons[].
- Balloon: id, type, text runs[], shape, tail anchor, style ref.
- TextRun: content, font, size, tracking, leading, style refs.
- Style: fonts, colors, stroke, fill, effects.
- Asset: type (font, image, ref), path, license metadata.

## Logging
Based on the concept and roadmap (offline-first, cross‑platform, structured JSON manifest, crash-safe autosave, optional telemetry), the project uses slog with a small wrapper for consistency and configuration.

- Primary: slog (log/slog in stdlib) with a custom handler
  - Reasons: standard API, structured fields, levels, easy context propagation across modules (storage, exporters, rendering), stable long-term.
  - Paired with:
    - lumberjack for file rotation (optional)
    - a pretty console text handler for development, JSON for machine consumption
    - a minimal wrapper package (internal/log) to centralize config and fields like component/op

Implementation details:
- Emit structured logs with consistent keys (component, op, project, issue, page, panel, asset, path).
- Levels: DEBUG for geometry/layout details, INFO for user actions and exports, WARN for recoverable validation issues, ERROR for failed I/O or rendering.
- Rotation: lumberjack; optional file logging controlled by env var.
- Crash hook handled in internal/crash (see below).

Current implementation (Phase 0):
- internal/log initializes slog and enriches records with app/version/time.
- Environment configuration:
  - GCW_LOG_LEVEL=debug|info|warn|error (default: info)
  - GCW_LOG_FORMAT=console|json (default: console)
  - GCW_LOG_SOURCE=true|false (default: false)
  - GCW_LOG_FILE=<path> (optional; enables rotating JSON file logs)

Crash handling and autosave (implemented):
- internal/crash.Recover captures panics, logs a stack trace, and writes a crash report file named crash-YYYYMMDD-HHMMSS.log to <project>/backups (or system temp if no project).
- storage.AutosaveCrashSnapshot writes backups/comic.json.crash-YYYYMMDD-HHMMSS.autosave without touching the main manifest, aiding recovery.
- The CLI exits with a non-zero status on crash.


## Testing Concept and Strategy

Goals:
- Ensure determinism (same inputs → same outputs) across platforms for storage, rendering, and exports.
- Catch regressions early with fast unit tests and targeted golden tests.
- Validate data integrity of the project manifest (comic.json) and safe I/O (backups, autosave, crash recovery).
- Provide confidence to refactor core geometry/text layout and exporters.

### Test Layers
1) Unit tests (Go testing) — fast, isolated
- internal/domain: model invariants (IDs, ordering, geometry bounds), JSON tags and defaults.
- internal/storage: load/save round-trips, versioning, migrations; atomic writes; backup/restore behavior.
- internal/log: configuration, level filtering, and structured fields presence.
- internal/crash: panic capture, file breadcrumbs, and autosave hooks (using temp dirs).

2) Schema validation
- Validate comic.json against docs/comic.schema.json in tests using a JSON Schema validator.
- Round-trip tests: marshal domain types → JSON → validate → unmarshal; ensure equality or compatible normalization.

3) Property and fuzz tests (where high value)
- Geometry operations (panels, balloons, hit testing) with randomized inputs to surface edge cases.
- Text segmentation/shaping boundaries with fuzzed strings (Unicode, RTL markers, emoji, ZWJ sequences) once the text stack lands.

4) Golden tests (deterministic outputs)
- Exporters: compare PDF/PNG/SVG bytes or normalized representations against versioned golden files.
- Rendering/layout: store small canonical scenes (few panels, balloons) and assert bounding boxes, line breaks, and z-ordering.
- Use testdata/golden under each package; include tools to re-generate goldens with explicit env var (e.g., UPDATE_GOLDEN=1).

5) Integration flows
- CLI smoke tests (cmd/gocomic): create project → add minimal data → save/export; assert files exist and validate against schema.
- Migration tests: open older manifest versions and verify automatic upgrade produces expected new-version JSON.
- Backend E2E/parity: spin up backend locally (Docker); verify auth and search endpoints; compare representative queries against the embedded SQLite index.
- Crash/Recovery: simulate a panic during save and assert journal/backup allows lossless recovery.

6) Performance and stability
- Benchmarks (go test -bench) for serialization, layout passes, and export of a mid-size page.
- Memory and CPU budgets tracked per milestone; add benches to Phase 9 checklist.

### Cross-Platform Determinism
- Normalize floating point precision in stored JSON (fixed decimals) to avoid drift.
- Use a single shaping/layout engine for text; test identical inputs across OS via golden assets.
- Font-dependent tests pin to test fonts committed in testdata/fonts with clear licenses.

### Test Data Organization
- Per-package testdata/ folders with:
  - projects/ (tiny sample manifests),
  - scenes/ (JSON describing panels/balloons),
  - fonts/ (test-only fonts),
  - golden/ (expected exporter outputs),
  - migrations/ (older manifest snapshots).

### CI and Quality Gates
- Run unit + schema + fast golden tests on every commit; heavy exporter goldens can be nightly.
- Static analysis: go vet and staticcheck; lint commit messages for migration/version bumps.
- Backend parity tests: SQLite↔Postgres search semantics validated in CI (quick subset) and nightly (full suite).
- Coverage targets: Phase 0–2 ≥60% core packages; Phase 3–5 ≥80% for domain/storage/exporters (UI excluded).

### Manual and Exploratory Testing (UI)
- While the desktop UI is built, maintain lightweight manual test scripts per feature (tools, snapping, text editing).
- Screenshot-based comparison can be introduced later for the canvas (pixel or vector structure diffs).

Alignment with Definition of Done:
- Each feature merges with unit tests for models/storage/exporters and, when applicable, goldens or schema checks.
- Add/update a manual UI checklist until automated UI coverage exists.

---

## Milestones and Task List

### Phase 0 — Foundation
- [✓] Initialize Go modules and workspace layout.
- [✓] Define domain models and JSON schema for comic.json.
- [✓] Implement slog logging with custom handler.
- [✓] Implement logging, error reporting, and crash-safe autosave.
- [✓] Implement file I/O: create/open/save project; transactional writes; backups.
- [✓] Implement fist tests to start testing early.
- [✓] Implement basic UI shell with canvas editor.
- [✓] Implement basic storage layer with JSON manifest.

### Phase 1 — Core Rendering & Geometry
- [✓] Build vector primitives and text layout abstraction.
- [✓] Implement page canvas with trim/bleed/gutter guides.
- [✓] Implement shapes: rectangles, ellipses, rounded boxes, paths.
- [✓] Implement hit testing and selection; transform handles (move/scale/rotate).

### Phase 2 — Script Editor
- [✓] Implement a structured script editor with scene/character syntax support.
- [✓] Character/location bible with auto-complete and tagging.
- [✓] Beat tagging and a sidebar outline; search/filter.
- [✓] Map beats to pages/panels; show unmapped beat warnings.

### Phase 3 — Page & Panel Planner
- [✓] Issue setup dialog (trim, bleed, dpi, reading direction).
- [✓] Grid templates and custom grids; apply per page.
- [✓] Panel creation, ordering, and metadata editing.
- [✓] Beat coverage overlay; page-turn pacing indicators.

### Phase 4 — Lettering System
- [✓] Balloon and caption tools with snapping and smart guides.
- [✓] Tail drawing with speaker anchor and auto-orient.
- [✓] Typography engine: font loading, style presets, kerning/leading/tracking.
- [✓] SFX tool with outline/fill/effects and text-on-path.
- [✓] Auto layout suggestion for balloons with collision avoidance.
- [✓] Style sheets (global, per-issue, per-page).

### Phase 5 — Exporters
- [✓] PDF export (with trim/bleed, vector text when possible).
- [✓] PNG/SVG per page with DPI control.
- [✓] CBZ packaging with metadata manifest.
- [✓] Export presets (web, print) and batch export.

### Phase 5a — Data & Indexing (Embedded SQLite; see "Database and Indexing — Selected Approach")
- [✓] Establish per-project index store at `project/.gcw/index.sqlite`; enable WAL; add `meta/version` tables.
- [✓] Define schema: `documents` (doc_id, type, path, page_id, character_id, text), `fts_documents` (FTS5, contentless with external content), `cross_refs` (from_id → to_id), `assets` (hash, path, type), `previews` (page_id/panel_id, thumb_blob, updated_at), `snapshots` (page_id, ts, delta_blob).
- [✓] Implement background indexer: initial full build from `comic.json` and incremental updates on save; safe rebuild command ("Rebuild Index").
- [✓] Add search service in-app: full-text with filters (character, scene, page range, tags) and "where-used" via `cross_refs`.
- [✓] Wire UI: search panel/omnibox; navigate results to issue/page/panel; highlight hits.
- [✓] Add caching pipeline: generate/stash thumbnails and geometry caches in `previews`; LRU eviction and max-size cap.
- [✓] Migrations and tests: schema migration scripts, corruption/rebuild path, performance baselines; fixtures to validate FTS5 and cross-ref queries.
- [✓] Docs and ops: clarify DB is derived/rebuildable; backup guidance; vacuum schedule.

### Phase 6 — Project UX & Assets
- [ ] Project dashboard with recent files and templates.
- [ ] Asset library with previews and drag-and-drop into pages.
- [ ] Style pack manager (install/export styles and templates).
- [ ] Undo/redo stack with snapshots and performance safeguards.

### Phase 7 — Collaboration (Optional Early Access)
- [ ] Commenting and review mode on script and pages.
- [ ] Change tracking in script editor.
- [ ] Merge-friendly project format guidance and diff tips.

### Phase 7a — Thin Backend (PostgreSQL Version 17+; see "Database and Indexing — Selected Approach")
- [ ] Define minimal backend service (Go) using PostgreSQL Version 17+: schema for projects, `documents`, full-text via `tsvector`+GIN, `cross_refs`, assets metadata; migrations.
- [ ] API endpoints: auth (token), list projects, pull index snapshot; later: push deltas and comments.
- [ ] Desktop integration (behind feature flag): manual "Connect to Server"; read-only listing and search first; file-based `comic.json` remains the source of truth.
- [ ] Sync prototype: op-log format and stable IDs; `created_at/updated_at/version` columns; basic push/pull over HTTPS (no conflict resolution yet).
- [ ] Security/ops: TLS, per-user auth, Docker dev stack (Postgres + MinIO optional), config via env; health checks.
- [ ] Tests: parity checks for search between SQLite and Postgres; end-to-end integration tests.

### Phase 8 — Packaging & Distribution
- [ ] Cross-platform builds and installers.
- [ ] Crash reporting and opt-in telemetry (anonymous usage metrics).
- [ ] Documentation site with tutorials and templates.

### Phase 9 — QA & Performance
- [ ] Performance benchmarks on large issues; profiling and optimizations.
- [ ] Automated validation: manifest schema, font embedding checks.
- [ ] Accessibility audit and keyboard shortcut coverage.

---

## Definition of Done (Per Feature)
- Documented user flows and shortcuts.
- Deterministic rendering and export parity across platforms.
- Search/indexing parity: embedded SQLite search matches backend search semantics where applicable; successful full index rebuild from comic.json.
- Automated tests for model, storage, exporters, and search; manual test plans for UI.
- Performance baseline met or improved versus previous milestone.

---

## Stretch Goals
- Mobile companion app for script review and annotations.
- Right-to-left scripts and vertical text support.
- Advanced color management and preflight checks for print.
- Template marketplace and community style packs.

---

## Risks and Mitigations
- Font licensing and embedding: allow user-provided fonts and document licensing; support subset embedding.
- Cross-platform text rendering parity: rely on a single shaping/layout engine and comprehensive test scenes.
- Large project performance: incremental rendering, background export, and careful memory management.
- Merge conflicts in binary assets: keep manifests text-based and assets external; provide diff tooling for JSON.


---

## Database and Indexing — Selected Approach

Selected approach
- Source of truth: the project folder with a human-readable `comic.json` manifest and external assets. This remains canonical across all phases.
- Required local index (Phase 5a): an embedded per-project SQLite database under `project/.gcw/index.sqlite` providing full‑text search (FTS5), cross‑references, thumbnails, geometry caches, and autosave snapshots.
- Required thin backend (Phase 7a): a minimal Go service backed by PostgreSQL to provide organization‑wide search and lay the foundation for future sync. The desktop app stays offline‑first and continues to operate directly on the folder project model.

Why a DB and what it’s used for
1) Fast search and navigation
- Full‑text search across dialog, SFX, character names, notes, and the story Bible.
- Faceted queries (character, scene, page range, tags) that would be slow to compute from raw JSON on every request.
- Chosen tech:
  - Embedded: SQLite with FTS5 (contentless or external content tables) for full‑text.
  - Server: PostgreSQL with `tsvector` + GIN for full‑text; keep tokenization/config aligned with SQLite for parity.

2) Asset and preview caching
- Thumbnails, rasterized previews, and derived vector outlines to avoid recomputation on canvas zoom/pan.
- Asset hashes (content‑addressed) to deduplicate across pages/projects.
- Chosen tech:
  - SQLite BLOBs for thumbs/caches; consider a KV store only if needed later.

3) Autosave, snapshots, and history
- Periodic autosave snapshots/crash recovery separate from the main `comic.json`.
- Lightweight rolling checkpoints for panels/pages (not a full VCS).
- Schema: `snapshots(project_id, page_id, ts, delta_blob)` with retention policy.

4) Cross‑file references and integrity
- Unique IDs and backlinks (character ⇄ appearances, assets ⇄ usage) enabling fast "where used" and integrity checks.

5) Collaboration/sync readiness
- While real‑time is a later milestone, the backend schema supports comments, cross‑project search, and future op‑log/CRDT state.

What remains in files vs DB
- Files (source of truth):
  - `comic.json` per `docs/comic.schema.json` (Issue, Page, Panel, GridSpec, Bible, etc.).
  - Linked binary assets (fonts, images, placed SVGs) in project folders.
- Database (derived/ephemeral):
  - Full‑text index of project text fields.
  - Thumbnails, render caches, geometry caches, font lookup caches.
  - Autosave snapshots and recovery metadata.
  - Cross‑reference tables.

Embedded architecture (Phase 5a)
- Folder layout is unchanged:
  - `project/manifest/comic.json` (authoritative), `project/assets/…`, `project/.gcw/` for index/caches.
- Embedded DB at `project/.gcw/index.sqlite` (WAL enabled).
- Background indexer builds initially from `comic.json` and incrementally on saves; single "Rebuild Index" command for recovery.
- Optional global app DB: `~/.gocomicwriter/app.sqlite` for recents, font discovery cache, and settings.

Thin backend (Phase 7a)
- Go service + PostgreSQL schema for projects, `documents` (for search), `cross_refs`, assets metadata; FTS via `tsvector` + GIN.
- Endpoints: auth (token), list projects, pull index snapshot; subsequent milestones add push deltas/comments.
- Desktop integration (feature‑flagged UI path is acceptable) with manual "Connect to Server"; read‑only listing/search first; folder project remains the source of truth.
- Sync prototype: stable IDs and op‑log format with `created_at/updated_at/version` columns; basic push/pull over HTTPS.

Operational considerations
- Reliability: SQLite WAL. The embedded index is disposable and can be rebuilt from `comic.json` and assets at any time.
- Backups: back up the project folder (comic.json, script/, pages/, assets/, styles/, exports/, and backups/). You can exclude `.gcw/` — the index/caches are derived and rebuildable.
- Portability: projects open even if `.gcw/index.sqlite` is missing; the app detects this and creates/rebuilds the index.
- Maintenance (embedded SQLite):
  - Defaults: WAL enabled; prefer `auto_vacuum=INCREMENTAL`; keep `wal_autocheckpoint` around ~1000 pages.
  - Schedule: best effort maintenance on open/close when idle and either (a) last maintenance > 7 days, or (b) DB size > ~128 MiB.
    - Run `PRAGMA optimize;` and `INSERT INTO fts_documents(fts_documents) VALUES('optimize');` to tune FTS.
    - Run `PRAGMA incremental_vacuum;` to reclaim free pages without a full rewrite.
  - After large deletions (e.g., removing many pages/assets), consider a one‑off full `VACUUM` or simply delete `.gcw/index.sqlite` and let the app rebuild.
- Security: TLS for backend; no secrets stored in local DB; sanitize cached text used for FTS.

Testing and parity
- Parity tests: ensure SQLite and PostgreSQL return equivalent search results for shared tokenization/config.
- Corruption/rebuild: fixtures and tests for damaged indexes and successful rebuilds.
- Performance baselines: indexing, query latency, cache sizes.

Commitments
- JSON manifest remains canonical for editing and interchange.
- Embedded SQLite index is a required feature from Phase 5a onward.
- Thin backend (PostgreSQL) is a required milestone in Phase 7a, initially providing org‑wide search with read‑only integration and forming the basis for future sync.
