// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package type1

import (
	"math"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

// Glyph coordinates, stem edges and widths are held as multiples of
// 1/gridScale.  Sums and differences of such values are again multiples of
// 1/gridScale and are computed exactly in float64, and every grid value
// below [maxOperand] can be written to a charstring exactly as an integer or
// as the quotient of two integers.  Together, this makes a read-write-read
// cycle reproduce glyphs exactly.
//
// gridScale is the largest power of two below [maxOperand], so that the
// divisor of a div operator is never a large number.  Some interpreters
// decode a large divisor incorrectly.
const gridScale = 1 << 14

// snap rounds x to the nearest multiple of 1/gridScale.
func snap(x float64) float64 {
	return math.Round(x*gridScale) / gridScale
}

// numberUsable reports whether x can be written as a charstring operand.
func numberUsable(x float64) bool {
	return math.Abs(x) < maxOperand
}

// outlineUsable reports whether every step between consecutive points of
// the outline, starting from the origin, can be written as charstring
// operands.  These are exactly the differences which encodeCharString
// writes.
func outlineUsable(o *path.Data) bool {
	if o == nil {
		return true
	}
	var prev vec.Vec2
	for _, p := range o.Coords {
		d := p.Sub(prev)
		if !numberUsable(d.X) || !numberUsable(d.Y) {
			return false
		}
		prev = p
	}
	return true
}

// snapped returns a copy of the glyph with all values rounded to the grid.
func (g *Glyph) snapped() *Glyph {
	snapAll := func(xx []float64) []float64 {
		if xx == nil {
			return nil
		}
		res := make([]float64, len(xx))
		for i, x := range xx {
			res[i] = snap(x)
		}
		return res
	}

	res := &Glyph{
		HStem:  snapAll(g.HStem),
		VStem:  snapAll(g.VStem),
		WidthX: snap(g.WidthX),
		WidthY: snap(g.WidthY),
	}
	if g.Outline != nil {
		res.Outline = &path.Data{
			Cmds:   g.Outline.Cmds,
			Coords: make([]vec.Vec2, len(g.Outline.Coords)),
		}
		for i, p := range g.Outline.Coords {
			res.Outline.Coords[i] = vec.Vec2{X: snap(p.X), Y: snap(p.Y)}
		}
	}
	return res
}
