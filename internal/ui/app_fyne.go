//go:build fyne && cgo

/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"math"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"gocomicwriter/internal/crash"
	"gocomicwriter/internal/domain"
	applog "gocomicwriter/internal/log"
	"gocomicwriter/internal/script"
	"gocomicwriter/internal/storage"
	"gocomicwriter/internal/vector"
)

// Run starts the Fyne-based desktop UI shell with a basic canvas editor placeholder.
func Run(projectDir string) error {
	applog.Init(applog.FromEnv())
	l := applog.WithComponent("ui")
	l.Info("starting UI")

	var ph *storage.ProjectHandle
	defer func() { crash.Recover(ph) }()

	fyneApp := app.New()
	w := fyneApp.NewWindow("Go Comic Writer")
	w.Resize(fyne.NewSize(1200, 800))

	status := widget.NewLabel("Ready")
	canvasWidget := NewPageCanvas()

	// Canvas layout panes
	left := container.NewVBox(widget.NewLabel("Pages"), widget.NewSeparator(), widget.NewLabel("(placeholder)"))
	right := container.NewVBox(widget.NewLabel("Inspector"), widget.NewSeparator(), widget.NewLabel("(placeholder)"))
	canvasCenter := container.NewMax(canvasWidget)
	canvasPane := container.NewBorder(nil, nil, left, right, canvasCenter)

	// Script editor UI
	scriptEntry := widget.NewMultiLineEntry()
	scriptEntry.SetPlaceHolder("Type your script here. Use scene headers like \"# Scene Title\" and character lines like \"ALICE: Hello\". Indent continuation lines with two spaces.")
	// Outline data structures
	type outlineItem struct {
		kind      string   // scene, dialogue, caption, beat
		display   string   // final display string
		character string   // for dialogue
		tags      []string // extracted @tags from parser
	}
	outlineItems := []outlineItem{}
	outlineData := []string{}
	outlineFilter := ""

	scriptOutline := widget.NewList(
		func() int { return len(outlineData) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(outlineData[i]) },
	)

	applyOutlineFilter := func() {
		// rebuild visible strings from items according to filter
		outlineData = outlineData[:0]
		q := strings.TrimSpace(outlineFilter)
		if q == "" {
			for _, it := range outlineItems {
				outlineData = append(outlineData, it.display)
			}
			scriptOutline.Refresh()
			return
		}
		tokens := strings.Fields(strings.ToLower(q))
		for _, it := range outlineItems {
			displayLower := strings.ToLower(it.display)
			match := true
			for _, tok := range tokens {
				if strings.HasPrefix(tok, "@") {
					tag := strings.TrimPrefix(tok, "@")
					found := false
					for _, tg := range it.tags {
						if tg == tag {
							found = true
							break
						}
					}
					if !found {
						match = false
						break
					}
				} else if strings.HasPrefix(tok, "char:") {
					name := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(tok, "char:")))
					if it.character != name {
						match = false
						break
					}
				} else if strings.HasPrefix(tok, "is:") || strings.HasPrefix(tok, "type:") {
					idx := strings.Index(tok, ":")
					typeVal := tok[idx+1:]
					if it.kind != typeVal {
						match = false
						break
					}
				} else {
					if !strings.Contains(displayLower, tok) {
						match = false
						break
					}
				}
			}
			if match {
				outlineData = append(outlineData, it.display)
			}
		}
		scriptOutline.Refresh()
	}
	// Search/filter entry for outline
	outlineSearch := widget.NewEntry()
	outlineSearch.SetPlaceHolder("Filter outline (text, @tag, char:NAME, is:beat|dialogue|caption|scene)")
	outlineSearch.OnChanged = func(q string) {
		outlineFilter = strings.ToLower(strings.TrimSpace(q))
		applyOutlineFilter()
	}

	scriptErr := widget.NewLabel("")
	scriptErr.Wrapping = fyne.TextWrapWord

	// Bible data and UI state
	charNames := []string{}
	locNames := []string{}
	tagNames := []string{}
	var charList *widget.List
	var locList *widget.List
	var tagList *widget.List
	selectedChar := -1
	selectedLoc := -1
	selectedTag := -1

	refreshBible := func() {
		if ph == nil {
			charNames = charNames[:0]
			locNames = locNames[:0]
			tagNames = tagNames[:0]
		} else {
			charNames = charNames[:0]
			for _, c := range ph.Project.Bible.Characters {
				n := strings.TrimSpace(c.Name)
				if n != "" {
					charNames = append(charNames, n)
				}
			}
			locNames = locNames[:0]
			for _, c := range ph.Project.Bible.Locations {
				n := strings.TrimSpace(c.Name)
				if n != "" {
					locNames = append(locNames, n)
				}
			}
			tagNames = tagNames[:0]
			for _, t := range ph.Project.Bible.Tags {
				n := strings.TrimSpace(t.Name)
				if n != "" {
					tagNames = append(tagNames, n)
				}
			}
		}
		if charList != nil {
			charList.Refresh()
		}
		if locList != nil {
			locList.Refresh()
		}
		if tagList != nil {
			tagList.Refresh()
		}
	}

	var updateOutline func(string)

	// Insert helpers using bible
	insertCharacterLine := func(name string) {
		if strings.TrimSpace(name) == "" {
			return
		}
		txt := scriptEntry.Text
		if len(txt) > 0 && !strings.HasSuffix(txt, "\n") {
			txt += "\n"
		}
		txt += strings.ToUpper(name) + ": "
		scriptEntry.SetText(txt)
		updateOutline(txt)
	}
	insertTag := func(tag string) {
		if strings.TrimSpace(tag) == "" {
			return
		}
		txt := scriptEntry.Text
		if len(txt) > 0 && !strings.HasSuffix(txt, " ") && !strings.HasSuffix(txt, "\n") {
			txt += " "
		}
		txt += "@" + tag
		scriptEntry.SetText(txt)
		updateOutline(txt)
	}

	updateOutline = func(txt string) {
		sc, errs := script.Parse(txt)
		// build outline items and compute unmapped beat warnings
		mapped := map[string]struct{}{}
		if ph != nil {
			mapped = storage.MappedBeatSet(ph.Project)
		}
		totalBeats := 0
		unmappedBeats := 0
		outlineItems = outlineItems[:0]
		for si, scn := range sc.Scenes {
			st := strings.TrimSpace(scn.Title)
			outlineItems = append(outlineItems, outlineItem{kind: "scene", display: "Scene: " + st})
			for _, ln := range scn.Lines {
				switch ln.Type {
				case script.LineDialogue:
					preview := ln.Text
					if len(preview) > 60 {
						preview = preview[:60] + "…"
					}
					outlineItems = append(outlineItems, outlineItem{kind: "dialogue", display: "  " + ln.Character + ": " + preview, character: ln.Character, tags: ln.Tags})
				case script.LineCaption:
					preview := ln.Text
					if len(preview) > 60 {
						preview = preview[:60] + "…"
					}
					outlineItems = append(outlineItems, outlineItem{kind: "caption", display: "  [CAPTION] " + preview, tags: ln.Tags})
				case script.LineBeat:
					totalBeats++
					preview := ln.Text
					if len(preview) > 60 {
						preview = preview[:60] + "…"
					}
					id := storage.BeatIDFor(si, ln)
					display := "  [" + ln.Character + "] " + preview
					if _, ok := mapped[id]; !ok {
						// not mapped to any panel -> warn
						unmappedBeats++
						display += "  ⚠ unmapped"
					}
					outlineItems = append(outlineItems, outlineItem{kind: "beat", display: display, tags: ln.Tags})
				default:
					// skip notes/unknown in outline for now
				}
			}
		}
		// apply filter to build visible data
		applyOutlineFilter()
		if len(errs) > 0 {
			scriptErr.SetText(errs[0].Message)
		} else {
			scriptErr.SetText("")
		}
		// Update status with beat coverage information
		if totalBeats > 0 {
			status.SetText(fmt.Sprintf("Script: %d beats (%d unmapped)", totalBeats, unmappedBeats))
		} else {
			status.SetText("Script: no beats detected")
		}
	}
	scriptEntry.OnChanged = func(s string) { updateOutline(s) }

	// Script insertion controls leveraging the bible
	insertCharBtn := widget.NewButton("Insert Character", func() {
		if ph == nil || len(ph.Project.Bible.Characters) == 0 {
			dialog.ShowInformation("Insert Character", "No project open or no characters in bible.", w)
			return
		}
		// ensure names are current
		refreshBible()
		sel := widget.NewSelect(charNames, nil)
		sel.PlaceHolder = "Choose character"
		dialog.NewCustomConfirm("Insert Character", "Insert", "Cancel", sel, func(ok bool) {
			if ok && sel.Selected != "" {
				insertCharacterLine(sel.Selected)
			}
		}, w).Show()
	})
	insertTagBtn := widget.NewButton("Insert @Tag", func() {
		if ph == nil || len(ph.Project.Bible.Tags) == 0 {
			dialog.ShowInformation("Insert Tag", "No project open or no tags in bible.", w)
			return
		}
		refreshBible()
		sel := widget.NewSelect(tagNames, nil)
		sel.PlaceHolder = "Choose tag"
		dialog.NewCustomConfirm("Insert Tag", "Insert", "Cancel", sel, func(ok bool) {
			if ok && sel.Selected != "" {
				insertTag(sel.Selected)
			}
		}, w).Show()
	})
	scriptControls := container.NewHBox(insertCharBtn, insertTagBtn)

	// script pane
	outlineBox := container.NewBorder(container.NewVBox(widget.NewLabel("Outline"), outlineSearch), nil, nil, nil, scriptOutline)
	scriptSplit := container.NewHSplit(scriptEntry, outlineBox)
	scriptSplit.Offset = 0.7
	scriptPane := container.NewBorder(scriptControls, scriptErr, nil, nil, scriptSplit)

	// Bible management UI
	charList = widget.NewList(
		func() int { return len(charNames) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(charNames[i]) },
	)
	charList.OnSelected = func(id widget.ListItemID) { selectedChar = int(id) }
	addCharEntry := widget.NewEntry()
	addCharEntry.SetPlaceHolder("Add character name")
	addCharBtn := widget.NewButton("Add", func() {
		if ph == nil {
			dialog.ShowInformation("Characters", "Open a project first.", w)
			return
		}
		name := strings.TrimSpace(addCharEntry.Text)
		if name == "" {
			return
		}
		ph.Project.Bible.Characters = append(ph.Project.Bible.Characters, domain.BibleCharacter{Name: name})
		addCharEntry.SetText("")
		refreshBible()
	})
	delCharBtn := widget.NewButton("Delete", func() {
		if ph == nil || selectedChar < 0 || selectedChar >= len(ph.Project.Bible.Characters) {
			return
		}
		ph.Project.Bible.Characters = append(ph.Project.Bible.Characters[:selectedChar], ph.Project.Bible.Characters[selectedChar+1:]...)
		selectedChar = -1
		refreshBible()
	})
	charBox := container.NewVBox(widget.NewLabel("Characters"), charList, container.NewHBox(addCharEntry, addCharBtn, delCharBtn))

	// Locations
	locList = widget.NewList(
		func() int { return len(locNames) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(locNames[i]) },
	)
	locList.OnSelected = func(id widget.ListItemID) { selectedLoc = int(id) }
	addLocEntry := widget.NewEntry()
	addLocEntry.SetPlaceHolder("Add location name")
	addLocBtn := widget.NewButton("Add", func() {
		if ph == nil {
			dialog.ShowInformation("Locations", "Open a project first.", w)
			return
		}
		name := strings.TrimSpace(addLocEntry.Text)
		if name == "" {
			return
		}
		ph.Project.Bible.Locations = append(ph.Project.Bible.Locations, domain.BibleLocation{Name: name})
		addLocEntry.SetText("")
		refreshBible()
	})
	delLocBtn := widget.NewButton("Delete", func() {
		if ph == nil || selectedLoc < 0 || selectedLoc >= len(ph.Project.Bible.Locations) {
			return
		}
		ph.Project.Bible.Locations = append(ph.Project.Bible.Locations[:selectedLoc], ph.Project.Bible.Locations[selectedLoc+1:]...)
		selectedLoc = -1
		refreshBible()
	})
	locBox := container.NewVBox(widget.NewLabel("Locations"), locList, container.NewHBox(addLocEntry, addLocBtn, delLocBtn))

	// Tags
	tagList = widget.NewList(
		func() int { return len(tagNames) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(tagNames[i]) },
	)
	tagList.OnSelected = func(id widget.ListItemID) { selectedTag = int(id) }
	addTagEntry := widget.NewEntry()
	addTagEntry.SetPlaceHolder("Add tag")
	addTagBtn := widget.NewButton("Add", func() {
		if ph == nil {
			dialog.ShowInformation("Tags", "Open a project first.", w)
			return
		}
		name := strings.TrimSpace(addTagEntry.Text)
		if name == "" {
			return
		}
		ph.Project.Bible.Tags = append(ph.Project.Bible.Tags, domain.BibleTag{Name: name})
		addTagEntry.SetText("")
		refreshBible()
	})
	delTagBtn := widget.NewButton("Delete", func() {
		if ph == nil || selectedTag < 0 || selectedTag >= len(ph.Project.Bible.Tags) {
			return
		}
		ph.Project.Bible.Tags = append(ph.Project.Bible.Tags[:selectedTag], ph.Project.Bible.Tags[selectedTag+1:]...)
		selectedTag = -1
		refreshBible()
	})
	tagBox := container.NewVBox(widget.NewLabel("Tags"), tagList, container.NewHBox(addTagEntry, addTagBtn, delTagBtn))

	biblePane := container.NewGridWithColumns(3, charBox, locBox, tagBox)
	refreshBible()

	// Tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Canvas", canvasPane),
		container.NewTabItem("Script", scriptPane),
		container.NewTabItem("Bible", biblePane),
	)
	content := container.NewBorder(nil, status, nil, nil, tabs)
	w.SetContent(content)

	// Build menus
	openItem := fyne.NewMenuItem("Open…", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				l.Error("open dialog error", slog.Any("err", err))
				return
			}
			if uri == nil {
				return
			}
			abs := uri.Path()
			if err := openProject(abs, &ph, w, l, status); err != nil {
				l.Error("open project failed", slog.Any("err", err))
				dialog.ShowError(err, w)
			}
			// Load script text after successful open
			if ph != nil {
				if txt, rerr := storage.ReadScript(ph); rerr == nil {
					scriptEntry.SetText(txt)
					updateOutline(txt)
					refreshBible()
				} else {
					l.Error("read script failed", slog.Any("err", rerr))
				}
			}
		}, w)
		fd.Show()
	})
	saveItem := fyne.NewMenuItem("Save", func() {
		if ph == nil {
			dialog.ShowInformation("Save", "No project open.", w)
			return
		}
		if err := storage.Save(ph); err != nil {
			l.Error("save failed", slog.Any("err", err))
			dialog.ShowError(err, w)
			return
		}
		if err := storage.WriteScript(ph, scriptEntry.Text); err != nil {
			l.Error("save script failed", slog.Any("err", err))
			dialog.ShowError(err, w)
			return
		}
		status.SetText("Saved project (manifest + script).")
	})
	exitItem := fyne.NewMenuItem("Exit", func() { w.Close() })

	fileMenu := fyne.NewMenu("File", openItem, saveItem, fyne.NewMenuItemSeparator(), exitItem)
	w.SetMainMenu(fyne.NewMainMenu(fileMenu))

	// Try to open a project if provided
	if projectDir != "" {
		if err := openProject(projectDir, &ph, w, l, status); err != nil {
			l.Error("auto-open project failed", slog.Any("err", err))
			// not fatal; continue
		} else {
			if txt, rerr := storage.ReadScript(ph); rerr == nil {
				scriptEntry.SetText(txt)
				updateOutline(txt)
				refreshBible()
			} else {
				l.Error("read script failed", slog.Any("err", rerr))
			}
		}
	}

	w.ShowAndRun()
	return nil
}

func openProject(dir string, ph **storage.ProjectHandle, w fyne.Window, l *slog.Logger, status *widget.Label) error {
	abs, _ := filepath.Abs(dir)
	l.Info("open project", slog.String("root", abs))
	h, err := storage.Open(abs)
	if err != nil {
		return err
	}
	*ph = h
	w.SetTitle(fmt.Sprintf("Go Comic Writer — %s", h.Project.Name))
	status.SetText(fmt.Sprintf("Opened project: %s", abs))
	return nil
}

// PageCanvas is a minimal interactive canvas placeholder that draws a page rectangle
// and simple trim/bleed guides. Supports pan with mouse drag and zoom with Ctrl+wheel.
type PageCanvas struct {
	widget.BaseWidget
	// Interaction
	zoom    float32
	offsetX float32
	offsetY float32
	// Geometry (logical units at 72dpi defaults)
	pageW       float32
	pageH       float32
	bleedMargin float32
	trimMargin  float32
	gutterSize  float32 // inner margin width
	gutterLeft  bool    // if false, gutter is drawn on the right

	// Scene graph (demo) and selection
	scene    []vector.Node
	selected int // index into scene, -1 if none
	// Interaction state for transforms
	dragMode  dragMode
	startPage vector.Pt
	startXf   vector.Affine2D
	// For scale/rotate operations
	anchor vector.Pt
}

// dragMode represents current interaction kind
// dragNone: idle; dragPan: background pan; dragMove: moving selection; dragScale*: corner scaling; dragRotate: rotation handle
// We keep minimal 4 corners and 1 rotation handle.
type dragMode int

const (
	dragNone dragMode = iota
	dragPan
	dragMove
	dragScaleNW
	dragScaleNE
	dragScaleSW
	dragScaleSE
	dragRotate
)

func NewPageCanvas() *PageCanvas {
	pc := &PageCanvas{
		zoom:        0.5,
		pageW:       595, // A4 portrait width in pt (72dpi)
		pageH:       842, // A4 portrait height in pt
		bleedMargin: 18,  // ~0.25in
		trimMargin:  9,   // ~0.125in
		gutterSize:  18,  // ~0.25in inner margin
		gutterLeft:  true,
		selected:    -1,
	}
	// Demo scene: two rectangles
	r1 := vector.NewRect(vector.R(100, 100, 160, 120), vector.Fill{Enabled: true, Color: vector.Color{R: 220, G: 120, B: 120, A: 255}}, vector.Stroke{Enabled: true, Color: vector.Black, Width: 2})
	r2 := vector.NewRect(vector.R(300, 220, 180, 100), vector.Fill{Enabled: true, Color: vector.Color{R: 120, G: 180, B: 220, A: 255}}, vector.Stroke{Enabled: true, Color: vector.Black, Width: 2})
	// Give second a slight rotation for testing rotate handler later
	r2.SetTransform(r2.Transform().Mul(vector.Translate(390, 270)).Mul(vector.Rotate(0.2)).Mul(vector.Translate(-390, -270)))
	pc.scene = []vector.Node{r1, r2}

	pc.ExtendBaseWidget(pc)
	return pc
}

// CreateRenderer builds the simple vector-like objects we position manually.
func (p *PageCanvas) CreateRenderer() fyne.WidgetRenderer {
	// Background
	bg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 34, A: 255})

	// Page base and guides
	page := canvas.NewRectangle(color.White)
	page.StrokeColor = color.RGBA{R: 20, G: 20, B: 20, A: 255}
	page.StrokeWidth = 2

	trim := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 0})
	trim.StrokeColor = color.RGBA{R: 200, G: 0, B: 0, A: 200}
	trim.StrokeWidth = 1

	bleed := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 0})
	bleed.StrokeColor = color.RGBA{R: 0, G: 120, B: 255, A: 180}
	bleed.StrokeWidth = 1

	gutter := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 0})
	gutter.StrokeColor = color.RGBA{R: 120, G: 200, B: 0, A: 200}
	gutter.FillColor = color.RGBA{R: 120, G: 200, B: 0, A: 40}
	gutter.StrokeWidth = 1

	// Node rectangles (use Rectangle instead of Polygon to match Fyne v2.6 API)
	var rects []*canvas.Rectangle
	for j := 0; j < len(p.scene); j++ {
		r := canvas.NewRectangle(color.RGBA{R: 220, G: 220, B: 220, A: 255})
		r.StrokeColor = color.RGBA{R: 30, G: 30, B: 30, A: 255}
		r.StrokeWidth = 1
		rects = append(rects, r)
	}

	// Selection overlay: bbox and 4 corner handles + rotation handle
	bbox := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
	bbox.StrokeColor = color.RGBA{R: 0, G: 170, B: 255, A: 255}
	bbox.StrokeWidth = 1
	bbox.Hide()

	handles := []*canvas.Rectangle{
		canvas.NewRectangle(color.RGBA{R: 0, G: 170, B: 255, A: 255}),
		canvas.NewRectangle(color.RGBA{R: 0, G: 170, B: 255, A: 255}),
		canvas.NewRectangle(color.RGBA{R: 0, G: 170, B: 255, A: 255}),
		canvas.NewRectangle(color.RGBA{R: 0, G: 170, B: 255, A: 255}),
	}
	for _, h := range handles {
		h.Hide()
	}
	rot := canvas.NewCircle(color.RGBA{R: 255, G: 170, B: 0, A: 255})
	rot.Hide()

	// Draw order: background, bleed (outside), page base, then guides, then nodes and selection overlay on top
	objs := []fyne.CanvasObject{bg, bleed, page, trim, gutter}
	for _, r := range rects {
		objs = append(objs, r)
	}
	objs = append(objs, bbox)
	for _, h := range handles {
		objs = append(objs, h)
	}
	objs = append(objs, rot)

	return &pageCanvasRenderer{pc: p, objects: objs, bg: bg, page: page, trim: trim, bleed: bleed, gutter: gutter, rects: rects, bbox: bbox, handles: handles, rot: rot}
}

// PreferredSize sets a decent default size for the widget.
func (p *PageCanvas) PreferredSize() fyne.Size { return fyne.NewSize(800, 600) }

// Coordinate helpers: page <-> screen mapping
func (p *PageCanvas) pageOriginAndScale() (cx, cy, scale float32) {
	size := p.Size()
	scaledW := p.pageW * p.zoom
	scaledH := p.pageH * p.zoom
	cx = float32(size.Width)/2 - scaledW/2 + p.offsetX
	cy = float32(size.Height)/2 - scaledH/2 + p.offsetY
	return cx, cy, p.zoom
}
func (p *PageCanvas) toScreen(pt vector.Pt) fyne.Position {
	cx, cy, s := p.pageOriginAndScale()
	x := cx + pt.X*s
	y := cy + pt.Y*s
	return fyne.NewPos(float32ToFixed(x), float32ToFixed(y))
}
func (p *PageCanvas) toPage(pos fyne.Position) vector.Pt {
	cx, cy, s := p.pageOriginAndScale()
	return vector.Pt{X: (float32(pos.X) - cx) / s, Y: (float32(pos.Y) - cy) / s}
}

// Hit test scene and return top-most index
func (p *PageCanvas) hitTest(pagePt vector.Pt) int {
	for i := len(p.scene) - 1; i >= 0; i-- {
		if p.scene[i].Hit(pagePt) {
			return i
		}
	}
	return -1
}

// Light-weight rectangle type for handle geometry
type fRect struct{ X, Y, Width, Height float32 }

func newFRect(x, y, w, h float32) fRect { return fRect{X: x, Y: y, Width: w, Height: h} }

// Handle rectangles in screen coords around selection bbox
func (p *PageCanvas) handleRects() (bbox fRect, corners [4]fRect, rot fRect, ok bool) {
	if p.selected < 0 || p.selected >= len(p.scene) {
		return fRect{}, [4]fRect{}, fRect{}, false
	}
	b := p.scene[p.selected].Bounds() // page coords
	p0 := p.toScreen(vector.Pt{X: b.X, Y: b.Y})
	p1 := p.toScreen(vector.Pt{X: b.X + b.W, Y: b.Y + b.H})
	bx := float32ToFixed(p0.X)
	by := float32ToFixed(p0.Y)
	bw := float32ToFixed(float32(p1.X - p0.X))
	bh := float32ToFixed(float32(p1.Y - p0.Y))
	bbox = newFRect(bx, by, bw, bh)
	sz := float32(8)
	hh := sz
	hw := sz
	corners = [4]fRect{
		newFRect(bx-hw/2, by-hh/2, hw, hh),       // NW
		newFRect(bx+bw-hw/2, by-hh/2, hw, hh),    // NE
		newFRect(bx-hw/2, by+bh-hh/2, hw, hh),    // SW
		newFRect(bx+bw-hw/2, by+bh-hh/2, hw, hh), // SE
	}
	// Rotation handle above top center
	rcx := bx + bw/2
	rcy := by - 24
	rot = newFRect(rcx-6, rcy-6, 12, 12)
	return bbox, corners, rot, true
}

// Tapped selects a node using hit testing
func (p *PageCanvas) Tapped(e *fyne.PointEvent) {
	pagePt := p.toPage(e.Position)
	idx := p.hitTest(pagePt)
	p.selected = idx
	p.dragMode = dragNone
	p.Refresh()
}

// Dragging and scrolling support
func (p *PageCanvas) Dragged(e *fyne.DragEvent) {
	pos := e.Position
	if p.dragMode == dragNone {
		// Determine action by start position
		if p.selected >= 0 {
			_, corners, rot, ok := p.handleRects()
			if ok {
				if pos.X >= rot.X && pos.X <= rot.X+rot.Width && pos.Y >= rot.Y && pos.Y <= rot.Y+rot.Height {
					p.dragMode = dragRotate
				} else if pos.X >= corners[0].X && pos.X <= corners[0].X+corners[0].Width && pos.Y >= corners[0].Y && pos.Y <= corners[0].Y+corners[0].Height {
					p.dragMode = dragScaleNW
				} else if pos.X >= corners[1].X && pos.X <= corners[1].X+corners[1].Width && pos.Y >= corners[1].Y && pos.Y <= corners[1].Y+corners[1].Height {
					p.dragMode = dragScaleNE
				} else if pos.X >= corners[2].X && pos.X <= corners[2].X+corners[2].Width && pos.Y >= corners[2].Y && pos.Y <= corners[2].Y+corners[2].Height {
					p.dragMode = dragScaleSW
				} else if pos.X >= corners[3].X && pos.X <= corners[3].X+corners[3].Width && pos.Y >= corners[3].Y && pos.Y <= corners[3].Y+corners[3].Height {
					p.dragMode = dragScaleSE
				}
			}
		}
		if p.dragMode == dragNone {
			// If hit on selection body -> move; else pan
			pagePt := p.toPage(pos)
			if p.selected >= 0 && p.scene[p.selected].Hit(pagePt) {
				p.dragMode = dragMove
			} else {
				p.dragMode = dragPan
			}
		}
		p.startPage = p.toPage(pos)
		if p.selected >= 0 {
			p.startXf = p.scene[p.selected].Transform()
			b := p.scene[p.selected].Bounds()
			// default anchor: center; for scale set based on handle later
			p.anchor = vector.Pt{X: b.X + b.W/2, Y: b.Y + b.H/2}
		}
	}

	switch p.dragMode {
	case dragPan:
		p.offsetX += float32(e.Dragged.DX)
		p.offsetY += float32(e.Dragged.DY)
	case dragMove:
		cur := p.toPage(pos)
		dx := cur.X - p.startPage.X
		dy := cur.Y - p.startPage.Y
		if p.selected >= 0 {
			newXf := vector.Translate(dx, dy).Mul(p.startXf)
			p.scene[p.selected].SetTransform(newXf)
		}
	case dragScaleNW, dragScaleNE, dragScaleSW, dragScaleSE:
		if p.selected >= 0 {
			b := p.scene[p.selected].Bounds()
			// Set anchor to opposite corner
			var ax, ay float32
			switch p.dragMode {
			case dragScaleNW:
				ax, ay = b.X+b.W, b.Y+b.H
			case dragScaleNE:
				ax, ay = b.X, b.Y+b.H
			case dragScaleSW:
				ax, ay = b.X+b.W, b.Y
			case dragScaleSE:
				ax, ay = b.X, b.Y
			}
			p.anchor = vector.Pt{X: ax, Y: ay}
			cur := p.toPage(pos)
			// Compute scale factors relative to bbox
			var sx, sy float32 = 1, 1
			if b.W != 0 {
				sx = (cur.X - p.anchor.X) / (p.startPage.X - p.anchor.X)
			}
			if b.H != 0 {
				sy = (cur.Y - p.anchor.Y) / (p.startPage.Y - p.anchor.Y)
			}
			// Guard against NaN or inf
			if sx == 0 {
				sx = 0.001
			}
			if sy == 0 {
				sy = 0.001
			}
			xf := vector.Translate(p.anchor.X, p.anchor.Y).Mul(vector.Scale(sx, sy)).Mul(vector.Translate(-p.anchor.X, -p.anchor.Y)).Mul(p.startXf)
			p.scene[p.selected].SetTransform(xf)
		}
	case dragRotate:
		if p.selected >= 0 {
			b := p.scene[p.selected].Bounds()
			c := vector.Pt{X: b.X + b.W/2, Y: b.Y + b.H/2}
			p.anchor = c
			start := p.startPage
			cur := p.toPage(pos)
			// Angle between center->point vectors
			dx0, dy0 := start.X-c.X, start.Y-c.Y
			dx1, dy1 := cur.X-c.X, cur.Y-c.Y
			ang0 := float32(math.Atan2(float64(dy0), float64(dx0)))
			ang1 := float32(math.Atan2(float64(dy1), float64(dx1)))
			dang := ang1 - ang0
			xf := vector.Translate(c.X, c.Y).Mul(vector.Rotate(dang)).Mul(vector.Translate(-c.X, -c.Y)).Mul(p.startXf)
			p.scene[p.selected].SetTransform(xf)
		}
	}
	p.Refresh()
}
func (p *PageCanvas) DragEnd() { p.dragMode = dragNone }

// Scroll changes zoom when Ctrl pressed, else pans vertically.
func (p *PageCanvas) Scrolled(e *fyne.ScrollEvent) {
	// Fyne v2.6 does not expose modifier keys on ScrollEvent; keep it simple and
	// always use the wheel to zoom. This keeps the demo usable across platforms.
	step := float32(e.Scrolled.DY) * 0.05
	p.zoom += step
	if p.zoom < 0.1 {
		p.zoom = 0.1
	}
	if p.zoom > 4.0 {
		p.zoom = 4.0
	}
	p.Refresh()
}

// pageCanvasRenderer handles layout of the drawable objects based on zoom/offset.
type pageCanvasRenderer struct {
	pc          *PageCanvas
	objects     []fyne.CanvasObject
	bg, page    *canvas.Rectangle
	trim, bleed *canvas.Rectangle
	gutter      *canvas.Rectangle
	// scene visuals
	rects []*canvas.Rectangle
	// selection visuals
	bbox    *canvas.Rectangle
	handles []*canvas.Rectangle
	rot     *canvas.Circle
}

func (r *pageCanvasRenderer) Destroy()                     {}
func (r *pageCanvasRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *pageCanvasRenderer) MinSize() fyne.Size           { return r.pc.PreferredSize() }
func (r *pageCanvasRenderer) Refresh()                     { r.Layout(r.pc.Size()); canvas.Refresh(r.pc) }

func (r *pageCanvasRenderer) Layout(size fyne.Size) {
	// Fill background
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))

	// Define logical page and margins from widget configuration.
	logicalW := r.pc.pageW
	logicalH := r.pc.pageH
	bleedMargin := r.pc.bleedMargin
	trimMargin := r.pc.trimMargin
	gutterSize := r.pc.gutterSize

	// Apply zoom
	scaledW := logicalW * r.pc.zoom
	scaledH := logicalH * r.pc.zoom

	// Center in the available space, then add pan offset
	cx := float32(size.Width)/2 - scaledW/2 + r.pc.offsetX
	cy := float32(size.Height)/2 - scaledH/2 + r.pc.offsetY

	// Page rectangle
	r.page.Resize(fyne.NewSize(float32ToFixed(scaledW), float32ToFixed(scaledH)))
	r.page.Move(fyne.NewPos(float32ToFixed(cx), float32ToFixed(cy)))

	// Trim and bleed boxes
	trimW := (logicalW - 2*trimMargin) * r.pc.zoom
	trimH := (logicalH - 2*trimMargin) * r.pc.zoom
	trimX := cx + trimMargin*r.pc.zoom
	trimY := cy + trimMargin*r.pc.zoom

	r.trim.Resize(fyne.NewSize(float32ToFixed(trimW), float32ToFixed(trimH)))
	r.trim.Move(fyne.NewPos(float32ToFixed(trimX), float32ToFixed(trimY)))

	bleedW := (logicalW + 2*bleedMargin) * r.pc.zoom
	bleedH := (logicalH + 2*bleedMargin) * r.pc.zoom
	bleedX := cx - bleedMargin*r.pc.zoom
	bleedY := cy - bleedMargin*r.pc.zoom

	r.bleed.Resize(fyne.NewSize(float32ToFixed(bleedW), float32ToFixed(bleedH)))
	r.bleed.Move(fyne.NewPos(float32ToFixed(bleedX), float32ToFixed(bleedY)))

	// Gutter guide: inner margin strip on left or right inside the page
	gW := gutterSize * r.pc.zoom
	gH := scaledH
	var gX float32
	if r.pc.gutterLeft {
		gX = cx
	} else {
		gX = cx + scaledW - gW
	}
	gY := cy
	r.gutter.Resize(fyne.NewSize(float32ToFixed(gW), float32ToFixed(gH)))
	r.gutter.Move(fyne.NewPos(float32ToFixed(gX), float32ToFixed(gY)))

	// Scene nodes as axis-aligned rectangles using their Bounds()
	for i, n := range r.pc.scene {
		if i >= len(r.rects) {
			break
		}
		b := n.Bounds()
		p0 := r.pc.toScreen(vector.Pt{X: b.X, Y: b.Y})
		p1 := r.pc.toScreen(vector.Pt{X: b.X + b.W, Y: b.Y + b.H})
		rc := r.rects[i]
		rc.Resize(fyne.NewSize(float32ToFixed(float32(p1.X-p0.X)), float32ToFixed(float32(p1.Y-p0.Y))))
		rc.Move(fyne.NewPos(float32ToFixed(p0.X), float32ToFixed(p0.Y)))
		// Color per node demo
		if i%2 == 0 {
			rc.FillColor = color.RGBA{R: 240, G: 160, B: 160, A: 255}
		} else {
			rc.FillColor = color.RGBA{R: 160, G: 200, B: 240, A: 255}
		}
		rc.Refresh()
	}

	// Selection overlay
	if r.pc.selected >= 0 {
		bbox, corners, rot, ok := r.pc.handleRects()
		if ok {
			r.bbox.Show()
			r.bbox.Resize(fyne.NewSize(bbox.Width, bbox.Height))
			r.bbox.Move(fyne.NewPos(bbox.X, bbox.Y))
			for i := 0; i < len(r.handles); i++ {
				r.handles[i].Show()
				r.handles[i].Resize(fyne.NewSize(corners[i].Width, corners[i].Height))
				r.handles[i].Move(fyne.NewPos(corners[i].X, corners[i].Y))
			}
			r.rot.Show()
			r.rot.Resize(fyne.NewSize(rot.Width, rot.Height))
			r.rot.Move(fyne.NewPos(rot.X, rot.Y))
		}
	} else {
		r.bbox.Hide()
		for _, h := range r.handles {
			h.Hide()
		}
		r.rot.Hide()
	}
}

func float32ToFixed(v float32) float32 { return fyne.NewSize(v, 0).Width }
