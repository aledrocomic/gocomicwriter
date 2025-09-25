/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

import "math"

// TailOptions controls the generated tail geometry.
// Units are the same as the canvas (points/pixels).
// Deterministic results are ensured by rounding output points to 3 decimals.
type TailOptions struct {
	// BaseWidth is the width where the tail attaches to the balloon edge.
	BaseWidth float32
	// Length is the tail length measured from the base center towards the speaker anchor.
	// If the anchor is farther than Length, the tip will stop at Length; if nearer and
	// outside direction would point inside the balloon, the tip will extend outward by Length.
	Length float32
	// Curved, when true, produces a slightly curved tail using quadratic beziers.
	// For minimal deterministic implementation we default to triangular when false.
	Curved bool
}

// TailGeometry describes the generated tail points and its path.
type TailGeometry struct {
	BaseLeft   Pt
	BaseRight  Pt
	BaseCenter Pt
	Tip        Pt
	Angle      float32 // radians, direction from balloon center to anchor
	Side       string  // approximate side: left/right/top/bottom
	Path       Path
}

// ComputeBalloonTailEllipse creates a tail for an elliptical balloon defined by rect.
// The ellipse is inscribed in rect (rx = rect.W/2, ry = rect.H/2). The tail auto-orients
// towards the speaker anchor and exits at the ellipse point where the ray from center to
// anchor meets the boundary.
func ComputeBalloonTailEllipse(balloon Rect, anchor Pt, opts TailOptions) TailGeometry {
	// sanity defaults
	if opts.BaseWidth <= 0 {
		opts.BaseWidth = max(8, min(balloon.W, balloon.H)*0.1) // 10% of minor axis or 8
	}
	if opts.Length <= 0 {
		opts.Length = max(16, min(balloon.W, balloon.H)*0.2)
	}

	cx, cy := balloon.X+balloon.W/2, balloon.Y+balloon.H/2
	rx, ry := balloon.W/2, balloon.H/2
	vx, vy := anchor.X-cx, anchor.Y-cy

	// If anchor coincides with center, pick upward direction deterministically.
	if vx == 0 && vy == 0 {
		vy = -1
	}

	// Direction unit from center to anchor
	mag := float32(math.Hypot(float64(vx), float64(vy)))
	ux, uy := vx/mag, vy/mag

	// Distance from center to ellipse boundary along direction u.
	// d = 1 / sqrt((ux^2/rx^2) + (uy^2/ry^2))
	den := (ux*ux)/(rx*rx) + (uy*uy)/(ry*ry)
	if den == 0 {
		den = 1 // avoid div-by-zero; degenerate rect
	}
	d := 1 / float32(math.Sqrt(float64(den)))

	bc := Pt{X: FloatRound(cx+ux*d, 3), Y: FloatRound(cy+uy*d, 3)}

	// Perpendicular direction to form the base chord
	px, py := -uy, ux
	halfW := opts.BaseWidth / 2
	bl := Pt{X: FloatRound(bc.X+px*halfW, 3), Y: FloatRound(bc.Y+py*halfW, 3)}
	br := Pt{X: FloatRound(bc.X-px*halfW, 3), Y: FloatRound(bc.Y-py*halfW, 3)}

	// Decide tip placement.
	// If anchor lies in outward direction (from base along u) and is closer than Length, use anchor.
	// Otherwise, place the tip at fixed length along outward direction.
	// Determine if anchor is outside relative to the base direction by dot((anchor-bc), u).
	dot := (anchor.X-bc.X)*ux + (anchor.Y-bc.Y)*uy
	var tip Pt
	if dot > 0 {
		// Candidate tip towards anchor, but clamp by Length.
		distToAnchor := float32(math.Hypot(float64(anchor.X-bc.X), float64(anchor.Y-bc.Y)))
		if distToAnchor <= opts.Length {
			tip = Pt{X: FloatRound(anchor.X, 3), Y: FloatRound(anchor.Y, 3)}
		} else {
			tip = Pt{X: FloatRound(bc.X+ux*opts.Length, 3), Y: FloatRound(bc.Y+uy*opts.Length, 3)}
		}
	} else {
		// Anchor is inside or behind; extend outward by fixed length.
		tip = Pt{X: FloatRound(bc.X+ux*opts.Length, 3), Y: FloatRound(bc.Y+uy*opts.Length, 3)}
	}

	angle := float32(math.Atan2(float64(uy), float64(ux)))
	side := classifySide(ux, uy)

	var path Path
	if opts.Curved {
		// Use quadratic curves from base points to tip with a control point slightly offset
		// along the outgoing direction for a subtle curve. Deterministic small offset.
		off := opts.Length * 0.35
		cx1 := FloatRound(bc.X+ux*off+px*(halfW*0.4), 3)
		cy1 := FloatRound(bc.Y+uy*off+py*(halfW*0.4), 3)
		cx2 := FloatRound(bc.X+ux*off-px*(halfW*0.4), 3)
		cy2 := FloatRound(bc.Y+uy*off-py*(halfW*0.4), 3)

		path.MoveTo(bl.X, bl.Y)
		path.QuadTo(cx1, cy1, tip.X, tip.Y)
		path.LineTo(br.X, br.Y)
		path.QuadTo(cx2, cy2, bl.X, bl.Y)
		path.Close()
	} else {
		// Simple triangular tail
		path.MoveTo(bl.X, bl.Y)
		path.LineTo(tip.X, tip.Y)
		path.LineTo(br.X, br.Y)
		path.Close()
	}

	return TailGeometry{
		BaseLeft:   bl,
		BaseRight:  br,
		BaseCenter: bc,
		Tip:        tip,
		Angle:      angle,
		Side:       side,
		Path:       path,
	}
}

func classifySide(ux, uy float32) string {
	// Determine the dominant axis of the direction vector.
	ax, ay := float32(math.Abs(float64(ux))), float32(math.Abs(float64(uy)))
	if ax >= ay {
		if ux >= 0 {
			return "right"
		}
		return "left"
	}
	if uy >= 0 {
		return "bottom"
	}
	return "top"
}
