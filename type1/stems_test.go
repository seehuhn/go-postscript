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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRepairStems(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want []float64
	}{
		{"empty", nil, nil},
		{"unpaired only", []float64{5}, nil},
		{"trailing value", []float64{0, 20, 50}, []float64{0, 20}},
		{"sorted", []float64{100, 120, 0, 20}, []float64{0, 20, 100, 120}},
		// a ghost hint has its edges in descending order
		{"ghost hint", []float64{700, 680, 0, 20}, []float64{0, 20, 700, 680}},
		{"duplicate", []float64{0, 20, 0, 20}, []float64{0, 20}},
		{"zero width duplicate", []float64{5, 5, 5, 5}, []float64{5, 5}},
		{"overlap", []float64{0, 30, 25, 60, 570, 600}, []float64{0, 30, 570, 600}},
		{"touching", []float64{0, 30, 30, 60}, []float64{0, 30, 30, 60}},
		{"overlap with later", []float64{0, 100, 10, 20, 50, 60}, []float64{0, 100}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := RepairStems(c.in)
			if d := cmp.Diff(c.want, got); d != "" {
				t.Errorf("wrong result (-want +got):\n%s", d)
			}
		})
	}
}

// TestDecodeCharStringStemSets checks how the stems of several hint sets
// are merged: a repeated hstem3 contributes its stems once, stems declared
// before an hstem3 are kept, and a stem which overlaps an earlier one is
// dropped.
func TestDecodeCharStringStemSets(t *testing.T) {
	hstem3 := func(cs []byte) []byte {
		cs = appendPair(cs, 100, 20)
		cs = appendPair(cs, 200, 20)
		cs = appendPair(cs, 300, 20)
		return append(cs, t1hstem3.Bytes()...)
	}

	var cs []byte
	cs = appendPair(cs, 0, 500)
	cs = append(cs, t1hsbw.Bytes()...)
	cs = appendPair(cs, 0, 20)
	cs = append(cs, t1hstem.Bytes()...)
	cs = hstem3(cs)
	cs = hstem3(cs)
	cs = appendPair(cs, 10, 5) // overlaps the first stem
	cs = append(cs, t1hstem.Bytes()...)
	cs = append(cs, t1endchar.Bytes()...)

	info := &decodeInfo{budget: newTestBudget()}
	g := info.decodeCharString(cs, "test")

	want := []float64{0, 20, 100, 120, 200, 220, 300, 320}
	if d := cmp.Diff(want, g.HStem); d != "" {
		t.Errorf("HStem mismatch (-want +got):\n%s", d)
	}
	if g.VStem != nil {
		t.Errorf("unexpected VStem %v", g.VStem)
	}
}
