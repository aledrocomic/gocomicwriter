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
	"time"

	"gocomicwriter/internal/domain"
	"gocomicwriter/internal/storage"
)

// EPUBOptions controls EPUB export behavior.
// Minimal set for MVP fixed-layout EPUB.
//
//nolint:revive // clarity
type EPUBOptions struct {
	IncludeGuides bool
	DPI           int
	Pages         []int
	Title         string
	Author        string
	Language      string // e.g., "en"
	Publisher     string
	Description   string
	Series        string
	SeriesIndex   int
	CoverIndex    int  // page index to use as cover; -1 => first page
	FixedLayout   bool // default true
}

// ExportIssueEPUB exports the specified issue as an EPUB 3 fixed-layout package.
func ExportIssueEPUB(ph *storage.ProjectHandle, issueIndex int, outPath string, opt EPUBOptions) error {
	if ph == nil {
		return fmt.Errorf("project handle is nil")
	}
	if issueIndex < 0 || issueIndex >= len(ph.Project.Issues) {
		return fmt.Errorf("issue index out of range")
	}
	iss := ph.Project.Issues[issueIndex]

	// Defaults
	if opt.Language == "" {
		opt.Language = "en"
	}
	if opt.FixedLayout == false {
		// default to fixed layout
		opt.FixedLayout = true
	}
	if opt.CoverIndex < 0 {
		opt.CoverIndex = 0
	}
	// Metadata fallbacks from project
	proj := ph.Project
	if opt.Title == "" {
		if proj.Metadata.IssueTitle != "" {
			opt.Title = proj.Metadata.IssueTitle
		} else {
			opt.Title = fmt.Sprintf("Issue %d", issueIndex+1)
		}
	}
	if opt.Author == "" {
		opt.Author = proj.Metadata.Creators
	}
	if opt.Series == "" {
		if proj.Metadata.Series != "" {
			opt.Series = proj.Metadata.Series
		} else if proj.Name != "" {
			opt.Series = proj.Name
		}
	}
	if opt.Description == "" {
		opt.Description = proj.Metadata.Notes
	}

	// Resolve output path
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(ph.Root, "exports", outPath)
	}
	if !strings.HasSuffix(strings.ToLower(outPath), ".epub") {
		outPath += ".epub"
	}

	// Prepare ZIP writer
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("ensure out dir: %w", err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create epub: %w", err)
	}
	defer func() { _ = f.Close() }()
	zw := zip.NewWriter(f)

	// 1) Write mimetype first, uncompressed
	if err := addStoredZipFile(zw, "mimetype", []byte("application/epub+zip")); err != nil {
		_ = zw.Close()
		return fmt.Errorf("write mimetype: %w", err)
	}

	// 2) META-INF/container.xml
	containerXML := "" +
		"<?xml version=\"1.0\" encoding=\"utf-8\"?>\n" +
		"<container version=\"1.0\" xmlns=\"urn:oasis:names:tc:opendocument:xmlns:container\">\n" +
		"  <rootfiles>\n" +
		"    <rootfile full-path=\"OEBPS/content.opf\" media-type=\"application/oebps-package+xml\"/>\n" +
		"  </rootfiles>\n" +
		"</container>\n"
	if err := addZipFile(zw, "META-INF/container.xml", []byte(containerXML)); err != nil {
		_ = zw.Close()
		return fmt.Errorf("write container.xml: %w", err)
	}

	// 3) Render pages to PNG bytes and build page XHTML/nav/manifest
	pages := pageIndexes(len(iss.Pages), opt.Pages)
	if len(pages) == 0 {
		_ = zw.Close()
		return fmt.Errorf("no pages to export")
	}
	// determine pixel dimensions
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
	scale := float64(dpi) / 72.0
	pixW := int(math.Round(mediaW * scale))
	pixH := int(math.Round(mediaH * scale))
	bx := int(math.Round(bleed * scale))
	by := int(math.Round(bleed * scale))

	// Styling defaults consistent with PNG/CBZ
	guideCol := domain.Color{R: 255, G: 0, B: 0, A: 255}
	panelStroke := domain.Stroke{Color: domain.Color{R: 0, G: 0, B: 0, A: 255}, Width: 1}
	balloonStroke := domain.Stroke{Color: domain.Color{R: 0, G: 0, B: 0, A: 255}, Width: 1}
	balloonFill := domain.Color{R: 255, G: 255, B: 255, A: 255}

	css := "html, body, .page { margin:0; padding:0; width:100%; height:100%; }\n" +
		"img { width:100%; height:100%; object-fit:contain; }\n" +
		"body { background:black; }\n"
	if err := addZipFile(zw, "OEBPS/styles/epub.css", []byte(css)); err != nil {
		_ = zw.Close()
		return fmt.Errorf("write css: %w", err)
	}

	pad := 1
	if n := len(pages); n >= 1000 {
		pad = 4
	} else if n >= 100 {
		pad = 3
	} else if n >= 10 {
		pad = 2
	}

	imgIDs := make([]string, 0, len(pages))
	pageIDs := make([]string, 0, len(pages))
	navBuf := &bytes.Buffer{}
	navBuf.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	navBuf.WriteString("<html xmlns=\"http://www.w3.org/1999/xhtml\" xmlns:epub=\"http://www.idpf.org/2007/ops\">\n<head><title>Table of Contents</title></head>\n<body>\n")
	navBuf.WriteString("<nav epub:type=\"toc\" id=\"toc\"><ol>\n")

	imgBuf := &bytes.Buffer{}
	for i, pidx := range pages {
		if pidx < 0 || pidx >= len(iss.Pages) {
			continue
		}
		pg := iss.Pages[pidx]

		// render
		img := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
		// background white
		draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
		// guides
		if opt.IncludeGuides {
			gc := toRGBA(guideCol)
			strokeRect(img, 0, 0, pixW-1, pixH-1, gc)
			strokeRect(img, bx, by, int(math.Round(trimW*scale))+bx-1, int(math.Round(trimH*scale))+by-1, gc)
		}
		// panels & balloons
		pc := toRGBA(panelStroke.Color)
		for _, pnl := range pg.Panels {
			r := pnl.Geometry
			x := int(math.Round((r.X + bleed) * scale))
			y := int(math.Round((r.Y + bleed) * scale))
			w := int(math.Round(r.Width * scale))
			h := int(math.Round(r.Height * scale))
			strokeRect(img, x, y, x+w-1, y+h-1, pc)
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
			_ = zw.Close()
			return fmt.Errorf("encode png: %w", err)
		}
		namePNG := fmt.Sprintf("OEBPS/images/page-%0*d.png", pad, i+1)
		if err := addZipFile(zw, namePNG, imgBuf.Bytes()); err != nil {
			_ = zw.Close()
			return fmt.Errorf("zip add image: %w", err)
		}
		imgID := fmt.Sprintf("img-%0*d", pad, i+1)
		pageID := fmt.Sprintf("page-%0*d", pad, i+1)
		imgIDs = append(imgIDs, imgID)
		pageIDs = append(pageIDs, pageID)

		// page XHTML
		pageXHTML := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"+
			"<html xmlns=\"http://www.w3.org/1999/xhtml\">\n<head>\n"+
			"<meta charset=\"utf-8\"/>\n"+
			"<meta name=\"viewport\" content=\"width=device-width, height=device-height\"/>\n"+
			"<title>Page %d</title>\n"+
			"<link rel=\"stylesheet\" type=\"text/css\" href=\"styles/epub.css\"/>\n"+
			"</head>\n<body>\n<div class=\"page\"><img src=\"images/page-%0*d.png\" alt=\"Page %d\"/></div>\n"+
			"</body>\n</html>\n", i+1, pad, i+1, i+1)
		if err := addZipFile(zw, fmt.Sprintf("OEBPS/page-%0*d.xhtml", pad, i+1), []byte(pageXHTML)); err != nil {
			_ = zw.Close()
			return fmt.Errorf("write page xhtml: %w", err)
		}
		navBuf.WriteString(fmt.Sprintf("<li><a href=\"page-%0*d.xhtml\">Page %d</a></li>\n", pad, i+1, i+1))
	}
	navBuf.WriteString("</ol></nav>\n</body>\n</html>\n")
	if err := addZipFile(zw, "OEBPS/nav.xhtml", navBuf.Bytes()); err != nil {
		_ = zw.Close()
		return fmt.Errorf("write nav.xhtml: %w", err)
	}

	// 4) content.opf
	reading := iss.ReadingDirection
	ppd := "ltr"
	if reading == "rtl" || strings.EqualFold(reading, "right-to-left") {
		ppd = "rtl"
	}
	mod := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	uid := fmt.Sprintf("urn:uuid:%d", time.Now().UnixNano())

	manifest := &bytes.Buffer{}
	manifest.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	manifest.WriteString("<package version=\"3.0\" unique-identifier=\"pub-id\" xmlns=\"http://www.idpf.org/2007/opf\">\n")
	manifest.WriteString("  <metadata xmlns:dc=\"http://purl.org/dc/elements/1.1/\" xmlns:opf=\"http://www.idpf.org/2007/opf\">\n")
	manifest.WriteString(fmt.Sprintf("    <dc:identifier id=\"pub-id\">%s</dc:identifier>\n", uid))
	manifest.WriteString(fmt.Sprintf("    <dc:title>%s</dc:title>\n", xmlEsc(opt.Title)))
	manifest.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", xmlEsc(opt.Language)))
	if strings.TrimSpace(opt.Author) != "" {
		manifest.WriteString(fmt.Sprintf("    <dc:creator>%s</dc:creator>\n", xmlEsc(opt.Author)))
	}
	if strings.TrimSpace(opt.Publisher) != "" {
		manifest.WriteString(fmt.Sprintf("    <dc:publisher>%s</dc:publisher>\n", xmlEsc(opt.Publisher)))
	}
	if strings.TrimSpace(opt.Description) != "" {
		manifest.WriteString(fmt.Sprintf("    <dc:description>%s</dc:description>\n", xmlEsc(opt.Description)))
	}
	manifest.WriteString(fmt.Sprintf("    <meta property=\"dcterms:modified\">%s</meta>\n", mod))
	if opt.FixedLayout {
		manifest.WriteString("    <meta property=\"rendition:layout\">pre-paginated</meta>\n")
		manifest.WriteString("    <meta property=\"rendition:orientation\">auto</meta>\n")
		manifest.WriteString("    <meta property=\"rendition:spread\">auto</meta>\n")
	}
	manifest.WriteString("  </metadata>\n")
	manifest.WriteString("  <manifest>\n")
	manifest.WriteString("    <item id=\"nav\" href=\"nav.xhtml\" media-type=\"application/xhtml+xml\" properties=\"nav\"/>\n")
	manifest.WriteString("    <item id=\"css\" href=\"styles/epub.css\" media-type=\"text/css\"/>\n")
	for i := range imgIDs {
		manifest.WriteString(fmt.Sprintf("    <item id=\"%s\" href=\"images/page-%0*d.png\" media-type=\"image/png\"%s/>\n",
			imgIDs[i], pad, i+1, func() string {
				if i == opt.CoverIndex {
					return " properties=\"cover-image\""
				}
				return ""
			}()))
		manifest.WriteString(fmt.Sprintf("    <item id=\"%s\" href=\"page-%0*d.xhtml\" media-type=\"application/xhtml+xml\"/>\n", pageIDs[i], pad, i+1))
	}
	manifest.WriteString("  </manifest>\n")
	manifest.WriteString(fmt.Sprintf("  <spine page-progression-direction=\"%s\">\n", ppd))
	for i := range pageIDs {
		manifest.WriteString(fmt.Sprintf("    <itemref idref=\"%s\"/>\n", pageIDs[i]))
	}
	manifest.WriteString("  </spine>\n")
	manifest.WriteString("</package>\n")
	if err := addZipFile(zw, "OEBPS/content.opf", manifest.Bytes()); err != nil {
		_ = zw.Close()
		return fmt.Errorf("write content.opf: %w", err)
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("close zip: %w", err)
	}
	return nil
}

// addStoredZipFile writes an entry with STORE method (no compression), required for EPUB mimetype.
func addStoredZipFile(zw *zip.Writer, name string, data []byte) error {
	hdr := &zip.FileHeader{Name: name, Method: zip.Store}
	// Set modification time without using deprecated SetModTime
	hdr.Modified = time.Now()
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
