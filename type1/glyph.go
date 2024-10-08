// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2024  Jochen Voss <voss@seehuhn.de>
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
	"fmt"

	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/postscript/funit"
)

// Glyph represents a glyph in a Type 1 font.
//
// TODO(voss): use float64 instead of funit.Int16?
type Glyph struct {
	Cmds   []GlyphOp
	HStem  []funit.Int16
	VStem  []funit.Int16
	WidthX float64
	WidthY float64
}

// NewGlyph creates a new glyph with the given name and width.
func (f *Font) NewGlyph(name string, width float64) *Glyph {
	g := &Glyph{
		WidthX: width,
	}
	f.Glyphs[name] = g
	return g
}

// MoveTo starts a new sub-path and moves the current point to (x, y).
// The previous sub-path, if any, is closed.
func (g *Glyph) MoveTo(x, y float64) {
	g.Cmds = append(g.Cmds, GlyphOp{
		Op:   OpMoveTo,
		Args: []float64{x, y},
	})
}

// LineTo adds a straight line to the current sub-path.
func (g *Glyph) LineTo(x, y float64) {
	g.Cmds = append(g.Cmds, GlyphOp{
		Op:   OpLineTo,
		Args: []float64{x, y},
	})
}

// CurveTo adds a cubic Bezier curve to the current sub-path.
func (g *Glyph) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	g.Cmds = append(g.Cmds, GlyphOp{
		Op:   OpCurveTo,
		Args: []float64{x1, y1, x2, y2, x3, y3},
	})
}

// ClosePath closes the current sub-path.
func (g *Glyph) ClosePath() {
	g.Cmds = append(g.Cmds, GlyphOp{Op: OpClosePath})
}

// BBox computes the bounding box of the glyph in glyph space units.
func (g *Glyph) BBox() rect.Rect {
	var left, right, top, bottom float64
	first := true
cmdLoop:
	for _, cmd := range g.Cmds {
		var x, y float64
		switch cmd.Op {
		case OpMoveTo, OpLineTo:
			x = cmd.Args[0]
			y = cmd.Args[1]
		case OpCurveTo:
			x = cmd.Args[4]
			y = cmd.Args[5]
		default:
			continue cmdLoop
		}
		if first || x < left {
			left = x
		}
		if first || x > right {
			right = x
		}
		if first || y < bottom {
			bottom = y
		}
		if first || y > top {
			top = y
		}
		first = false
	}
	return rect.Rect{
		LLx: left,
		LLy: bottom,
		URx: right,
		URy: top,
	}
}

// GlyphOp is a Type 1 glyph drawing command.
type GlyphOp struct {
	Op   GlyphOpType
	Args []float64
}

// GlyphOpType is the type of a Type 1 glyph drawing command.
type GlyphOpType byte

func (op GlyphOpType) String() string {
	switch op {
	case OpMoveTo:
		return "moveto"
	case OpLineTo:
		return "lineto"
	case OpCurveTo:
		return "curveto"
	case OpClosePath:
		return "closepath"
	default:
		return fmt.Sprintf("CommandType(%d)", op)
	}
}

const (
	// OpMoveTo tarts a new subpath at the given point.
	OpMoveTo GlyphOpType = iota + 1

	// OpLineTo appends a straight line segment from the previous point to the
	// given point.
	OpLineTo

	// OpCurveTo appends a Bezier curve segment from the previous point to the
	// given point.
	OpCurveTo

	// OpClosePath closes the current subpath by appending a straight line from
	// the current point to the starting point of the current subpath.  This
	// does not change the current point.
	OpClosePath
)

func (c GlyphOp) String() string {
	return fmt.Sprint("cmd", c.Args, c.Op)
}
