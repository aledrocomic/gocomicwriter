package version

import "testing"

func TestVersionStringNonEmpty(t *testing.T) {
	if s := String(); s == "" {
		t.Fatalf("version string is empty")
	}
}
