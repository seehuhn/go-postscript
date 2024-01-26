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

package afm

import (
	"sort"

	"golang.org/x/exp/maps"
	"seehuhn.de/go/postscript/funit"
)

// TODO(voss): use float64 instead of funit.Int16?
type Info struct {
	Glyphs   map[string]*GlyphInfo
	Encoding []string

	// PostScript language name (FontName or CIDFontName) of the font.
	FontName string

	// FullName is a unique, human-readable name for an individual font.
	FullName string

	CapHeight funit.Int16
	XHeight   funit.Int16
	Ascent    funit.Int16
	Descent   funit.Int16 // negative

	// UnderlinePosition is the recommended distance from the baseline for
	// positioning underlining strokes. This number is the y coordinate (in the
	// glyph coordinate system) of the center of the stroke.
	UnderlinePosition funit.Float64

	// UnderlineThickness is the recommended stroke width for underlining, in
	// units of the glyph coordinate system.
	UnderlineThickness funit.Float64

	// ItalicAngle is the angle, in degrees counterclockwise from the vertical,
	// of the dominant vertical strokes of the font.
	ItalicAngle float64

	// IsFixedPitch is a flag indicating whether the font is a fixed-pitch
	// (monospaced) font.
	IsFixedPitch bool

	Kern []*KernPair
}

// NumGlyphs returns the number of glyphs in the font (including the .notdef glyph).
func (f *Info) NumGlyphs() int {
	n := len(f.Glyphs)
	if _, ok := f.Glyphs[".notdef"]; !ok {
		n++
	}
	return n
}

// GlyphList returns a list of all glyph names in the font.
// The list starts with the ".notdef" glyph, followed by the glyphs in the
// Encoding vector, followed by the remaining glyphs in alphabetical order
// of their names.
func (f *Info) GlyphList() []string {
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

type GlyphInfo struct {
	WidthX    funit.Int16
	BBox      funit.Rect16
	Ligatures map[string]string
}

// KernPair represents a kerning pair.
type KernPair struct {
	Left, Right string
	Adjust      funit.Int16 // negative = move glyphs closer together
}