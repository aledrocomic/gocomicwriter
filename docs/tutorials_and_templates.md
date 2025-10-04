---
title: Go Comic Writer — Tutorials and Templates
author: Go Comic Writer Team
date: 2025-10-02
subtitle: Practical guide to projects, scripts, pages, lettering, search, export, and re‑usable templates
---

Introduction

This guide complements docs/go_comic_writer_concept.md. It covers hands‑on workflows and provides ready‑to‑copy templates you can adapt for your projects. A build script is provided to generate this guide as a PDF.

Audience
- Comic writers, writer–letterers, and small teams adopting Go Comic Writer.
- Contributors who need practical examples aligned with the concept and schema.

How to read this document
- Tutorials first, then templates. Copy code blocks into your project and tweak.
- See docs/comic.schema.json for the authoritative field reference.

1. Quick Start Tutorial

Prerequisites
- Go 1.24+
- Optional (for UI): build with `-tags fyne` and a working C toolchain.

Create a project
1) Create a new folder for your project.
2) Create a minimal manifest at project-root/comic.json (see Templates → Minimal comic.json skeleton).
3) Add assets (fonts/images) under project-root/assets.
4) (Optional) Add script files under project-root/script.

Open in Go Comic Writer (UI build)
- Launch the app, choose File → Open, and select your project folder containing comic.json.
- Use File → New Project to scaffold from one of the starter templates if available.

Add an issue
- In the UI: Project → New Issue… set trim, bleed, DPI, reading direction.
- Or manual: Add an issue entry to comic.json and run the app to validate.

Plan pages and panels
- Choose a grid preset (e.g., 3×3) to scaffold page layout.
- Add panels, then link beats from your script to panels for coverage.

Lettering basics
- Add balloons (speech/whisper/thought) and captions.
- Set speaker, style, and text. Use stylesheets for consistent typography.

Export
- From the UI: Export → PDF (issue) or PNG/SVG/CBZ/EPUB.
- Respect trim/bleed settings. See Templates → Export profiles for presets.

2. Script → Pages Workflow

- Write scenes and beats in script files under script/ using a simple, Fountain‑like style (example template provided).
- Tag beats with IDs (e.g., BEAT: ULID) so you can link them to panels in the UI.
- Maintain a character/location bible for auto‑complete and search filters.
- In the UI, link beats to panels to track coverage; unmapped beats are highlighted.

Script conventions (lightweight)
- Scene headings: INT./EXT. LOCATION — TIME
- Beats: `[[beat: <ID>]]` as an inline marker or heading.
- Dialogue: CHARACTER: line… Additional lines indented.
- Captions/SFX: `CAP:` and `SFX:` prefixes.

3. Page Grids and Panels

Choosing a grid
- Common presets: 3×3, 2×3, 4×2, or custom. Start with a preset, then tweak.
- Grids help pacing and page turns. Reserve bottom‑right for reveals when LTR.

Panels
- Each panel has geometry, z‑order, and optional notes.
- Keep stable IDs; don’t reuse after deletion to keep diffs clean.

Beat coverage
- Link beats to panels. The app visualizes unmapped beats and page‑turn hints.

4. Lettering and Styles

Stylesheets
- Define text styles (font family, size, leading, tracking, fill/stroke colors).
- Use style packs to reuse across projects.

Balloons and tails
- Shapes: ellipse, rounded‑rect; tails snap to speakers.
- Keep contrast and margins legible at print size.

SFX and text on path
- Use outlines, strokes, and path‑aligned text for dynamic effects.

5. Search and Cross‑References

- Global full‑text search across script, captions, SFX, and bible entries.
- Filters: character, scene, page range, tags.
- “Where‑used” lookups via cross‑references (e.g., where a character appears).

6. Backend (Optional, Early Access)

- A thin Go backend with PostgreSQL provides organization‑wide search and listing.
- The local comic.json remains the source of truth; sync is opt‑in and minimal.

7. Exports

- PDF: multi‑page issue export honoring trim/bleed and font embedding.
- PNG/SVG: per‑page exports for web/social or vector workflows.
- CBZ: packaged images.
- EPUB (fixed layout): digital reading with metadata.

Tips
- Keep assets external (assets/). Don’t embed binaries in JSON.
- Rebuild the local index if search seems stale; the index is disposable.
- Use stable IDs and canonical ordering to keep diffs small.

Templates

A. Minimal comic.json skeleton

Save as comic.json at the project root. Adjust IDs to ULID/UUIDv7. This aligns with docs/go_comic_writer_concept.md and docs/comic.schema.json.

{
  "project": {
    "id": "01JABCDTESTID",
    "name": "My Comic",
    "version": 1
  },
  "issues": [
    {
      "id": "01JABCDISSUE01",
      "number": 1,
      "title": "Issue #1",
      "readingDirection": "LTR",
      "trim": { "width": 210, "height": 297, "unit": "mm" },
      "bleed": { "all": 3, "unit": "mm" },
      "dpi": 300,
      "pages": [
        {
          "id": "01JABCDPAGE01",
          "number": 1,
          "grid": "3x3",
          "panels": [
            { "id": "01JABCDPAN001", "zOrder": 1, "notes": "Establishing shot" }
          ],
          "balloons": [],
          "sfx": []
        }
      ]
    }
  ],
  "styles": {
    "textStyles": [
      {
        "id": "01JABCDTXT001",
        "name": "Dialogue",
        "font": { "family": "CCWildwords", "size": 9, "leading": 11 },
        "fill": "#000000"
      }
    ]
  },
  "fonts": [
    { "family": "CCWildwords", "path": "assets/fonts/CCWildwords-Regular.otf" }
  ],
  "bible": { "characters": [], "locations": [], "tags": [] }
}

B. Page grid presets (inline examples)

- 3×3: "grid": "3x3"
- 2×3: "grid": "2x3"
- Custom: omit grid and specify explicit panel geometry in pages[].panels[].

C. Style pack skeleton (reusable)

styles/
  pack.json
  fonts/
  balloons/

Example pack.json
{
  "id": "01JABCDPACK01",
  "name": "Clean Dialogue + SFX",
  "version": 1,
  "textStyles": [
    { "id": "01JABCDTXT001", "name": "Dialogue", "font": { "family": "CCWildwords", "size": 9, "leading": 11 }, "fill": "#000" },
    { "id": "01JABCDTXT002", "name": "Caption",  "font": { "family": "CCWildwords", "size": 8.5, "leading": 10.5 }, "fill": "#111" },
    { "id": "01JABCDTXT003", "name": "SFX",      "font": { "family": "Badaboom",    "size": 18,  "tracking": 20 }, "fill": "#F30", "stroke": { "color": "#000", "width": 1 } }
  ],
  "balloonStyles": [
    { "id": "01JABAL001", "name": "Ellipse", "shape": "ellipse", "fill": "#FFF", "stroke": { "color": "#000", "width": 1 } }
  ]
}

D. Script template (lightweight)

# ISSUE 1 — DRAFT A

INT. LAB — NIGHT
[[beat: 01JBEATAAAA01]]
CAP: The city sleeps while the lab hums.

EXT. ROOFTOP — NIGHT
[[beat: 01JBEATAAAA02]]
HERO: We move at dawn.
SIDEKICK: Dawn’s late.
SFX: WHOOSH

E. Export profiles (conceptual)

- Web: PNG, 144 DPI, sRGB, 1080px width, no bleed.
- Print: PDF, 300 DPI, CMYK or tagged RGB, include trim/bleed.
- Social: Square crops (PNG), large type, high contrast.

Building this guide as PDF

A PowerShell script is provided to build a PDF locally using Pandoc.

Requirements
- Windows PowerShell 5+ or PowerShell 7+
- Pandoc installed and available on PATH: https://pandoc.org/installing.html

Command

# From repository root
scripts\build_docs.ps1

Output
- docs\gocomicwriter_tutorials_templates.pdf

Notes
- If Pandoc is missing, the script prints instructions and returns a non‑zero code.
- Fonts in code samples are illustrative; ensure you have appropriate licenses.

Further reading
- docs/go_comic_writer_concept.md — Product concept and plan
- docs/comic.schema.json — Manifest schema
- README.md — Building, running, and project layout


8. Storyboard and Colorization Tabs (Quick Start)

Storyboard basics
1) Open your project and switch to the Storyboard tab.
2) Use the Page dropdown to choose a page. The left list shows panels in z-order with their IDs and any note preview.
3) Click a panel to view details. In Notes, type your storyboard notes and click "Save Notes". Notes are saved to the panel in comic.json.
4) To link script beats: in Unmapped Beats, select a beat ID, then select a panel in the left list and click "Map Selected Beat to Panel". The linked beat IDs appear in the panel details.

Colorization basics
1) On the Canvas tab, click to select a shape (rectangle/ellipse/path/etc.).
2) Switch to the Colorize tab.
3) Adjust R, G, B, A sliders and (optionally) Stroke width. Toggle Fill Enabled / Stroke Enabled as needed.
4) Click "Apply Fill to Selected" and/or "Apply Stroke to Selected" to update the selected shape.
5) Optionally, use "Pick From Selected" to load the selected shape's current style back into the controls.
6) File → Save to persist changes in your project.

See also
- Developer details and limitations: docs/developer-guide.md#storyboard-tab and docs/developer-guide.md#colorization-tab
- Concept and roadmap context: docs/go_comic_writer_concept.md
