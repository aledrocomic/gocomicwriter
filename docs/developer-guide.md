# Go Comic Writer — Developer Guide

Audience: contributors, integrators, and maintainers working on the Go Comic Writer codebase.

This guide complements the README (user‑oriented quick start) and the product concept/roadmap in docs/go_comic_writer_concept.md. It focuses on developer setup, architecture, coding/testing conventions, build tags, CI/CD, and release flow.

- Project home: https://github.com/aledrocomic/gocomicwriter
- License: Apache 2.0 (see LICENSE)
- Code of Conduct: CODE_OF_CONDUCT.md


## Quick start for developers

Prerequisites
- Go 1.24+
- Git
- Optional (for UI builds with Fyne): a working C toolchain and CGO enabled
  - Windows: MSYS2/MinGW-w64 (or LLVM/clang), ensure gcc/clang on PATH; enable CGO
  - macOS: Xcode Command Line Tools (`xcode-select --install`)
  - Linux: build-essential (Debian/Ubuntu) or base-devel (Arch)

Clone and build (UI build):
- Windows PowerShell
  - go build -tags fyne -o bin\gocomicwriter.exe ./cmd/gocomicwriter
- macOS/Linux
  - go build -tags fyne -o bin/gocomicwriter ./cmd/gocomicwriter

Run (UI):
- Windows PowerShell
  - go run -tags fyne ./cmd/gocomicwriter
- macOS/Linux
  - GOFLAGS='' go run -tags fyne ./cmd/gocomicwriter

Headless / no-UI stub:
- go run ./cmd/gocomicwriter
  - Without the `fyne` build tag, a helpful stub binary is built (no desktop UI). Useful in CI and servers.

Helpful environment variables (logging):
- GCW_LOG_LEVEL=debug|info|warn|error
- GCW_LOG_FORMAT=console|json
- GCW_LOG_FILE=<path> (enables rotating JSON file logs)
- GCW_LOG_SOURCE=true|false (include source file/line)


## Repository layout and entry points

- cmd/gocomicwriter
  - Main program entry. Build with `-tags fyne` for the desktop UI, or without for a CLI/stub.
- internal/ui
  - Desktop UI implemented with Fyne v2.
  - Build-tagged variants:
    - app_fyne.go — real UI when `fyne` and CGO are enabled.
    - app_fyne_nocgo.go — tells you CGO is needed if `fyne` is set but CGO is off.
    - app_stub.go — stub when built without `fyne`.
- internal/domain
  - Core types for issues, pages, panels, and supporting structures.
- internal/storage
  - Project persistence layer (transactional save, backups, validation against schema).
  - Fall‑back open: if manifest is unreadable, auto‑selects latest valid backup.
- internal/export
  - Exporters for PDF, PNG, SVG, CBZ, and EPUB.
- internal/script
  - Lightweight script parser and types; beat extraction; outline & filters.
- internal/textlayout
  - Abstractions for text layout and SFX; typography groundwork.
- internal/vector
  - 2D geometry primitives, paths, styles, transforms, handles, smart guides.
- internal/log
  - Centralized slog setup (console/JSON, rotating file via lumberjack). Env‑driven.
- internal/crash
  - Crash recovery utilities: panic capture, crash report file, and autosave snapshot.
- internal/version
  - Single source of truth for the application version string.
- docs/
  - Concept/roadmap, JSON schema, CI/CD notes, and this guide.


## Build matrix and tags

- UI build (desktop app): add build tag `fyne`.
  - Requires CGO and an OpenGL-capable environment.
- Headless build (default): no `fyne` tag → compiles the stub app.
- Example commands:
  - go build -tags fyne -o bin/gocomicwriter ./cmd/gocomicwriter
  - go run -tags fyne ./cmd/gocomicwriter
  - go build -o bin/gocomicwriter ./cmd/gocomicwriter   # stub build

Windows CGO quick tips
- Install MSYS2 and MinGW‑w64 packages (64‑bit).
- Ensure gcc.exe on PATH from a MinGW64 shell or add to PowerShell PATH.
- Enable CGO for your session: `$env:CGO_ENABLED = '1'`

macOS CGO quick tips
- Install Xcode CLT. CGO is on by default.

Linux CGO quick tips
- Install build-essential/clang and necessary OpenGL dev libs.

Run with race detector (useful in tests)
- go test -race ./...
- go run -race -tags fyne ./cmd/gocomicwriter

Cross compilation (non‑UI)
- GOOS=windows GOARCH=amd64 go build -o dist/gocomicwriter-windows-amd64.exe ./cmd/gocomicwriter
- GOOS=linux   GOARCH=amd64 go build -o dist/gocomicwriter-linux-amd64       ./cmd/gocomicwriter


## Data model and schema

- The project’s source of truth is a human‑readable JSON manifest at <project>/comic.json.
- A public JSON Schema lives at docs/comic.schema.json.
- Saves are transactional; previous manifests are backed up under <project>/backups/ as comic.json.YYYYMMDD-HHMMSS.bak.
- On open, storage falls back to the latest valid backup if the current manifest is unreadable.

Beat and script integration (experimental)
- The script editor extracts beats. Beats have stable IDs like `b:<lineNo>`.
- Panels link beats via `linkedBeats` in the manifest.


## Storage & Indexing (embedded SQLite) — ops notes

- Location: per-project embedded index at `<project>\\.gcw\\index.sqlite` providing full‑text search (FTS5), cross‑references, thumbnails, and geometry caches.
- Derived/rebuildable: the index is derived from `comic.json` and assets. It is safe to delete; the app recreates/rebuilds it on open. The JSON manifest remains canonical.
- SQLite settings: WAL mode enabled; FTS5 contentless index kept in sync via triggers; prefer `auto_vacuum=INCREMENTAL`; keep `wal_autocheckpoint` around ~1000 pages.
- Backups: include the project folder (`comic.json`, `script/`, `pages/`, `assets/`, `styles/`, `exports/`, and `backups/`). You may exclude `.gcw/` entirely — it contains only derived state.
- Maintenance schedule (recommendation):
  - Weekly or when DB > ~128 MiB: run `PRAGMA optimize;` and FTS optimize via `INSERT INTO fts_documents(fts_documents) VALUES('optimize');`, then `PRAGMA incremental_vacuum;`.
  - After large deletions: optionally run a full `VACUUM` or delete `index.sqlite` to force a clean rebuild.
- Recovery path: if corruption is detected, back up `index.sqlite` (optional), remove it, and launch the app or call the storage `RebuildIndex` helper to repopulate from `comic.json`.

## Logging and crash handling

- Logging is centrally configured by internal/log with slog.
- Configure via env (see Quick start). Default: INFO, pretty console.
- To add context from code, use:
  - applog.WithComponent("component") to add a component label.
  - applog.WithOperation(l, "opName") to mark operations.
- Crash handling (internal/crash):
  - Defer crash.Recover(ph) in top‑level UI flows to capture panics.
  - On panic: writes a crash report to <project>/backups or system temp and attempts an autosave snapshot.


## Testing

Run all tests
- go test ./...

With coverage
- go test -cover ./...

Race detector
- go test -race ./...

UI tests (optional)
- These are gated behind the `fyne` tag to avoid CI display dependencies.
- Run locally only when you have Fyne deps and a display:
  - go test -tags fyne ./internal/ui

Package‑specific notes
- internal/storage: tests validate schema compatibility and transactional backups.
- internal/export: tests verify exporters generate files of expected shape/metadata.
- internal/vector, internal/textlayout: heavy geometry and layout invariants.


## Coding standards and tools

Formatting & linting
- gofmt is mandatory (CI checks formatting; run `gofmt -s -w .`).
- go vet should be clean.
- Optionally use staticcheck and govulncheck locally.

Project conventions
- Error handling: prefer wrapping with context using `%w` and errors.Join as needed.
- Logging: avoid log.Printf; use internal/log and structured fields.
- Package boundaries: keep UI‑free logic in internal/* packages so it’s testable without Fyne.
- Build tags: keep the `fyne` UI isolated; provide stubs where appropriate.
- Determinism: exporters and layout code should be deterministic across platforms.

Commit messages
- Conventional style appreciated (feat:, fix:, docs:, test:, refactor:, chore:). Reference issue numbers where applicable.


## Versioning and releases

- Version lives in internal/version/version.go (const Version).
- Use semantic versioning (MAJOR.MINOR.PATCH‑pre) and keep CHANGELOG.md updated.

Release checklist (manual)
1) Ensure tests are green locally and in CI.
2) Update internal/version.Version to the new tag (e.g., 0.6.0).
3) Update CHANGELOG.md.
4) Tag the commit: `git tag v0.6.0 && git push --tags`.
5) Build release artifacts:
   - Cross‑compile non‑UI binaries as needed.
   - For UI binaries, build natively per OS with `-tags fyne`.
6) Optionally publish artifacts to S3 or GitHub Releases.

CI/CD on AWS (optional)
- See docs/ci-cd-aws.md and docs/aws-codepipeline.yml.
- The template provisions CodePipeline + CodeBuild to build/test and optionally upload artifacts to S3.
- Adjust the inline buildspec to point to ./cmd/gocomicwriter (see doc for details).


## Local developer workflow

- Typical loop
  - Edit code in internal/*, run `go test ./...`.
  - For UI: run `go run -tags fyne ./cmd/gocomicwriter` and iterate.
  - Use GCW_LOG_LEVEL=debug for richer logs.
- Troubleshooting UI startup
  - If you see OpenGL/GLFW errors, ensure you have a display and drivers.
  - If you see “UI not built in this binary”, you forgot `-tags fyne`.
  - If build fails with CGO disabled, enable CGO and install a C toolchain (see Quick start).


## Contributing

- Please read CODE_OF_CONDUCT.md.
- Open a PR with a clear description and small, focused changes.
- Add or update tests for your changes.
- Keep docs in sync (README, this guide, and the JSON schema if data model changes).


## Roadmap and design

- The product concept, milestones, and selected technical approaches are documented in docs/go_comic_writer_concept.md.
- Use it to understand the phases (rendering, text layout, exporters, search/indexing, packaging, optional thin backend) and DoD per feature.


## FAQ / Troubleshooting

Build complains about missing OpenGL / Fyne deps
- Ensure CGO is enabled and Fyne prerequisites are installed (see Fyne docs).

CI fails UI tests
- UI tests are behind `fyne`. CI intentionally runs without them. Run locally with `-tags fyne`.

Exports look different on Windows vs macOS
- File a bug with a minimal reproduction and include OS/arch/Go version. Exporters aim to be deterministic across platforms.

Where are logs?
- Console by default. If GCW_LOG_FILE is set, rotating JSON logs are written to that path.

Where are crash reports?
- <project>/backups/crash-YYYYMMDD-HHMMSS.log (or system temp if no project).

