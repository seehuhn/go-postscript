// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/matrix"
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
