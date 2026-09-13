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
	"math"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/matrix"
)

func TestRepair(t *testing.T) {
	sevenPairs := []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	fivePairs := []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	cases := []struct {
		name string
		in   PrivateDict
		want PrivateDict
	}{
		{
			name: "unchanged",
			in:   PrivateDict{BlueValues: sevenPairs, OtherBlues: fivePairs, BlueScale: 0.05, BlueShift: 8, BlueFuzz: 2, StdHW: 10, StdVW: 20},
			want: PrivateDict{BlueValues: sevenPairs, OtherBlues: fivePairs, BlueScale: 0.05, BlueShift: 8, BlueFuzz: 2, StdHW: 10, StdVW: 20},
		},
		{
			name: "odd number of zone values",
			in:   PrivateDict{BlueValues: []float64{0, 10, 40}, BlueScale: DefaultBlueScale},
			want: PrivateDict{BlueValues: []float64{0, 10}, BlueScale: DefaultBlueScale},
		},
		{
			name: "descending pair",
			in:   PrivateDict{BlueValues: []float64{0, 10, 50, 40, 100, 110}, BlueScale: DefaultBlueScale},
			want: PrivateDict{BlueValues: []float64{0, 10}, BlueScale: DefaultBlueScale},
		},
		{
			name: "first pair broken",
			in:   PrivateDict{BlueValues: []float64{10, 0}, BlueScale: DefaultBlueScale},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
		{
			name: "too many blue pairs",
			in:   PrivateDict{BlueValues: append(sevenPairs, 14, 15), BlueScale: DefaultBlueScale},
			want: PrivateDict{BlueValues: sevenPairs, BlueScale: DefaultBlueScale},
		},
		{
			name: "too many other blue pairs",
			in:   PrivateDict{OtherBlues: append(fivePairs, 10, 11), BlueScale: DefaultBlueScale},
			want: PrivateDict{OtherBlues: fivePairs, BlueScale: DefaultBlueScale},
		},
		{
			name: "zone edge not finite",
			in:   PrivateDict{BlueValues: []float64{0, 10, 40, math.Inf(1)}, OtherBlues: []float64{math.NaN(), 0}, BlueScale: DefaultBlueScale},
			want: PrivateDict{BlueValues: []float64{0, 10}, BlueScale: DefaultBlueScale},
		},
		{
			name: "fractional and large zone edges are kept",
			in:   PrivateDict{BlueValues: []float64{-0.5, 0, 40000, 40010.5}, BlueScale: DefaultBlueScale},
			want: PrivateDict{BlueValues: []float64{-0.5, 0, 40000, 40010.5}, BlueScale: DefaultBlueScale},
		},
		{
			name: "blue scale zero",
			in:   PrivateDict{},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
		{
			name: "blue scale negative",
			in:   PrivateDict{BlueScale: -1},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
		{
			name: "blue scale too large",
			in:   PrivateDict{BlueScale: 2},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
		{
			name: "blue scale NaN",
			in:   PrivateDict{BlueScale: math.NaN()},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
		{
			name: "negative blue shift and fuzz",
			in:   PrivateDict{BlueScale: DefaultBlueScale, BlueShift: -1, BlueFuzz: -1},
			want: PrivateDict{BlueScale: DefaultBlueScale, BlueShift: DefaultBlueShift, BlueFuzz: DefaultBlueFuzz},
		},
		{
			name: "blue distances not finite",
			in:   PrivateDict{BlueScale: DefaultBlueScale, BlueShift: math.NaN(), BlueFuzz: math.Inf(1)},
			want: PrivateDict{BlueScale: DefaultBlueScale, BlueShift: DefaultBlueShift, BlueFuzz: DefaultBlueFuzz},
		},
		{
			name: "fractional blue distances are kept",
			in:   PrivateDict{BlueScale: DefaultBlueScale, BlueShift: 7.5, BlueFuzz: 0.5},
			want: PrivateDict{BlueScale: DefaultBlueScale, BlueShift: 7.5, BlueFuzz: 0.5},
		},
		{
			name: "negative stem widths",
			in:   PrivateDict{BlueScale: DefaultBlueScale, StdHW: -1, StdVW: -2},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
		{
			name: "stem widths too large",
			in:   PrivateDict{BlueScale: DefaultBlueScale, StdHW: MaxStemWidth + 1, StdVW: math.Inf(1)},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
		{
			name: "stem widths NaN",
			in:   PrivateDict{BlueScale: DefaultBlueScale, StdHW: math.NaN(), StdVW: math.NaN()},
			want: PrivateDict{BlueScale: DefaultBlueScale},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.in
			got.Repair()
			if d := cmp.Diff(c.want, got); d != "" {
				t.Errorf("repair gave the wrong result (-want +got):\n%s", d)
			}

			// repair is idempotent, and its result passes validation
			again := got
			again.Repair()
			if d := cmp.Diff(got, again); d != "" {
				t.Errorf("repair is not idempotent (-once +twice):\n%s", d)
			}
			if err := got.Validate(); err != nil {
				t.Errorf("repaired dict does not validate: %v", err)
			}
		})
	}
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name string
		in   PrivateDict
	}{
		{"odd number of zone values", PrivateDict{BlueValues: []float64{0, 10, 40}}},
		{"descending blue pair", PrivateDict{BlueValues: []float64{10, 0}}},
		{"descending other blue pair", PrivateDict{OtherBlues: []float64{10, 0}}},
		{"too many blue pairs", PrivateDict{BlueValues: make([]float64, 2*MaxBlueValuePairs+2)}},
		{"too many other blue pairs", PrivateDict{OtherBlues: make([]float64, 2*MaxOtherBluePairs+2)}},
		{"zone edge NaN", PrivateDict{BlueValues: []float64{math.NaN(), 0}}},
		{"zone edge infinite", PrivateDict{OtherBlues: []float64{0, math.Inf(1)}}},
		{"negative blue scale", PrivateDict{BlueScale: -1}},
		{"blue scale too large", PrivateDict{BlueScale: MaxBlueScale + 1}},
		{"blue scale NaN", PrivateDict{BlueScale: math.NaN()}},
		{"negative blue shift", PrivateDict{BlueShift: -1}},
		{"negative blue fuzz", PrivateDict{BlueFuzz: -1}},
		{"blue shift NaN", PrivateDict{BlueShift: math.NaN()}},
		{"blue fuzz infinite", PrivateDict{BlueFuzz: math.Inf(1)}},
		{"negative StdHW", PrivateDict{StdHW: -1}},
		{"negative StdVW", PrivateDict{StdVW: -1}},
		{"StdHW too large", PrivateDict{StdHW: MaxStemWidth + 1}},
		{"StdVW too large", PrivateDict{StdVW: MaxStemWidth + 1}},
		{"StdHW NaN", PrivateDict{StdHW: math.NaN()}},
		{"StdVW NaN", PrivateDict{StdVW: math.NaN()}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.in.Validate(); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestValidateZero checks that a dict left at its zero value can be written.
func TestValidateZero(t *testing.T) {
	if err := (&PrivateDict{}).Validate(); err != nil {
		t.Errorf("zero dict does not validate: %v", err)
	}
}

// makePrivateTestFont returns a minimal font with the given private dict.
func makePrivateTestFont(private *PrivateDict) *Font {
	encoding := makeEmptyEncoding()
	encoding[1] = "A"
	F := &Font{
		FontInfo: &FontInfo{
			FontName:   "Test",
			FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private:  private,
			Glyphs:   map[string]*Glyph{},
			Encoding: encoding,
		},
	}
	g := F.NewGlyph(".notdef", 100)
	g.MoveTo(10, 10)
	g.LineTo(20, 10)
	g.LineTo(20, 20)
	g.ClosePath()
	g = F.NewGlyph("A", 200)
	g.MoveTo(0, 10)
	g.LineTo(200, 10)
	g.LineTo(100, 110)
	g.ClosePath()
	return F
}

// TestWriteRefusesNonIntegerPrivate checks that the writer refuses a
// BlueValues, OtherBlues, BlueShift or BlueFuzz entry the format cannot spell.
// The fields are float64 because CFF allows a fraction there; a Type 1 file
// holds PostScript integers.
func TestWriteRefusesNonIntegerPrivate(t *testing.T) {
	for _, p := range []*PrivateDict{
		{BlueShift: 7.5},
		{BlueFuzz: 0.5},
		{BlueValues: []float64{0, 0.5}},
		{OtherBlues: []float64{-10.5, 0}},
		{BlueValues: []float64{0, 1e10}},
	} {
		F := makePrivateTestFont(p)
		if err := F.Write(&bytes.Buffer{}, nil); err == nil {
			t.Errorf("Write accepted %+v", p)
		}
	}
}

// TestReadRoundsFractionalBlueDistance checks that a fraction in a file is
// rounded on read, so that the font can be written out again.
func TestReadRoundsFractionalBlueDistance(t *testing.T) {
	F := makePrivateTestFont(&PrivateDict{
		BlueScale: DefaultBlueScale,
		BlueShift: DefaultBlueShift,
		BlueFuzz:  DefaultBlueFuzz,
	})
	G := writeAndRead(t, F, "/ForceBold false def", "/BlueShift 7.5 def\n/ForceBold false def")

	if got := G.Private.BlueShift; got != 8 {
		t.Errorf("BlueShift: got %v, want 8", got)
	}
	if err := G.Write(&bytes.Buffer{}, nil); err != nil {
		t.Errorf("the font cannot be written back out: %v", err)
	}
}

// TestWriteLargeIntegers checks that a large whole value is written as an
// integer, not in the exponent form a real would take.
func TestWriteLargeIntegers(t *testing.T) {
	F := makePrivateTestFont(&PrivateDict{
		BlueValues: []float64{0, 10, 1000000, 1000010},
		OtherBlues: []float64{-2000000, -1000000},
		BlueScale:  DefaultBlueScale,
		BlueShift:  3000000,
		BlueFuzz:   4000000,
	})
	buf := &bytes.Buffer{}
	if err := F.Write(buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/BlueValues [0 10 1000000 1000010] def",
		"/OtherBlues [-2000000 -1000000] def",
		"/BlueShift 3000000 def",
		"/BlueFuzz 4000000 def",
	} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("missing %q", want)
		}
	}

	G, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(F.Private, G.Private); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}

func TestWriteRefusesInvalidPrivate(t *testing.T) {
	F := makePrivateTestFont(&PrivateDict{BlueValues: []float64{10, 0}})

	if err := F.Write(&bytes.Buffer{}, nil); err == nil {
		t.Error("Write accepted an invalid private dict")
	}
	if _, _, err := F.WritePDF(&bytes.Buffer{}); err == nil {
		t.Error("WritePDF accepted an invalid private dict")
	}
}

// writeAndRead writes the font as plain text, applies the given replacement to
// the file, and reads the result back.
func writeAndRead(t *testing.T, F *Font, old, new string) *Font {
	t.Helper()

	buf := &bytes.Buffer{}
	if err := F.Write(buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	body := buf.String()
	if old != "" {
		if !strings.Contains(body, old) {
			t.Fatalf("font does not contain %q", old)
		}
		body = strings.Replace(body, old, new, 1)
	}

	G, err := Read(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return G
}

// TestReadRepairsPrivate checks that values a font cannot hold are put right
// on read, so that the font can be written out again.
func TestReadRepairsPrivate(t *testing.T) {
	cases := []struct {
		name string
		old  string
		new  string
		want PrivateDict
	}{
		{
			name: "descending pair",
			old:  "/BlueValues [0 10 40 50] def",
			new:  "/BlueValues [0 10 50 40] def",
			want: PrivateDict{BlueValues: []float64{0, 10}, BlueScale: 0.05, StdHW: 10, StdVW: 20},
		},
		{
			name: "fractional zone edge is rounded",
			old:  "/BlueValues [0 10 40 50] def",
			new:  "/BlueValues [0 10 40 50.5] def",
			want: PrivateDict{BlueValues: []float64{0, 10, 40, 51}, BlueScale: 0.05, StdHW: 10, StdVW: 20},
		},
		{
			name: "zone edge beyond 16 bits is kept",
			old:  "/BlueValues [0 10 40 50] def",
			new:  "/BlueValues [0 10 40000 40050] def",
			want: PrivateDict{BlueValues: []float64{0, 10, 40000, 40050}, BlueScale: 0.05, StdHW: 10, StdVW: 20},
		},
		{
			name: "zone edge beyond a PostScript integer",
			old:  "/BlueValues [0 10 40 50] def",
			new:  "/BlueValues [0 10 40 1e300] def",
			want: PrivateDict{BlueScale: 0.05, StdHW: 10, StdVW: 20},
		},
		{
			name: "blue shift beyond a PostScript integer",
			old:  "/ForceBold false def",
			new:  "/BlueShift 1e300 def\n/ForceBold false def",
			want: PrivateDict{
				BlueValues: []float64{0, 10, 40, 50},
				BlueScale:  0.05, BlueShift: DefaultBlueShift, StdHW: 10, StdVW: 20,
			},
		},
		{
			name: "too many pairs",
			old:  "/BlueValues [0 10 40 50] def",
			new:  "/BlueValues [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15] def",
			want: PrivateDict{
				BlueValues: []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
				BlueScale:  0.05, StdHW: 10, StdVW: 20,
			},
		},
		{
			name: "blue scale out of range",
			old:  "/BlueScale 0.05 def",
			new:  "/BlueScale -0.05 def",
			want: PrivateDict{
				BlueValues: []float64{0, 10, 40, 50},
				BlueScale:  DefaultBlueScale, StdHW: 10, StdVW: 20,
			},
		},
		{
			name: "blue scale near the default",
			old:  "/BlueScale 0.05 def",
			new:  "/BlueScale 0.0396255 def",
			want: PrivateDict{
				BlueValues: []float64{0, 10, 40, 50},
				BlueScale:  0.0396255, StdHW: 10, StdVW: 20,
			},
		},
		{
			name: "negative stem width",
			old:  "/StdHW [10] def",
			new:  "/StdHW [-10] def",
			want: PrivateDict{
				BlueValues: []float64{0, 10, 40, 50},
				BlueScale:  0.05, StdVW: 20,
			},
		},
		{
			name: "negative blue shift",
			old:  "/ForceBold false def",
			new:  "/BlueShift -3 def\n/ForceBold false def",
			want: PrivateDict{
				BlueValues: []float64{0, 10, 40, 50},
				BlueScale:  0.05, BlueShift: DefaultBlueShift, StdHW: 10, StdVW: 20,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			F := makePrivateTestFont(&PrivateDict{
				BlueValues: []float64{0, 10, 40, 50},
				BlueScale:  0.05,
				BlueShift:  DefaultBlueShift,
				BlueFuzz:   DefaultBlueFuzz,
				StdHW:      10,
				StdVW:      20,
			})
			G := writeAndRead(t, F, c.old, c.new)

			want := c.want
			// the reader fills in the values of the omitted entries
			if want.BlueShift == 0 {
				want.BlueShift = DefaultBlueShift
			}
			if want.BlueFuzz == 0 {
				want.BlueFuzz = DefaultBlueFuzz
			}
			if d := cmp.Diff(&want, G.Private); d != "" {
				t.Errorf("private dict differs (-want +got):\n%s", d)
			}

			// the repaired font can be written out and read back unchanged
			H := writeAndRead(t, G, "", "")
			if d := cmp.Diff(G.Private, H.Private); d != "" {
				t.Errorf("round trip failed (-first +second):\n%s", d)
			}
		})
	}
}

// TestWriteEmptyBlueValues checks that BlueValues, which a font must give, is
// written even when the font declares no alignment zones.
func TestWriteEmptyBlueValues(t *testing.T) {
	F := makePrivateTestFont(&PrivateDict{
		BlueScale: DefaultBlueScale,
		BlueShift: DefaultBlueShift,
		BlueFuzz:  DefaultBlueFuzz,
	})
	G := writeAndRead(t, F, "", "")

	if G.Private.BlueValues != nil {
		t.Errorf("BlueValues: got %v, want nil", G.Private.BlueValues)
	}

	buf := &bytes.Buffer{}
	if err := F.Write(buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("/BlueValues [] def")) {
		t.Error("BlueValues entry is missing")
	}
}
