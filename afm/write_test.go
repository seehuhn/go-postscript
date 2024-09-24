// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2024  Jochen Voss <voss@seehuhn.de>
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

package afm

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"seehuhn.de/go/postscript/funit"
)

var (
	testMetrics = &Metrics{
		Glyphs: map[string]*GlyphInfo{
			".notdef": {
				WidthX: 500,
				BBox: funit.Rect16{
					URx: 500,
					URy: 800,
				},
			},
			"f": {
				WidthX: 400,
				BBox: funit.Rect16{
					LLx: 20,
					LLy: -100,
					URx: 500,
					URy: 800,
				},
				Ligatures: map[string]string{"f": "ff"},
			},
			"ff": {
				WidthX: 700,
				BBox: funit.Rect16{
					LLx: 20,
					LLy: 100,
					URx: 750,
					URy: 810,
				},
			},
		},
		Encoding: []string{
			".notdef",
			"f",
			".notdef",
			"ff",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
		},
		FontName:           "Test",
		FullName:           "Test Font",
		CapHeight:          750,
		XHeight:            451,
		Ascent:             812,
		Descent:            -203,
		UnderlinePosition:  -400,
		UnderlineThickness: 5,
		ItalicAngle:        -6,
		IsFixedPitch:       false,
		Kern: []*KernPair{
			{"f", "f", -20},
		},
	}
)

func TestWriteReadCycle(t *testing.T) {
	buf := &bytes.Buffer{}

	err := testMetrics.Write(buf)
	if err != nil {
		t.Fatal(err)
	}

	m2, err := Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(testMetrics, m2); d != "" {
		t.Fatalf("mismatch (-want +got):\n%s", d)
	}
}

func FuzzReadAFM(f *testing.F) {
	buf := &bytes.Buffer{}
	err := testMetrics.Write(buf)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(buf.Bytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		info, err := Read(bytes.NewReader(data))
		if err != nil {
			return
		}

		buf := &bytes.Buffer{}
		err = info.Write(buf)
		if err != nil {
			t.Fatal(err)
		}

		info2, err := Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(info, info2) {
			t.Fatalf("mismatch: %s", cmp.Diff(info, info2))
		}
	})
}
