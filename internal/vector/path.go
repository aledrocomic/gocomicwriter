/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

// Path commands and shapes.

type PathOp uint8

const (
	MoveTo PathOp = iota
	LineTo
	QuadTo  // quadratic bezier (cx, cy, x, y)
	CubicTo // cubic bezier (cx1, cy1, cx2, cy2, x, y)
	Close
)

type PathCmd struct {
	Op   PathOp
	Data [6]float32 // enough for cubic; unused slots are zero
}

type Path struct{ Cmds []PathCmd }

func (p *Path) MoveTo(x, y float32) {
	p.Cmds = append(p.Cmds, PathCmd{Op: MoveTo, Data: [6]float32{x, y}})
}
func (p *Path) LineTo(x, y float32) {
	p.Cmds = append(p.Cmds, PathCmd{Op: LineTo, Data: [6]float32{x, y}})
}
func (p *Path) QuadTo(cx, cy, x, y float32) {
	p.Cmds = append(p.Cmds, PathCmd{Op: QuadTo, Data: [6]float32{cx, cy, x, y}})
}
func (p *Path) CubicTo(cx1, cy1, cx2, cy2, x, y float32) {
	p.Cmds = append(p.Cmds, PathCmd{Op: CubicTo, Data: [6]float32{cx1, cy1, cx2, cy2, x, y}})
}
func (p *Path) Close() { p.Cmds = append(p.Cmds, PathCmd{Op: Close}) }

// Bounds returns an axis-aligned bounding box of the path using a simple
// approximation by considering control points. This is sufficient for UI layout
// and selection rectangles; exporters can use tighter bounds later.
func (p *Path) Bounds() Rect {
	minX, minY := float32(+1e9), float32(+1e9)
	maxX, maxY := float32(-1e9), float32(-1e9)
	cur := Pt{}
	for _, c := range p.Cmds {
		switch c.Op {
		case MoveTo, LineTo:
			x, y := c.Data[0], c.Data[1]
			cur = Pt{x, y}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		case QuadTo:
			pts := []Pt{cur, {c.Data[0], c.Data[1]}, {c.Data[2], c.Data[3]}}
			for _, p := range pts {
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
			cur = Pt{c.Data[2], c.Data[3]}
		case CubicTo:
			pts := []Pt{cur, {c.Data[0], c.Data[1]}, {c.Data[2], c.Data[3]}, {c.Data[4], c.Data[5]}}
			for _, p := range pts {
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
			cur = Pt{c.Data[4], c.Data[5]}
		case Close:
			// no-op for bounds
		}
	}
	if minX > maxX || minY > maxY {
		return Rect{}
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}
