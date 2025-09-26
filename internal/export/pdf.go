/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package export

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

// PDFOptions controls PDF export behavior.
// Units are points (pt) unless otherwise noted.
// Vector text is used whenever possible; we rely on built-in Helvetica for portability.
// In later phases, font embedding can be added using TTFs.
//
// Coordinates:
// - Page origin is top-left.
// - All Rect values are assumed to be in page coordinates.
// - Bleed is applied as an outer margin beyond trim.
//
// Boxes:
// - MediaBox = trim + 2*bleed (full page size in PDF)
// - TrimBox drawn as a guide rectangle (non-printing intent is not encoded; we draw hairlines)
// - BleedBox drawn as outermost guide
//
// NOTE: This is a minimal first exporter honoring the concept doc; it will evolve.
//
//nolint:revive // keep options grouped and explicit for clarity
type PDFOptions struct {
	IncludeGuides bool
	EmbedFonts    bool // reserved; not used yet
	GuideColor    domain.Color
	PanelStroke   domain.Stroke
	BalloonStroke domain.Stroke
	BalloonFill   domain.Color
	Pages         []int // if empty, export all pages
}

// ExportIssuePDF exports the specified issue to a single multi-page PDF placed at outPath.
func ExportIssuePDF(ph *storage.ProjectHandle, issueIndex int, outPath string, opt PDFOptions) error {
	if ph == nil {
		return fmt.Errorf("project handle is nil")
	}
	if issueIndex < 0 || issueIndex >= len(ph.Project.Issues) {
		return fmt.Errorf("issue index out of range")
	}
	iss := ph.Project.Issues[issueIndex]

	// Default styles
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

	trimW := iss.TrimWidth
	trimH := iss.TrimHeight
	bleed := iss.Bleed
	mediaW := trimW + 2*bleed
	mediaH := trimH + 2*bleed

	// Use points for 1:1 mapping from model to PDF
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "pt",
		Size:    gofpdf.SizeType{Wd: mediaW, Ht: mediaH},
		// We'll set orientation automatically by size
		OrientationStr: "",
	})
	pdf.SetTitle(fmt.Sprintf("%s — Issue PDF", ph.Project.Name), false)
	pdf.SetAuthor("Go Comic Writer", false)

	// Built-in Helvetica keeps text vector without embedding
	pdf.SetFont("Helvetica", "", 12)

	pages := pageIndexes(len(iss.Pages), opt.Pages)
	for _, pidx := range pages {
		if pidx < 0 || pidx >= len(iss.Pages) {
			continue
		}
		pg := iss.Pages[pidx]
		pdf.AddPageFormat("", gofpdf.SizeType{Wd: mediaW, Ht: mediaH})

		// Draw bleed and trim guides if requested
		if opt.IncludeGuides {
			setDrawColor(pdf, guideCol)
			pdf.SetLineWidth(0.2)
			// Bleed (outer border = media box)
			pdf.Rect(0, 0, mediaW, mediaH, "D")
			// Trim box
			pdf.Rect(bleed, bleed, trimW, trimH, "D")
		}

		// Panels
		setDrawColor(pdf, panelStroke.Color)
		pdf.SetLineWidth(panelStroke.Width)
		for _, pnl := range pg.Panels {
			r := pnl.Geometry
			// Shift by bleed to map to media coordinates
			x := r.X + bleed
			y := r.Y + bleed
			pdf.Rect(x, y, r.Width, r.Height, "D")

			// Balloons within panel (coordinates assumed absolute already)
			for _, b := range pnl.Balloons {
				br := b.Shape.Rect
				bx := br.X + bleed
				by := br.Y + bleed
				// Shape
				setFillColor(pdf, balloonFill)
				setDrawColor(pdf, balloonStroke.Color)
				pdf.SetLineWidth(balloonStroke.Width)
				switch b.Shape.Kind {
				case "ellipse":
					pdf.Ellipse(bx+br.Width/2, by+br.Height/2, br.Width/2, br.Height/2, 0, "FD")
				case "roundedBox":
					r := b.Shape.Radius
					roundedRect(pdf, bx, by, br.Width, br.Height, r, "FD")
				default:
					pdf.Rect(bx, by, br.Width, br.Height, "FD")
				}
				// Text (simple top-left flow)
				pad := 6.0
				cx := bx + pad
				cy := by + pad + 12 // approx baseline offset for 12pt
				for _, run := range b.TextRuns {
					fsz := run.Size
					if fsz <= 0 {
						fsz = 12
					}
					pdf.SetFont("Helvetica", "", fsz)
					pdf.Text(cx, cy, run.Content)
					cy += fsz * 1.2
				}
			}
		}
	}

	// Ensure output path is under project exports folder if relative
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(ph.Root, "exports", outPath)
	}
	// Ensure directory exists
	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure out dir: %w", err)
	}
	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}
	return nil
}

func pageIndexes(total int, specific []int) []int {
	if len(specific) == 0 {
		out := make([]int, total)
		for i := range out {
			out[i] = i
		}
		return out
	}
	return specific
}

func setDrawColor(pdf *gofpdf.Fpdf, c domain.Color) {
	pdf.SetDrawColor(int(c.R), int(c.G), int(c.B))
}

func setFillColor(pdf *gofpdf.Fpdf, c domain.Color) {
	pdf.SetFillColor(int(c.R), int(c.G), int(c.B))
}

func roundedRect(pdf *gofpdf.Fpdf, x, y, w, h, r float64, style string) {
	// parameter r reserved for future rounded corners; mark as used to avoid warnings
	_ = r
	// Fallback for older gofpdf versions: draw as a normal rectangle
	pdf.Rect(x, y, w, h, style)
}
