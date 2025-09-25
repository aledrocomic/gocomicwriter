# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog (https://keepachangelog.com/en/1.1.0/),
and this project adheres to Semantic Versioning. This is a pre-release
(0.x) and APIs may change at any time.

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
- Rendering/lettering engine and exporters (PDF/PNG/SVG/CBZ) are not implemented yet.

