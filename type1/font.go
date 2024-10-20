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
	"math"
	"sort"
	"time"

	"golang.org/x/exp/maps"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/rect"
)

// Font represents a Type 1 font.
//
// TODO(voss): make this more similar to cff.Font?
type Font struct {
	*FontInfo

	Glyphs map[string]*Glyph

	Private *PrivateDict

	Encoding []string

	CreationDate time.Time
}

// NumGlyphs returns the number of glyphs in the font (including the .notdef glyph).
func (f *Font) NumGlyphs() int {
	n := len(f.Glyphs)
	if _, ok := f.Glyphs[".notdef"]; !ok {
		n++
	}
	return n
}

// GlyphList returns a list of all glyph names in the font.
// The list starts with ".notdef", followed by the glyphs in the Encoding
// vector, followed by the remaining glyph names in alphabetical order.
func (f *Font) GlyphList() []string {
	glyphNames := maps.Keys(f.Glyphs)
	if _, ok := f.Glyphs[".notdef"]; !ok {
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

// GetEncoding returns the built-in encoding of the font.
func (f *Font) GetEncoding() []string {
	return f.Encoding
}

func (f *Font) WidthsMapPDF() map[string]float64 {
	q := f.FontMatrix[0]
	if math.Abs(f.FontMatrix[3]) > 1e-6 {
		q -= f.FontMatrix[1] * f.FontMatrix[2] / f.FontMatrix[3]
	}
	q *= 1000

	widths := make(map[string]float64)
	for name, glyph := range f.Glyphs {
		widths[name] = glyph.WidthX * q
	}
	return widths
}

// FontBBox returns the font bounding box in glyph space units.
// This is the smallest rectangle enclosing all glyphs in the font.
//
// TODO(voss): remove in favour of FontBBoxPDF
func (f *Font) FontBBox() (bbox rect.Rect) {
	first := true
	for _, glyph := range f.Glyphs {
		thisBBox := glyph.BBox()
		if thisBBox.IsZero() {
			continue
		}
		if first {
			bbox = thisBBox
			first = false
		} else {
			bbox.Extend(thisBBox)
		}
	}
	return bbox
}

// FontBBoxPDF returns the font bounding box in PDF glyph space units.
// This is the smallest rectangle enclosing all individual glyphs bounding boxes.
func (f *Font) FontBBoxPDF() (fontBBox rect.Rect) {
	first := true
	for glyphName := range f.Glyphs {
		glyphBBox := f.GlyphBBoxPDF(glyphName)
		if glyphBBox.IsZero() {
			continue
		}
		if first {
			fontBBox = glyphBBox
			first = false
		} else {
			fontBBox.Extend(glyphBBox)
		}
	}
	return fontBBox
}

// GlyphBBoxPDF computes the bounding box of a glyph in PDF glyph space units.
// If the glyph does not exist or is blank, the zero rectangle is returned.
func (f *Font) GlyphBBoxPDF(name string) (bbox rect.Rect) {
	g, ok := f.Glyphs[name]
	if !ok {
		return
	}

	M := f.FontMatrix.Mul(matrix.Scale(1000, 1000))

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

		x, y = M.Apply(x, y)

		if first || x < bbox.LLx {
			bbox.LLx = x
		}
		if first || x > bbox.URx {
			bbox.URx = x
		}
		if first || y < bbox.LLy {
			bbox.LLy = y
		}
		if first || y > bbox.URy {
			bbox.URy = y
		}
		first = false
	}

	return bbox
}

func (f *Font) GlyphWidthPDF(name string) float64 {
	g, ok := f.Glyphs[name]
	if !ok {
		return 0
	}

	q := f.FontMatrix[0]
	if math.Abs(f.FontMatrix[3]) > 1e-6 {
		q -= f.FontMatrix[1] * f.FontMatrix[2] / f.FontMatrix[3]
	}

	w := g.WidthX
	return w * (q * 1000)
}
