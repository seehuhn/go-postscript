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
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/matrix"
)

// TestDecodeCharStringMalformedNoPanic checks that decodeCharString
// returns a blank stub rather than panicking on malformed input.
// The inputs exercise the bounds and arity checks added to
// t1callothersubr.
func TestDecodeCharStringMalformedNoPanic(t *testing.T) {
	cases := []struct {
		name string
		cs   []byte
	}{
		// 0, 0, callothersubr — flex-end with argN=0
		{"othersubr0_argN0", []byte{0x8b, 0x8b, 0x0c, 0x10}},
		// 0, 0, 2, 1, callothersubr — flex-start with argN=2
		{"othersubr1_argN2", []byte{0x8b, 0x8b, 0x8d, 0x8c, 0x0c, 0x10}},
		// 0, 1, 2, callothersubr — flex coord pair with argN=1
		{"othersubr2_argN1", []byte{0x8b, 0x8c, 0x8d, 0x0c, 0x10}},
		// 0, 3, callothersubr — hint replacement with argN=0
		{"othersubr3_argN0", []byte{0x8b, 0x8e, 0x0c, 0x10}},
		// -1 (5-byte int), 0, callothersubr — negative argN
		{"argN_negative", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0x8b, 0x0c, 0x10}},
		// 2_000_000 (5-byte int), 0, callothersubr — argN above the threshold
		{"argN_excessive", []byte{0xff, 0x00, 0x1e, 0x84, 0x80, 0x8b, 0x0c, 0x10}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := &decodeInfo{}
			g := info.decodeCharString(tc.cs, "test")
			if g == nil {
				t.Fatal("expected non-nil stub")
			}
			if len(g.Outline.Cmds) != 0 || len(g.HStem) != 0 || len(g.VStem) != 0 {
				t.Errorf("expected blank stub, got %+v", g)
			}
		})
	}
}

// TestReadMalformedCharstringRoundTrip checks that a font containing
// a malformed glyph charstring reads successfully (with the bad glyph
// substituted by a blank glyph), and that the resulting font writes
// back and re-reads identically.
func TestReadMalformedCharstringRoundTrip(t *testing.T) {
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		FontInfo: &FontInfo{
			FontName:   "Test",
			FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{},
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

	var buf bytes.Buffer
	if err := F.Write(&buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}

	// replace /A's charstring with `0 200 hsbw 0 0 callothersubr` —
	// the leading hsbw sets WidthX before the trailing bytes panic, so
	// reading must preserve the width on the substituted glyph.
	malicious := obfuscateCharstring(
		[]byte{0x8b, 0xf7, 0x5c, 0x0d, 0x8b, 0x8b, 0x0c, 0x10},
		[]byte{0, 0, 0, 0})
	patched, ok := patchCharstring(buf.Bytes(), "A", malicious)
	if !ok {
		t.Fatal("could not locate /A charstring in PFA output")
	}

	// reading must succeed and substitute a blank glyph for /A
	F1, err := Read(bytes.NewReader(patched))
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	gA, ok := F1.Glyphs["A"]
	if !ok || gA == nil {
		t.Fatal("glyph A missing after substitution")
	}
	if gA.Outline == nil || len(gA.Outline.Cmds) != 0 {
		t.Errorf("glyph A: expected blank outline, got %d cmds", len(gA.Outline.Cmds))
	}
	if gA.WidthX != 200 {
		t.Errorf("glyph A: WidthX not preserved, got %v, want 200", gA.WidthX)
	}
	if len(gA.HStem) != 0 || len(gA.VStem) != 0 {
		t.Errorf("glyph A: hint state must not leak into the substituted glyph (HStem=%v, VStem=%v)",
			gA.HStem, gA.VStem)
	}
	gNotdef, ok := F1.Glyphs[".notdef"]
	if !ok || gNotdef == nil || len(gNotdef.Outline.Cmds) == 0 {
		t.Errorf(".notdef should be intact, got %v", gNotdef)
	}

	// round-trip: write F1 and read again, must match F1 exactly
	var buf2 bytes.Buffer
	if err := F1.Write(&buf2, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	F2, err := Read(bytes.NewReader(buf2.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(F1, F2); diff != "" {
		t.Errorf("round-trip differs (-F1 +F2):\n%s", diff)
	}
}

// patchCharstring replaces the obfuscated bytes of glyph `name` in a
// FormatNoEExec PFA, returning the modified buffer and true on success.
// The input format is `/<name> <len> RD <bytes...> ND`.
func patchCharstring(pfa []byte, name string, newBytes []byte) ([]byte, bool) {
	needle := []byte("/" + name + " ")
	i := bytes.Index(pfa, needle)
	if i < 0 {
		return nil, false
	}
	j := i + len(needle)
	k := j
	for k < len(pfa) && pfa[k] >= '0' && pfa[k] <= '9' {
		k++
	}
	oldLen, err := strconv.Atoi(string(pfa[j:k]))
	if err != nil {
		return nil, false
	}
	if !bytes.HasPrefix(pfa[k:], []byte(" RD ")) {
		return nil, false
	}
	bytesStart := k + len(" RD ")
	bytesEnd := bytesStart + oldLen
	if bytesEnd > len(pfa) {
		return nil, false
	}
	out := make([]byte, 0, len(pfa)+len(newBytes)-oldLen+8)
	out = append(out, pfa[:j]...)
	out = append(out, strconv.Itoa(len(newBytes))...)
	out = append(out, " RD "...)
	out = append(out, newBytes...)
	out = append(out, pfa[bytesEnd:]...)
	return out, true
}
