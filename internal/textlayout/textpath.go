/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package textlayout

// Text-on-path utilities that remain rendering-agnostic.
// These produce deterministic glyph poses (position + tangent angle)
// that a renderer can use to draw text along a vector path.

import (
	"math"
	"unicode/utf8"

	"golang.org/x/image/font"

	"gocomicwriter/internal/vector"
)

// GlyphPose is the placement of one glyph along a path.
// Angle is in radians and represents the local tangent orientation.
type GlyphPose struct {
	Rune    rune
	Pos     vector.Pt
	Angle   float32
	Advance float32 // advance used for this glyph
}

// LayoutOnPath lays out the given text along the provided path using the
// font Provider for measurement. Tracking is applied between glyphs.
// startOffset specifies the distance (in px) from the path start before the
// first glyph center is placed.
func LayoutOnPath(provider Provider, text string, fontSpec FontSpec, path *vector.Path, tracking, startOffset float32) ([]GlyphPose, float32) {
	if provider == nil {
		provider = BasicProvider{}
	}
	if path == nil || len(path.Cmds) == 0 || text == "" {
		return nil, 0
	}
	poly := flattenPath(path, 12)
	if len(poly) < 2 {
		return nil, 0
	}
	segs, total := buildSegments(poly)
	face, _ := provider.Resolve(fontSpec)
	d := &font.Drawer{Face: face}
	var poses []GlyphPose
	advanceSoFar := startOffset
	prevRune := rune(-1)
	for i, off := 0, 0; off < len(text); {
		r, sz := utf8.DecodeRuneInString(text[off:])
		if r == utf8.RuneError && sz == 1 {
			break
		}
		// Measure advance of this glyph with optional kerning context.
		adv := measureRuneAdvance(d, prevRune, r)
		if i > 0 && tracking != 0 {
			adv += tracking
		}
		// Center position of this glyph along the path.
		centerS := advanceSoFar + adv/2
		pos, angle, ok := pointAt(segs, centerS)
		if !ok {
			break // remaining glyphs do not fit
		}
		poses = append(poses, GlyphPose{Rune: r, Pos: pos, Angle: angle, Advance: adv})
		advanceSoFar += adv
		prevRune = r
		i++
		off += sz
	}
	return poses, total
}

// measureRuneAdvance measures one rune's advance and applies kerning against prev if supported by the face.
// We approximate kerning by measuring the pair and subtracting the previous advance.
func measureRuneAdvance(d *font.Drawer, prev, cur rune) float32 {
	if prev <= 0 {
		return float32(d.MeasureString(string(cur)) >> 6)
	}
	pair := string([]rune{prev, cur})
	pairAdv := float32(d.MeasureString(pair) >> 6)
	prevAdv := float32(d.MeasureString(string(prev)) >> 6)
	adv := pairAdv - prevAdv
	if adv <= 0 {
		// Fallback if font does not support kerning
		adv = float32(d.MeasureString(string(cur)) >> 6)
	}
	return adv
}

// segment represents a straight line with cached length and angle.
type segment struct {
	A, B       vector.Pt
	Len, Angle float32
}

func buildSegments(pts []vector.Pt) ([]segment, float32) {
	segs := make([]segment, 0, len(pts)-1)
	var total float32
	for i := 0; i < len(pts)-1; i++ {
		a, b := pts[i], pts[i+1]
		dx := b.X - a.X
		dy := b.Y - a.Y
		segLen := float32(math.Hypot(float64(dx), float64(dy)))
		if segLen <= 0 {
			continue
		}
		ang := float32(math.Atan2(float64(dy), float64(dx)))
		segs = append(segs, segment{A: a, B: b, Len: segLen, Angle: ang})
		total += segLen
	}
	return segs, total
}

// pointAt returns the position and tangent angle at distance s along the polyline.
func pointAt(segs []segment, s float32) (vector.Pt, float32, bool) {
	var acc float32
	for _, sg := range segs {
		if s <= acc+sg.Len {
			t := (s - acc) / sg.Len
			return vector.Pt{X: sg.A.X + (sg.B.X-sg.A.X)*t, Y: sg.A.Y + (sg.B.Y-sg.A.Y)*t}, sg.Angle, true
		}
		acc += sg.Len
	}
	return vector.Pt{}, 0, false
}

// flattenPath approximates curves with fixed-step sampling to a polyline.
// stepsCurve defines the number of segments used for curves; lines remain 1.
func flattenPath(p *vector.Path, stepsCurve int) []vector.Pt {
	if stepsCurve < 2 {
		stepsCurve = 2
	}
	var pts []vector.Pt
	var cur, start vector.Pt
	for i, c := range p.Cmds {
		switch c.Op {
		case vector.MoveTo:
			cur = vector.Pt{X: c.Data[0], Y: c.Data[1]}
			start = cur
			pts = append(pts, cur)
		case vector.LineTo:
			next := vector.Pt{X: c.Data[0], Y: c.Data[1]}
			pts = append(pts, next)
			cur = next
		case vector.QuadTo:
			c1 := vector.Pt{X: c.Data[0], Y: c.Data[1]}
			end := vector.Pt{X: c.Data[2], Y: c.Data[3]}
			for s := 1; s <= stepsCurve; s++ {
				t := float32(s) / float32(stepsCurve)
				pt := quadAt(cur, c1, end, t)
				pts = append(pts, pt)
			}
			cur = end
		case vector.CubicTo:
			c1 := vector.Pt{X: c.Data[0], Y: c.Data[1]}
			c2 := vector.Pt{X: c.Data[2], Y: c.Data[3]}
			end := vector.Pt{X: c.Data[4], Y: c.Data[5]}
			for s := 1; s <= stepsCurve; s++ {
				t := float32(s) / float32(stepsCurve)
				pt := cubicAt(cur, c1, c2, end, t)
				pts = append(pts, pt)
			}
			cur = end
		case vector.Close:
			// Close the current subpath if needed.
			if i > 0 && (cur.X != start.X || cur.Y != start.Y) {
				pts = append(pts, start)
				cur = start
			}
		}
	}
	return pts
}

func quadAt(p0, p1, p2 vector.Pt, t float32) vector.Pt {
	// B(t) = (1-t)^2 p0 + 2(1-t)t p1 + t^2 p2
	u := 1 - t
	return vector.Pt{
		X: u*u*p0.X + 2*u*t*p1.X + t*t*p2.X,
		Y: u*u*p0.Y + 2*u*t*p1.Y + t*t*p2.Y,
	}
}

func cubicAt(p0, p1, p2, p3 vector.Pt, t float32) vector.Pt {
	// B(t) = (1-t)^3 p0 + 3(1-t)^2 t p1 + 3(1-t) t^2 p2 + t^3 p3
	u := 1 - t
	u2 := u * u
	t2 := t * t
	return vector.Pt{
		X: u2*u*p0.X + 3*u2*t*p1.X + 3*u*t2*p2.X + t2*t*p3.X,
		Y: u2*u*p0.Y + 3*u2*t*p1.Y + 3*u*t2*p2.Y + t2*t*p3.Y,
	}
}
