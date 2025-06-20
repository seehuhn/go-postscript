// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2023  Jochen Voss <voss@seehuhn.de>
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
	"time"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/postscript/funit"
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
					BlueValues: []funit.Int16{0, 10, 40, 50, 100, 120},
					OtherBlues: []funit.Int16{-20, -10},
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
