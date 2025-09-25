/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * Licensed under the Apache License, Version 2.0.
 */
package textlayout

import "testing"

func TestStyleSheet_ResolvePrecedence(t *testing.T) {
	ss := NewStyleSheet()
	// Base builtin Dialogue exists
	b, ok := ss.Resolve("Dialogue")
	if !ok {
		t.Fatalf("expected builtin Dialogue")
	}

	// Issue overrides Dialogue tracking
	iss := TextStyle{Name: "Dialogue", Font: b.Font, Tracking: 1.25, Leading: b.Leading}
	// Page overrides Dialogue leading
	pag := TextStyle{Name: "Dialogue", Font: b.Font, Tracking: iss.Tracking, Leading: 9}

	ss = ss.WithIssue(map[string]TextStyle{"Dialogue": iss})
	got, ok := ss.Resolve("Dialogue")
	if !ok {
		t.Fatalf("resolve after issue override failed")
	}
	if got.Tracking != 1.25 {
		t.Fatalf("issue override not applied: got tracking=%v", got.Tracking)
	}
	if got.Leading != b.Leading {
		t.Fatalf("issue override should not change leading: got leading=%v want %v", got.Leading, b.Leading)
	}

	ss = ss.WithPage(map[string]TextStyle{"Dialogue": pag})
	got2, ok := ss.Resolve("Dialogue")
	if !ok {
		t.Fatalf("resolve after page override failed")
	}
	if got2.Leading != 9 {
		t.Fatalf("page override not applied: got leading=%v", got2.Leading)
	}
	if got2.Tracking != 1.25 {
		t.Fatalf("page should inherit issue tracking when not overridden: got tracking=%v", got2.Tracking)
	}
}

func TestStyleSheet_FallbackBuiltin(t *testing.T) {
	ss := &StyleSheet{Global: map[string]TextStyle{}, Issue: map[string]TextStyle{}, Page: map[string]TextStyle{}}
	// Should still resolve builtins
	if _, ok := ss.Resolve("Caption"); !ok {
		t.Fatalf("expected builtin fallback for Caption")
	}
	if _, ok := ss.Resolve("SFX"); !ok {
		t.Fatalf("expected builtin fallback for SFX")
	}
	// Unknown should fail
	if _, ok := ss.Resolve("Nonexistent"); ok {
		t.Fatalf("unexpected resolve of unknown style")
	}
}

func TestStyleSheet_NamesDeterministic(t *testing.T) {
	ss := NewStyleSheet()
	// Add a new custom style only at page level
	ss = ss.WithPage(map[string]TextStyle{"Narration": {Name: "Narration", Font: FontSpec{Family: "Comic Sans MS", SizePt: 8}}})
	names := ss.Names()
	if len(names) < 4 {
		t.Fatalf("expected at least 4 names, got %v", names)
	}
	// Builtins should come first in stable order
	if names[0] != "Dialogue" || names[1] != "Caption" || names[2] != "SFX" {
		t.Fatalf("unexpected initial order: %v", names)
	}
}
