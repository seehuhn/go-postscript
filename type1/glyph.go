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

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
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

func (g *Glyph) IsBlank() bool {
	for _, cmd := range g.Cmds {
		if cmd.Op == OpLineTo || cmd.Op == OpCurveTo {
			return false
		}
	}
	return true
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

func (g *Glyph) Path() path.Path {
	return func(yield func(path.Command, []vec.Vec2) bool) {
		var buf [3]vec.Vec2

		for _, cmd := range g.Cmds {
			switch cmd.Op {
			case OpMoveTo:
				buf[0] = vec.Vec2{X: cmd.Args[0], Y: cmd.Args[1]}
				if !yield(path.CmdMoveTo, buf[:1]) {
					return
				}
			case OpLineTo:
				buf[0] = vec.Vec2{X: cmd.Args[0], Y: cmd.Args[1]}
				if !yield(path.CmdLineTo, buf[:1]) {
					return
				}
			case OpCurveTo:
				buf[0] = vec.Vec2{X: cmd.Args[0], Y: cmd.Args[1]} // control point 1
				buf[1] = vec.Vec2{X: cmd.Args[2], Y: cmd.Args[3]} // control point 2
				buf[2] = vec.Vec2{X: cmd.Args[4], Y: cmd.Args[5]} // end point
				if !yield(path.CmdCubeTo, buf[:3]) {
					return
				}
			case OpClosePath:
				if !yield(path.CmdClose, nil) {
					return
				}
			}
		}
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
