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
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/postscript/funit"
)

// Glyph represents a glyph in a Type 1 font.
//
// TODO(voss): use float64 instead of funit.Int16?
type Glyph struct {
	Outline *path.Data
	HStem   []funit.Int16
	VStem   []funit.Int16
	WidthX  float64
	WidthY  float64
}

// NewGlyph creates a new glyph with the given name and width.
func (f *Font) NewGlyph(name string, width float64) *Glyph {
	g := &Glyph{
		WidthX: width,
	}
	f.Glyphs[name] = g
	return g
}

// IsBlank returns true if the glyph has no visible outline.
func (g *Glyph) IsBlank() bool {
	return g.Outline.IsBlank()
}

// MoveTo starts a new sub-path and moves the current point to (x, y).
// The previous sub-path, if any, is closed.
func (g *Glyph) MoveTo(x, y float64) {
	if g.Outline == nil {
		g.Outline = &path.Data{}
	}
	g.Outline.MoveTo(vec.Vec2{X: x, Y: y})
}

// LineTo adds a straight line to the current sub-path.
func (g *Glyph) LineTo(x, y float64) {
	if g.Outline == nil {
		g.Outline = &path.Data{}
	}
	g.Outline.LineTo(vec.Vec2{X: x, Y: y})
}

// CurveTo adds a cubic Bezier curve to the current sub-path.
func (g *Glyph) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	if g.Outline == nil {
		g.Outline = &path.Data{}
	}
	g.Outline.CubeTo(vec.Vec2{X: x1, Y: y1}, vec.Vec2{X: x2, Y: y2}, vec.Vec2{X: x3, Y: y3})
}

// ClosePath closes the current sub-path.
func (g *Glyph) ClosePath() {
	if g.Outline == nil {
		g.Outline = &path.Data{}
	}
	g.Outline.Close()
}

// Path returns the glyph outline as a path.
func (g *Glyph) Path() path.Path {
	if g.Outline == nil {
		return func(yield func(path.Command, []vec.Vec2) bool) {}
	}
	return g.Outline.Iter()
}
