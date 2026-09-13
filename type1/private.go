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
	"fmt"
	"math"
)

// The conventional values for the corresponding Private dictionary entries.
// Use these unless the font calls for something else.
const (
	DefaultBlueScale = 0.039625
	DefaultBlueShift = 7
	DefaultBlueFuzz  = 1
)

// The number of pairs the alignment zone arrays may hold.
const (
	MaxBlueValuePairs = 7
	MaxOtherBluePairs = 5
)

// MaxBlueScale is the largest BlueScale a font can hold.
//
// BlueScale relates the text size at which overshoot suppression stops to the
// device resolution, as BlueScale = (pointsize - 0.49) / 240.  On top of that
// there is a mandatory restriction: the product of (pointsize - 0.49) and the
// maximum alignment zone height must stay below 240, so that suppression ends
// before an overshoot reaches a whole device pixel.  Substituting the first
// into the second leaves
//
//	BlueScale * maxZoneHeight < 1
//
// which bounds nothing on its own, since a zone may have equal ends and so no
// height at all.  The limit here is therefore ours, and generous on purpose:
// it is the point beyond which a font with a zone even one unit tall breaks
// the restriction.  Real fonts sit far below it, the default 0.039625
// allowing zones up to 25 units tall.
const MaxBlueScale = 1.0

// MaxStemWidth is the largest dominant stem width a font can hold.
//
// No specification states one: the Type 1 format calls StdHW optional and says
// no more, and CFF lists it as a bare number with no default.  The limit is
// therefore ours, and generous on purpose -- 10000 units is ten times the em
// at the usual 1000 units per em -- so that it can only catch a value no font
// would give.
const MaxStemWidth = 10000.0

// UsableStemWidth reports whether v is a dominant stem width a font could
// give.  Zero is excluded: it says nothing a renderer could act on, and is
// spent on "no stem width given" instead.  Writing the test in the positive
// also rejects a NaN supplied through the API.
func UsableStemWidth(v float64) bool {
	return v > 0 && v <= MaxStemWidth
}

// UsableBlueDistance reports whether v is a distance a font could give for
// BlueShift or BlueFuzz.  Both extend an alignment zone, so zero is a setting
// in its own right rather than a marker for "not given"; only a negative
// distance, an infinity or a NaN says nothing a renderer could act on.  The
// comparison rejects a NaN supplied through the API.
func UsableBlueDistance(v float64) bool {
	return v >= 0 && !math.IsInf(v, 1)
}

// RepairBlueScale returns a usable BlueScale for a value read from a font.
//
// Only (0, MaxBlueScale] is reachable for a conforming font: the defining
// formula yields a positive number at any text size, and MaxBlueScale caps it.
// A value outside carries no intent to preserve, so it falls back to the
// default rather than to the nearest bound -- clamping would land on 0, which
// is itself not a value a font can hold.  The negated comparison also rejects
// a NaN supplied through the API.
func RepairBlueScale(v float64) float64 {
	if !(v > 0) || v > MaxBlueScale {
		return DefaultBlueScale
	}
	return v
}

// UsableZones returns the length of the leading part of an alignment zone
// array which obeys the rules the format sets for it: an even number of
// finite values, taken in pairs whose first value does not exceed the second,
// and at most maxPairs of them.  Anything from the first broken pair on is
// excluded.
//
// Keeping the good prefix rather than discarding the array is what a zone list
// can stand: the pairs are independent of one another, so the ones ahead of a
// broken pair remain exactly as useful as they were.
//
// Two further rules the format states are left alone, since neither can be
// repaired without guessing which of two entries is the wrong one: that pairs
// stay at least BlueFuzz-adjusted units apart, here and across into
// OtherBlues, and that a pair is no taller than BlueScale allows.
func UsableZones(vals []float64, maxPairs int) int {
	n := min(len(vals)/2, maxPairs)
	for i := range n {
		if !usableZone(vals[2*i], vals[2*i+1]) {
			return 2 * i
		}
	}
	return 2 * n
}

// usableZone reports whether lo..hi is an alignment zone a font could give.
// The comparison rejects a NaN supplied through the API.
func usableZone(lo, hi float64) bool {
	return lo <= hi && !math.IsInf(lo, 0) && !math.IsInf(hi, 0)
}

// checkZones reports an error for an alignment zone array a font cannot hold.
// The name is used in the error message.
func checkZones(name string, vals []float64, maxPairs int) error {
	if len(vals)%2 != 0 {
		return fmt.Errorf("%s has %d values, want an even number", name, len(vals))
	}
	if len(vals)/2 > maxPairs {
		return fmt.Errorf("%s has %d pairs, at most %d allowed",
			name, len(vals)/2, maxPairs)
	}
	for i := range len(vals) / 2 {
		if !usableZone(vals[2*i], vals[2*i+1]) {
			return fmt.Errorf("%s pair %d is %v..%v, want finite and ascending",
				name, i, vals[2*i], vals[2*i+1])
		}
	}
	return nil
}

// trimZones cuts an alignment zone array back to the part UsableZones accepts.
// An array with nothing left becomes nil rather than an empty slice, which is
// the shape a reader gives for an absent entry -- the writer omits either, so
// the two have to agree for a file to read back the same.
func trimZones(vals []float64, maxPairs int) []float64 {
	n := UsableZones(vals, maxPairs)
	if n == 0 {
		return nil
	}
	return vals[:n]
}

// Repair adjusts Private dictionary values which fall outside the range that
// gives them meaning.
//
// Readers call this, so that a dictionary handed to a caller always holds
// values a writer accepts and a second read gives back unchanged.  An entry a
// font may omit falls back to the value an absent entry would produce, rather
// than to the nearest bound: the latter would claim a value the font never
// gave.
func (p *PrivateDict) Repair() {
	p.BlueValues = trimZones(p.BlueValues, MaxBlueValuePairs)
	p.OtherBlues = trimZones(p.OtherBlues, MaxOtherBluePairs)

	p.BlueScale = RepairBlueScale(p.BlueScale)
	if !UsableBlueDistance(p.BlueShift) {
		p.BlueShift = DefaultBlueShift
	}
	if !UsableBlueDistance(p.BlueFuzz) {
		p.BlueFuzz = DefaultBlueFuzz
	}

	// An unusable stem width is dropped rather than moved to a bound: zero is
	// how a stem width the font does not give is recorded, and a renderer then
	// takes each stem from the charstring, which is a truer report than naming
	// a width the font never claimed.
	if !UsableStemWidth(p.StdHW) {
		p.StdHW = 0
	}
	if !UsableStemWidth(p.StdVW) {
		p.StdVW = 0
	}
}

// Validate reports an error for Private dictionary values a font cannot hold.
//
// Writers call this.  Values read from a file have been put right by
// [PrivateDict.Repair] already, so a failure here means the caller supplied
// them.  Zero passes for BlueScale, StdHW and StdVW, since the writer then
// omits the entry.
func (p *PrivateDict) Validate() error {
	if err := checkZones("BlueValues", p.BlueValues, MaxBlueValuePairs); err != nil {
		return err
	}
	if err := checkZones("OtherBlues", p.OtherBlues, MaxOtherBluePairs); err != nil {
		return err
	}

	if p.BlueScale != 0 && RepairBlueScale(p.BlueScale) != p.BlueScale {
		return fmt.Errorf("BlueScale %v outside (0, %v]", p.BlueScale, MaxBlueScale)
	}
	if !UsableBlueDistance(p.BlueShift) {
		return fmt.Errorf("BlueShift %v is not a usable distance", p.BlueShift)
	}
	if !UsableBlueDistance(p.BlueFuzz) {
		return fmt.Errorf("BlueFuzz %v is not a usable distance", p.BlueFuzz)
	}

	for _, w := range []struct {
		name string
		val  float64
	}{{"StdHW", p.StdHW}, {"StdVW", p.StdVW}} {
		if w.val != 0 && !UsableStemWidth(w.val) {
			return fmt.Errorf("%s %v outside (0, %v]", w.name, w.val, MaxStemWidth)
		}
	}
	return nil
}
