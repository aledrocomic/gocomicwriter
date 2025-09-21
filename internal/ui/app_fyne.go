//go:build fyne

package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"path/filepath"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"gocomicwriter/internal/crash"
	applog "gocomicwriter/internal/log"
	"gocomicwriter/internal/storage"
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

	// Layout: left placeholder nav, center canvas, right placeholder inspector
	left := container.NewVBox(widget.NewLabel("Pages"), widget.NewSeparator(), widget.NewLabel("(placeholder)"))
	right := container.NewVBox(widget.NewLabel("Inspector"), widget.NewSeparator(), widget.NewLabel("(placeholder)"))
	center := container.NewMax(canvasWidget)
	content := container.NewBorder(nil, status, left, right, center)
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
			if !uri.IsDirectory() {
				abs = filepath.Dir(abs)
			}
			if err := openProject(abs, &ph, w, l, status); err != nil {
				l.Error("open project failed", slog.Any("err", err))
				dialog.ShowError(err, w)
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
		status.SetText("Saved project and created a backup.")
	})
	exitItem := fyne.NewMenuItem("Exit", func() { w.Close() })

	fileMenu := fyne.NewMenu("File", openItem, saveItem, fyne.NewMenuItemSeparator(), exitItem)
	w.SetMainMenu(fyne.NewMainMenu(fileMenu))

	// Try to open a project if provided
	if projectDir != "" {
		if err := openProject(projectDir, &ph, w, l, status); err != nil {
			l.Error("auto-open project failed", slog.Any("err", err))
			// not fatal; continue
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
	zoom    float32
	offsetX float32
	offsetY float32
}

func NewPageCanvas() *PageCanvas {
	pc := &PageCanvas{zoom: 0.5}
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

	objs := []fyne.CanvasObject{bg, bleed, trim, page}

	return &pageCanvasRenderer{pc: p, objects: objs, bg: bg, page: page, trim: trim, bleed: bleed}
}

// PreferredSize sets a decent default size for the widget.
func (p *PageCanvas) PreferredSize() fyne.Size { return fyne.NewSize(800, 600) }

// Dragging and scrolling support
func (p *PageCanvas) Dragged(e *fyne.DragEvent) {
	p.offsetX += float32(e.Dragged.DX)
	p.offsetY += float32(e.Dragged.DY)
	p.Refresh()
}
func (p *PageCanvas) DragEnd() {}

// Scroll changes zoom when Ctrl pressed, else pans vertically.
func (p *PageCanvas) Scrolled(e *fyne.ScrollEvent) {
	if e.Modifiers&fyne.KeyModifierControl > 0 {
		step := float32(e.Scrolled.DY) * 0.05
		p.zoom += step
		if p.zoom < 0.1 {
			p.zoom = 0.1
		}
		if p.zoom > 4.0 {
			p.zoom = 4.0
		}
	} else {
		p.offsetY += float32(e.Scrolled.DY) * 10
	}
	p.Refresh()
}

// pageCanvasRenderer handles layout of the drawable objects based on zoom/offset.
type pageCanvasRenderer struct {
	pc          *PageCanvas
	objects     []fyne.CanvasObject
	bg, page    *canvas.Rectangle
	trim, bleed *canvas.Rectangle
}

func (r *pageCanvasRenderer) Destroy()                     {}
func (r *pageCanvasRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *pageCanvasRenderer) MinSize() fyne.Size           { return r.pc.PreferredSize() }
func (r *pageCanvasRenderer) Refresh()                     { r.Layout(r.pc.Size()); canvas.Refresh(r.pc) }

func (r *pageCanvasRenderer) Layout(size fyne.Size) {
	// Fill background
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))

	// Define a logical page size (e.g., A4 portrait in points).
	logicalW := float32(595)   // A4 width at 72dpi
	logicalH := float32(842)   // A4 height at 72dpi
	bleedMargin := float32(18) // 0.25in at 72dpi approx
	trimMargin := float32(9)   // 0.125in approx

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
}

func float32ToFixed(v float32) float32 { return fyne.NewSize(v, 0).Width }
