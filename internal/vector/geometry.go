/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

// Basic 2D geometry and transforms for resolution-independent drawing.
// Float values use float32 for compactness and to align with many UI libs.

import "math"

// Pt is a 2D point.
type Pt struct{ X, Y float32 }

// Size is a width/height pair.
type Size struct{ W, H float32 }

// Rect is an axis-aligned rectangle defined by min corner and size.
type Rect struct {
	X, Y float32
	W, H float32
}

func R(x, y, w, h float32) Rect { return Rect{X: x, Y: y, W: w, H: h} }

func (r Rect) Min() Pt { return Pt{r.X, r.Y} }
func (r Rect) Max() Pt { return Pt{r.X + r.W, r.Y + r.H} }

func (r Rect) Contains(p Pt) bool {
	return p.X >= r.X && p.Y >= r.Y && p.X <= r.X+r.W && p.Y <= r.Y+r.H
}

// Inset returns a rectangle inset by dx,dy on all sides (negative grows).
func (r Rect) Inset(dx, dy float32) Rect {
	return Rect{X: r.X + dx, Y: r.Y + dy, W: r.W - 2*dx, H: r.H - 2*dy}
}

// Union returns the minimal rect containing both.
func (r Rect) Union(o Rect) Rect {
	minX := min(r.X, o.X)
	minY := min(r.Y, o.Y)
	maxX := max(r.X+r.W, o.X+o.W)
	maxY := max(r.Y+r.H, o.Y+o.H)
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

// Affine2D represents a 2D affine transform as matrix:
// | a c e |
// | b d f |
// | 0 0 1 |
// stored as [a b c d e f].
type Affine2D struct{ A, B, C, D, E, F float32 }

var Identity = Affine2D{A: 1, D: 1}

func (m Affine2D) Mul(n Affine2D) Affine2D {
	return Affine2D{
		A: m.A*n.A + m.C*n.B,
		B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D,
		D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E,
		F: m.B*n.E + m.D*n.F + m.F,
	}
}

func (m Affine2D) Apply(p Pt) Pt {
	return Pt{
		X: m.A*p.X + m.C*p.Y + m.E,
		Y: m.B*p.X + m.D*p.Y + m.F,
	}
}

func Translate(tx, ty float32) Affine2D { return Affine2D{A: 1, D: 1, E: tx, F: ty} }
func Scale(sx, sy float32) Affine2D     { return Affine2D{A: sx, D: sy} }
func Rotate(rad float32) Affine2D {
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))
	return Affine2D{A: c, B: s, C: -s, D: c}
}

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// FloatRound rounds v to n decimal places deterministically.
func FloatRound(v float32, places int) float32 {
	if places < 0 {
		return v
	}
	pow := float32(math.Pow(10, float64(places)))
	return float32(math.Round(float64(v*pow))) / pow
}
