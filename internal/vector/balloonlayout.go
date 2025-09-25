/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import (
	"math"
	sortpkg "sort"
)

// SuggestOptions controls the auto layout suggestion for balloons.
// All units are in the same coordinate space as Rect.
// The algorithm is deterministic for identical inputs.
//
// ReadingDirection influences search ordering when no anchor is provided.
// Use "ltr" or "rtl" (default ltr).
//
// Padding is added around the content size to form the balloon rect.
// Margin is the clearance to keep from panel edges.
// GridStep controls the search granularity in pixels; lower values are slower but find tighter fits.
//
// Anchor, when provided (HasAnchor=true), biases search toward positions whose center is closest
// to the anchor (e.g., speaker location) while still avoiding collisions.
// If no collision-free placement exists, the algorithm returns the least-overlapping candidate.
// In all cases, the returned rect is clamped to be within the panel inset by Margin.
// Attempts is the number of candidates evaluated.
type SuggestOptions struct {
	ReadingDirection string
	Padding          float32
	Margin           float32
	GridStep         float32
	Anchor           Pt
	HasAnchor        bool
}

// SuggestBalloonLayout proposes a placement Rect for a balloon given:
// - panel bounds
// - content size (text box, before padding)
// - obstacles to avoid (other balloons/captions/SFX as rects)
// The returned Rect includes the padding.
func SuggestBalloonLayout(panel Rect, content Size, obstacles []Rect, opts SuggestOptions) (Rect, int) {
	// defaults
	if opts.Padding <= 0 {
		opts.Padding = 8
	}
	if opts.Margin <= 0 {
		opts.Margin = 8
	}
	if opts.GridStep <= 0 {
		opts.GridStep = 8
	}
	if opts.ReadingDirection == "" {
		opts.ReadingDirection = "ltr"
	}

	inner := panel.Inset(opts.Margin, opts.Margin)
	bw := max(0, content.W+2*opts.Padding)
	bh := max(0, content.H+2*opts.Padding)
	if bw > inner.W {
		bw = inner.W
	}
	if bh > inner.H {
		bh = inner.H
	}

	// Candidate grid of potential top-left positions within inner bounds.
	x0 := inner.X
	y0 := inner.Y
	x1 := inner.X + inner.W - bw
	y1 := inner.Y + inner.H - bh
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}

	var candidates []Rect
	// Build grid according to GridStep (ensure last cell at x1/y1 is included).
	for y := y0; ; y += opts.GridStep {
		if y > y1 {
			y = y1
		}
		if opts.ReadingDirection == "rtl" {
			for x := x1; x >= x0-1e-3; x -= opts.GridStep {
				if x < x0 {
					x = x0
				}
				candidates = append(candidates, R(FloatRound(x, 3), FloatRound(y, 3), FloatRound(bw, 3), FloatRound(bh, 3)))
				if x == x0 {
					break
				}
			}
		} else {
			for x := x0; x <= x1+1e-3; x += opts.GridStep {
				if x > x1 {
					x = x1
				}
				candidates = append(candidates, R(FloatRound(x, 3), FloatRound(y, 3), FloatRound(bw, 3), FloatRound(bh, 3)))
				if x == x1 {
					break
				}
			}
		}
		if y == y1 {
			break
		}
	}

	// If we have an anchor, sort candidates by distance to anchor first (stable sort preserves row order ties).
	if opts.HasAnchor {
		sortpkg.SliceStable(candidates, func(i, j int) bool {
			ci := rectCenter(candidates[i])
			cj := rectCenter(candidates[j])
			di := hypot(ci.X-opts.Anchor.X, ci.Y-opts.Anchor.Y)
			dj := hypot(cj.X-opts.Anchor.X, cj.Y-opts.Anchor.Y)
			if di == dj { // tie-break by y,x to keep deterministic
				if candidates[i].Y == candidates[j].Y {
					return candidates[i].X < candidates[j].X
				}
				return candidates[i].Y < candidates[j].Y
			}
			return di < dj
		})
	}

	bestRect := candidates[0]
	bestCost := float32(+1e9)
	attempts := 0

	for _, c := range candidates {
		attempts++
		ovArea := totalOverlapArea(c, obstacles)
		if ovArea <= 0.0001 { // no collision
			// Early return on the first collision-free candidate in the current ordering
			bestRect = c
			break
		}
		// No perfect fit; compute cost and keep the best
		cost := ovArea * 10_000 // strong penalty in px^2
		// Distance to anchor (if any) — prefer closer to anchor
		if opts.HasAnchor {
			cc := rectCenter(c)
			cost += hypot(cc.X-opts.Anchor.X, cc.Y-opts.Anchor.Y)
		}
		// Reading preference: prefer higher (smaller y); tiny bias to left for LTR and right for RTL
		cost += c.Y * 0.01
		if opts.ReadingDirection == "ltr" {
			cost += c.X * 0.001
		} else {
			// prefer right side: smaller bias for larger X (invert by using remaining distance)
			cost += (inner.X + inner.W - (c.X + c.W)) * 0.001
		}
		if cost < bestCost {
			bestCost = cost
			bestRect = c
		}
	}

	// Clamp to inner bounds just in case of numeric drift.
	bestRect = clampRectTo(bestRect, inner)
	return bestRect, attempts
}

// --- helpers ---

func rectCenter(r Rect) Pt { return Pt{r.X + r.W/2, r.Y + r.H/2} }

func hypot(dx, dy float32) float32 { return float32(math.Hypot(float64(dx), float64(dy))) }

func clampRectTo(r Rect, bounds Rect) Rect {
	if r.X < bounds.X {
		r.X = bounds.X
	}
	if r.Y < bounds.Y {
		r.Y = bounds.Y
	}
	if r.X+r.W > bounds.X+bounds.W {
		r.X = bounds.X + bounds.W - r.W
	}
	if r.Y+r.H > bounds.Y+bounds.H {
		r.Y = bounds.Y + bounds.H - r.H
	}
	return r
}

func (r Rect) Intersects(o Rect) bool {
	return r.X < o.X+o.W && r.X+r.W > o.X && r.Y < o.Y+o.H && r.Y+r.H > o.Y
}

func (r Rect) Intersection(o Rect) Rect {
	x0 := max(r.X, o.X)
	y0 := max(r.Y, o.Y)
	x1 := min(r.X+r.W, o.X+o.W)
	y1 := min(r.Y+r.H, o.Y+o.H)
	w := x1 - x0
	h := y1 - y0
	if w <= 0 || h <= 0 {
		return R(0, 0, 0, 0)
	}
	return R(x0, y0, w, h)
}

func area(r Rect) float32 {
	if r.W <= 0 || r.H <= 0 {
		return 0
	}
	return r.W * r.H
}

func totalOverlapArea(r Rect, obstacles []Rect) float32 {
	var sum float32
	for _, o := range obstacles {
		if r.Intersects(o) {
			sum += area(r.Intersection(o))
		}
	}
	return sum
}
