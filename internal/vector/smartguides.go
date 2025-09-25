/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

// Smart guides and snapping helpers for interactive tools (balloons, captions, etc.).
// These utilities are UI-agnostic and deterministic to enable unit testing and
// reuse across different frontends.

import "math"

// SnapOptions controls which guide candidates are considered and the threshold.
type SnapOptions struct {
	// Threshold is the maximum distance (in the same units as Rect) at which
	// snapping occurs. Typical UI values are 6–8 points/pixels.
	Threshold float32
	// Snap to edges (left, right, top, bottom)
	SnapToEdges bool
	// Snap to centers (cx, cy)
	SnapToCenters bool
}

// Anchor represents a static reference rect (e.g., a panel bounds or another balloon).
// Weight can be used to bias selection when distances tie (higher = preferred).
// When uncertain, set Weight to 1.
type Anchor struct {
	Rect   Rect
	Weight float32
}

// GuideLine describes a visual guide generated during a snap alignment.
// Orientation is "vertical" or "horizontal".
// Kind indicates which features aligned: "edge" or "center".
// From and To denote the guide extents for rendering.
// Position is the x (vertical) or y (horizontal) coordinate of the guide.
// For deterministic behavior, values are rounded to 3 decimal places.
type GuideLine struct {
	Orientation string
	Kind        string
	Position    float32
	From        Pt
	To          Pt
}

// ComputeSmartGuides computes snapping adjustments for a moving rectangle
// against a set of anchors. It returns the snapped rectangle and any guide
// lines to render for visual feedback. Snapping happens independently in X and Y.
func ComputeSmartGuides(moving Rect, anchors []Anchor, opts SnapOptions) (Rect, []GuideLine) {
	if opts.Threshold <= 0 {
		opts.Threshold = 6
	}
	var guides []GuideLine

	// Horizontal (X) snapping candidates: left, centerX, right
	bestDX, bestDXDist, bestDXGuide := float32(0), float32(+1e9), (GuideLine{})
	// Vertical (Y) snapping: top, centerY, bottom
	bestDY, bestDYDist, bestDYGuide := float32(0), float32(+1e9), (GuideLine{})

	mxL, mxR, mxT, mxB, mxCX, mxCY := moving.X, moving.X+moving.W, moving.Y, moving.Y+moving.H, moving.X+moving.W/2, moving.Y+moving.H/2

	for _, a := range anchors {
		axL, axR, axT, axB := a.Rect.X, a.Rect.X+a.Rect.W, a.Rect.Y, a.Rect.Y+a.Rect.H
		axCX, axCY := a.Rect.X+a.Rect.W/2, a.Rect.Y+a.Rect.H/2

		// X axis
		if opts.SnapToEdges {
			// left-to-left
			d := mxL - axL
			considerX(&bestDX, &bestDXDist, &bestDXGuide, d, opts.Threshold, a.Weight, guideForVertical(axL, moving, a.Rect, "edge"))
			// right-to-right
			d = mxR - axR
			considerX(&bestDX, &bestDXDist, &bestDXGuide, d, opts.Threshold, a.Weight, guideForVertical(axR, moving, a.Rect, "edge"))
			// left-to-right (abut) and right-to-left
			d = mxL - axR
			considerX(&bestDX, &bestDXDist, &bestDXGuide, d, opts.Threshold, a.Weight, guideForVertical(axR, moving, a.Rect, "edge"))
			d = mxR - axL
			considerX(&bestDX, &bestDXDist, &bestDXGuide, d, opts.Threshold, a.Weight, guideForVertical(axL, moving, a.Rect, "edge"))
		}
		if opts.SnapToCenters {
			d := mxCX - axCX
			considerX(&bestDX, &bestDXDist, &bestDXGuide, d, opts.Threshold, a.Weight, guideForVertical(axCX, moving, a.Rect, "center"))
		}

		// Y axis
		if opts.SnapToEdges {
			// top-to-top
			d := mxT - axT
			considerY(&bestDY, &bestDYDist, &bestDYGuide, d, opts.Threshold, a.Weight, guideForHorizontal(axT, moving, a.Rect, "edge"))
			// bottom-to-bottom
			d = mxB - axB
			considerY(&bestDY, &bestDYDist, &bestDYGuide, d, opts.Threshold, a.Weight, guideForHorizontal(axB, moving, a.Rect, "edge"))
			// top-to-bottom and bottom-to-top
			d = mxT - axB
			considerY(&bestDY, &bestDYDist, &bestDYGuide, d, opts.Threshold, a.Weight, guideForHorizontal(axB, moving, a.Rect, "edge"))
			d = mxB - axT
			considerY(&bestDY, &bestDYDist, &bestDYGuide, d, opts.Threshold, a.Weight, guideForHorizontal(axT, moving, a.Rect, "edge"))
		}
		if opts.SnapToCenters {
			d := mxCY - axCY
			considerY(&bestDY, &bestDYDist, &bestDYGuide, d, opts.Threshold, a.Weight, guideForHorizontal(axCY, moving, a.Rect, "center"))
		}
	}

	snapped := moving
	if bestDXDist <= opts.Threshold {
		snapped.X = FloatRound(moving.X-bestDX, 3)
		guides = append(guides, bestDXGuide)
	}
	if bestDYDist <= opts.Threshold {
		snapped.Y = FloatRound(moving.Y-bestDY, 3)
		guides = append(guides, bestDYGuide)
	}
	return snapped, guides
}

func considerX(bestDX *float32, bestD *float32, bestGuide *GuideLine, delta float32, threshold float32, weight float32, g GuideLine) {
	dist := float32(math.Abs(float64(delta)))
	if dist > threshold {
		return
	}
	score := dist / max32(1, weight)
	if score < *bestD {
		*bestD = dist
		*bestDX = delta
		*bestGuide = g
	}
}

func considerY(bestDY *float32, bestD *float32, bestGuide *GuideLine, delta float32, threshold float32, weight float32, g GuideLine) {
	dist := float32(math.Abs(float64(delta)))
	if dist > threshold {
		return
	}
	score := dist / max32(1, weight)
	if score < *bestD {
		*bestD = dist
		*bestDY = delta
		*bestGuide = g
	}
}

func guideForVertical(x float32, a Rect, b Rect, kind string) GuideLine {
	minY := min(a.Y, b.Y)
	maxY := max(a.Y+a.H, b.Y+b.H)
	x = FloatRound(x, 3)
	return GuideLine{
		Orientation: "vertical",
		Kind:        kind,
		Position:    x,
		From:        Pt{x, minY},
		To:          Pt{x, maxY},
	}
}

func guideForHorizontal(y float32, a Rect, b Rect, kind string) GuideLine {
	minX := min(a.X, b.X)
	maxX := max(a.X+a.W, b.X+b.W)
	y = FloatRound(y, 3)
	return GuideLine{
		Orientation: "horizontal",
		Kind:        kind,
		Position:    y,
		From:        Pt{minX, y},
		To:          Pt{maxX, y},
	}
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
