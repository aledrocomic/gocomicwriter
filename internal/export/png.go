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
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

// PNGOptions controls PNG export behavior.
// - DPI: when > 0 overrides issue DPI for output pixel size
// - IncludeGuides: draw trim/bleed hairlines similar to PDF
// - Pages: if empty, export all
// - Styles control colors and stroke widths; reasonable defaults are applied if zero values.
//
//nolint:revive // clarity is preferred
type PNGOptions struct {
	IncludeGuides bool
	DPI           int
	GuideColor    domain.Color
	PanelStroke   domain.Stroke
	BalloonStroke domain.Stroke
	BalloonFill   domain.Color
	Pages         []int
}

// ExportIssuePNGPages exports each page of an issue as a separate PNG file.
// Output files will be named issue-<issue+1>-page-<pageNumber>.png under the project's exports folder
// unless outDir is absolute or contains an explicit filename pattern with %d for page number.
func ExportIssuePNGPages(ph *storage.ProjectHandle, issueIndex int, outDir string, opt PNGOptions) error {
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

	// Calculate pixel dimensions from points (1pt = 1/72")
	scale := float64(dpi) / 72.0
	pixW := int(math.Round(mediaW * scale))
	pixH := int(math.Round(mediaH * scale))
	bx := int(math.Round(bleed * scale))
	by := int(math.Round(bleed * scale))

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

		img := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
		// Background white
		draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)

		// Guides
		if opt.IncludeGuides {
			gc := toRGBA(guideCol)
			strokeRect(img, 0, 0, pixW-1, pixH-1, gc)
			// trim box
			strokeRect(img, bx, by, int(math.Round(trimW*scale))+bx-1, int(math.Round(trimH*scale))+by-1, gc)
		}

		// Panels
		pc := toRGBA(panelStroke.Color)
		for _, pnl := range pg.Panels {
			r := pnl.Geometry
			x := int(math.Round((r.X + bleed) * scale))
			y := int(math.Round((r.Y + bleed) * scale))
			w := int(math.Round(r.Width * scale))
			h := int(math.Round(r.Height * scale))
			strokeRect(img, x, y, x+w-1, y+h-1, pc)

			// Balloons
			fc := toRGBA(balloonFill)
			bc := toRGBA(balloonStroke.Color)
			for _, b := range pnl.Balloons {
				br := b.Shape.Rect
				bxp := int(math.Round((br.X + bleed) * scale))
				byp := int(math.Round((br.Y + bleed) * scale))
				bw := int(math.Round(br.Width * scale))
				bh := int(math.Round(br.Height * scale))
				fillRect(img, bxp, byp, bxp+bw-1, byp+bh-1, fc)
				strokeRect(img, bxp, byp, bxp+bw-1, byp+bh-1, bc)
			}
		}

		name := filepath.Join(outDir, fmt.Sprintf("issue-%d-page-%d.png", issueIndex+1, pg.Number))
		f, err := os.Create(name)
		if err != nil {
			return fmt.Errorf("create png: %w", err)
		}
		if err := png.Encode(f, img); err != nil {
			_ = f.Close()
			return fmt.Errorf("encode png: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close png: %w", err)
		}
	}
	return nil
}

func toRGBA(c domain.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// strokeRect draws a 1px axis-aligned rectangle border inclusive of endpoints.
func strokeRect(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	// top and bottom
	for x := x0; x <= x1; x++ {
		img.SetRGBA(x, y0, col)
		img.SetRGBA(x, y1, col)
	}
	// left and right
	for y := y0; y <= y1; y++ {
		img.SetRGBA(x0, y, col)
		img.SetRGBA(x1, y, col)
	}
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			img.SetRGBA(x, y, col)
		}
	}
}
