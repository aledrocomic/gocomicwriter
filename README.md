# Go Comic Writer (gocomic)

A Go-powered project aiming to become a writing, planning, and lettering toolchain for comics — from script to precisely lettered pages — with reliable exports for print and screen.

This repository currently provides a development skeleton: a minimal CLI, an evolving domain model, and a public JSON schema for the project manifest. The product concept and roadmap live in docs/go_comic_writer_concept.md.

- Vision: Empower comic creators to go from script to lettered pages in one streamlined, offline‑first tool.
- Status: Early stage (0.0.0‑dev). Not production‑ready.
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
- Minimal CLI with a version command.
- Versioned module structure for future expansion.
- Domain model types aligned with the planned data model (internal/domain).
- Public JSON schema for the project manifest at docs/comic.schema.json.
- Docs for setting up CI/CD and release automation.

What’s not here yet:
- Rendering/lettering engine and UI.
- Real project load/save flows.
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
Version: 0.0.0-dev
```

## Usage
The current CLI only supports printing its version:

```bash
gocomicwriter -v
```

Future commands will handle project creation, validation, rendering, and export.

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
- internal/version — version string helper
- docs/go_comic_writer_concept.md — product concept, pillars, and milestones
- docs/comic.schema.json — JSON schema for comic.json
- docs/ci-cd-aws.md — guide to CI, releases, and optional AWS artifact hosting
- .github/workflows/ci.yml — CI checks on pushes/PRs (build/vet/module hygiene)
- .github/workflows/release.yml — release builds on tags (cross‑platform binaries)
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
This repo includes GitHub Actions workflows for CI and releases, along with an AWS‑friendly guide if you want to mirror artifacts to S3 using OIDC:
- CI workflow: .github/workflows/ci.yml
- Release workflow: .github/workflows/release.yml
- Setup guide: docs/ci-cd-aws.md

How releases will work (when the project reaches usable milestones):
- Create a tag like v0.1.0
- CI builds cross‑platform artifacts and attaches them to the GitHub Release
- Optionally mirror artifacts to S3 (see the AWS guide)

## Contributing and conduct
Contributions are welcome once core foundations stabilize. Until then, feel free to:
- File issues with feedback or questions
- Discuss the data model and schema
- Propose improvements to docs and developer experience

Please review the Code of Conduct before participating:
- CODE_OF_CONDUCT.md

## License
Apache License 2.0 — see LICENSE for details.
