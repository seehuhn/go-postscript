// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2023  Jochen Voss <voss@seehuhn.de>
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
	"math"
	"sort"
	"time"

	"golang.org/x/exp/maps"
	"seehuhn.de/go/postscript/funit"
)

// Font represents a Type 1 font.
//
// TODO(voss): make this more similar to cff.Font
type Font struct {
	CreationDate time.Time
	UnitsPerEm   uint16

	Encoding []string

	Ascent    funit.Int16
	Descent   funit.Int16 // negative
	CapHeight funit.Int16
	XHeight   funit.Int16

	FontInfo *FontInfo
	Private  *PrivateDict

	Outlines  map[string]*Glyph
	GlyphInfo map[string]*GlyphInfo
	Kern      []*KernPair
}

// NumGlyphs returns the number of glyphs in the font (including the .notdef glyph).
func (f *Font) NumGlyphs() int {
	n := len(f.GlyphInfo)
	if _, ok := f.GlyphInfo[".notdef"]; !ok {
		n++
	}
	return n
}

func (f *Font) BBox() (bbox funit.Rect16) {
	first := true
	for _, glyph := range f.GlyphInfo {
		if glyph.BBox.IsZero() {
			continue
		}
		if first {
			bbox = glyph.BBox
		} else {
			bbox.Extend(glyph.BBox)
		}
	}
	return bbox
}

// GlyphList returns a list of all glyph names in the font.
// The list starts with the ".notdef" glyph, followed by the glyphs in the
// Encoding vector, followed by the remaining glyphs in alphabetical order
// of their names.
func (f *Font) GlyphList() []string {
	glyphNames := maps.Keys(f.GlyphInfo)
	if _, ok := f.GlyphInfo[".notdef"]; !ok {
		glyphNames = append(glyphNames, ".notdef")
	}

	order := make(map[string]int, len(glyphNames))
	for _, name := range glyphNames {
		order[name] = 256
	}
	order[".notdef"] = -1
	for i, name := range f.Encoding {
		if name != ".notdef" {
			order[name] = i
		}
	}
	sort.Slice(glyphNames, func(i, j int) bool {
		oi := order[glyphNames[i]]
		oj := order[glyphNames[j]]
		if oi != oj {
			return oi < oj
		}
		return glyphNames[i] < glyphNames[j]
	})
	return glyphNames
}

// Glyph represents a glyph in a Type 1 font.
type Glyph struct {
	Cmds  []GlyphOp
	HStem []funit.Int16
	VStem []funit.Int16
}

func (f *Font) NewGlyph(name string, width funit.Int16) *Glyph {
	g := &Glyph{}
	gi := &GlyphInfo{
		WidthX: width,
	}
	f.Outlines[name] = g
	f.GlyphInfo[name] = gi
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

func (g *Glyph) ClosePath() {
	g.Cmds = append(g.Cmds, GlyphOp{Op: OpClosePath})
}

func (g *Glyph) computeExt() funit.Rect16 {
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
	return funit.Rect16{
		LLx: funit.Int16(math.Floor(left)),
		LLy: funit.Int16(math.Floor(bottom)),
		URx: funit.Int16(math.Ceil(right)),
		URy: funit.Int16(math.Ceil(top)),
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

type GlyphInfo struct {
	WidthX    funit.Int16
	WidthY    funit.Int16
	BBox      funit.Rect16
	Ligatures map[string]string
}

type KernPair struct {
	Left, Right string
	Adjust      funit.Int16 // negative = move glyphs closer together
}
