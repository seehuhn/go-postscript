// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
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
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"

	"seehuhn.de/go/postscript/funit"
)

// corner-master tolerance: a master coordinate is treated as 0 or 1 if it lies
// within this distance of that value.
const mmCornerEps = 1e-9

// VariationAxis describes one design axis of a multiple master font.
type VariationAxis struct {
	Name    string  // axis name from /BlendAxisTypes, e.g. "Weight"
	Min     float64 // smallest design coordinate (first BlendDesignMap point)
	Default float64 // design coordinate of the font's own WeightVector
	Max     float64 // largest design coordinate (last BlendDesignMap point)
}

// VariationAxes returns the design axes of a multiple master font.
// It returns nil if the font is not a multiple master font.
func (f *Font) VariationAxes() []VariationAxis {
	if f.MM == nil || len(f.MM.Axes) == 0 {
		return nil
	}
	def := f.MM.defaultDesignCoords()
	axes := make([]VariationAxis, len(f.MM.Axes))
	for j, a := range f.MM.Axes {
		axes[j] = VariationAxis{
			Name:    a.Name,
			Min:     a.Map[0].Design,
			Default: def[j],
			Max:     a.Map[len(a.Map)-1].Design,
		}
	}
	return axes
}

// Instantiate returns a single-master snapshot of a multiple master font.
// coords maps axis names (e.g. "Weight") to design-space coordinates;
// missing axes use the font's default coordinates, values are clamped to
// the axis range, and unknown axis names cause an error.
//
// The result is an ordinary Font with MM == nil that the writer can embed
// without special handling.
func (f *Font) Instantiate(coords map[string]float64) (*Font, error) {
	mm := f.MM
	if mm == nil {
		return nil, errors.New("not a multiple master font")
	}
	if len(mm.Axes) == 0 || len(mm.Masters) == 0 {
		return nil, errors.New("font has incomplete blend data")
	}
	n := len(mm.Axes)

	// API args are trusted, so an unknown axis name is a hard error
	for name := range coords {
		if !slices.ContainsFunc(mm.Axes, func(a MMAxis) bool { return a.Name == name }) {
			return nil, fmt.Errorf("unknown axis %q", name)
		}
	}

	// resolve design coordinates (default, override, clamp) and normalize
	def := mm.defaultDesignCoords()
	design := make([]float64, n)
	x := make([]float64, n)
	for j, a := range mm.Axes {
		d := def[j]
		if v, ok := coords[a.Name]; ok {
			d = v
		}
		lo := a.Map[0].Design
		hi := a.Map[len(a.Map)-1].Design
		d = max(lo, min(hi, d))
		design[j] = d
		x[j] = normalizeDesign(a.Map, d)
	}

	// corner-master weight vector: w_i = Π_j (Masters[i][j]==1 ? x_j : 1-x_j)
	k := len(mm.Masters)
	w := make([]float64, k)
	for i := range k {
		prod := 1.0
		for j := range n {
			switch m := mm.Masters[i][j]; {
			case math.Abs(m-1) < mmCornerEps:
				prod *= x[j]
			case math.Abs(m) < mmCornerEps:
				prod *= 1 - x[j]
			default:
				return nil, errors.New("unsupported master positions")
			}
		}
		w[i] = prod
	}

	// re-decode the glyphs at the instance weight vector, mirroring Read
	codeBytes := 0
	for _, s := range mm.subrs {
		codeBytes += len(s)
	}
	for _, s := range mm.charstrings {
		codeBytes += len(s)
	}
	encoding := slices.Clone(f.Encoding)
	glyphs := decodeGlyphs(mm.charstrings, mm.subrs, w, encoding, codeBytes)

	// blended Private and FontInfo entries
	private := *f.Private
	fi := *f.FontInfo
	if b := mm.Blend; b != nil {
		if b.BlueValues != nil {
			private.BlueValues = blendInt16(b.BlueValues, w)
		}
		if b.OtherBlues != nil {
			private.OtherBlues = blendInt16(b.OtherBlues, w)
		}
		if b.StdHW != nil {
			private.StdHW = blend1D(b.StdHW, w)
		}
		if b.StdVW != nil {
			private.StdVW = blend1D(b.StdVW, w)
		}
		// StemSnapH, StemSnapV and FontBBox have no destination field:
		// PrivateDict lacks StemSnap entries and the writer emits a fixed
		// FontBBox, so their blend data is dropped.
		if b.UnderlinePosition != nil {
			fi.UnderlinePosition = funit.Float64(blend1D(b.UnderlinePosition, w))
		}
		if b.UnderlineThickness != nil {
			fi.UnderlineThickness = funit.Float64(blend1D(b.UnderlineThickness, w))
		}
		if b.ItalicAngle != nil {
			fi.ItalicAngle = blend1D(b.ItalicAngle, w)
		}
	}

	// instance name: base + "_" + design coordinate per axis (TN #5088)
	name := f.FontName
	for j := range mm.Axes {
		name += "_" + strconv.FormatFloat(design[j], 'f', -1, 64)
	}
	fi.FontName = name

	return &Font{
		CreationDate: f.CreationDate,
		FontInfo:     &fi,
		MM:           nil,
		Outlines: &Outlines{
			Private:  &private,
			Glyphs:   glyphs,
			Encoding: encoding,
		},
	}, nil
}

// defaultDesignCoords derives the design-space coordinates of the font's own
// WeightVector.  The normalized default is x_j = Σ WeightVector[i] over the
// masters at position 1 on axis j, the exact inverse of the corner-master
// product formula; it is exact for corner-master layouts.  Each normalized
// value is then mapped back to design space through the axis' design map.
func (mm *MMInfo) defaultDesignCoords() []float64 {
	n := len(mm.Axes)
	design := make([]float64, n)
	for j := range n {
		var sum float64
		for i, m := range mm.Masters {
			if math.Abs(m[j]-1) < mmCornerEps {
				sum += mm.WeightVector[i]
			}
		}
		norm := max(0, min(1, sum))
		design[j] = designFromNormalized(mm.Axes[j].Map, norm)
	}
	return design
}

// normalizeDesign maps a design coordinate to a normalized one by piecewise
// linear interpolation through the axis' design map.
func normalizeDesign(pts []MMMapPoint, design float64) float64 {
	if design <= pts[0].Design {
		return pts[0].Normalized
	}
	last := len(pts) - 1
	if design >= pts[last].Design {
		return pts[last].Normalized
	}
	for i := 0; i < last; i++ {
		d0, d1 := pts[i].Design, pts[i+1].Design
		if design <= d1 {
			if d1 == d0 { // guard; the reader forbids this, but stay safe
				return pts[i].Normalized
			}
			t := (design - d0) / (d1 - d0)
			return pts[i].Normalized + t*(pts[i+1].Normalized-pts[i].Normalized)
		}
	}
	return pts[last].Normalized
}

// designFromNormalized inverts the axis' design map, mapping a normalized
// coordinate back to design space.  For an equal-Normalized segment it returns
// the segment start.  It assumes the map is monotone in Normalized.
func designFromNormalized(pts []MMMapPoint, norm float64) float64 {
	if norm <= pts[0].Normalized {
		return pts[0].Design
	}
	last := len(pts) - 1
	if norm >= pts[last].Normalized {
		return pts[last].Design
	}
	for i := 0; i < last; i++ {
		n0, n1 := pts[i].Normalized, pts[i+1].Normalized
		if norm <= n1 {
			if n1 == n0 { // equal-Normalized segment
				return pts[i].Design
			}
			t := (norm - n0) / (n1 - n0)
			return pts[i].Design + t*(pts[i+1].Design-pts[i].Design)
		}
	}
	return pts[last].Design
}

// blend1D blends a per-master vector v with weights w.
func blend1D(v, w []float64) float64 {
	var sum float64
	for i, vi := range v {
		sum += w[i] * vi
	}
	return sum
}

// blendInt16 blends element-major per-master values with weights w, rounding
// each blended element to funit.Int16.
func blendInt16(vals [][]float64, w []float64) []funit.Int16 {
	res := make([]funit.Int16, len(vals))
	for i, v := range vals {
		res[i] = funit.Int16(math.Round(blend1D(v, w)))
	}
	return res
}
