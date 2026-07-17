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
	"strings"
	"testing"

	"math"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/postscript"
)

// baseFontForMM builds a minimal single master font program (plaintext, no
// eexec) into which multiple master entries can be injected.
func baseFontForMM(t *testing.T) []byte {
	t.Helper()
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		FontInfo: &FontInfo{
			FontName:           "MMTest",
			FontMatrix:         matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
			UnderlinePosition:  100,
			UnderlineThickness: 50,
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{StdHW: 10, StdVW: 20},
			Glyphs:   map[string]*Glyph{},
			Encoding: encoding,
		},
	}
	g := F.NewGlyph(".notdef", 100)
	g.MoveTo(0, 0)
	g.LineTo(10, 0)
	g.LineTo(10, 10)
	g.ClosePath()
	g = F.NewGlyph("A", 200)
	g.MoveTo(0, 0)
	g.LineTo(100, 0)
	g.LineTo(50, 100)
	g.ClosePath()

	buf := &bytes.Buffer{}
	if err := F.Write(buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// injectMM inserts the given FontInfo-level and font-dict-level snippets into
// a plaintext font program produced by baseFontForMM.
func injectMM(t *testing.T, base []byte, fontInfo, fontDict string) []byte {
	t.Helper()
	s := string(base)
	if !strings.Contains(s, "end def\n/FontName") {
		t.Fatal("FontInfo anchor not found")
	}
	s = strings.Replace(s, "end def\n/FontName", fontInfo+"end def\n/FontName", 1)
	if !strings.Contains(s, "currentdict end") {
		t.Fatal("font dict anchor not found")
	}
	s = strings.Replace(s, "currentdict end", fontDict+"currentdict end", 1)
	return []byte(s)
}

const mmFontInfo = `/BlendAxisTypes [/Weight /Width] def
/BlendDesignPositions [[0 0][1 0][0 1][1 1]] def
/BlendDesignMap [[[50 0][900 1]] [[100 0][800 1]]] def
`

const mmFontDict = `/WeightVector [0.25 0.25 0.25 0.25] def
/Blend 3 dict dup begin
/FontBBox [[0 0 0 0][0 0 0 0][700 710 720 730][680 690 700 710]] def
/Private 6 dict dup begin
/BlueValues [[0 0 0 0][10 10 10 10]] def
/OtherBlues [[-20 -20 -20 -20][-10 -10 -10 -10]] def
/StemSnapH [[10 11 12 13]] def
/StemSnapV [[20 21 22 23]] def
/StdHW [[10 11 12 13]] def
/StdVW [[20 21 22 23]] def
end def
/FontInfo 3 dict dup begin
/UnderlinePosition [100 101 102 103] def
/UnderlineThickness [50 51 52 53] def
/ItalicAngle [0 -1 -2 -3] def
end def
end def
`

func TestReadMMFull(t *testing.T) {
	data := injectMM(t, baseFontForMM(t), mmFontInfo, mmFontDict)

	F, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	want := &MMInfo{
		Axes: []MMAxis{
			{Name: "Weight", Map: []MMMapPoint{{50, 0}, {900, 1}}},
			{Name: "Width", Map: []MMMapPoint{{100, 0}, {800, 1}}},
		},
		Masters:      [][]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}},
		WeightVector: []float64{0.25, 0.25, 0.25, 0.25},
		Blend: &MMBlend{
			FontBBox:           [][]float64{{0, 0, 0, 0}, {0, 0, 0, 0}, {700, 710, 720, 730}, {680, 690, 700, 710}},
			BlueValues:         [][]float64{{0, 0, 0, 0}, {10, 10, 10, 10}},
			OtherBlues:         [][]float64{{-20, -20, -20, -20}, {-10, -10, -10, -10}},
			StemSnapH:          [][]float64{{10, 11, 12, 13}},
			StemSnapV:          [][]float64{{20, 21, 22, 23}},
			StdHW:              []float64{10, 11, 12, 13},
			StdVW:              []float64{20, 21, 22, 23},
			UnderlinePosition:  []float64{100, 101, 102, 103},
			UnderlineThickness: []float64{50, 51, 52, 53},
			ItalicAngle:        []float64{0, -1, -2, -3},
		},
	}

	if F.MM == nil {
		t.Fatal("MM data missing")
	}
	if d := cmp.Diff(want, F.MM, cmpopts.IgnoreUnexported(MMInfo{})); d != "" {
		t.Errorf("MM mismatch (-want +got):\n%s", d)
	}

	// the deobfuscated charstrings and subrs must be retained
	if len(F.MM.charstrings) == 0 {
		t.Error("charstrings not retained")
	}
}

func TestReadMMDegraded(t *testing.T) {
	base := baseFontForMM(t)

	tests := []struct {
		name     string
		fontInfo string
		fontDict string
		wantMM   bool // MM != nil
		wantAxes bool // Axes populated
	}{
		{
			name:     "non-array WeightVector",
			fontInfo: mmFontInfo,
			fontDict: "/WeightVector (bad) def\n",
			wantMM:   false,
		},
		{
			name:     "too few masters",
			fontInfo: mmFontInfo,
			fontDict: "/WeightVector [1] def\n",
			wantMM:   false,
		},
		{
			name:     "truncated BlendDesignPositions",
			fontInfo: "/BlendAxisTypes [/Weight /Width] def\n/BlendDesignPositions [[0 0][1 0][0 1]] def\n/BlendDesignMap [[[50 0][900 1]] [[100 0][800 1]]] def\n",
			fontDict: "/WeightVector [0.25 0.25 0.25 0.25] def\n",
			wantMM:   true,
			wantAxes: false,
		},
		{
			name:     "row length mismatch",
			fontInfo: "/BlendAxisTypes [/Weight /Width] def\n/BlendDesignPositions [[0 0][1 0][0 1][1] ] def\n/BlendDesignMap [[[50 0][900 1]] [[100 0][800 1]]] def\n",
			fontDict: "/WeightVector [0.25 0.25 0.25 0.25] def\n",
			wantMM:   true,
			wantAxes: false,
		},
		{
			name:     "descending design map",
			fontInfo: "/BlendAxisTypes [/Weight /Width] def\n/BlendDesignPositions [[0 0][1 0][0 1][1 1]] def\n/BlendDesignMap [[[900 0][50 1]] [[100 0][800 1]]] def\n",
			fontDict: "/WeightVector [0.25 0.25 0.25 0.25] def\n",
			wantMM:   true,
			wantAxes: false,
		},
		{
			name:     "no MM entries",
			fontInfo: "",
			fontDict: "",
			wantMM:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := injectMM(t, base, tc.fontInfo, tc.fontDict)
			F, err := Read(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("read failed: %v", err)
			}
			if (F.MM != nil) != tc.wantMM {
				t.Fatalf("MM present: got %t, want %t", F.MM != nil, tc.wantMM)
			}
			if tc.wantMM {
				if (len(F.MM.Axes) > 0) != tc.wantAxes {
					t.Errorf("axes present: got %t, want %t", len(F.MM.Axes) > 0, tc.wantAxes)
				}
				if !tc.wantAxes && F.MM.Blend != nil {
					t.Error("degraded MM must have nil Blend")
				}
			}
			// glyphs must still decode
			if len(F.Glyphs) == 0 {
				t.Error("no glyphs decoded")
			}
		})
	}
}

// TestReadMMInfoNonFinite checks that a non-finite weight or design value
// discards all MM data.
func TestReadMMInfoNonFinite(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		fd := postscript.Dict{
			"WeightVector": postscript.Array{
				postscript.Real(0.25), postscript.Real(0.25),
				postscript.Real(0.25), postscript.Real(bad),
			},
		}
		if mm := readMMInfo(fd, nil); mm != nil {
			t.Errorf("non-finite weight %v must discard MM data", bad)
		}
	}
}
