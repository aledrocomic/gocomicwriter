//go:build !fyne

package ui

import "fmt"

// Run starts the desktop UI. In non-fyne builds, this is a stub so CI remains headless.
// Pass an optional project directory to open immediately.
func Run(_ string) error {
	return fmt.Errorf("UI not built in this binary. Rebuild with: go run -tags fyne ./cmd/gocomicwriter ui [projectDir]")
}
