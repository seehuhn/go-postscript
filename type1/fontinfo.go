// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2022  Jochen Voss <voss@seehuhn.de>
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
	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/postscript/funit"
)

// FontInfo contains information about a font.
type FontInfo struct {
	// PostScript language name (FontName or CIDFontName) of the font.
	FontName string

	// Version is the version number of the font file.
	Version string

	// Notice is used to save any trademark notice/information for the font.
	Notice string

	// The copyright notice of the font.
	Copyright string

	// FullName is a unique, human-readable name for an individual font.
	FullName string

	// FamilyName is a human-readable name for a group of fonts that are
	// stylistic variants of a single design.  All fonts that are members of
	// such a group should have exactly the same FamilyName value.
	FamilyName string

	// A human-readable name for the weight, or "boldness," of a font.
	Weight string

	// ItalicAngle is the angle, in degrees counterclockwise from the vertical,
	// of the dominant vertical strokes of the font.
	ItalicAngle float64

	// IsFixedPitch is a flag indicating whether the font is a fixed-pitch
	// (monospaced) font.
	IsFixedPitch bool

	// UnderlinePosition is the recommended distance from the baseline for
	// positioning underlining strokes. This number is the y coordinate of the
	// center of the stroke (in font design units).
	UnderlinePosition funit.Float64

	// UnderlineThickness is the recommended stroke width for underlining, in
	// units of the glyph coordinate system.
	UnderlineThickness funit.Float64

	// FontMatrix is the transformation from font design units to text space
	// units.
	FontMatrix matrix.Matrix
}

// PrivateDict contains information about a font's private dictionary.
type PrivateDict struct {
	// BlueValues is an array containing an even number of values, in glyph
	// space units.  The first value in each pair is less than or equal to
	// the second value.  The first pair is the baseline overshoot position
	// and the baseline.  All subsequent pairs describe alignment zones for
	// the tops of character features.  At most [MaxBlueValuePairs] pairs may
	// be given.
	//
	// The Type 1 format allows only whole numbers here, so a font with a
	// fractional zone edge cannot be written as Type 1.
	BlueValues []float64

	// OtherBlues describes alignment zones for the bottoms of character
	// features, in the same form as BlueValues and with at most
	// [MaxOtherBluePairs] pairs.
	OtherBlues []float64

	// BlueScale is the text size at which overshoot suppression stops,
	// expressed as (pointsize - 0.49) / 240 on a 300 dpi device. Valid values
	// are (0, [MaxBlueScale]], and BlueScale times the height of the tallest
	// alignment zone must stay below 1.
	//
	// On write, 0 can be used as a shorthand for [DefaultBlueScale].
	BlueScale float64

	// BlueShift is the height a character feature must overshoot its zone by
	// before it is rendered curved rather than flat, in glyph space units.
	// Valid values are finite and zero or above.
	//
	// The Type 1 format allows only whole numbers here, so a font with a
	// fractional BlueShift cannot be written as Type 1.
	BlueShift float64

	// BlueFuzz is the distance by which an alignment zone is extended when
	// deciding whether a horizontal stem falls inside it, in glyph space
	// units.  Valid values are finite and zero or above.
	//
	// The Type 1 format allows only whole numbers here, so a font with a
	// fractional BlueFuzz cannot be written as Type 1.
	BlueFuzz float64

	// StdHW is the dominant width of horizontal stems for glyphs in the font.
	// Valid values are (0, [MaxStemWidth] ].
	// Zero indicates that the font gives no dominant width.
	StdHW float64

	// StdVW the dominant width of vertical stems.
	// Typically, this will be the width of straight stems in lower case letters.
	// Valid values are (0, [MaxStemWidth] ].
	// Zero indicates that the font gives no dominant width.
	StdVW float64

	// TODO(voss): StemSnapH, StemSnapV

	ForceBold bool
}
