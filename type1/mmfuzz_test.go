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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/postscript/internal/debug"
)

// FuzzInstantiate exercises Font.Instantiate across the design space of a
// multiple master font.  Once an instance has been produced, it must obey
// the same idempotence-after-first-write contract as FuzzFont: writing and
// re-reading it twice must yield identical results.
func FuzzInstantiate(f *testing.F) {
	mmPFA := debug.MakeMMFont()
	seeds := [][2]uint16{
		{0, 0},
		{0, 65535},
		{65535, 0},
		{65535, 65535},
		{32767, 32767},
	}
	for _, s := range seeds {
		f.Add(mmPFA, s[0], s[1])
	}

	// add PFB-encoded form of the same MM font
	mmFont, err := Read(bytes.NewReader(mmPFA))
	if err != nil {
		f.Fatal(err)
	}
	mmPFB := &bytes.Buffer{}
	if err := mmFont.Write(mmPFB, &WriterOptions{Format: FormatPFB}); err != nil {
		f.Fatal(err)
	}
	for _, s := range seeds {
		f.Add(mmPFB.Bytes(), s[0], s[1])
	}

	f.Fuzz(func(t *testing.T, data []byte, w1, w2 uint16) {
		font, err := Read(bytes.NewReader(data))
		if err != nil {
			return
		}
		if font.MM == nil || len(font.MM.Axes) == 0 {
			return
		}

		axes := font.VariationAxes()
		ws := [2]uint16{w1, w2}
		coords := make(map[string]float64, len(axes))
		for j, a := range axes {
			if j >= len(ws) {
				break
			}
			frac := float64(ws[j]) / 65535
			coords[a.Name] = a.Min + (a.Max-a.Min)*frac
		}

		inst, err := font.Instantiate(coords)
		if err != nil {
			// degraded MM fonts (e.g. non-corner masters) may legitimately
			// fail to instantiate
			return
		}

		buf := &bytes.Buffer{}
		if err := inst.Write(buf, &WriterOptions{Format: FormatBinary}); err != nil {
			t.Fatal(err)
		}
		i2, err := Read(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		if i2.MM != nil {
			t.Fatal("instance must not carry MM data")
		}

		buf2 := &bytes.Buffer{}
		if err := i2.Write(buf2, &WriterOptions{Format: FormatBinary}); err != nil {
			t.Fatal(err)
		}
		i3, err := Read(bytes.NewReader(buf2.Bytes()))
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(i2, i3) {
			t.Fatal(cmp.Diff(i2, i3))
		}
	})
}
