# Go Comic Writer (gocomicwriter)

A Go-powered project aiming to become a writing, planning, and lettering toolchain for comics — from script to precisely lettered pages — with reliable exports for print and screen.

This repository currently provides a development skeleton: a minimal CLI, an evolving domain model, and a public JSON schema for the project manifest. The product concept and roadmap live in docs/go_comic_writer_concept.md.

- Vision: Empower comic creators to go from script to lettered pages in one streamlined, offline‑first tool.
- Status: Early stage (0.1.2‑dev). Not production‑ready.
- License: Apache 2.0

## Contents
- What is this? (short overview)
- Current features and what’s next
- Install and quick start
- Usage
- Project manifest (comic.json) and schema
- Repository layout
- Roadmap and concept
- CI/CD and releases
- Contributing and conduct
- License

## What is this?
Go Comic Writer is an in‑progress toolchain for comic writing and lettering. It’s designed to be:
- Writing‑first
- Page‑aware
- Precise for typography/balloons/SFX
- Deterministic in rendering and export
- Cross‑platform and offline‑first

The long‑term plan is a desktop application with a canvas editor and exporters (PDF/PNG/SVG/CBZ). See the concept document for details.

## Current features (alpha)
- CLI commands: version, init, open, save, ui.
- Transactional project storage with a human‑readable manifest (comic.json) and timestamped backups under backups/.
- Crash safety: on panic, write a crash report and autosave snapshot; on open, fall back to the latest valid backup if the manifest is unreadable.
- Structured logging via Go's slog with simple env configuration; optional rotating file via GCW_LOG_FILE.
- Core domain model in internal/domain and a public JSON schema at docs/comic.schema.json.
- Basic desktop UI shell (behind build tag `fyne`) with a placeholder canvas editor that shows page/trim/bleed guides, simple pan/zoom, and File→Open/Save.
- Sample project manifest at tmp_proj/comic.json (with backups under tmp_proj/backups/).
- Unit tests for core packages (storage, logging, crash, version, schema).

What’s not here yet:
- Rendering/lettering engine (beyond the placeholder canvas) and exporters.
- Exporters (PDF/PNG/SVG/CBZ).

## Install and quick start
Prerequisites:
- Go 1.23 or newer
- A supported OS (Windows/macOS/Linux)

Install the CLI from source:

```bash
# From within a clone of this repository
go build -o bin/gocomicwriter ./cmd/gocomicwriter

# Or install into your GOPATH/bin (adjust module path if needed)
go install ./cmd/gocomicwriter
```

Verify it runs:

```bash
bin/gocomicwriter
bin/gocomicwriter -v
bin/gocomicwriter --version
```

Expected output resembles:

```
Go Comic Writer — development skeleton
Version: 0.1.2-dev
```

## Usage
Current CLI commands:

```
gocomicwriter version | -v | --version   Show version
gocomicwriter init <dir> <name>           Create a new project at <dir> with name <name>
gocomicwriter open <dir>                  Open project at <dir> and print summary
gocomicwriter save <dir>                  Save project at <dir> (creates backup)
gocomicwriter ui [<dir>]                  Launch desktop UI (build with -tags fyne)
```

Examples:
- Build locally, then run (PowerShell):
  - .\bin\gocomicwriter.exe -v
  - .\bin\gocomicwriter.exe init .\tmp_proj "My Series"
  - .\bin\gocomicwriter.exe open .\tmp_proj
  - .\bin\gocomicwriter.exe save .\tmp_proj
- Or if installed into PATH: gocomicwriter -v

### Run the basic UI (experimental)
The repository includes a minimal desktop UI shell guarded by the `fyne` build tag.

Build and run directly (no binary):

```bash
# Open the sample project (Windows PowerShell)
go run -tags fyne ./cmd/gocomicwriter ui .\tmp_proj

# On macOS/Linux
GOFLAGS='' go run -tags fyne ./cmd/gocomicwriter ui ./tmp_proj
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
bin/gocomicwriter-ui ui ./tmp_proj
```

Notes and controls:
- File → Open to select a project folder; File → Save writes comic.json (with a timestamped backup of the previous manifest).
- Center canvas shows a page rectangle with bleed (blue) and trim (red) guides.
- Drag with mouse to pan.
- Ctrl + Mouse Wheel to zoom in/out.
- Window title shows the project name when opened.

Troubleshooting:
- Linux may require OpenGL drivers and a working X11/Wayland setup. On headless CI this UI is not built by default.
- If you see "UI not built in this binary", rebuild with `-tags fyne`.
- If you see a build error like: `imports github.com/go-gl/gl/v2.1/gl: build constraints exclude all Go files`, it means cgo is disabled and the OpenGL backend cannot compile.
  - On Windows, install a C toolchain (MSYS2/MinGW-w64) so gcc is available, then enable cgo:
    - Start an MSYS2 MinGW64 shell or ensure `gcc` is on PATH in PowerShell.
    - PowerShell: `setx CGO_ENABLED 1` (or for the current session: `$env:CGO_ENABLED='1'`)
    - Then: `go run -tags fyne ./cmd/gocomicwriter ui .\tmp_proj`
  - On macOS: Xcode Command Line Tools are usually sufficient. Ensure `CGO_ENABLED=1`.
  - On Linux: install build-essential (Debian/Ubuntu) or base-devel (Arch), ensure `CGO_ENABLED=1`.
- If cgo is still disabled, the binary will fall back to a helpful stub error when running the `ui` command with `-tags fyne`.

Notes:
- init scaffolds standard subfolders (script, pages, assets, styles, exports, backups) and writes comic.json.
- save writes comic.json transactionally and copies the previous manifest into backups/comic.json.YYYYMMDD-HHMMSS.bak.
- open attempts to read comic.json; if it fails, it falls back to the latest backup.

## Logging configuration
The CLI uses structured logging (slog). Configure via environment variables:
- GCW_LOG_LEVEL=debug|info|warn|error (default: info)
- GCW_LOG_FORMAT=console|json (default: console)
- GCW_LOG_SOURCE=true|false (default: false)
- GCW_LOG_FILE=<path> (optional; enables rotating JSON file logs)

Examples:
- PowerShell: `$env:GCW_LOG_LEVEL='debug'; .\bin\gocomicwriter.exe open .\tmp_proj`
- Bash: `GCW_LOG_FORMAT=json GCW_LOG_FILE=gcw.log gocomicwriter open tmp_proj`

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

Note: The schema defines richer structures for pages, panels, balloons, styles, etc. See docs/comic.schema.json for all fields.

A sample work‑in‑progress manifest lives at tmp_proj/comic.json (with automatic timestamped backups under tmp_proj/backups/).

## Repository layout
- cmd/gocomicwriter — CLI entrypoint
- internal/domain — core data model types (Project, Issue, Page, Panel, Balloon, etc.)
- internal/storage — project I/O (init/open/save), backups, autosave snapshot
- internal/crash — panic recovery and crash reports
- internal/log — slog setup, env options, optional rotating file handler
- internal/version — version string helper
- docs/go_comic_writer_concept.md — product concept, pillars, and milestones
- docs/comic.schema.json — JSON schema for comic.json
- .github/workflows/go.yml — CI checks (build, vet, test)
- tmp_proj/ — scratch space for local experiments and sample files

## Roadmap and concept
The full product concept, architecture overview, and milestone plan are maintained here:
- docs/go_comic_writer_concept.md

Highlights:
- Script editor with beat tagging and character/location bible
- Page & panel planner with grids and coverage tracking
- Lettering tools (balloons, tails, SFX) with pro typography
- Deterministic exporters: PDF, PNG/SVG, CBZ

## CI/CD and releases
- CI: .github/workflows/go.yml runs build, vet, and tests on pushes/PRs.
- Releases: not configured yet; will be added once usable milestones are reached.

## Contributing and conduct
Contributions are welcome once core foundations stabilize. Until then, feel free to:
- File issues with feedback or questions
- Discuss the data model and schema
- Propose improvements to docs and developer experience

Please review the Code of Conduct before participating:
- CODE_OF_CONDUCT.md

## License
Apache License 2.0 — see LICENSE for details.
