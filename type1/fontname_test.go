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
	"unicode/utf8"

	"seehuhn.de/go/geom/matrix"
)

func TestRepairFontName(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"Quire-Regular", "Quire-Regular"},
		{"ABCDEF+Quire-Regular", "ABCDEF+Quire-Regular"},
		{"", ""},

		// names outside ASCII survive
		{"Grüße-Regular", "Grüße-Regular"},
		{"宋体-Regular", "宋体-Regular"},

		// white space and delimiters are removed
		{"Times New Roman", "TimesNewRoman"},
		{"Foo(Bar)-Regular", "FooBar-Regular"},
		{"a\tb\nc\fd\re/f%g<h>i[j]k{l}m", "abcdefghijklm"},

		// characters which cannot be shown are removed: the C0 controls
		// PostScript does not count as white space, DEL, the C1 controls, the
		// format characters, and the spaces outside ASCII
		{"a\vb", "ab"},
		{"a\x01b\x1fc", "abc"},
		{"a\x7fb", "ab"},
		{"a\u0085b", "ab"},
		{"a\u200db", "ab"},
		{"a\u00a0b", "ab"},

		// invalid UTF-8 is removed, a correctly encoded U+FFFD is not
		{"a\xffb", "ab"},
		{"a\ufffdb", "a\ufffdb"},

		// a name which is still too long is dropped, not truncated
		{strings.Repeat("x", MaxFontNameLen), strings.Repeat("x", MaxFontNameLen)},
		{strings.Repeat("x", MaxFontNameLen+1), ""},
		{strings.Repeat("x", MaxFontNameLen) + " ", strings.Repeat("x", MaxFontNameLen)},
	} {
		if got := RepairFontName(tc.in); got != tc.want {
			t.Errorf("RepairFontName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Every name RepairFontName returns must be one CheckFontName accepts,
// otherwise a font read from a file could not be written back out again.
func TestRepairFontNameIsWritable(t *testing.T) {
	for _, in := range []string{
		"Quire-Regular", "", "Grüße-Regular", "宋体-Regular",
		"Times New Roman", "Foo(Bar)-Regular", "a\vb\x7fc\u00a0d",
		"a\xffb", strings.Repeat("x", MaxFontNameLen+1),
		strings.Repeat("宋", MaxFontNameLen),
	} {
		if err := CheckFontName(RepairFontName(in)); err != nil {
			t.Errorf("CheckFontName(RepairFontName(%q)): %v", in, err)
		}
	}
}

func TestCheckFontName(t *testing.T) {
	for _, valid := range []string{
		"", "Quire-Regular", "ABCDEF+Quire-Regular", "Grüße-Regular",
		"宋体-Regular", strings.Repeat("x", MaxFontNameLen),
	} {
		if err := CheckFontName(valid); err != nil {
			t.Errorf("CheckFontName(%q) = %v, want nil", valid, err)
		}
	}

	for _, invalid := range []string{
		strings.Repeat("x", MaxFontNameLen+1),
		"Times New Roman",
		"Foo(Bar)",
		"a/b",
		"a\vb",
		"a\x7fb",
		"a\u00a0b",
		"a\xffb",
	} {
		if err := CheckFontName(invalid); err == nil {
			t.Errorf("CheckFontName(%q) = nil, want error", invalid)
		}
	}
}

// The PostScript scanner accepts font names the writer refuses: it allows up
// to 4096 bytes and treats every byte above 127 as an ordinary name
// character.  Read must repair such a name, so that a font read from a file
// can be written back out again.
func TestReadRepairsFontName(t *testing.T) {
	for _, tc := range []struct {
		name, want string
	}{
		{"Foo\x01Bar", "FooBar"},       // a control character
		{"Foo\xffBar", "FooBar"},       // not valid UTF-8
		{"Grüße", "Grüße"},             // UTF-8 is kept
		{strings.Repeat("x", 200), ""}, // too long, so dropped
	} {
		src := strings.Replace(string(makeNamedFont(t, "Good")),
			"/FontName /Good def", "/FontName /"+tc.name+" def", 1)
		if strings.Contains(src, "/FontName /Good def") {
			t.Fatal("font name not patched")
		}

		F, err := Read(strings.NewReader(src))
		if err != nil {
			t.Errorf("name %q: read: %v", tc.name, err)
			continue
		}
		if F.FontName != tc.want {
			t.Errorf("name %q: got %q, want %q", tc.name, F.FontName, tc.want)
		}

		buf := &bytes.Buffer{}
		if err := F.Write(buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
			t.Errorf("name %q: write: %v", tc.name, err)
			continue
		}
		G, err := Read(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Errorf("name %q: second read: %v", tc.name, err)
			continue
		}
		if G.FontName != F.FontName {
			t.Errorf("name %q: round trip gave %q, want %q",
				tc.name, G.FontName, F.FontName)
		}
	}
}

// Write and WritePDF must refuse a name they cannot emit, instead of letting
// the PostScript name writer panic.
func TestWriteRejectsFontName(t *testing.T) {
	F, err := Read(bytes.NewReader(makeNamedFont(t, "Good")))
	if err != nil {
		t.Fatal(err)
	}
	F.FontName = "Times New Roman"

	if err := F.Write(&bytes.Buffer{}, nil); err == nil {
		t.Error("Write accepted an invalid font name")
	}
	if _, _, err := F.WritePDF(&bytes.Buffer{}); err == nil {
		t.Error("WritePDF accepted an invalid font name")
	}
}

// makeNamedFont returns a minimal Type 1 font in cleartext PFA form, so that
// tests can patch the /FontName entry.
func makeNamedFont(t *testing.T, name string) []byte {
	t.Helper()

	F := &Font{
		FontInfo: &FontInfo{
			FontName:   name,
			FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{},
			Glyphs:   map[string]*Glyph{},
			Encoding: makeEmptyEncoding(),
		},
	}
	g := F.NewGlyph(".notdef", 100)
	g.MoveTo(10, 10)
	g.LineTo(20, 10)
	g.LineTo(20, 20)
	g.ClosePath()

	buf := &bytes.Buffer{}
	if err := F.Write(buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCheckGlyphName(t *testing.T) {
	valid := []string{
		"A", ".notdef", "uni0041", "a_b", "one-half", "9", "Aä",
		"Overlapping-thick\\thin-line-border-design---character-3",
		strings.Repeat("x", maxGlyphNameLen),
		strings.Repeat("ä", maxGlyphNameLen/2),
	}
	for _, s := range valid {
		if err := CheckGlyphName(s); err != nil {
			t.Errorf("%q rejected: %v", s, err)
		}
	}

	invalid := []string{
		"", "A B", "A\tB", "A\nB", "A(B", "A)B", "A/B", "A%B",
		"A<B", "A>B", "A[B", "A]B", "A{B", "A}B", "A\x01B",
		"A\xffB", "A B",
		strings.Repeat("x", maxGlyphNameLen+1),
		strings.Repeat("ä", maxGlyphNameLen),
	}
	for _, s := range invalid {
		if err := CheckGlyphName(s); err == nil {
			t.Errorf("%q accepted", s)
		}
	}
}

func TestRepairGlyphName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"A", "A"},
		{".notdef", ".notdef"},
		{"A(B", "AB"},
		{"A B", "AB"},
		{"A\xffB", "AB"},
		{"(/)", ""},
		{strings.Repeat("x", maxGlyphNameLen), strings.Repeat("x", maxGlyphNameLen)},
		{strings.Repeat("x", maxGlyphNameLen+1), ""},
		// the repair can bring an over-long name back under the limit
		{strings.Repeat("x", maxGlyphNameLen) + "()", strings.Repeat("x", maxGlyphNameLen)},
	}
	for _, c := range cases {
		if got := RepairGlyphName(c.in); got != c.want {
			t.Errorf("RepairGlyphName(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// a repaired name is always one which can be written again
	for _, s := range []string{"A B", "A(B", "A\xffB", "x"} {
		if r := RepairGlyphName(s); r != "" {
			if err := CheckGlyphName(r); err != nil {
				t.Errorf("repaired %q to %q, still invalid: %v", s, r, err)
			}
		}
	}
}

// TestGlyphNameFastPath checks that the ASCII fast path agrees with the
// rune-by-rune rule it stands in for.
func TestGlyphNameFastPath(t *testing.T) {
	reference := func(s string) bool {
		if s == "" || len(s) > maxGlyphNameLen || !utf8.ValidString(s) {
			return false
		}
		for _, r := range s {
			if !allowedInName(r) {
				return false
			}
		}
		return true
	}

	var samples []string
	for c := range 256 {
		samples = append(samples, string([]byte{byte(c)}), "A"+string([]byte{byte(c)})+"B")
	}
	samples = append(samples, "", "Aä", "ä", "A B", "A​B", "\xff", "A\xffB")

	for _, s := range samples {
		want := reference(s)
		if got := CheckGlyphName(s) == nil; got != want {
			t.Errorf("CheckGlyphName(%q) == nil is %v, want %v", s, got, want)
		}
		if want && RepairGlyphName(s) != s {
			t.Errorf("RepairGlyphName(%q) changed a valid name", s)
		}
	}
}
