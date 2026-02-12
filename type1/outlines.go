// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2025  Jochen Voss <voss@seehuhn.de>
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
	"maps"
	"slices"
	"sort"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/rect"
)

// Outlines contains the glyph outlines and encoding for a Type 1 font.
type Outlines struct {
	Glyphs map[string]*Glyph

	Private *PrivateDict

	Encoding []string
}

// NumGlyphs returns the number of glyphs in the font (including the .notdef glyph).
func (o *Outlines) NumGlyphs() int {
	n := len(o.Glyphs)
	if _, ok := o.Glyphs[".notdef"]; !ok {
		n++
	}
	return n
}

// GlyphList returns a list of all glyph names in the font.
// The list starts with ".notdef", followed by the glyphs in the Encoding
// vector, followed by the remaining glyph names in alphabetical order.
func (o *Outlines) GlyphList() []string {
	glyphNames := slices.Collect(maps.Keys(o.Glyphs))
	if _, ok := o.Glyphs[".notdef"]; !ok {
		glyphNames = append(glyphNames, ".notdef")
	}

	order := make(map[string]int, len(glyphNames))
	for _, name := range glyphNames {
		order[name] = 256
	}
	order[".notdef"] = -1
	for i, name := range o.Encoding {
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

// BuiltinEncoding returns the built-in encoding of the font.
func (o *Outlines) BuiltinEncoding() []string {
	return o.Encoding
}

// IsBlank returns true if the glyph with the given name does not add marks to the page.
func (o *Outlines) IsBlank(name string) bool {
	g, exists := o.Glyphs[name]
	if !exists {
		name = ".notdef"
		g, exists = o.Glyphs[name]
		if !exists {
			return true
		}
	}

	return g.IsBlank()
}

// GlyphBBox computes the bounding box of a glyph, after the matrix M has been
// applied to the glyph outline. If the glyph is missing, the bounding box of
// the ".notdef" glyph is returned instead. If the glyph is blank, the zero
// rectangle is returned.
func (o *Outlines) GlyphBBox(M matrix.Matrix, name string) (bbox rect.Rect) {
	g, ok := o.Glyphs[name]
	if !ok {
		name = ".notdef"
		g, ok = o.Glyphs[name]
		if !ok {
			return
		}
	}

	return g.Path().Transform(M).BBox()
}
