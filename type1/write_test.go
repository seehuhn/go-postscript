// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"bytes"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/postscript"
	"seehuhn.de/go/postscript/psenc"
)

func TestWrite(t *testing.T) {
	for _, useStdEnc := range []bool{false, true} {
		encoding := makeEmptyEncoding()
		if useStdEnc {
			encoding[psenc.StandardEncodingRev["A"]] = "A"
		} else {
			encoding[1] = "A"
		}
		F := &Font{
			CreationDate: time.Now().Round(time.Second),
			FontInfo: &FontInfo{
				FontName:           "Test",
				Version:            "1.000",
				Notice:             "Notice",
				Copyright:          "Copyright",
				FullName:           "Test Font",
				FamilyName:         "Test Family",
				Weight:             "Bold",
				ItalicAngle:        -11.5,
				IsFixedPitch:       false,
				UnderlinePosition:  12,
				UnderlineThickness: 14,
				FontMatrix:         matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
			},
			Outlines: &Outlines{
				Private: &PrivateDict{
					BlueValues: []float64{0, 10, 40, 50, 100, 120},
					OtherBlues: []float64{-20, -10},
					BlueScale:  0.1,
					BlueShift:  8,
					BlueFuzz:   2,
					StdHW:      10,
					StdVW:      20,
					ForceBold:  true,
				},
				Glyphs:   map[string]*Glyph{},
				Encoding: encoding,
			},
		}
		g := F.NewGlyph(".notdef", 100)
		g.MoveTo(10, 10)
		g.LineTo(20, 10)
		g.LineTo(20, 20)
		g.LineTo(10, 20)
		g.ClosePath()
		g = F.NewGlyph("A", 200)
		g.MoveTo(0, 10)
		g.LineTo(200, 10)
		g.LineTo(100, 110)
		g.ClosePath()

		buf := &bytes.Buffer{}
		for _, format := range []FileFormat{FormatPFA, FormatPFB, FormatBinary, FormatNoEExec} {
			buf.Reset()
			err := F.Write(buf, &WriterOptions{Format: format})
			if err != nil {
				t.Fatal(err)
			}

			isStdEnc := bytes.Contains(buf.Bytes(), []byte("/Encoding StandardEncoding def\n"))
			if isStdEnc != useStdEnc {
				t.Errorf("standardencoding: got %t, want %t", isStdEnc, useStdEnc)
			}

			r := bytes.NewReader(buf.Bytes())
			G, err := Read(r)
			if err != nil {
				t.Fatal(err)
			}

			if d := cmp.Diff(F, G); d != "" {
				t.Errorf("F and G differ: %s", d)
			}
		}
	}
}

// TestCreationDate ensures that the %%CreationDate comment is not included,
// if the CreationDate field is unset.
func TestCreationDate(t *testing.T) {
	for _, dateWanted := range []bool{true, false} {
		F := &Font{
			FontInfo: &FontInfo{
				FontName:   "Test",
				FontMatrix: [6]float64{0.001, 0, 0, 0.001, 0, 0},
			},
			Outlines: &Outlines{
				Private:  &PrivateDict{},
				Glyphs:   map[string]*Glyph{},
				Encoding: psenc.StandardEncoding[:],
			},
		}
		g := F.NewGlyph(".notdef", 100)
		g.MoveTo(10, 10)
		g.LineTo(20, 10)
		g.LineTo(20, 20)
		g.LineTo(10, 20)
		g.ClosePath()
		g = F.NewGlyph("A", 200)
		g.MoveTo(0, 10)
		g.LineTo(200, 10)
		g.LineTo(100, 110)
		g.ClosePath()

		if dateWanted {
			F.CreationDate = time.Now()
		}

		buf := &bytes.Buffer{}
		_, _, err := F.WritePDF(buf)
		if err != nil {
			t.Fatal(err)
		}

		body := buf.String()

		dateFound := strings.Contains(body, "%%CreationDate")
		if dateFound != dateWanted {
			t.Errorf("date: got %t, want %t", dateFound, dateWanted)
		}
	}
}

// stemTestFont returns a font with a single glyph carrying the given stems.
func stemTestFont(hStem, vStem []float64) *Font {
	F := &Font{
		FontInfo: &FontInfo{
			FontName:   "Test",
			FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{BlueScale: DefaultBlueScale},
			Glyphs:   map[string]*Glyph{},
			Encoding: makeEmptyEncoding(),
		},
	}
	g := F.NewGlyph(".notdef", 500)
	g.MoveTo(0, 0)
	g.LineTo(400, 0)
	g.LineTo(400, 700)
	g.ClosePath()
	g.HStem = hStem
	g.VStem = vStem
	return F
}

// TestWriteStemRoundTrip checks that fractional stem hints survive a
// write/read cycle.
func TestWriteStemRoundTrip(t *testing.T) {
	hStem := []float64{10.5, 30.25, 700, 679}
	vStem := []float64{-20, 15.5}
	F := stemTestFont(hStem, vStem)

	buf := &bytes.Buffer{}
	if err := F.Write(buf, nil); err != nil {
		t.Fatal(err)
	}
	G, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	g := G.Glyphs[".notdef"]
	if d := cmp.Diff(hStem, g.HStem); d != "" {
		t.Errorf("HStem round trip failed (-want +got):\n%s", d)
	}
	if d := cmp.Diff(vStem, g.VStem); d != "" {
		t.Errorf("VStem round trip failed (-want +got):\n%s", d)
	}
}

// TestWriteStemInvalid checks that the writer rejects stem hints which a
// charstring cannot express.
func TestWriteStemInvalid(t *testing.T) {
	cases := []struct {
		name         string
		hStem, vStem []float64
	}{
		{"odd count", []float64{0, 10, 20}, nil},
		{"edge out of range", nil, []float64{32000, 32010}},
		{"width out of range", []float64{-100, 31950}, nil},
		{"not finite", []float64{0, math.Inf(1)}, nil},
		{"overlap", nil, []float64{100, 130, 0, 110}},
		{"duplicate", []float64{0, 20, 0, 20}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			F := stemTestFont(c.hStem, c.vStem)
			if err := F.Write(&bytes.Buffer{}, nil); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// makeBBoxTestFont returns a font whose glyph extremes lie at fractional
// coordinates, some of them between curve control points.
func makeBBoxTestFont() *Font {
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		FontInfo: &FontInfo{
			FontName:   "BBoxTest",
			FontMatrix: matrix.Matrix{0.002, 0, 0, 0.002, 0, 0},
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{BlueValues: []float64{-10, 0}},
			Glyphs:   map[string]*Glyph{},
			Encoding: encoding,
		},
	}
	F.NewGlyph(".notdef", 250)
	g := F.NewGlyph("A", 500)
	g.MoveTo(10.25, -20.5)
	g.LineTo(300.75, -20.5)
	g.LineTo(300.75, 400)
	g.ClosePath()
	g = F.NewGlyph("B", 500)
	g.MoveTo(0, 0)
	g.CurveTo(0, 800, 200, 800, 200, 0) // the top lies at y = 600
	g.ClosePath()
	return F
}

// fontDictEntry writes the font without eexec and returns an entry of the
// font dictionary, as a PostScript interpreter sees it.
func fontDictEntry(t *testing.T, F *Font, key postscript.Name) postscript.Object {
	t.Helper()
	buf := &bytes.Buffer{}
	err := F.Write(buf, &WriterOptions{Format: FormatNoEExec})
	if err != nil {
		t.Fatal(err)
	}
	intp := postscript.NewInterpreter()
	err = intp.Execute(buf)
	if err != nil {
		t.Fatal(err)
	}
	for _, fd := range intp.FontDirectory {
		return fd.(postscript.Dict)[key]
	}
	t.Fatal("no font found")
	return nil
}

func TestWriteFontBBox(t *testing.T) {
	F := makeBBoxTestFont()

	obj := fontDictEntry(t, F, "FontBBox")
	arr, ok := obj.(postscript.Array)
	if !ok || len(arr) != 4 {
		t.Fatalf("FontBBox = %v", obj)
	}
	var got [4]float64
	for i, x := range arr {
		v, ok := x.(postscript.Integer)
		if !ok {
			t.Fatalf("FontBBox entry %d = %v, want an integer", i, x)
		}
		got[i] = float64(v)
	}
	bbox := rect.Rect{LLx: got[0], LLy: got[1], URx: got[2], URy: got[3]}

	// the box encloses every glyph, and no smaller integer box does
	var union rect.Rect
	for name := range F.Glyphs {
		glyphBBox := F.GlyphBBox(matrix.Identity, name)
		if glyphBBox.IsZero() {
			continue
		}
		if !bbox.Covers(glyphBBox) {
			t.Errorf("FontBBox %v does not contain glyph %q with box %v", bbox, name, glyphBBox)
		}
		union.Extend(glyphBBox)
	}
	if union.LLx-bbox.LLx >= 1 || union.LLy-bbox.LLy >= 1 ||
		bbox.URx-union.URx >= 1 || bbox.URy-union.URy >= 1 {
		t.Errorf("FontBBox %v is not tight around %v", bbox, union)
	}
}

func TestWriteFontBBoxNoInk(t *testing.T) {
	F := makeBBoxTestFont()
	F.Glyphs = map[string]*Glyph{}
	F.NewGlyph(".notdef", 250)
	F.NewGlyph("space", 250)

	obj := fontDictEntry(t, F, "FontBBox")
	want := postscript.Array{postscript.Integer(0), postscript.Integer(0), postscript.Integer(0), postscript.Integer(0)}
	if d := cmp.Diff(want, obj); d != "" {
		t.Errorf("FontBBox mismatch (-want +got):\n%s", d)
	}
}

func TestWriteEncodingLength(t *testing.T) {
	for _, n := range []int{0, 255, 257} {
		F := makeBBoxTestFont()
		if n == 0 {
			F.Encoding = nil
		} else {
			F.Encoding = make([]string, n)
			for i := range F.Encoding {
				F.Encoding[i] = ".notdef"
			}
		}
		err := F.Write(io.Discard, nil)
		if err == nil {
			t.Errorf("%d entries: expected an error", n)
		}
	}
}

// TestWriteEncodingRoundTrip checks that an encoding which leaves the code of
// an existing glyph unused reads back unchanged.
func TestWriteEncodingRoundTrip(t *testing.T) {
	F := makeBBoxTestFont() // code 65 ("A") is used, code 66 ("B") is not

	buf := &bytes.Buffer{}
	err := F.Write(buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	G, err := Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(F.Encoding, G.Encoding); d != "" {
		t.Errorf("encoding changed (-want +got):\n%s", d)
	}
}
