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

	// MM holds the multiple master data of the font, or nil for an
	// ordinary single master font.
	MM *MMInfo
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
// of the ".notdef" glyph is returned instead. If the glyph is blank, the zero
// rectangle is returned.
func (f *Font) GlyphBBoxPDF(name string) (bbox rect.Rect) {
	M := f.FontMatrix.Mul(matrix.Scale(1000, 1000))
	return f.Outlines.GlyphBBox(M, name)
}

// CapHeightChars lists the characters whose glyphs are measured to find a
// font's cap height, in order of preference.  Several are tried because a
// subsetted font need not contain "H".  The standard glyph name of each of
// these characters is the character itself, so the list serves fonts keyed by
// glyph name as well as fonts keyed by character code.
const CapHeightChars = "HIKLT"

// XHeightChars lists the characters whose glyphs are measured to find a font's
// x-height, in order of preference.  See [CapHeightChars].
const XHeightChars = "xuvwz"

// CapHeightPDF returns the font's cap height in PDF glyph space units.
// The Type 1 format does not record one, so this measures the glyphs listed in
// [CapHeightChars].  The result is 0 if the height cannot be determined.
func (f *Font) CapHeightPDF() float64 {
	return f.glyphHeightPDF(CapHeightChars)
}

// XHeightPDF returns the font's x-height in PDF glyph space units.
// The Type 1 format does not record one, so this measures the glyphs listed in
// [XHeightChars].  The result is 0 if the height cannot be determined.
func (f *Font) XHeightPDF() float64 {
	return f.glyphHeightPDF(XHeightChars)
}

func (f *Font) glyphHeightPDF(chars string) float64 {
	M := f.FontMatrix.Mul(matrix.Scale(1000, 1000))
	for _, r := range chars {
		g, ok := f.Glyphs[string(r)]
		if !ok {
			continue
		}
		h := g.Path().Transform(M).BBox().URy
		if h > 0 { // the test also rejects NaN
			return h
		}
	}
	return 0
}
