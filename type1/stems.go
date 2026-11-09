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
	"cmp"
	"fmt"
	"slices"
)

// maxOperand bounds the absolute value of a charstring operand.  The Type 1
// specification requires larger numbers to be brought below this limit by a
// div operator.
const maxOperand = 32000

// stemUsable reports whether the stem with edges a and b can be written back
// to a charstring.  The writer gives the edge a relative to a zero
// sidebearing, followed by the width b - a, and each operand must be finite
// and stay below [maxOperand] in absolute value.
func stemUsable(a, b float64) bool {
	return numberUsable(a) && numberUsable(b-a)
}

// appendStems appends the edges of the stems given by the (position, width)
// operand pairs in args, measured from the sidebearing coordinate sb.
// The edges are rounded to the grid, and stems which cannot be written back
// are dropped.
func appendStems(stems []float64, sb float64, args []float64) []float64 {
	for i := 0; i+1 < len(args); i += 2 {
		a := sb + args[i]
		b := snap(a + args[i+1])
		a = snap(a)
		if stemUsable(a, b) {
			stems = append(stems, a, b)
		}
	}
	return stems
}

// RepairStems brings a list of stem hints into the form which a single set
// of hints requires: the pairs of edges are sorted by increasing bottom
// edge, and pairs which overlap an earlier pair are dropped.  A trailing
// unpaired value is ignored, and nil is returned if no pairs remain.
//
// A Type 1 charstring which uses hint replacement switches between several
// sets of hints, and stems from different sets may overlap.  A [Glyph] holds
// a single set, so the reader merges the sets and uses this function to
// remove the conflicts.  The function can be used in the same way for hints
// from other sources.
func RepairStems(stems []float64) []float64 {
	n := len(stems) / 2
	if n == 0 {
		return nil
	}

	pairs := make([][2]float64, n)
	for i := range pairs {
		pairs[i] = [2]float64{stems[2*i], stems[2*i+1]}
	}
	slices.SortStableFunc(pairs, func(p, q [2]float64) int {
		return cmp.Compare(stemBottom(p), stemBottom(q))
	})

	res := make([]float64, 0, 2*n)
	var last [2]float64
	for i, p := range pairs {
		if i > 0 && stemsOverlap(last, p) {
			continue
		}
		res = append(res, p[0], p[1])
		last = p
	}
	return res
}

// Ghost hints list their edges in descending order, so the bottom edge of a
// pair is the smaller of the two values.
func stemBottom(p [2]float64) float64 { return min(p[0], p[1]) }
func stemTop(p [2]float64) float64    { return max(p[0], p[1]) }

// stemsOverlap reports whether the stem q overlaps the stem p, where q does
// not start below p.  Stems which only touch do not overlap, but two copies
// of the same stem do.
func stemsOverlap(p, q [2]float64) bool {
	return stemBottom(q) < stemTop(p) || p == q
}

// checkStems returns an error unless stems holds pairs of edges which a
// charstring can express as a single set of hints.
func checkStems(glyphName string, stems []float64) error {
	if len(stems)%2 != 0 {
		return fmt.Errorf("glyph %q: odd number of stem edges", glyphName)
	}
	pairs := make([][2]float64, 0, len(stems)/2)
	for i := 0; i < len(stems); i += 2 {
		if !stemUsable(stems[i], stems[i+1]) {
			return fmt.Errorf("glyph %q: stem %v..%v out of range",
				glyphName, stems[i], stems[i+1])
		}
		pairs = append(pairs, [2]float64{stems[i], stems[i+1]})
	}

	// Once the stems are sorted by bottom edge, an overlap between any two
	// stems implies one between neighbours.
	slices.SortFunc(pairs, func(p, q [2]float64) int {
		return cmp.Compare(stemBottom(p), stemBottom(q))
	})
	for i := 1; i < len(pairs); i++ {
		if stemsOverlap(pairs[i-1], pairs[i]) {
			return fmt.Errorf("glyph %q: stems %v and %v overlap",
				glyphName, pairs[i-1], pairs[i])
		}
	}
	return nil
}
