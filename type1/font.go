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
	"time"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/rect"
)

// Font represents a Type 1 font.
type Font struct {
	*FontInfo
	*Outlines
	CreationDate time.Time
}

// GlyphWidthPDF computes the width of a glyph in PDF glyph space units.
// If the glyph does not exist, the width of the .notdef glyph is returned.
func (f *Font) GlyphWidthPDF(name string) float64 {
	g, ok := f.Glyphs[name]
	if !ok {
		g, ok = f.Glyphs[".notdef"]
	}
	if !ok {
		return 0
	}

	q := f.FontMatrix[0]
	if math.Abs(f.FontMatrix[3]) > 1e-6 {
		q -= f.FontMatrix[1] * f.FontMatrix[2] / f.FontMatrix[3]
	}

	return g.WidthX * (q * 1000)
}

// FontBBoxPDF returns the font bounding box in PDF glyph space units.
// This is the smallest rectangle enclosing all individual glyphs bounding boxes.
func (f *Font) FontBBoxPDF() (fontBBox rect.Rect) {
	M := f.FontMatrix.Mul(matrix.Scale(1000, 1000))

	first := true
	for glyphName := range f.Glyphs {
		glyphBBox := f.Outlines.GlyphBBox(M, glyphName)
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

// GlyphBBoxPDF computes the bounding box of a glyph in PDF glyph space units
// (1/1000th of a text space unit). If the glyph is missing, the bounding box
// of the ".notdef" glyph is returned intead. If the glyph is blank, the zero
// rectangle is returned.
func (f *Font) GlyphBBoxPDF(name string) (bbox rect.Rect) {
	M := f.FontMatrix.Mul(matrix.Scale(1000, 1000))
	return f.Outlines.GlyphBBox(M, name)
}
