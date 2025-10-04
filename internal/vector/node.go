/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

// Node is a scene-graph item that can be rendered by different backends.
// It supports basic transforms, styling, bounds, and hit-testing.

type Node interface {
	Bounds() Rect
	Transform() Affine2D
	SetTransform(Affine2D)
	Fill() Fill
	Stroke() Stroke
	SetFill(Fill)
	SetStroke(Stroke)
	Hit(p Pt) bool
}

type baseNode struct {
	xf     Affine2D
	fill   Fill
	stroke Stroke
}

func (b *baseNode) Transform() Affine2D     { return b.xf }
func (b *baseNode) SetTransform(m Affine2D) { b.xf = m }
func (b *baseNode) Fill() Fill              { return b.fill }
func (b *baseNode) Stroke() Stroke          { return b.stroke }
func (b *baseNode) SetFill(f Fill)          { b.fill = f }
func (b *baseNode) SetStroke(s Stroke)      { b.stroke = s }

// Backwards-compatible unexported setters used by some tests
func (b *baseNode) setFill(f Fill)     { b.fill = f }
func (b *baseNode) setStroke(s Stroke) { b.stroke = s }

// RectNode draws an axis-aligned rectangle before transform.
type RectNode struct {
	baseNode
	rect Rect
}

func NewRect(r Rect, f Fill, s Stroke) *RectNode {
	return &RectNode{baseNode: baseNode{xf: Identity, fill: f, stroke: s}, rect: r}
}

func (n *RectNode) Bounds() Rect {
	// approximate by transforming 4 corners
	minX, minY := float32(+1e9), float32(+1e9)
	maxX, maxY := float32(-1e9), float32(-1e9)
	corners := []Pt{{n.rect.X, n.rect.Y}, {n.rect.X + n.rect.W, n.rect.Y}, {n.rect.X, n.rect.Y + n.rect.H}, {n.rect.X + n.rect.W, n.rect.Y + n.rect.H}}
	for _, c := range corners {
		p := n.xf.Apply(c)
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

func (n *RectNode) Hit(p Pt) bool {
	// inverse-transform p by approximate inverse (assume no shear singularity)
	inv := invert(n.xf)
	q := inv.Apply(p)
	return n.rect.Contains(q)
}

// EllipseNode represents an ellipse inside rect.
type EllipseNode struct {
	baseNode
	rect Rect
}

func NewEllipse(r Rect, f Fill, s Stroke) *EllipseNode {
	return &EllipseNode{baseNode: baseNode{xf: Identity, fill: f, stroke: s}, rect: r}
}

func (n *EllipseNode) Bounds() Rect { return n.RectBounds() }
func (n *EllipseNode) RectBounds() Rect {
	// same bbox as the rect pre-transform, but apply transform on 4 corners for safety
	minX, minY := float32(+1e9), float32(+1e9)
	maxX, maxY := float32(-1e9), float32(-1e9)
	corners := []Pt{{n.rect.X, n.rect.Y}, {n.rect.X + n.rect.W, n.rect.Y}, {n.rect.X, n.rect.Y + n.rect.H}, {n.rect.X + n.rect.W, n.rect.Y + n.rect.H}}
	for _, c := range corners {
		p := n.xf.Apply(c)
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

func (n *EllipseNode) Hit(p Pt) bool {
	inv := invert(n.xf)
	q := inv.Apply(p)
	// point-in-ellipse: ((x-cx)/rx)^2 + ((y-cy)/ry)^2 <= 1
	cx := n.rect.X + n.rect.W/2
	ry := n.rect.Y + n.rect.H/2
	rx := n.rect.W / 2
	ryr := n.rect.H / 2
	if rx == 0 || ryr == 0 {
		return false
	}
	dx := (q.X - cx) / rx
	dy := (q.Y - ry) / ryr
	return dx*dx+dy*dy <= 1
}

// RoundedRectNode uses uniform radii for simplicity.
type RoundedRectNode struct {
	baseNode
	rect Rect
	r    float32
}

func NewRoundedRect(r Rect, radius float32, f Fill, s Stroke) *RoundedRectNode {
	return &RoundedRectNode{baseNode: baseNode{xf: Identity, fill: f, stroke: s}, rect: r, r: radius}
}

func (n *RoundedRectNode) Bounds() Rect { return NewRect(n.rect, n.fill, n.stroke).Bounds() }
func (n *RoundedRectNode) Hit(p Pt) bool {
	inv := invert(n.xf)
	q := inv.Apply(p)
	// quick reject outside rect
	if !n.rect.Contains(q) {
		return false
	}
	// If inside the core (rect inset by r), it's a hit
	core := n.rect.Inset(n.r, n.r)
	if core.W > 0 && core.H > 0 && core.Contains(q) {
		return true
	}
	// Otherwise test the four quarter-circles
	cx := []float32{n.rect.X + n.r, n.rect.X + n.rect.W - n.r}
	cy := []float32{n.rect.Y + n.r, n.rect.Y + n.rect.H - n.r}
	r2 := n.r * n.r
	for _, x := range cx {
		for _, y := range cy {
			dx := q.X - x
			dy := q.Y - y
			if dx*dx+dy*dy <= r2 {
				return true
			}
		}
	}
	return false
}

// PathNode references a path geometry.
type PathNode struct {
	baseNode
	path Path
	bbox Rect // cached approx bounds
}

func NewPath(p Path, f Fill, s Stroke) *PathNode {
	return &PathNode{baseNode: baseNode{xf: Identity, fill: f, stroke: s}, path: p, bbox: p.Bounds()}
}

func (n *PathNode) Bounds() Rect {
	// transform bbox corners
	minX, minY := float32(+1e9), float32(+1e9)
	maxX, maxY := float32(-1e9), float32(-1e9)
	b := n.bbox
	corners := []Pt{{b.X, b.Y}, {b.X + b.W, b.Y}, {b.X, b.Y + b.H}, {b.X + b.W, b.Y + b.H}}
	for _, c := range corners {
		p := n.xf.Apply(c)
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

func (n *PathNode) Hit(p Pt) bool {
	// Simple bbox hit; exporters or advanced tools can do point-in-path later.
	inv := invert(n.xf)
	q := inv.Apply(p)
	return n.bbox.Contains(q)
}

// Group is a container for child nodes with its own transform.
type Group struct {
	baseNode
	Children []Node
}

func NewGroup(children ...Node) *Group {
	g := &Group{baseNode: baseNode{xf: Identity}}
	g.Children = append(g.Children, children...)
	return g
}

func (g *Group) Bounds() Rect {
	var b Rect
	first := true
	for _, c := range g.Children {
		cb := c.Bounds()
		if first {
			b = cb
			first = false
		} else {
			b = b.Union(cb)
		}
	}
	return b
}

func (g *Group) Hit(p Pt) bool {
	inv := invert(g.xf)
	q := inv.Apply(p)
	for i := len(g.Children) - 1; i >= 0; i-- { // top-most first
		if g.Children[i].Hit(q) {
			return true
		}
	}
	return false
}

// invert computes the inverse of an affine matrix (if invertible).
func invert(m Affine2D) Affine2D {
	det := m.A*m.D - m.B*m.C
	if det == 0 {
		return Identity
	}
	invDet := 1 / det
	return Affine2D{
		A: m.D * invDet,
		B: -m.B * invDet,
		C: -m.C * invDet,
		D: m.A * invDet,
		E: (m.C*m.F - m.D*m.E) * invDet,
		F: (m.B*m.E - m.A*m.F) * invDet,
	}
}
