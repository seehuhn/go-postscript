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

package afm

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"seehuhn.de/go/postscript/type1"
)

const nameBody = `StartFontMetrics 4.1
FontName %s
StartCharMetrics 1
C 65 ; WX 100 ; N A ; B 0 0 100 100 ;
EndCharMetrics
EndFontMetrics
`

// An AFM file states the font name without restricting it, so the reader must
// repair the name in the same way as the Type 1 reader does.
func TestReadRepairsFontName(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"Test", "Test"},
		{"Foo(Bar)", "FooBar"},
		{"Foo\x01Bar", "FooBar"},
		{"Foo\xffBar", "FooBar"},
		{"Grüße", "Grüße"},
		{strings.Repeat("x", type1.MaxFontNameLen+1), ""},
	} {
		m, err := Read(strings.NewReader(fmt.Sprintf(nameBody, tc.in)))
		if err != nil {
			t.Errorf("name %q: %v", tc.in, err)
			continue
		}
		if m.FontName != tc.want {
			t.Errorf("name %q: got %q, want %q", tc.in, m.FontName, tc.want)
		}

		buf := &bytes.Buffer{}
		if err := m.Write(buf); err != nil {
			t.Errorf("name %q: write: %v", tc.in, err)
			continue
		}
		m2, err := Read(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Errorf("name %q: second read: %v", tc.in, err)
			continue
		}
		if m2.FontName != m.FontName {
			t.Errorf("name %q: round trip gave %q, want %q",
				tc.in, m2.FontName, m.FontName)
		}
	}
}

// Write must refuse a font name it cannot emit.
func TestWriteRejectsFontName(t *testing.T) {
	m, err := Read(strings.NewReader(fmt.Sprintf(nameBody, "Test")))
	if err != nil {
		t.Fatal(err)
	}
	m.FontName = "Times New Roman"
	if err := m.Write(&bytes.Buffer{}); err == nil {
		t.Error("Write accepted an invalid font name")
	}
}
