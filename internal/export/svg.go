/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package export

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

// SVGOptions controls SVG export behavior.
// - DPI defines the physical pixel size; width/height attributes use pixels derived from DPI.
// - The coordinate system matches the model (points). A viewBox is provided to scale.
//
//nolint:revive // clarity is preferred
type SVGOptions struct {
	IncludeGuides bool
	DPI           int
	GuideColor    domain.Color
	PanelStroke   domain.Stroke
	BalloonStroke domain.Stroke
	BalloonFill   domain.Color
	Pages         []int
}

// ExportIssueSVGPages exports each page of an issue as a separate SVG file.
// Files will be named issue-<issue+1>-page-<pageNumber>.svg under outDir or project's exports.
func ExportIssueSVGPages(ph *storage.ProjectHandle, issueIndex int, outDir string, opt SVGOptions) error {
	if ph == nil {
		return fmt.Errorf("project handle is nil")
	}
	if issueIndex < 0 || issueIndex >= len(ph.Project.Issues) {
		return fmt.Errorf("issue index out of range")
	}
	iss := ph.Project.Issues[issueIndex]

	// Defaults
	guideCol := opt.GuideColor
	if guideCol.A == 0 && guideCol.R == 0 && guideCol.G == 0 && guideCol.B == 0 {
		guideCol = domain.Color{R: 255, G: 0, B: 0, A: 255}
	}
	panelStroke := opt.PanelStroke
	if panelStroke.Width == 0 {
		panelStroke = domain.Stroke{Color: domain.Color{R: 0, G: 0, B: 0, A: 255}, Width: 1}
	}
	balloonStroke := opt.BalloonStroke
	if balloonStroke.Width == 0 {
		balloonStroke = domain.Stroke{Color: domain.Color{R: 0, G: 0, B: 0, A: 255}, Width: 1}
	}
	balloonFill := opt.BalloonFill
	if balloonFill.A == 0 && balloonFill.R == 0 && balloonFill.G == 0 && balloonFill.B == 0 {
		balloonFill = domain.Color{R: 255, G: 255, B: 255, A: 255}
	}
	// DPI
	dpi := iss.DPI
	if opt.DPI > 0 {
		dpi = opt.DPI
	}
	if dpi <= 0 {
		dpi = 300
	}

	trimW := iss.TrimWidth
	trimH := iss.TrimHeight
	bleed := iss.Bleed
	mediaW := trimW + 2*bleed
	mediaH := trimH + 2*bleed

	// Derived pixel size for width/height attributes
	scale := float64(dpi) / 72.0
	pxW := int(math.Round(mediaW * scale))
	pxH := int(math.Round(mediaH * scale))

	// Resolve output directory
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(ph.Root, "exports", outDir)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("ensure out dir: %w", err)
	}

	pages := pageIndexes(len(iss.Pages), opt.Pages)
	for _, pidx := range pages {
		if pidx < 0 || pidx >= len(iss.Pages) {
			continue
		}
		pg := iss.Pages[pidx]

		var buf bytes.Buffer
		var werr error
		wf := func(format string, args ...any) {
			if werr != nil {
				return
			}
			_, werr = fmt.Fprintf(&buf, format, args...)
		}

		wf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
		wf("<svg xmlns=\"http://www.w3.org/2000/svg\" version=\"1.1\" width=\"%dpx\" height=\"%dpx\" viewBox=\"0 0 %g %g\">\n", pxW, pxH, mediaW, mediaH)
		// Background white
		wf("  <rect x=\"0\" y=\"0\" width=\"%g\" height=\"%g\" fill=\"#ffffff\"/>\n", mediaW, mediaH)

		if opt.IncludeGuides {
			gc := svgColor(guideCol)
			wf("  <rect x=\"0\" y=\"0\" width=\"%g\" height=\"%g\" fill=\"none\" stroke=\"%s\" stroke-width=\"0.2\"/>\n", mediaW, mediaH, gc)
			wf("  <rect x=\"%g\" y=\"%g\" width=\"%g\" height=\"%g\" fill=\"none\" stroke=\"%s\" stroke-width=\"0.2\"/>\n", bleed, bleed, trimW, trimH, gc)
		}

		pc := svgColor(panelStroke.Color)
		bc := svgColor(balloonStroke.Color)
		bf := svgColor(balloonFill)

		for _, pnl := range pg.Panels {
			r := pnl.Geometry
			wf("  <rect x=\"%g\" y=\"%g\" width=\"%g\" height=\"%g\" fill=\"none\" stroke=\"%s\" stroke-width=\"%g\"/>\n", r.X+bleed, r.Y+bleed, r.Width, r.Height, pc, panelStroke.Width)
			for _, b := range pnl.Balloons {
				br := b.Shape.Rect
				x := br.X + bleed
				y := br.Y + bleed
				switch b.Shape.Kind {
				case "ellipse":
					cx := x + br.Width/2
					cy := y + br.Height/2
					rx := br.Width / 2
					ry := br.Height / 2
					wf("  <ellipse cx=\"%g\" cy=\"%g\" rx=\"%g\" ry=\"%g\" fill=\"%s\" stroke=\"%s\" stroke-width=\"%g\"/>\n", cx, cy, rx, ry, bf, bc, balloonStroke.Width)
				case "roundedBox":
					radius := b.Shape.Radius
					wf("  <rect x=\"%g\" y=\"%g\" width=\"%g\" height=\"%g\" rx=\"%g\" ry=\"%g\" fill=\"%s\" stroke=\"%s\" stroke-width=\"%g\"/>\n", x, y, br.Width, br.Height, radius, radius, bf, bc, balloonStroke.Width)
				default:
					wf("  <rect x=\"%g\" y=\"%g\" width=\"%g\" height=\"%g\" fill=\"%s\" stroke=\"%s\" stroke-width=\"%g\"/>\n", x, y, br.Width, br.Height, bf, bc, balloonStroke.Width)
				}
				// Text runs: simple top-left stacking
				pad := 6.0
				cx := x + pad
				cy := y + pad + 12
				for _, run := range b.TextRuns {
					fsz := run.Size
					if fsz <= 0 {
						fsz = 12
					}
					// We don't embed fonts here; the font family is a hint only.
					font := run.Font
					if font == "" {
						font = "Helvetica, Arial, sans-serif"
					}
					wf("  <text x=\"%g\" y=\"%g\" font-family=\"%s\" font-size=\"%g\" fill=\"#000\">%s</text>\n", cx, cy, escAttr(font), fsz, escText(run.Content))
					cy += fsz * 1.2
				}
			}
		}

		wf("</svg>\n")

		if werr != nil {
			return fmt.Errorf("build svg: %w", werr)
		}

		name := filepath.Join(outDir, fmt.Sprintf("issue-%d-page-%d.svg", issueIndex+1, pg.Number))
		if err := os.WriteFile(name, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write svg: %w", err)
		}
	}
	return nil
}

func svgColor(c domain.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func escAttr(s string) string {
	// naive escaping sufficient for our simple usage
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '"':
			out = append(out, '&', 'q', 'u', 'o', 't', ';')
		case '\n':
			out = append(out, ' ')
		case '\r':
			// skip
		default:
			out = append(out, ch)
		}
	}
	return string(out)
}

func escText(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '&':
			out = append(out, '&', 'a', 'm', 'p', ';')
		case '<':
			out = append(out, '&', 'l', 't', ';')
		case '>':
			out = append(out, '&', 'g', 't', ';')
		default:
			out = append(out, ch)
		}
	}
	return string(out)
}
