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
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/postscript/funit"
)

func makeEmptyEncoding() []string {
	encoding := make([]string, 256)
	for i := range encoding {
		encoding[i] = ".notdef"
	}
	return encoding
}

func FuzzFont(f *testing.F) {
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		CreationDate: time.Now(),
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
			FontMatrix:         [6]float64{0.001, 0, 0, 0.001, 0, 0},
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
	g = F.NewGlyph("A", 200)
	g.MoveTo(0, 10)
	g.LineTo(200, 10)
	g.LineTo(100, 110)

	buf := &bytes.Buffer{}
	ff := []FileFormat{FormatPFA, FormatPFB, FormatBinary, FormatNoEExec}
	for _, format := range ff {
		buf.Reset()
		err := F.Write(buf, &WriterOptions{Format: format})
		if err != nil {
			f.Fatal(err)
		}
		f.Add(buf.Bytes(), uint8(format))
	}

	f.Fuzz(func(t *testing.T, data []byte, format uint8) {
		i1, err := Read(bytes.NewReader(data))
		if err != nil {
			return
		}

		buf := &bytes.Buffer{}
		err = i1.Write(buf, &WriterOptions{Format: FileFormat(format % uint8(len(ff)))})
		if err != nil {
			t.Fatal(err)
		}

		i2, err := Read(bytes.NewReader(buf.Bytes()))
		if err != nil {
			os.WriteFile("debug.pfa", buf.Bytes(), 0644)
			t.Fatal(err)
		}

		if !reflect.DeepEqual(i1, i2) {
			t.Fatal(cmp.Diff(i1, i2))
		}
	})
}
