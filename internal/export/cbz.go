/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

// CBZOptions controls CBZ export behavior.
// Pages are exported as PNG with optional guides, similar to PNGOptions.
//
//nolint:revive // clarity
type CBZOptions struct {
	IncludeGuides bool
	DPI           int
	GuideColor    domain.Color
	PanelStroke   domain.Stroke
	BalloonStroke domain.Stroke
	BalloonFill   domain.Color
	Pages         []int
}

// ExportIssueCBZ packages selected issue pages as PNG images into a CBZ (ZIP) archive
// and adds a ComicInfo.xml metadata manifest for reader compatibility.
func ExportIssueCBZ(ph *storage.ProjectHandle, issueIndex int, outPath string, opt CBZOptions) error {
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

	// Ensure output path is under project exports folder if relative
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(ph.Root, "exports", outPath)
	}
	// Enforce .cbz extension
	if !strings.HasSuffix(strings.ToLower(outPath), ".cbz") {
		outPath = outPath + ".cbz"
	}

	// Create ZIP writer
	zw, f, err := createZip(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	pages := pageIndexes(len(iss.Pages), opt.Pages)
	// Zero padding width based on count
	pad := 3
	if n := len(pages); n >= 1000 {
		pad = 4
	} else if n >= 100 {
		pad = 3
	} else if n >= 10 {
		pad = 2
	} else {
		pad = 1
	}

	imgBuf := &bytes.Buffer{}
	for i, pidx := range pages {
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

		// Panels and balloons
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

		imgBuf.Reset()
		if err := png.Encode(imgBuf, img); err != nil {
			return fmt.Errorf("encode png: %w", err)
		}
		name := fmt.Sprintf("%0*d.png", pad, i+1)
		if err := addZipFile(zw, name, imgBuf.Bytes()); err != nil {
			return fmt.Errorf("zip add image: %w", err)
		}
	}

	// Add ComicInfo.xml manifest
	manifest, merr := buildComicInfoXML(ph, issueIndex, len(pages))
	if merr != nil {
		return fmt.Errorf("build manifest: %w", merr)
	}
	if err := addZipFile(zw, "ComicInfo.xml", []byte(manifest)); err != nil {
		return fmt.Errorf("zip add manifest: %w", err)
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("close zip: %w", err)
	}
	return nil
}

func createZip(outPath string) (*zip.Writer, *os.File, error) {
	// Ensure directory exists
	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("ensure out dir: %w", err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create cbz: %w", err)
	}
	return zip.NewWriter(f), f, nil
}

func addZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func buildComicInfoXML(ph *storage.ProjectHandle, issueIndex, pageCount int) (string, error) {
	proj := ph.Project
	iss := proj.Issues[issueIndex]
	series := proj.Metadata.Series
	if series == "" {
		series = proj.Name
	}
	title := proj.Metadata.IssueTitle
	if title == "" {
		title = fmt.Sprintf("Issue %d", issueIndex+1)
	}
	writer := proj.Metadata.Creators
	summary := proj.Metadata.Notes
	reading := iss.ReadingDirection
	if reading == "rtl" || strings.EqualFold(reading, "right-to-left") {
		reading = "RightToLeft"
	} else {
		reading = "LeftToRight"
	}
	buf := &bytes.Buffer{}
	var werr error
	wf := func(format string, args ...any) {
		if werr != nil {
			return
		}
		_, werr = fmt.Fprintf(buf, format, args...)
	}
	wf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	wf("<ComicInfo xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">\n")
	wf("  <Series>%s</Series>\n", xmlEsc(series))
	wf("  <Title>%s</Title>\n", xmlEsc(title))
	wf("  <Number>%d</Number>\n", issueIndex+1)
	wf("  <PageCount>%d</PageCount>\n", pageCount)
	if writer != "" {
		wf("  <Writer>%s</Writer>\n", xmlEsc(writer))
	}
	if summary != "" {
		wf("  <Summary>%s</Summary>\n", xmlEsc(summary))
	}
	wf("  <ReadingDirection>%s</ReadingDirection>\n", reading)
	wf("</ComicInfo>\n")
	if werr != nil {
		return "", fmt.Errorf("build xml: %w", werr)
	}
	return buf.String(), nil
}

func xmlEsc(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			out = append(out, '&', 'a', 'm', 'p', ';')
		case '<':
			out = append(out, '&', 'l', 't', ';')
		case '>':
			out = append(out, '&', 'g', 't', ';')
		case '"':
			out = append(out, '&', 'q', 'u', 'o', 't', ';')
		case '\'':
			out = append(out, '&', 'a', 'p', 'o', 's', ';')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
