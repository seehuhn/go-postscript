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
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/postscript/psenc"
)

// fraction returns the charstring encoding of p/q.
func fraction(p, q int) []byte {
	cs := appendPair(nil, p, q)
	return append(cs, t1div.Bytes()...)
}

// TestGridRoundTrip checks that values which the reader forms as sums of
// fractions survive a write/read cycle exactly.
func TestGridRoundTrip(t *testing.T) {
	var cs []byte
	// sbw with sidebearing (-199.5, 1/3) and width (1001/2, 0)
	cs = append(cs, fraction(-399, 2)...)
	cs = append(cs, fraction(1, 3)...)
	cs = append(cs, fraction(1001, 2)...)
	cs = append(cs, encInt(0)...)
	cs = append(cs, t1sbw.Bytes()...)
	// stems measured from the fractional sidebearing
	cs = append(cs, fraction(193, 3)...)
	cs = append(cs, encInt(10)...)
	cs = append(cs, t1vstem.Bytes()...)
	cs = append(cs, fraction(2, 7)...)
	cs = append(cs, fraction(1, 7)...)
	cs = append(cs, t1hstem.Bytes()...)
	// outline points with fractional coordinates
	cs = append(cs, fraction(1, 3)...)
	cs = append(cs, fraction(5, 11)...)
	cs = append(cs, t1rmoveto.Bytes()...)
	cs = append(cs, fraction(100, 7)...)
	cs = append(cs, t1hlineto.Bytes()...)
	cs = append(cs, fraction(1, 9)...)
	cs = append(cs, fraction(2, 9)...)
	cs = append(cs, fraction(3, 9)...)
	cs = append(cs, fraction(4, 9)...)
	cs = append(cs, fraction(5, 9)...)
	cs = append(cs, fraction(6, 9)...)
	cs = append(cs, t1rrcurveto.Bytes()...)
	cs = append(cs, t1closepath.Bytes()...)
	cs = append(cs, t1endchar.Bytes()...)

	g1 := (&decodeInfo{budget: newTestBudget()}).decodeCharString(cs, "test")
	if len(g1.Outline.Cmds) == 0 || len(g1.HStem) != 2 || len(g1.VStem) != 2 {
		t.Fatalf("test is broken: %v", g1)
	}
	if g1.WidthX != 500.5 {
		t.Errorf("wrong width: got %g, want 500.5", g1.WidthX)
	}

	cs2 := g1.encodeCharString()
	g2 := (&decodeInfo{budget: newTestBudget()}).decodeCharString(cs2, "test")
	if d := cmp.Diff(g1, g2); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}

// TestDecodeOutlineOutOfRange checks that a glyph with an outline which
// cannot be written back is read as a blank glyph which keeps its width.
func TestDecodeOutlineOutOfRange(t *testing.T) {
	cases := map[string][]byte{
		"large step":     encInt(40000),
		"division by 0":  fraction(1, 0),
		"large quotient": fraction(1<<30, 2),
	}
	for name, arg := range cases {
		t.Run(name, func(t *testing.T) {
			var cs []byte
			cs = appendPair(cs, 0, 500)
			cs = append(cs, t1hsbw.Bytes()...)
			cs = appendPair(cs, 0, 20)
			cs = append(cs, t1hstem.Bytes()...)
			cs = append(cs, arg...)
			cs = append(cs, t1hmoveto.Bytes()...)
			cs = append(cs, t1endchar.Bytes()...)

			g := (&decodeInfo{budget: newTestBudget()}).decodeCharString(cs, "test")
			if len(g.Outline.Cmds) != 0 || g.HStem != nil {
				t.Errorf("expected a blank glyph, got %v", g)
			}
			if g.WidthX != 500 {
				t.Errorf("wrong width: got %g, want 500", g.WidthX)
			}
		})
	}
}

// TestDecodeWidthOutOfRange checks that a glyph whose width cannot be
// written back is dropped, and that the encoding no longer refers to it.
func TestDecodeWidthOutOfRange(t *testing.T) {
	var wide []byte
	wide = appendPair(wide, 0, 40000)
	wide = append(wide, t1hsbw.Bytes()...)
	wide = append(wide, t1endchar.Bytes()...)

	var ok []byte
	ok = appendPair(ok, 0, 500)
	ok = append(ok, t1hsbw.Bytes()...)
	ok = append(ok, t1endchar.Bytes()...)

	charstrings := map[string][]byte{".notdef": ok, "A": wide, "B": ok}
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	encoding[66] = "B"

	glyphs := decodeGlyphs(charstrings, nil, nil, encoding, 1000)
	if _, found := glyphs["A"]; found {
		t.Error("glyph with out-of-range width was kept")
	}
	if _, found := glyphs["B"]; !found {
		t.Error("glyph B missing")
	}
	if encoding[65] != ".notdef" || encoding[66] != "B" {
		t.Errorf("wrong encoding: %q, %q", encoding[65], encoding[66])
	}
}

// TestSeacOutOfRange checks that a seac composite whose outline cannot be
// written back is left as decoded.
func TestSeacOutOfRange(t *testing.T) {
	var base []byte
	base = appendPair(base, 0, 500)
	base = append(base, t1hsbw.Bytes()...)
	base = appendPair(base, 31000, 0)
	base = append(base, t1rmoveto.Bytes()...)
	base = append(base, t1endchar.Bytes()...)

	var accent []byte
	accent = appendPair(accent, 0, 300)
	accent = append(accent, t1hsbw.Bytes()...)
	accent = appendPair(accent, 0, 0)
	accent = append(accent, t1rmoveto.Bytes()...)
	accent = append(accent, t1endchar.Bytes()...)

	// the accent lands at x = -31000, a step of 62000 from the base
	var composite []byte
	composite = appendPair(composite, 0, 500)
	composite = append(composite, t1hsbw.Bytes()...)
	composite = append(composite, encInt(0)...)
	composite = appendPair(composite, -31000, 0)
	composite = appendPair(composite, 65, 194)
	composite = append(composite, t1seac.Bytes()...)

	if psenc.StandardEncoding[65] != "A" || psenc.StandardEncoding[194] != "acute" {
		t.Fatal("test is broken")
	}
	charstrings := map[string][]byte{"A": base, "acute": accent, "Aacute": composite}
	glyphs := decodeGlyphs(charstrings, nil, nil, makeEmptyEncoding(), 1000)
	g := glyphs["Aacute"]
	if g == nil {
		t.Fatal("glyph Aacute missing")
	}
	if g.Outline != nil && len(g.Outline.Cmds) != 0 {
		t.Errorf("unexpected outline %v", g.Outline)
	}
}

// TestWriteOutOfRange checks that the writer rejects glyphs with widths or
// outline steps which a charstring cannot express.
func TestWriteOutOfRange(t *testing.T) {
	cases := map[string]func(g *Glyph){
		"width":   func(g *Glyph) { g.WidthX = 32000 },
		"width y": func(g *Glyph) { g.WidthY = -32000 },
		"step":    func(g *Glyph) { g.Outline.Coords[1].X = 32400 },
		"first":   func(g *Glyph) { g.Outline.Coords[0].Y = 32000 },
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			F := stemTestFont(nil, nil)
			modify(F.Glyphs[".notdef"])
			if err := F.Write(&bytes.Buffer{}, nil); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestWriteSnapped checks that the writer rounds API values to the grid.
func TestWriteSnapped(t *testing.T) {
	F := stemTestFont([]float64{0.1, 20.3}, nil)
	g := F.Glyphs[".notdef"]
	g.WidthX = 500.7
	g.Outline.Coords[1] = vec.Vec2{X: 400 + 1.0/3, Y: 2.0 / 3}

	buf := &bytes.Buffer{}
	if err := F.Write(buf, nil); err != nil {
		t.Fatal(err)
	}
	G, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(g.snapped(), G.Glyphs[".notdef"]); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}
