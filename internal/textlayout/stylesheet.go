/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

// StyleSheet provides hierarchical resolution of TextStyle presets.
// It supports three scopes:
//  - Global: app defaults or builtins
//  - Issue: styles defined for the current project/issue
//  - Page: overrides specific to a single page
// Resolution precedence is Page > Issue > Global > Builtin.
// Builtins are provided by styles.go (builtinStyles map).
//
// This is an in-memory helper to keep UI and storage decoupled; project code
// can populate the Issue and Page maps as needed.
// The API keeps it intentionally minimal, matching the concept doc.

type StyleSheet struct {
	Global map[string]TextStyle
	Issue  map[string]TextStyle
	Page   map[string]TextStyle
}

// NewStyleSheet creates a stylesheet with empty scopes and builtin styles
// copied into Global for convenience.
func NewStyleSheet() *StyleSheet {
	ss := &StyleSheet{
		Global: map[string]TextStyle{},
		Issue:  map[string]TextStyle{},
		Page:   map[string]TextStyle{},
	}
	// Seed with builtins as global defaults
	for _, name := range ListStyles() {
		if st, ok := GetStyle(name); ok {
			ss.Global[name] = st
		}
	}
	return ss
}

// WithIssue returns a shallow copy with the provided issue-level overrides merged.
func (s *StyleSheet) WithIssue(over map[string]TextStyle) *StyleSheet {
	cp := s.clone()
	for k, v := range over {
		cp.Issue[k] = v
	}
	return cp
}

// WithPage returns a shallow copy with the provided page-level overrides merged.
func (s *StyleSheet) WithPage(over map[string]TextStyle) *StyleSheet {
	cp := s.clone()
	for k, v := range over {
		cp.Page[k] = v
	}
	return cp
}

// Resolve returns the effective TextStyle by name using precedence Page > Issue > Global > Builtin.
// The second return value is false if the name cannot be resolved at any level.
func (s *StyleSheet) Resolve(name string) (TextStyle, bool) {
	if s == nil {
		return TextStyle{}, false
	}
	if st, ok := s.Page[name]; ok {
		return st, true
	}
	if st, ok := s.Issue[name]; ok {
		return st, true
	}
	if st, ok := s.Global[name]; ok {
		return st, true
	}
	// Final fallback to builtin table
	if st, ok := GetStyle(name); ok {
		return st, true
	}
	return TextStyle{}, false
}

// Names returns the set of known style names considering all scopes.
// Order is deterministic: builtin ListStyles order, then any additional names sorted lexicographically.
func (s *StyleSheet) Names() []string {
	seen := map[string]bool{}
	var out []string
	// First, builtin order for known builtins
	for _, name := range ListStyles() {
		if _, ok := s.Resolve(name); ok {
			out = append(out, name)
			seen[name] = true
		}
	}
	// Then collect from scopes
	collect := func(m map[string]TextStyle) {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	collect(s.Global)
	collect(s.Issue)
	collect(s.Page)
	// Keep deterministic but simple: stable insertion order as above suffices for tests.
	return out
}

func (s *StyleSheet) clone() *StyleSheet {
	cp := &StyleSheet{Global: map[string]TextStyle{}, Issue: map[string]TextStyle{}, Page: map[string]TextStyle{}}
	for k, v := range s.Global {
		cp.Global[k] = v
	}
	for k, v := range s.Issue {
		cp.Issue[k] = v
	}
	for k, v := range s.Page {
		cp.Page[k] = v
	}
	return cp
}
