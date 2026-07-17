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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/postscript/afm"
	"seehuhn.de/go/postscript/internal/debug"
)

// wantMMInfo is the multiple master data of the synthetic font built by
// debug.MakeMMFont, used as the expected value in the reader tests.
var wantMMInfo = &MMInfo{
	Axes: []MMAxis{
		{Name: "Weight", Map: []MMMapPoint{{100, 0}, {400, 0.5}, {900, 1}}},
		{Name: "Width", Map: []MMMapPoint{{50, 0}, {100, 0.5}, {200, 1}}},
	},
	Masters:      [][]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}},
	WeightVector: []float64{0.25, 0.25, 0.25, 0.25},
	Blend: &MMBlend{
		FontBBox:           [][]float64{{0, 0, 0, 0}, {-100, -100, -100, -100}, {700, 720, 740, 760}, {700, 720, 740, 760}},
		BlueValues:         [][]float64{{0, 0, 0, 0}, {700, 710, 720, 730}},
		OtherBlues:         [][]float64{{-100, -100, -100, -100}, {-90, -90, -90, -90}},
		StemSnapH:          [][]float64{{40, 42, 44, 46}},
		StemSnapV:          [][]float64{{80, 82, 84, 86}},
		StdHW:              []float64{40, 42, 44, 46},
		StdVW:              []float64{80, 82, 84, 86},
		UnderlinePosition:  []float64{-100, -100, -100, -100},
		UnderlineThickness: []float64{50, 51, 52, 53},
		ItalicAngle:        []float64{0, 0, 0, 0},
	},
}

func pt(x, y float64) vec.Vec2 { return vec.Vec2{X: x, Y: y} }

// wantOutlines holds the hand-computed default-instance outlines of the blended
// glyphs.  With WeightVector [0.25 0.25 0.25 0.25] every blended value equals
// its base plus 0.25 times the sum of its three master deltas.
var wantOutlines = map[string]struct {
	width   float64
	outline *path.Data
}{
	"A": {500, (&path.Data{}).
		MoveTo(pt(100, 0)).
		LineTo(pt(400, 0)).
		LineTo(pt(250, 700)).
		Close()},
	"B": {480, (&path.Data{}).
		MoveTo(pt(50, 0)).
		LineTo(pt(450, 0)).
		LineTo(pt(450, 600)).
		LineTo(pt(50, 600)).
		LineTo(pt(50, 300)).
		Close()},
	"D": {520, (&path.Data{}).
		MoveTo(pt(100, 100)).
		CubeTo(pt(140, 110), pt(180, 120), pt(220, 120)).
		CubeTo(pt(260, 120), pt(300, 110), pt(340, 100)).
		LineTo(pt(340, 300)).
		Close()},
	"acute": {200, (&path.Data{}).
		MoveTo(pt(0, 600)).
		LineTo(pt(50, 700)).
		LineTo(pt(30, 600)).
		Close()},
	// seac composite: A plus acute translated by (150, 200)
	"Aacute": {500, (&path.Data{}).
		MoveTo(pt(100, 0)).
		LineTo(pt(400, 0)).
		LineTo(pt(250, 700)).
		Close().
		MoveTo(pt(150, 800)).
		LineTo(pt(200, 900)).
		LineTo(pt(180, 800)).
		Close()},
}

func TestMMFontBuilderDeterministic(t *testing.T) {
	if !bytes.Equal(debug.MakeMMFont(), debug.MakeMMFont()) {
		t.Error("MakeMMFont is not deterministic")
	}
}

func TestMMFontInfo(t *testing.T) {
	F, err := Read(bytes.NewReader(debug.MakeMMFont()))
	if err != nil {
		t.Fatal(err)
	}
	if F.MM == nil {
		t.Fatal("MM data missing")
	}
	if d := cmp.Diff(wantMMInfo, F.MM, cmpopts.IgnoreUnexported(MMInfo{})); d != "" {
		t.Errorf("MM mismatch (-want +got):\n%s", d)
	}
}

func TestMMFontOutlines(t *testing.T) {
	F, err := Read(bytes.NewReader(debug.MakeMMFont()))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range wantOutlines {
		g := F.Glyphs[name]
		if g == nil {
			t.Errorf("glyph %q missing", name)
			continue
		}
		if g.WidthX != want.width {
			t.Errorf("glyph %q width: got %g, want %g", name, g.WidthX, want.width)
		}
		if d := cmp.Diff(want.outline, g.Outline); d != "" {
			t.Errorf("glyph %q outline mismatch (-want +got):\n%s", name, d)
		}
	}
}

// TestMMFontReal exercises the reader on real Adobe Multiple Master fonts, if
// the user has supplied any.  It looks for *.pfb files in the "mm" subdirectory
// of QUIRE_TESTFONTS and skips cleanly when none are present.
func TestMMFontReal(t *testing.T) {
	base := os.Getenv("QUIRE_TESTFONTS")
	if base == "" {
		t.Skip("external test fonts not available (set QUIRE_TESTFONTS)")
	}
	dir := filepath.Join(base, "mm")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no mm test fonts: %v", err)
	}

	found := 0
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".pfb") {
			continue
		}
		found++
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			F, err := Read(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if F.MM == nil {
				t.Fatal("font parsed without MM data")
			}
			if len(F.MM.Axes) == 0 {
				t.Error("MM font has no axes")
			}
			if len(F.Glyphs) == 0 {
				t.Error("MM font has no glyphs")
			}

			// spot-check default-instance advance widths against the AFM
			afmPath := strings.TrimSuffix(filepath.Join(dir, name), filepath.Ext(name)) + ".afm"
			checkAdvanceWidths(t, F, afmPath)
		})
	}
	if found == 0 {
		t.Skip("no *.pfb MM test fonts present")
	}
}

func checkAdvanceWidths(t *testing.T, F *Font, afmPath string) {
	t.Helper()
	fd, err := os.Open(afmPath)
	if err != nil {
		return // no metrics alongside the font
	}
	defer fd.Close()
	metrics, err := afm.Read(fd)
	if err != nil {
		t.Logf("skipping width check, AFM unreadable: %v", err)
		return
	}
	checked := 0
	for name, gi := range metrics.Glyphs {
		g := F.Glyphs[name]
		if g == nil {
			continue
		}
		if math.Abs(g.WidthX-gi.WidthX) > 1 {
			t.Errorf("glyph %q width: font %g, AFM %g", name, g.WidthX, gi.WidthX)
		}
		checked++
		if checked >= 5 {
			break
		}
	}
}
