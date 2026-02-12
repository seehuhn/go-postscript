// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2021  Jochen Voss <voss@seehuhn.de>
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

package names

import (
	"testing"
	"unicode"
)

func TestNotdef(t *testing.T) {
	text := ToUnicode(".notdef", "")
	if text != "" {
		t.Errorf("expected empty result for .notdef but got %q", text)
	}
}

func TestToUnicode(t *testing.T) {
	cases := []struct {
		glyph    string
		fontName string
		res      []rune
	}{
		{"space", "Test", []rune{0x0020}},
		{"A", "Test", []rune{0x0041}},
		{"Lcommaaccent", "Test", []rune{0x013B}},
		{"uni20AC0308", "Test", []rune{0x20AC, 0x0308}},
		{"u1040C", "Test", []rune{0x1040C}},
		{"uniD801DC0C", "Test", []rune{}},
		{"uni20ac", "Test", []rune{}},
		{"Lcommaaccent_uni20AC0308_u1040C.alternate",
			"Test", []rune{0x013B, 0x20AC, 0x0308, 0x1040C}},
		{"uni013B", "Test", []rune{0x013B}},
		{"u013B", "Test", []rune{0x013B}},
		{"foo", "Test", []rune{}},
		{".notdef", "Test", []rune{}},
		{"Ogoneksmall", "Test", []rune{0xF6FB}},
		{"a7", "ZapfDingbats", []rune{0x271E}},
	}
	for i, test := range cases {
		out := []rune(ToUnicode(test.glyph, test.fontName))
		equal := len(out) == len(test.res)
		if equal {
			for j, c := range out {
				if test.res[j] != c {
					equal = false
					break
				}
			}
		}
		if !equal {
			t.Errorf("%d: expected %q but got %q",
				i, string(test.res), string(out))
		}
	}
}

func equal(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestFromUnicode(t *testing.T) {
	if FromUnicode("ﬄ") != "f_f_l" {
		t.Error("wrong name for ffl-ligature")
	}
	seen := make(map[string]bool)
	for r := range rune(65537) {
		if !unicode.IsGraphic(r) {
			continue
		}
		name := FromUnicode(string(r))
		if seen[name] {
			t.Error("duplicate name " + name)
		}
		seen[name] = true

		out := []rune(ToUnicode(name, ""))
		switch len(out) {
		case 0:
			t.Errorf("no output %c -> %s -> xxx", r, name)
		case 1:
			if r != out[0] {
				t.Errorf("mismatch %c(%04x) -> %s -> %c(%04x)",
					r, r, name, out, out)
			}
		default:
			rr := expand(r)
			if !equal(rr, out) {
				t.Errorf("multi-rune %c -> %s -> %c", r, name, out)
			}
		}
	}
}

func TestFromUnicodeEmpty(t *testing.T) {
	if FromUnicode("") != "" {
		t.Error("expected empty result for empty input")
	}
}

func TestGlyphMap(t *testing.T) {
	cases := []struct {
		file, glyph string
		ok          bool
		res         rune
	}{
		{"zapfdingbats", "a100", true, 0x275E},
		{"zapfdingbats", "a128", true, 0x2468},
		{"zapfdingbats", "a9", true, 0x2720},
		{"zapfdingbats", "finger", false, 0},
		{"glyphlist", "A", true, 'A'},
		{"glyphlist", "Izhitsadblgravecyrillic", true, 0x0476},
		{"glyphlist", "zukatakana", true, 0x30BA},
		{"glyphlist", "END", false, 0},
	}

	for i, test := range cases {
		res, ok := glyph.lookup(test.file, test.glyph)
		if ok != test.ok || res != test.res {
			t.Errorf("%d: expected %t/%c but got %t/%c",
				i, test.ok, test.res, ok, res)
		}
	}
}
