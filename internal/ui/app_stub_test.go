//go:build !fyne

package ui

import (
	"strings"
	"testing"
)

func TestRunStub_ReturnsHelpfulError(t *testing.T) {
	err := Run("")
	if err == nil {
		t.Fatal("expected error from Run() in non-fyne build, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "UI not built") || !strings.Contains(msg, "-tags fyne") {
		t.Fatalf("unexpected error message: %q", msg)
	}
}
