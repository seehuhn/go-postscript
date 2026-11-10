// Copyright 2024 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

// Glyph represents a glyph in a Type 1 font.
//
// All coordinates, stem edges and widths are held to a precision of 1/16384
// unit.  The reader rounds values to this grid, and the writer rounds the
// values it is given, so that a write/read cycle reproduces a glyph read from
// a file exactly.
type Glyph struct {
	// Outline is the glyph outline in glyph space.  The difference between
	// consecutive points, and the first point itself, must be less than
	// 32000 in absolute value in each coordinate.
	Outline *path.Data

	// HStem and VStem hold the horizontal and vertical stem hints, as
	// pairs of edge coordinates in glyph space.  The edges of a pair are
	// normally in increasing order; a ghost hint lists them in decreasing
	// order.  The first edge of each pair, and the difference between its
	// two edges, must be less than 32000 in absolute value.  Stems in the
	// same direction must not overlap; [RepairStems] can be used to remove
	// overlapping pairs.
	HStem []float64
	VStem []float64

	// WidthX and WidthY give the advance width vector.  Both components
	// must be less than 32000 in absolute value.
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

// IsBlank returns true if the glyph has an empty outline.
func (g *Glyph) IsBlank() bool {
	return g.Outline == nil || g.Outline.IsBlank()
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
		return path.Empty
	}
	return g.Outline.Iter()
}
