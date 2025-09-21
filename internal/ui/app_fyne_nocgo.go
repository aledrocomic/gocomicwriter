//go:build fyne && !cgo

package ui

import "fmt"

// Run informs the user that Fyne UI requires cgo (OpenGL) and a C toolchain.
// This stub is compiled when the build uses -tags fyne but CGO is disabled.
func Run(_ string) error {
	return fmt.Errorf("Fyne UI requires cgo (OpenGL). Enable cgo and install a C toolchain. On Windows: install MSYS2/MinGW-w64, ensure gcc is on PATH, then run with CGO_ENABLED=1. Example: set CGO_ENABLED=1 && go run -tags fyne ./cmd/gocomicwriter ui [projectDir]")
}
