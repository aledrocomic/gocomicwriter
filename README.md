# Go Comic Writer

A Go-powered project aiming to become a writing, planning, and lettering toolchain for comics — from script to precisely lettered pages — with reliable exports for print and screen.

This repository provides the Go Comic Writer desktop app and backend server, an evolving domain model, and a public JSON schema for the project manifest. The product concept and roadmap live in docs/go_comic_writer_concept.md. For the 2.x plan and tasks, see docs/go_comic_writer_concept_2x.md.

- Vision: Empower comic creators to go from script to lettered pages in one streamlined, offline‑first tool.
- Status: Public Beta (as of 2025-10-02). Suitable for evaluation and small projects; expect some features to be incomplete. Back up your work regularly.
- License: Apache 2.0

## Contents
- Beta Program (Public Beta)
- What is this? (short overview)
- Tech stack and entry points
- Current features and what’s next
- Install and quick start
- Usage
- Backend (gcwserver) — run locally
- Common commands (scripts)
- Logging and environment variables
- Project manifest (comic.json) and schema
- Database, backups, and maintenance
- Repository layout
- Tests
- Developer Guide (for contributors): docs/developer-guide.md
- Roadmap and concept
- Tutorials & templates: docs/tutorials_and_templates.md (build PDF with scripts/build_docs.ps1)
- CI/CD and releases
- Contributing and conduct
- License

## Beta Program (Public Beta)
As of 2025-10-02, Go Comic Writer has entered Public Beta. The Beta is intended for evaluation, early adopters, and small real projects. Expect some rough edges and the occasional breaking change. Always keep backups of your projects.

What's included
- The core desktop app with canvas, project storage with timestamped backups, exporters (PDF/PNG/SVG/CBZ/EPUB), a structured script editor with outline and beat mapping, an assets panel, style packs, full-text search, and undo/redo. See “Current features (Beta)” below for details.

Known limitations
- Some lettering and pro typography tools are still in progress.
- Advanced export presets and fine-grained options are limited.
- See “What’s not in Beta yet” below.

Stability and data safety
- comic.json is the human-readable source of truth. Each save is transactional and writes a timestamped backup under backups/.
- On a crash, the app writes a crash log and an autosave snapshot.

Getting the Beta
- When available, prebuilt Beta binaries are published on the repository’s Releases page.
- Building from source is fully supported on Windows, macOS, and Linux; see “Install and quick start” below.

Feedback and support
- Please file bugs and feature requests in the issue tracker. Include steps to reproduce, logs, and screenshots where possible.
- Anonymous telemetry and crash uploads are OFF by default. You can opt in via environment variables; see “Telemetry (opt-in) and crash reporting” below.

## What is this?
Go Comic Writer is an in‑progress toolchain for comic writing and lettering. It’s designed to be:
- Writing‑first
- Page‑aware
- Precise for typography/balloons/SFX
- Deterministic in rendering and export
- Cross‑platform and offline‑first

The long‑term plan is a desktop application with a canvas editor and exporters (PDF/PNG/SVG/CBZ/EPUB). See the concept document for details.

## Tech stack and entry points
- Language: Go 1.24 (module-based; see go.mod)
- UI framework: Fyne v2 (build tag: `fyne`)
- Logging: standard library slog with optional rotating file output (lumberjack)
- JSON Schema validation: xeipuuv/gojsonschema
- Package manager: Go modules
- OS: Windows, macOS, Linux (UI requires OpenGL and cgo)

Entry points:
- cmd/gocomicwriter/main.go — main program. Build with `-tags fyne` to include the desktop UI; without it, a stub is compiled that prints a helpful message.
- cmd/gcwserver/main.go — thin backend server (gcwserver). Provides read-only APIs for listing projects, fetching index snapshots, search, and basic sync. See "Backend (gcwserver) — run locally" below.
- internal/ui — UI implementation. Build-tagged variants:
  - app_fyne.go — real UI when `fyne` and cgo are enabled.
  - app_fyne_nocgo.go — helpful message when `fyne` is set but cgo is disabled.
  - app_stub.go — stub when built without the `fyne` tag.

## Current features (Beta)
- Desktop UI launcher; optional project path argument to open on startup.
- Transactional project storage with a human‑readable manifest (comic.json) and timestamped backups under backups/.
- Crash safety: on panic, write a crash report and autosave snapshot; on open, fall back to the latest valid backup if the manifest is unreadable.
- Structured logging via Go's slog with simple env configuration; optional rotating file via GCW_LOG_FILE.
- Core domain model in internal/domain and a public JSON schema at docs/comic.schema.json.
- Basic desktop UI shell (behind build tag `fyne`) with a canvas editor that shows page/trim/bleed guides, pan/zoom, and File→New/Open/Save.
  - Keyboard shortcuts: Ctrl+N, Ctrl+O, Ctrl+S, Ctrl+Q.
  - Preferences persisted: window size and the Beat Coverage overlay toggle are saved between sessions.
  - The UI can start without a project and lets you create one from within the app.
- Project dashboard: recent projects list and starter templates (Blank, 3x3 Grid).
- Issue setup dialog: configure trim size, bleed, DPI, and reading direction (LTR/RTL) from the UI.
- Page grids: supported via the page's `grid` property in the manifest (e.g., "3x3") and previewed on the canvas; in-UI grid editing is planned.
- Panels: add from the Inspector (Add Panel), reorder Z with Move Up/Down, and edit metadata (ID, notes). A quick filter above the panel list helps find panels by ID/notes/text.
- Script integration (experimental): structured editor with outline and beat tagging; beats can be linked to panels; unmapped beat warnings in outline.
- Beat coverage overlay and page‑turn pacing indicators (experimental) in the canvas to aid layout/planning.
- Exporters (UI): Export menu for PDF (multi-page), PNG pages, SVG pages, CBZ package, and EPUB (fixed-layout). Exports include trim/bleed guides and respect issue settings.
- Assets pane: previews images from project/assets; click to arm and place into panels.
- Style Pack manager: import/export styles and templates via the Style Pack menu.
- Undo/Redo: snapshot-based undo/redo with safeguards (Edit → Undo/Redo).
- Search panel/omnibox: instant full-text search with filters (character, scene, page range, tags); navigate to results (issue/page/panel) and highlight hits.
- Commenting and review mode on script and pages (minimal; behind feature flag).
- Thin backend integration (feature-flagged): File → Server → Connect to Server… shows a read-only list of projects from a gcwserver instance and allows simple snapshot text search; comic.json remains the source of truth.
- Change tracking in script editor.
- Documentation: Merge-friendly project format guidance and diff tips (see “Merge-friendly Project Format & Diff Tips” in docs/go_comic_writer_concept.md).
- About menu with environment info (Go version, OS/arch, cgo/fyne status) and a Copyright dialog.
- Unit tests for core packages (storage, logging, crash, version, schema).
- Vector primitives and transforms with a small scene graph and hit testing (see internal/vector: geometry.go, node.go, path.go, style.go).
- Text layout abstraction scaffolding (internal/textlayout) to prepare for typography and balloon text.
- Page canvas with trim/bleed/gutter guides, pan/zoom, and selection in the experimental UI (build with `-tags fyne`).
- Shapes: rectangles, ellipses, rounded boxes, and paths, with axis-aligned bounds for layout/selection.
- Selection and transform handles enabling move, scale (corner handles), and rotate (rotation handle).

What’s not in Beta yet:
- Full-featured rendering/lettering engine and pro typography tools in the editor.
- Advanced export options and presets in the UI (basic PDF/PNG/SVG/CBZ exporters are implemented).

## Install and quick start

Note for the Beta: When available, prebuilt Beta binaries are published on the repository’s Releases page. If no release assets are provided for your platform yet, you can build from source using the steps below.
Prerequisites:
- Go 1.24 or newer
- A supported OS (Windows/macOS/Linux)

Build the desktop app (UI) from source:

```bash
# From within a clone of this repository (UI build; requires cgo and a C toolchain)
go build -tags fyne -o bin/gocomicwriter ./cmd/gocomicwriter

# Or install into your GOPATH/bin (adjust module path if needed)
go install -tags fyne ./cmd/gocomicwriter
```

Verify it launches (UI):

```bash
# Windows PowerShell
go run -tags fyne ./cmd/gocomicwriter

# macOS/Linux
GOFLAGS='' go run -tags fyne ./cmd/gocomicwriter

# Optionally, open a specific project on startup by passing its path
# Windows PowerShell
go run -tags fyne ./cmd/gocomicwriter C:\\path\\to\\your\\project
# macOS/Linux
GOFLAGS='' go run -tags fyne ./cmd/gocomicwriter /path/to/your/project
```

## Usage
This build is UI-only. Launch the app as shown above. Optionally pass a project directory path to open on startup.

### Run the basic UI (experimental)
The repository includes a minimal desktop UI shell guarded by the `fyne` build tag.

Build and run directly (no binary):

```bash
# Start the UI with no project (Windows PowerShell)
go run -tags fyne ./cmd/gocomicwriter
# Use File → New to create a project, or File → Open to open an existing one.

# Alternatively, open a specific project directly by passing its path
# Windows PowerShell
go run -tags fyne ./cmd/gocomicwriter C:\\path\\to\\your\\project

# On macOS/Linux
GOFLAGS='' go run -tags fyne ./cmd/gocomicwriter /path/to/your/project
```

Or build a binary with UI support:

```bash
# Windows
go build -tags fyne -o bin/gocomicwriter-ui.exe ./cmd/gocomicwriter

# macOS/Linux
go build -tags fyne -o bin/gocomicwriter-ui ./cmd/gocomicwriter
```

Then run:

```bash
bin/gocomicwriter-ui C:\\path\\to\\your\\project  # on Windows
# or on macOS/Linux
./bin/gocomicwriter-ui /path/to/your/project
```

Notes and controls:
- File → New/Open/Save (shortcuts: Ctrl+N/Ctrl+O/Ctrl+S; Close Project: Ctrl+W; Quit: Ctrl+Q). Saves are transactional with timestamped backups.
- Issue → Setup opens the Issue Setup dialog (trim size, bleed, DPI, reading direction). Changes apply to the current issue.
- Export menu: Export Issue as PDF…, PNG pages…, SVG pages…, CBZ…, or EPUB…. You will be prompted for a file or folder; exports include trim/bleed guides and respect issue settings.
- Panels (Inspector on the right): use Add Panel to create; select in the list to edit. Use Move Up/Down to change Z-order, Edit Metadata to change ID/notes, and the quick filter to find panels.
- Script integration: see the Script tab. Beats can be linked to panels; unmapped beats are highlighted in the outline.
- Overlays and pacing: toggle Beat Coverage Overlay in the Inspector; pacing info for the current page is shown above the panel list.
- Canvas: page rectangle with bleed (blue) and trim (red) guides. Drag on empty area to pan. Mouse Wheel to zoom in/out.
- Window title shows the project name when opened.

### Script Editor (experimental)
- Open the "Script" tab to write a structured script and see an outline update as you type.
- Supported syntax (minimal initial version):
  - Scene headers: lines starting with `#` (e.g., `# Opening Scene`) or `Scene: Title`.
  - Character dialogue: `NAME: text` (NAME is treated case-insensitively and shown uppercase in the outline).
    - Continuation lines: indent by two spaces to continue the previous dialogue/caption.
  - Captions/Narration: `CAPTION: text` or `NARRATION: text`.
  - Beat markers: `Panel 1 ...` or `Beat ...` are recognized and shown in the outline.
  - Notes: lines starting with `;` are notes (not shown in outline for now).
- Outline sidebar with search/filter:
  - Type to filter by free text, e.g., words from scene titles or lines.
  - Use @tags to filter by tags referenced in lines (e.g., `@prop`, `@theme-1`).
  - Use `char:NAME` to filter by character dialogue (e.g., `char:ALICE`).
  - Use `is:beat`, `is:dialogue`, `is:caption`, or `is:scene` to filter by item kind.
  - Combine terms (space-separated) for AND filtering.
- Save: File → Save writes both the manifest and your script to `<project>\script\script.txt`. 

### Beat-to-Panel Mapping and Unmapped Warnings (experimental)
- Beat lines in the script (those starting with "Panel ..." or "Beat ...") are assigned a stable identifier based on their source line number, in the form `b:<lineNo>` (e.g., `b:42`).
- Panels can link beats via `linkedBeats` (array of strings) in the project manifest (comic.json). Example:
  - "panels": [{"id": "p1", "zOrder": 0, "geometry": {"x":0,"y":0,"width":100,"height":100}, "linkedBeats": ["b:42"]}]
- The Script tab outline shows a warning marker for unmapped beats: a "⚠ unmapped" suffix appears on beats that are not linked from any panel in the current project. A summary is also shown in the status bar (e.g., `Script: 7 beats (3 unmapped)`).
- Programmatic mapping helper: `storage.MapBeatToPanel(ph, pageNumber, panelID, beatID)` adds a beat mapping to a panel if it exists. This is a building block ahead of a full UI for page/panel planning.

### Bible (characters, locations, tags) — experimental
- Open the "Bible" tab to manage reusable names and tags used in your script.
- Characters and Locations: add names via the text field and Add button; select an item and click Delete to remove it.
- Tags: add free-form tags (e.g., themes, props). Tags can be referenced in your script as `@tag`.
- In the Script tab, use the buttons above the editor to insert a character line (NAME: ) or an `@tag` from the bible. This simulates auto-complete.
- All bible data is saved in the project manifest (comic.json) under `bible`.

Troubleshooting:
- Linux may require OpenGL drivers and a working X11/Wayland setup. On headless CI this UI is not built by default.
- If you see "UI not built in this binary", rebuild with `-tags fyne`.
- If you see a build error like: `imports github.com/go-gl/gl/v2.1/gl: build constraints exclude all Go files`, it means cgo is disabled and the OpenGL backend cannot compile.
  - On Windows, install a C toolchain (MSYS2/MinGW-w64) so gcc is available, then enable cgo:
    - Start an MSYS2 MinGW64 shell or ensure `gcc` is on PATH in PowerShell.
    - PowerShell: `setx CGO_ENABLED 1` (or for the current session: `$env:CGO_ENABLED='1'`)
    - Then: `go run -tags fyne ./cmd/gocomicwriter`
  - On macOS: Xcode Command Line Tools are usually sufficient. Ensure `CGO_ENABLED=1`.
  - On Linux: install build-essential (Debian/Ubuntu) or base-devel (Arch), ensure `CGO_ENABLED=1`.
- If cgo is still disabled, the binary will fall back to a helpful stub error when running the app built with `-tags fyne`. 

Notes:
- Project operations (New/Open/Save) are available from the UI's File menu. Saves are transactional and copy the previous manifest into backups/comic.json.YYYYMMDD-HHMMSS.bak. Opening a project falls back to the latest valid backup if the manifest is unreadable.

## Common commands (scripts)
These shell snippets act as “scripts” you can copy-paste. Adjust paths for your OS.
- Build UI binary (Windows): `go build -tags fyne -o bin\\gocomicwriter.exe ./cmd/gocomicwriter`
- Build UI binary (macOS/Linux): `go build -tags fyne -o bin/gocomicwriter ./cmd/gocomicwriter`
- Run UI from source (Windows): `go run -tags fyne ./cmd/gocomicwriter`
- Run UI from source (macOS/Linux): `go run -tags fyne ./cmd/gocomicwriter`
- Exports: use the app's Export menu (CLI export commands have been removed).
- Format code: `gofmt -s -w .`
- Vet: `go vet ./...`

## Logging configuration
The app uses structured logging (slog). Configure via environment variables:
- GCW_LOG_LEVEL=debug|info|warn|error (default: info)
- GCW_LOG_FORMAT=console|json (default: console)
- GCW_LOG_SOURCE=true|false (default: false)
- GCW_LOG_FILE=<path> (optional; enables rotating JSON file logs)

Examples:
- PowerShell: `$env:GCW_LOG_LEVEL='debug'; go run -tags fyne ./cmd/gocomicwriter`
- Bash: `GCW_LOG_FORMAT=json GCW_LOG_FILE=gcw.log go run -tags fyne ./cmd/gocomicwriter`

## Feature flags
The app includes a few early, opt-in features that are hidden by default and can be enabled via environment variables.

- GCW_ENABLE_SERVER=true|1|on
  - Adds a “Server” menu with “Connect to Server…”.
  - Lets you connect to a running gcwserver backend (base URL + bearer token), list projects, and view an index snapshot per project.
  - Read-only: no data is written to your local project; comic.json on disk remains the source of truth.

## Telemetry (opt-in) and crash reporting
The app includes a tiny, privacy-respecting telemetry client that is disabled by default. When enabled by you, it sends anonymous usage events (like "app_start" and "project_open" with simple counts) and can upload crash reports. No project paths, filenames, or personal data are sent.

How to enable (env-based; default is OFF):
- GCW_TELEMETRY_OPT_IN=1
- GCW_TELEMETRY_URL=https://your.endpoint.example/telemetry  (receives JSON events)
- GCW_CRASH_UPLOAD_URL=https://your.endpoint.example/crash     (receives text/plain crash logs)
- GCW_TELEMETRY_TIMEOUT_MS=1500                                 (optional; default 1500ms)
- GCW_TELEMETRY_DEBUG=1                                         (optional; logs send attempts)

Notes:
- If URLs are not set, nothing is sent even if GCW_TELEMETRY_OPT_IN=1.
- Telemetry never blocks the UI; it uses a small async queue and drops on failure.
- Crash reports are also written locally under backups/ or the system temp folder regardless of opt-in; upload only happens when GCW_TELEMETRY_OPT_IN=1 and GCW_CRASH_UPLOAD_URL is configured.

## Crash reports and autosave
On an unexpected crash (panic), the app will:
- write a crash report file named `crash-YYYYMMDD-HHMMSS.log` under `<project>\backups` (or the system temp dir if no project is open),
- write an autosave snapshot of the manifest as `backups/comic.json.crash-YYYYMMDD-HHMMSS.autosave`,
- exit with a non-zero status code.

## Project manifest (comic.json) and schema
The project’s canonical manifest is intended to be a readable JSON file, usually named comic.json. A formal schema is provided to aid validation and tooling:

- Schema file: docs/comic.schema.json

A minimal example comic.json:

```json
{
  "name": "My Series",
  "metadata": {
    "series": "My Series",
    "issueTitle": "Issue #1",
    "creators": "Writer, Artist, Letterer"
  },
  "issues": [
    {
      "trimWidth": 210,
      "trimHeight": 297,
      "bleed": 3,
      "dpi": 300,
      "readingDirection": "ltr",
      "pages": [
        {
          "number": 1,
          "grid": "3x3",
          "panels": []
        }
      ]
    }
  ]
}
```

Note: The schema defines richer structures for pages, panels, balloons, styles, etc. For example, panels include fields like `id`, `zOrder`, `geometry {x,y,width,height}`, optional `notes`, and `linkedBeats` (array of beat IDs like `b:42`). See docs/comic.schema.json for all fields.

No sample project is bundled. Create a new one via File → New in the app, or open an existing project directory.

## Database, backups, and maintenance

Embedded index (SQLite):
- Per project, the app keeps an embedded SQLite database at `<project>\\.gcw\\index.sqlite` to power fast search (FTS5), cross‑references, and caches (thumbnails/geometry).
- This database is derived from your manifest and assets. It is disposable and can be rebuilt at any time. Your source of truth remains `comic.json` and your asset files.

Backups — what to include/exclude:
- Include in backups: the entire project folder except `.gcw/` — at minimum `comic.json`, `script/`, `pages/`, `assets/`, `styles/`, `exports/`, and the `backups/` directory with timestamped manifest backups.
- Exclude from backups (optional): `.gcw/` (contains `index.sqlite` and caches). If lost, the app will rebuild it automatically when you open the project.

Rebuild the index (if needed):
- Easiest: open the project; if `.gcw/index.sqlite` is missing or corrupt, the app detects it and performs a clean rebuild from `comic.json`.
- Manual: close the app, delete `<project>\\.gcw\\index.sqlite`, then reopen the project. The index will be recreated. No project content is lost.

Maintenance (SQLite VACUUM/optimize):
- Defaults: WAL mode on; prefer `auto_vacuum=INCREMENTAL` under the hood; reasonable `wal_autocheckpoint`.
- Recommended schedule (best effort, when idle):
  - Weekly or when the DB grows beyond ~128 MiB: run `PRAGMA optimize;` and FTS optimize via `INSERT INTO fts_documents(fts_documents) VALUES('optimize');`, then `PRAGMA incremental_vacuum;` to reclaim free pages.
  - After large deletions (many pages/assets removed): optionally run a full `VACUUM` or simply delete `index.sqlite` and let the app rebuild.
- Note: These steps are informational; typical users don’t need to do anything. The app maintains the index and can always rebuild it.

## Backend (gcwserver) — run locally

Overview
- A thin backend service offering read-only APIs to list projects, fetch the latest index snapshot, perform text search, and a simple op-log sync prototype. The desktop app remains file-first; the backend is optional and behind a feature flag.

Requirements
- PostgreSQL 17 or newer.
- Optional: MinIO (S3-compatible object storage) for future asset health checks (not required for basic API usage).
- See .env.example for all server environment variables.

Quick start with Docker Compose (PostgreSQL and optional MinIO)
- Start Postgres: `docker compose up -d postgres`
- (Optional) Start MinIO: `docker compose up -d minio`
- The compose file maps Postgres to localhost:5432.

Run the server from source
- Windows PowerShell:
    - `$env:GCW_PG_DSN='postgres://postgres:postgres@localhost:5432/gocomicwriter?sslmode=disable'`
    - `go run ./cmd/gcwserver`
- Or build a binary:
    - `go build -o bin\gcwserver.exe ./cmd/gcwserver`
    - `bin\gcwserver.exe`
- The server binds to :8080 by default. Override with `ADDR=:8080` or `PORT=8080`.

Issue a token (dev mode)
- Dev is the default auth mode. Request a token and use it as a Bearer token:
    - `curl -s -X POST http://localhost:8080/api/auth/token -H "Content-Type: application/json" -d "{\"email\":\"dev@example.com\",\"ttl_seconds\":3600}"`
- Response: `{ "token": "...", "subject": "dev@example.com", "expires_at": "..." }`

Connect from the desktop app
- Enable the feature flag and run the app:
    - PowerShell: `$env:GCW_ENABLE_SERVER='1'; go run -tags fyne ./cmd/gocomicwriter`
- In the app: Server → Connect to Server…
    - Base URL: `http://localhost:8080`
    - Token: paste the token obtained from /api/auth/token
- The integration is read-only: lists projects and shows an index snapshot; search is routed to the backend. Your local comic.json remains the source of truth.

Health and version endpoints
- `GET /healthz` — liveness; responds with status and version
- `GET /readyz` — readiness; verifies DB connectivity (and, optionally, object storage health)
- `GET /version` — plain-text version

API overview (subject to change)
- `POST /api/auth/token` — returns `{ token, expires_at }`. In `static` auth mode this requires an admin API key header `X-API-Key: <GCW_ADMIN_API_KEY>` and the subject must exist.
- `GET /api/projects` — list projects (Authorization: Bearer <token>)
- `GET /api/projects/{id}/index` — latest index snapshot envelope
- `GET /api/projects/{id}/search?text=&character=&scene=&tags=a,b&types=script,panel&page_from=1&page_to=10&limit=100&offset=0` — search
- `POST /api/projects/{id}/sync/push` — push ops (prototype, no conflict resolution)
- `GET /api/projects/{id}/sync/pull?since=0&limit=500` — pull ops

Key environment variables (see .env.example)
- Database: `GCW_PG_DSN` (preferred) or `DATABASE_URL`.
- Network: `ADDR` or `PORT`.
- TLS (optional): `GCW_TLS_ENABLE`, `GCW_TLS_CERT_FILE`, `GCW_TLS_KEY_FILE`.
- Auth: `GCW_AUTH_MODE` (dev|static), `GCW_AUTH_SECRET`, `GCW_ADMIN_API_KEY`.
- Object storage health (optional): `GCW_MINIO_ENDPOINT` or `GCW_OBJECT_HEALTH_URL`, `GCW_OBJECT_HEALTH_REQUIRED`.

Notes
- docker-compose.yml includes an example Postgres 17 service and an optional MinIO service. A containerized gcwserver service is provided as commented instructions within the file; adapt if you want to run the server inside Docker.
- For production, configure TLS and switch `GCW_AUTH_MODE` to `static`, provision users, and protect token issuance with the admin API key.


## Repository layout
Top‑level and key packages:
- cmd/gocomicwriter — UI entrypoint/launcher. Build with `-tags fyne` to include the desktop UI.
- cmd/gcwserver — backend server entrypoint (thin HTTP API over PostgreSQL).
- internal/ — core libraries:
  - domain — core data model types (Project, Issue, Page, Panel, Balloon, etc.); mirrors fields in docs/comic.schema.json.
  - storage — project I/O (init/open/save), transactional writes, timestamped backups, autosave snapshot; see doc.go and project.go.
  - backend — thin backend service and client:
    - db.go — HTTP handlers, config/env, migrations loader, auth, sync routes.
    - migrations/*.sql — PostgreSQL schema and migrations.
    - search_pg.go — Postgres-backed search implementation.
    - client.go — minimal Go client used by the desktop app under a feature flag.
  - log — slog setup and env configuration (GCW_LOG_*), optional rotating file handler.
  - crash — panic recovery and crash reports written to backups/.
  - version — version string helper used by the app.
  - vector — vector primitives and scene graph used by the editor: geometry.go (Pt/Rect/Affine2D), node.go (Rect/Ellipse/RoundedRect/Path/Group with transforms and hit testing), path.go (path ops), style.go (Fill/Stroke).
  - textlayout — initial text layout abstractions to support typography and balloons later.
  - ui — desktop UI shell (experimental):
    - app_fyne.go — real editor window using Fyne; build tags: `fyne && cgo`.
    - app_fyne_nocgo.go — helpful fallback when `fyne` is set but `cgo` is disabled.
    - app_stub.go — stub used when the binary is built without `-tags fyne`.
- docs/ — concept and schema:
  - go_comic_writer_concept.md — product concept, pillars, architecture, milestones.
  - comic.schema.json — JSON schema for comic.json projects.
  - ci-cd-aws.md — pipeline and release guide; aws-codepipeline.yml — CloudFormation template.
- bin/ — local build output (e.g., gocomicwriter, gocomicwriter-ui); not published.
- Dev helpers:
  - docker-compose.yml — local PostgreSQL 17 and optional MinIO.
  - .env.example — example server configuration.
- Root files:
  - README.md, CHANGELOG.md, LICENSE, CODE_OF_CONDUCT.md, go.mod, go.sum.

## Tests
- Run all tests: `go test ./...`
- With coverage: `go test ./... -coverprofile=cover.out` then `go tool cover -html=cover.out`
- Race detector (recommended): `go test -race ./...`
- Backend integration tests: require PostgreSQL 17+ and a DSN via `GCW_PG_DSN` (preferred) or `DATABASE_URL`. The tests will connect to this database and run migrations.
  - Example (PowerShell): `$env:GCW_PG_DSN='postgres://postgres:postgres@localhost:5432/gocomicwriter?sslmode=disable'; go test ./internal/backend -run E2E`
  - Or start Postgres via Docker Compose: `docker compose up -d postgres`

## Roadmap and concept
The full product concept, architecture overview, and milestone plan are maintained here:
- docs/go_comic_writer_concept.md

Highlights:
- Script editor with beat tagging and character/location bible
- Page & panel planner with grids and coverage tracking
- Lettering tools (balloons, tails, SFX) with pro typography
- Deterministic exporters: PDF, PNG/SVG, CBZ, EPUB

## Cross-platform builds and installers

This repository includes a GoReleaser configuration (.goreleaser.yaml) and helper scripts to produce cross-platform builds and basic installers for both applications:
- gocomicwriter — the desktop UI (Fyne). Built with CGO enabled and the build tag `fyne`.
- gcwserver — the thin backend server. Built without CGO by default.

Artifacts
- Windows: ZIP archives with .exe binaries.
- macOS: tar.gz/ZIP archives with binaries. App bundle packaging is planned; see docs/go_comic_writer_concept.md (Phase 8).
- Linux: tar.gz archives and .deb/.rpm packages (nfpm).

Where outputs go
- dist/ contains all build artifacts (archives, checksums, linux packages).

How to build locally (snapshot, no publish)
- PowerShell (Windows):
  - ./scripts/build_release.ps1
- Bash (Linux/macOS):
  - ./scripts/build_release.sh

Full release (publishing) requires tagging and additional setup; see GoReleaser docs. You can still run a local full pipeline with:
- PowerShell: ./scripts/build_release.ps1 -Release
- Bash: ./scripts/build_release.sh release

Prerequisites for GUI builds (gocomicwriter)
- Windows: Install MSYS2/MinGW-w64 and ensure gcc is on PATH (CGO). See https://www.msys2.org/ and verify `gcc --version` works in your shell.
- macOS: Install Xcode Command Line Tools (`xcode-select --install`).
- Linux: Install a C toolchain and GL/X11 headers. On Debian/Ubuntu:
  - sudo apt-get update && sudo apt-get install -y build-essential xorg-dev

Installers (Linux packages)
- After a build, install with:
  - Debian/Ubuntu: sudo dpkg -i dist/gocomicwriter_*.deb
  - RHEL/Fedora: sudo rpm -i dist/gocomicwriter_*.rpm
- These packages install both gocomicwriter and gcwserver into /usr/bin.

Versioning
- internal/version.Version is now settable at build-time via ldflags and will be populated by GoReleaser from the release version or git tag.

See also
- docs/go_comic_writer_concept.md → Phase 8 — Packaging & Distribution.

## CI/CD and releases

AWS CI/CD (CodePipeline):
- A ready-to-deploy AWS pipeline is provided as a CloudFormation template at docs/aws-codepipeline.yml. It creates:
  - A versioned, encrypted S3 bucket for pipeline artifacts (retained on stack deletion)
  - Least-privilege IAM roles for CodePipeline and CodeBuild
  - A CodeBuild project (Go 1.24; vet + formatting check; builds linux/amd64 and windows/amd64; optional S3 publish)
  - A CodePipeline with two stages: Source (GitHub via CodeStar Connections) and Build (CodeBuild)
- Full setup and operations guide: docs/ci-cd-aws.md

Quick deploy (PowerShell, run from the repo root):

```powershell
aws cloudformation deploy `
  --region eu-west-1 `
  --stack-name gocomicwriter-pipeline `
  --template-file docs/aws-codepipeline.yml `
  --capabilities CAPABILITY_NAMED_IAM `
  --parameter-overrides `
    ProjectName=gocomicwriter `
    PipelineName=gocomicwriter-pipeline `
    GitHubConnectionArn=arn:aws:codestar-connections:eu-west-1:<ACCOUNT_ID>:connection/<ID> `
    GitHubOwner=<GITHUB_OWNER> `
    GitHubRepo=gocomicwriter `
    BranchName=<branch> `
    ReleaseBucketName=<optional-s3-bucket-or-empty>
```

Notes:
- You must have a CodeStar Connection to GitHub in eu-west-1 and pass its ARN as GitHubConnectionArn.
- To publish artifacts to S3, create a bucket and pass its name as ReleaseBucketName; otherwise S3 upload is skipped.
- If your build entry point differs from the template defaults, see “Adjusting build paths/names” in docs/ci-cd-aws.md.
- Toolchain version: the template’s CodeBuild buildspec uses Go 1.24, aligned with this repository (see go.mod).

GitHub Actions (TODO):
- No GitHub Actions workflows are currently included in this repository. For an example release workflow and setup instructions, see docs/ci-cd-aws.md (section "GitHub Actions-based releases"). If you add workflows later, update this README accordingly.

## Contributing and conduct
Contributions are welcome during the Beta. Please focus on issues, docs, and UX feedback first; larger feature work may be deferred until after Beta.
- File issues with feedback or questions
- Discuss the data model and schema
- Propose improvements to docs and developer experience

Please review the Code of Conduct before participating:
- CODE_OF_CONDUCT.md

## License
Apache License 2.0 — see LICENSE for details.


## Tutorials & templates

- Read online: docs/tutorials_and_templates.md
- Build a PDF locally (Windows, requires Pandoc):
  - PowerShell: `scripts\build_docs.ps1`
  - Output: `docs\gocomicwriter_tutorials_templates.pdf`

If Pandoc is not installed, the script will print a helpful message and exit with a non-zero code. Install Pandoc from https://pandoc.org/installing.html.
