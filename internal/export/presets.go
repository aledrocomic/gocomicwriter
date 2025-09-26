/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * Licensed under the Apache License, Version 2.0
 */

package export

import (
	"fmt"
	"path/filepath"
	"strings"

	"gocomicwriter/internal/storage"
)

// PresetName represents a named export preset.
type PresetName string

const (
	PresetWeb   PresetName = "web"
	PresetPrint PresetName = "print"
)

// BatchOptions controls batch export across multiple formats/issues/pages.
// Minimal surface intended to satisfy concept doc: export presets (web, print) and batch export.
//
// Path semantics:
//   - If OutDir is empty or relative, it will be created under <project>/exports/<preset>/.
//   - For PDF/CBZ single-file outputs, names will be issue-<n>.pdf/cbz in OutDir.
//   - For PNG/SVG per-page outputs, files are issue-<n>-page-<m>.(png|svg) in subfolders png/ or svg/ inside OutDir.
//     This keeps assets grouped by preset and format.
//
// Pages applies to per-page exporters; PDF ignores Pages for simplicity and exports all pages (can be refined later).
//
//nolint:revive // keep fields explicit for clarity
type BatchOptions struct {
	Preset        PresetName
	Formats       []string // allowed: pdf, png, svg, cbz; empty means preset defaults
	Issues        []int    // zero-based indices; empty means all issues
	Pages         []int    // zero-based indices; empty means all pages
	DPIOverride   int      // when > 0 overrides raster/vector viewport DPI where applicable
	IncludeGuides *bool    // when set, overrides preset's default for guides
	OutDir        string   // base directory for outputs (created per preset if relative)
}

// BatchExport runs exports according to the given preset.
func BatchExport(ph *storage.ProjectHandle, opt BatchOptions) error {
	if ph == nil {
		return fmt.Errorf("project handle is nil")
	}
	if len(ph.Project.Issues) == 0 {
		return fmt.Errorf("project has no issues")
	}

	formats := opt.Formats
	if len(formats) == 0 {
		formats = presetDefaultFormats(opt.Preset)
	}
	// normalize format strings
	for i := range formats {
		formats[i] = strings.ToLower(strings.TrimSpace(formats[i]))
	}

	// Resolve output base directory
	baseOut := opt.OutDir
	if baseOut == "" {
		baseOut = string(opt.Preset)
	}
	if !filepath.IsAbs(baseOut) {
		baseOut = filepath.Join(ph.Root, "exports", baseOut)
	}

	// Resolve issues list
	issues := opt.Issues
	if len(issues) == 0 {
		issues = make([]int, len(ph.Project.Issues))
		for i := range issues {
			issues[i] = i
		}
	}

	// Helpers to compute IncludeGuides default
	guides := presetIncludeGuides(opt.Preset)
	if opt.IncludeGuides != nil {
		guides = *opt.IncludeGuides
	}

	for _, issueIdx := range issues {
		if issueIdx < 0 || issueIdx >= len(ph.Project.Issues) {
			continue
		}

		for _, f := range formats {
			switch f {
			case "pdf":
				// Single file per issue
				out := filepath.Join(baseOut, "pdf", fmt.Sprintf("issue-%d.pdf", issueIdx+1))
				po := PDFOptions{IncludeGuides: guides, Pages: nil}
				if err := ExportIssuePDF(ph, issueIdx, out, po); err != nil {
					return fmt.Errorf("pdf issue %d: %w", issueIdx+1, err)
				}
			case "cbz":
				out := filepath.Join(baseOut, "cbz", fmt.Sprintf("issue-%d.cbz", issueIdx+1))
				co := CBZOptions{IncludeGuides: guides}
				if opt.DPIOverride > 0 {
					co.DPI = opt.DPIOverride
				}
				if err := ExportIssueCBZ(ph, issueIdx, out, co); err != nil {
					return fmt.Errorf("cbz issue %d: %w", issueIdx+1, err)
				}
			case "png":
				outDir := filepath.Join(baseOut, "png")
				po := PNGOptions{IncludeGuides: guides, Pages: opt.Pages}
				if opt.DPIOverride > 0 {
					po.DPI = opt.DPIOverride
				}
				if err := ExportIssuePNGPages(ph, issueIdx, outDir, po); err != nil {
					return fmt.Errorf("png issue %d: %w", issueIdx+1, err)
				}
			case "svg":
				outDir := filepath.Join(baseOut, "svg")
				so := SVGOptions{IncludeGuides: guides, Pages: opt.Pages}
				if opt.DPIOverride > 0 {
					so.DPI = opt.DPIOverride
				}
				if err := ExportIssueSVGPages(ph, issueIdx, outDir, so); err != nil {
					return fmt.Errorf("svg issue %d: %w", issueIdx+1, err)
				}
			default:
				return fmt.Errorf("unknown format: %s", f)
			}
		}
	}
	return nil
}

func presetDefaultFormats(p PresetName) []string {
	switch p {
	case PresetWeb:
		return []string{"png", "svg", "cbz"}
	case PresetPrint:
		return []string{"pdf", "png"}
	default:
		return []string{"pdf"}
	}
}

func presetIncludeGuides(p PresetName) bool {
	switch p {
	case PresetWeb:
		return false
	case PresetPrint:
		return true
	default:
		return true
	}
}
