# Go Comic Writer 2.x — Product Concept and Roadmap

A focused evolution of Go Comic Writer that keeps the original vision intact while strengthening typography, collaboration, performance, and production workflows. Version 2.x is an incremental, compatible series of releases that scale the app from strong single‑user workflows to small‑team collaboration and larger projects, without sacrificing the offline‑first model.

## Vision (unchanged)
Empower writers and comic teams to go from script to lettered pages in one streamlined tool—fast, reliable, and offline‑first—with an ergonomic writing experience and precise layout control.

## What “2.x” Means
- Compatibility first: comic.json remains the source of truth; the embedded SQLite index (.gcw/index.sqlite) is derived and rebuildable.
- Pragmatic collaboration: A thin backend provides organization‑wide search and opt‑in sync for comments and selected artifacts; the desktop app remains fully capable offline.
- Production‑ready typography and exports: Advanced text shaping, internationalization, and preflight strengthen print and digital delivery.
- Responsiveness at scale: Large issues and asset‑heavy projects remain smooth, with predictable memory and CPU budgets.

## Primary Goals for 2.x
1) Typography and Internationalization
- Unified shaping and layout with consistent behavior across OSes.
- Robust support for ligatures, kerning, script shaping (incl. RTL), fallback fonts, and hyphenation.
- Variable fonts and style packs that capture advanced typographic options.

2) Collaboration & Sync (opt‑in, offline‑first)
- Comments and review modes for script and pages.
- Change tracking in the script editor; optional read‑only sharing.
- Thin backend evolves from search/index snapshotting to selective sync (comments, annotations, light metadata) with clear conflict handling.

3) Project UX & Asset System
- Project dashboard with recent files and templates.
- Asset library with previews and drag‑and‑drop; style pack manager.
- Undo/redo with snapshotting and safeguards.

4) Export & Preflight
- More robust PDF pipeline (embedding, subset, metadata), PNG/SVG refinements.
- Preflight checks: fonts present/embedded, color space warnings, trim/bleed validation.
- Export presets expanded (social, webtoon‑friendly, print variants) and batch automation.

5) Performance, Stability, and Observability
- Measurable speed‑ups for layout, search, and export on large issues.
- Crash resilience and autosave improvements; better index rebuild paths.
- Optional telemetry and crash reporting (opt‑in, privacy‑respecting).

## Non‑Goals for 2.x
- Cloud‑only workflows (desktop remains primary).
- Heavy real‑time multi‑cursor editing; 2.x focuses on comments/review and staged sync.
- AI authoring features; the scope stays on craft tooling, not content generation.

## Compatibility and Migration
- Manifest remains text‑based (comic.json); introduce versioned schema upgrades with in‑app migration and backups.
- SQLite index (.gcw/index.sqlite) stays disposable/rebuildable; migrations are automated and tested.
- Export outputs remain deterministic; goldens updated only on intentional changes.

## Architecture Evolutions (2.x)
- Text layout: consolidate shaping features in internal/textlayout, ensuring platform parity; add fallback/stacked font selection and hyphenation.
- Rendering: maintain vector‑first pipeline; refine snapping/smart guides and geometry caches to reduce recompute.
- Indexing: keep FTS5 contentless tables and cross_refs; add derived caches for thumbnails/geometry with LRU policies; parity checks with backend when connected.
- Backend: thin Go service with PostgreSQL 17+; extend from read‑only search to selective sync for comments/annotations; clear auth and transport.
- Extension points: promote style packs/templates; prepare scripting hooks for automation (behind a flag).

## Milestones and Task List (2.x)

### 2.0 — Foundations and Migration
- [ ] Bump manifest/schema version and provide in‑app migration with backups and validation.
- [ ] Strengthen crash handling paths and autosave cadence; verify restore UX.
- [ ] Deterministic serialization for geometry/text runs (normalized precision).
- [ ] CI gates: schema lint, golden stability checks, and platform matrix runs.
- [ ] Packaging groundwork: cross‑platform build scripts and notarization/signing plan.

### 2.1 — Typography & Internationalization
- [ ] Unified shaping: ligatures, kerning, RTL, and script‑aware line breaking.
- [ ] Fallback fonts and font stacks with per‑style overrides; variable font axis support.
- [ ] Hyphenation dictionaries with locale selection; soft‑hyphen handling.
- [ ] Style sheets v2: advanced typographic properties and inheritance.
- [ ] Golden tests for complex scripts; identical layout across OSes.

### 2.2 — Project UX & Assets
- [ ] Project dashboard: recent files, templates, sample projects onboarding.
- [ ] Asset library: previews, search, tags; drag‑drop into pages and script links.
- [ ] Style pack manager: install/export; per‑project and global packs.
- [ ] Undo/redo: snapshot strategy, memory caps, and conflict guards.
- [ ] Accessibility pass: keyboard coverage, focus order, high‑contrast themes.

### 2.3 — Collaboration & Thin Backend Evolution
- [ ] Commenting and review mode on script and pages (local first; syncable later).
- [ ] Change tracking in script editor with accept/reject workflow.
- [ ] Backend endpoints: auth (token), list projects, pull index snapshot (read‑only), push/pull comments/annotations (pilot).
- [ ] Desktop integration (feature flag): connect/disconnect, project listing, search parity checks.
- [ ] Sync prototype: op‑log format, stable IDs, created_at/updated_at/version; basic conflict surfaces and manual resolution.
- [ ] Security/ops: TLS, per‑user auth, Docker dev stack, health checks, and rate limits.

### 2.4 — Export & Preflight
- [ ] PDF: font embedding/subsetting audits; metadata, ICC profiles, and trim/bleed correctness.
- [ ] PNG/SVG: DPI controls and vector/text preservation improvements.
- [ ] CBZ: enriched metadata manifest and series packaging options.
- [ ] Preflight checks: missing fonts, color space mismatches, bleed/trim warnings; export blocker or warning configuration.
- [ ] Export presets v2 and batch export improvements; headless CLI for automation.

### 2.5 — Performance, QA, and Observability
- [ ] Index rebuild and search benchmarks on large projects; target budgets and regressions alarms.
- [ ] Layout passes: cache hits, reduced allocations; benchmarks in CI.
- [ ] Crash reporting and opt‑in telemetry (anonymous); consent UI and docs.
- [ ] Automated validation: schema, font embedding checks, exporter goldens.
- [ ] Documentation site refresh with tutorials and templates.

## Definition of Done (2.x)
- Vision preserved; user flows clarified and documented.
- Deterministic rendering and export parity across platforms.
- Search/indexing parity when online; successful full index rebuild from comic.json.
- Automated tests for domain, storage, text layout, and exporters; manual plans for UI; goldens for complex scripts.
- Performance budgets met; crash/restore paths tested.
- Packaging delivers signed builds for supported platforms.

## Stretch Goals (post‑2.5)
- Real‑time page‑level co‑editing for lettering (limited scope).
- Right‑to‑left reading direction enhancements and vertical text layout presets.
- Imposition options for print workflows and spot color handling.
- Scripting hooks for batch tasks and integrations.

## Risks and Mitigations
- Typography parity across OSes: choose a single shaping strategy and lock tests to licensed test fonts; keep goldens for complex scripts.
- Data model drift: maintain clear, versioned schema migrations with backups and validation prior to write.
- Large project performance: background computation, caches, and budgets; nightly performance checks.
- Merge conflicts in binary assets: keep manifests text‑based, assets external; provide diff tooling and guidance.

---

Notes
- The 2.x roadmap assumes the 1.x plan and foundations in docs/go_comic_writer_concept.md are complete. This document refines direction for the next major iteration while keeping the original vision and offline‑first principles unchanged.
