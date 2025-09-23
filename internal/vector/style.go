/*
 * Copyright (c) 2025 by Alexander Drost, Oldenburg, Germany.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.  You may obtain a copy of the License at
 *   http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.
 */

package vector

// Styles and paint definitions.

type Color struct{ R, G, B, A uint8 }

var (
	Black       = Color{0, 0, 0, 255}
	White       = Color{255, 255, 255, 255}
	Transparent = Color{0, 0, 0, 0}
)

type FillRule uint8

const (
	NonZero FillRule = iota
	EvenOdd
)

type Fill struct {
	Color   Color
	Rule    FillRule
	Enabled bool
}

type LineCap uint8

const (
	CapButt LineCap = iota
	CapRound
	CapSquare
)

type LineJoin uint8

const (
	JoinMiter LineJoin = iota
	JoinRound
	JoinBevel
)

type Stroke struct {
	Color    Color
	Width    float32
	Cap      LineCap
	Join     LineJoin
	MiterLim float32
	Enabled  bool
}
