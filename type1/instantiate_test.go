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
	"testing"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/postscript/funit"
	"seehuhn.de/go/postscript/internal/debug"
)

// readMMFixture reads the synthetic multiple master font built by
// debug.MakeMMFont.
func readMMFixture(t *testing.T) *Font {
	t.Helper()
	F, err := Read(bytes.NewReader(debug.MakeMMFont()))
	if err != nil {
		t.Fatal(err)
	}
	if F.MM == nil {
		t.Fatal("MM data missing")
	}
	return F
}

// TestInstantiateCorners pins each of the four corners of the design space and
// checks that the result reproduces the corresponding master outline exactly.
func TestInstantiateCorners(t *testing.T) {
	F := readMMFixture(t)

	type want struct {
		width   float64
		outline *path.Data
	}
	tests := []struct {
		name   string
		coords map[string]float64
		glyph  string
		want   want
	}{
		{
			name:   "master1 A",
			coords: map[string]float64{"Weight": 100, "Width": 50},
			glyph:  "A",
			want: want{440, (&path.Data{}).
				MoveTo(pt(60, 0)).LineTo(pt(360, 0)).LineTo(pt(250, 660)).Close()},
		},
		{
			name:   "master2 A",
			coords: map[string]float64{"Weight": 900, "Width": 50},
			glyph:  "A",
			want: want{520, (&path.Data{}).
				MoveTo(pt(100, 0)).LineTo(pt(400, 0)).LineTo(pt(250, 700)).Close()},
		},
		{
			name:   "master3 A",
			coords: map[string]float64{"Weight": 100, "Width": 200},
			glyph:  "A",
			want: want{520, (&path.Data{}).
				MoveTo(pt(100, 0)).LineTo(pt(400, 0)).LineTo(pt(250, 700)).Close()},
		},
		{
			name:   "master4 A",
			coords: map[string]float64{"Weight": 900, "Width": 200},
			glyph:  "A",
			want: want{520, (&path.Data{}).
				MoveTo(pt(140, 0)).LineTo(pt(440, 0)).LineTo(pt(250, 740)).Close()},
		},
		{
			name:   "master1 B",
			coords: map[string]float64{"Weight": 100, "Width": 50},
			glyph:  "B",
			want: want{480, (&path.Data{}).
				MoveTo(pt(50, 0)).LineTo(pt(450, 0)).LineTo(pt(450, 560)).
				LineTo(pt(90, 560)).LineTo(pt(90, 300)).Close()},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inst, err := F.Instantiate(tc.coords)
			if err != nil {
				t.Fatal(err)
			}
			g := inst.Glyphs[tc.glyph]
			if g == nil {
				t.Fatalf("glyph %q missing", tc.glyph)
			}
			if g.WidthX != tc.want.width {
				t.Errorf("width: got %g, want %g", g.WidthX, tc.want.width)
			}
			if d := cmp.Diff(tc.want.outline, g.Outline); d != "" {
				t.Errorf("outline mismatch (-want +got):\n%s", d)
			}
		})
	}
}

// TestInstantiateDefault checks that Instantiate(nil) and the design-space
// midpoint both reproduce the default-instance glyphs bit-identically to what
// Read decoded.
func TestInstantiateDefault(t *testing.T) {
	F := readMMFixture(t)

	nilInst, err := F.Instantiate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if nilInst.MM != nil {
		t.Error("instance must not carry MM data")
	}
	if d := cmp.Diff(F.Glyphs, nilInst.Glyphs); d != "" {
		t.Errorf("Instantiate(nil) glyphs differ from Read (-read +inst):\n%s", d)
	}

	// midpoint design coords equal the default instance
	mid, err := F.Instantiate(map[string]float64{"Weight": 400, "Width": 100})
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(nilInst.Glyphs, mid.Glyphs); d != "" {
		t.Errorf("midpoint glyphs differ from default (-nil +mid):\n%s", d)
	}
}

// TestInstantiateMetrics checks the blended Private and FontInfo entries of the
// default instance against hand-computed values.
func TestInstantiateMetrics(t *testing.T) {
	F := readMMFixture(t)
	inst, err := F.Instantiate(nil)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff([]funit.Int16{0, 715}, inst.Private.BlueValues); d != "" {
		t.Errorf("BlueValues mismatch (-want +got):\n%s", d)
	}
	if d := cmp.Diff([]funit.Int16{-100, -90}, inst.Private.OtherBlues); d != "" {
		t.Errorf("OtherBlues mismatch (-want +got):\n%s", d)
	}
	if inst.Private.StdHW != 43 {
		t.Errorf("StdHW: got %g, want 43", inst.Private.StdHW)
	}
	if inst.Private.StdVW != 83 {
		t.Errorf("StdVW: got %g, want 83", inst.Private.StdVW)
	}
	if inst.UnderlinePosition != -100 {
		t.Errorf("UnderlinePosition: got %g, want -100", float64(inst.UnderlinePosition))
	}
	if inst.UnderlineThickness != 51.5 {
		t.Errorf("UnderlineThickness: got %g, want 51.5", float64(inst.UnderlineThickness))
	}
	if inst.ItalicAngle != 0 {
		t.Errorf("ItalicAngle: got %g, want 0", inst.ItalicAngle)
	}

	// a corner selects a single master's blended values
	corner, err := F.Instantiate(map[string]float64{"Weight": 900, "Width": 50})
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff([]funit.Int16{0, 710}, corner.Private.BlueValues); d != "" {
		t.Errorf("corner BlueValues mismatch (-want +got):\n%s", d)
	}
	if corner.Private.StdVW != 82 {
		t.Errorf("corner StdVW: got %g, want 82", corner.Private.StdVW)
	}
}

// TestInstantiateName checks the derived instance FontName.
func TestInstantiateName(t *testing.T) {
	F := readMMFixture(t)

	def, err := F.Instantiate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if def.FontName != "QuireMMTest_400_100" {
		t.Errorf("default FontName: got %q, want %q", def.FontName, "QuireMMTest_400_100")
	}

	corner, err := F.Instantiate(map[string]float64{"Weight": 900, "Width": 200})
	if err != nil {
		t.Fatal(err)
	}
	if corner.FontName != "QuireMMTest_900_200" {
		t.Errorf("corner FontName: got %q, want %q", corner.FontName, "QuireMMTest_900_200")
	}
}

// TestVariationAxes checks the reported design axes and default round-trip.
func TestVariationAxes(t *testing.T) {
	F := readMMFixture(t)

	axes := F.VariationAxes()
	want := []VariationAxis{
		{Name: "Weight", Min: 100, Default: 400, Max: 900},
		{Name: "Width", Min: 50, Default: 100, Max: 200},
	}
	if d := cmp.Diff(want, axes); d != "" {
		t.Errorf("VariationAxes mismatch (-want +got):\n%s", d)
	}

	// the derived default normalizes back to the corner-weight sum (0.5 here)
	for j, a := range F.MM.Axes {
		norm := normalizeDesign(a.Map, axes[j].Default)
		if math.Abs(norm-0.5) > 1e-9 {
			t.Errorf("axis %q default normalized to %g, want 0.5", a.Name, norm)
		}
	}

	// a non-MM font has no axes
	if (&Font{}).VariationAxes() != nil {
		t.Error("non-MM font must return nil axes")
	}
}

// TestInstantiateErrors checks the error paths.
func TestInstantiateErrors(t *testing.T) {
	F := readMMFixture(t)

	if _, err := F.Instantiate(map[string]float64{"Bogus": 1}); err == nil {
		t.Error("unknown axis name must fail")
	}

	if _, err := (&Font{}).Instantiate(nil); err == nil {
		t.Error("non-MM font must fail")
	}

	incomplete := &Font{MM: &MMInfo{WeightVector: []float64{0.5, 0.5}}}
	if _, err := incomplete.Instantiate(nil); err == nil {
		t.Error("incomplete blend data must fail")
	}

	nonCorner := &Font{MM: &MMInfo{
		Axes:         []MMAxis{{Name: "Weight", Map: []MMMapPoint{{0, 0}, {1, 1}}}},
		Masters:      [][]float64{{0.5}, {1}},
		WeightVector: []float64{0.5, 0.5},
	}}
	if _, err := nonCorner.Instantiate(nil); err == nil {
		t.Error("non-corner masters must fail")
	}
}

// TestInstantiateClamp checks that out-of-range design coordinates are clamped.
func TestInstantiateClamp(t *testing.T) {
	F := readMMFixture(t)

	lo, err := F.Instantiate(map[string]float64{"Weight": -1000, "Width": -1000})
	if err != nil {
		t.Fatal(err)
	}
	corner, err := F.Instantiate(map[string]float64{"Weight": 100, "Width": 50})
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(corner.Glyphs["A"].Outline, lo.Glyphs["A"].Outline); d != "" {
		t.Errorf("clamped low glyph differs from corner (-corner +clamped):\n%s", d)
	}
}

// TestInstantiateWriteRoundTrip checks that an instance is an ordinary font the
// existing writer can embed and re-read.
func TestInstantiateWriteRoundTrip(t *testing.T) {
	F := readMMFixture(t)
	inst, err := F.Instantiate(nil)
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := inst.Write(buf, &WriterOptions{Format: FormatBinary}); err != nil {
		t.Fatal(err)
	}
	back, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if back.MM != nil {
		t.Error("re-read instance must not be a multiple master font")
	}
	for _, name := range []string{"A", "B", "D", "acute", "Aacute"} {
		g0 := inst.Glyphs[name]
		g1 := back.Glyphs[name]
		if g0 == nil || g1 == nil {
			t.Errorf("glyph %q missing after round trip", name)
			continue
		}
		if g0.WidthX != g1.WidthX {
			t.Errorf("glyph %q width: got %g, want %g", name, g1.WidthX, g0.WidthX)
		}
		if d := cmp.Diff(g0.Outline, g1.Outline); d != "" {
			t.Errorf("glyph %q outline changed by round trip (-inst +back):\n%s", name, d)
		}
	}
}
