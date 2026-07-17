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

// Package debug provides builders for synthetic PostScript font programs
// used in tests.
package debug

import (
	"bytes"
	"fmt"
	"sort"
)

// MakeMMFont returns a complete, valid two-axis Multiple Master Type 1 font
// program in PFA (plain-text, hex eexec) form.
//
// The font has two design axes (Weight and Width), four corner masters and a
// default WeightVector of [0.25 0.25 0.25 0.25], so the default instance is a
// genuine blend of all four masters.  Every blended charstring coordinate is
// chosen so its default-instance value is an exact integer, which makes the
// decoded outlines analytically computable.  The charstrings exercise the
// multiple-master blend OtherSubrs 14, 15 and 17, a seac composite glyph and a
// glyph combining flex with blend in a single charstring.
//
// The output is deterministic: repeated calls return identical bytes.
func MakeMMFont() []byte {
	priv := buildPrivate()

	buf := &bytes.Buffer{}
	buf.WriteString(clearText)
	buf.WriteString("currentfile eexec\n")
	writeHex(buf, eexecEncrypt(priv))
	buf.WriteString(trailer)
	return buf.Bytes()
}

// clearText is the unencrypted portion of the font program.  It defines the
// FontInfo (including the multiple-master axis descriptions), the top-level
// font dictionary with WeightVector and the per-master /Blend dictionary, and
// ends with "currentfile eexec".
const clearText = `%!FontType1-1.1: QuireMMTest 001.000
10 dict begin
/FontInfo 16 dict dup begin
/version (001.000) def
/FullName (Quire MM Test) def
/FamilyName (Quire MM Test) def
/Weight (All) def
/ItalicAngle 0 def
/isFixedPitch false def
/UnderlinePosition -100 def
/UnderlineThickness 50 def
/BlendAxisTypes [/Weight /Width] def
/BlendDesignPositions [[0 0][1 0][0 1][1 1]] def
/BlendDesignMap [[[100 0][400 0.5][900 1]] [[50 0][100 0.5][200 1]]] def
end def
/FontName /QuireMMTest def
/Encoding StandardEncoding def
/PaintType 0 def
/FontType 1 def
/FontMatrix [0.001 0 0 0.001 0 0] def
/FontBBox [0 -100 760 900] def
/WeightVector [0.25 0.25 0.25 0.25] def
/Blend 3 dict dup begin
/FontBBox [[0 0 0 0][-100 -100 -100 -100][700 720 740 760][700 720 740 760]] def
/Private 6 dict dup begin
/BlueValues [[0 0 0 0][700 710 720 730]] def
/OtherBlues [[-100 -100 -100 -100][-90 -90 -90 -90]] def
/StemSnapH [[40 42 44 46]] def
/StemSnapV [[80 82 84 86]] def
/StdHW [[40 42 44 46]] def
/StdVW [[80 82 84 86]] def
end def
/FontInfo 3 dict dup begin
/UnderlinePosition [-100 -100 -100 -100] def
/UnderlineThickness [50 51 52 53] def
/ItalicAngle [0 0 0 0] def
end def
end def
currentdict end
`

// trailer is the 512 zero bytes and cleartomark that terminate an eexec block.
const trailer = `
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
cleartomark
`

// buildPrivate assembles the plaintext of the eexec-encrypted Private
// dictionary, including the RD/ND/NP definitions, the default-instance Private
// entries and the CharStrings.
func buildPrivate() []byte {
	buf := &bytes.Buffer{}
	buf.WriteString("dup /Private 15 dict dup begin\n")
	buf.WriteString("/RD {string currentfile exch readstring pop} executeonly def\n")
	buf.WriteString("/ND {def} executeonly def\n")
	buf.WriteString("/NP {put} executeonly def\n")
	buf.WriteString("/Subrs 0 array\n")
	buf.WriteString("/BlueValues [-20 0 700 720] def\n")
	buf.WriteString("/OtherBlues [-110 -100] def\n")
	buf.WriteString("/StdHW [42] def\n")
	buf.WriteString("/StdVW [82] def\n")
	buf.WriteString("/ForceBold false def\n")
	buf.WriteString("/password 5839 def\n")
	buf.WriteString("/MinFeature {16 16} def\n")
	buf.WriteString("ND\n")

	glyphs := mmGlyphs()
	names := make([]string, 0, len(glyphs))
	for name := range glyphs {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(buf, "2 index /CharStrings %d dict dup begin\n", len(glyphs))
	for _, name := range names {
		enc := charstringEncrypt(glyphs[name])
		fmt.Fprintf(buf, "/%s %d RD ", name, len(enc))
		buf.Write(enc)
		buf.WriteString(" ND\n")
	}
	buf.WriteString("end\n")
	buf.WriteString("end\n")
	buf.WriteString("readonly put\n")
	buf.WriteString("put\n")
	buf.WriteString("dup /FontName get exch definefont pop\n")
	buf.WriteString("mark currentfile closefile\n")
	return buf.Bytes()
}

// cs assembles the bytes of a single Type 1 charstring.
type cs struct {
	b []byte
}

// Type 1 charstring operators.
var (
	opHsbw            = []byte{13}
	opEndchar         = []byte{14}
	opRMoveTo         = []byte{21}
	opRLineTo         = []byte{5}
	opHLineTo         = []byte{6}
	opVLineTo         = []byte{7}
	opClosePath       = []byte{9}
	opSeac            = []byte{12, 6}
	opCallOtherSubr   = []byte{12, 16}
	opPop             = []byte{12, 17}
	opSetCurrentPoint = []byte{12, 33}
)

// num appends an integer operand in the charstring number encoding.
func (c *cs) num(v int) {
	c.b = appendCSInt(c.b, v)
}

// op appends an operator.
func (c *cs) op(o []byte) {
	c.b = append(c.b, o...)
}

// bval describes one blended value: a base (master 1) value and the deltas of
// masters 2, 3 and 4 relative to it.
type bval struct {
	base int
	d    [3]int
}

// blendCall emits the operand block for a multiple-master blend OtherSubr and
// the callothersubr operator, leaving the m blended results on the PostScript
// stack for the following pop operators.  With four masters the OtherSubr
// number is 13+m (14 for one value, 15 for two, 17 for four).
func (c *cs) blendCall(vals []bval) {
	m := len(vals)
	for _, v := range vals {
		c.num(v.base)
	}
	for _, v := range vals {
		for _, d := range v.d {
			c.num(d)
		}
	}
	c.num(m * 4) // argument count = values * masters
	c.num(13 + m)
	c.op(opCallOtherSubr)
}

// pop moves one OtherSubr result from the PostScript stack to the main stack.
func (c *cs) pop() {
	c.op(opPop)
}

// mmGlyphs returns the plaintext charstrings of every glyph, keyed by name.
func mmGlyphs() map[string][]byte {
	return map[string][]byte{
		".notdef": glyphNotdef(),
		"space":   glyphSpace(),
		"A":       glyphA(),
		"B":       glyphB(),
		"D":       glyphD(),
		"acute":   glyphAcute(),
		"Aacute":  glyphAacute(),
	}
}

// glyphNotdef is a plain box, no blend.  Width 500.
func glyphNotdef() []byte {
	c := &cs{}
	c.num(0)
	c.num(500)
	c.op(opHsbw)
	c.num(50)
	c.num(0)
	c.op(opRMoveTo)
	c.num(400)
	c.op(opHLineTo)
	c.num(700)
	c.op(opVLineTo)
	c.num(-400)
	c.op(opHLineTo)
	c.op(opClosePath)
	c.op(opEndchar)
	return c.b
}

// glyphSpace has no outline.  Width 500.
func glyphSpace() []byte {
	c := &cs{}
	c.num(0)
	c.num(500)
	c.op(opHsbw)
	c.op(opEndchar)
	return c.b
}

// glyphA is a triangle whose sidebearing/width (OtherSubr 15) and two of its
// three vertices (OtherSubr 15) are blended.  Default instance: hsbw sbx 0,
// width 500; MoveTo(100,0), LineTo(400,0), LineTo(250,700), Close.
func glyphA() []byte {
	c := &cs{}
	// hsbw: sbx = 0, wx = 440 + 0.25*(80+80+80) = 500
	c.blendCall([]bval{{0, [3]int{0, 0, 0}}, {440, [3]int{80, 80, 80}}})
	c.pop()
	c.pop()
	c.op(opHsbw)
	// rmoveto to (100,0): dx = 60 + 0.25*(40+40+80) = 100, dy = 0
	c.blendCall([]bval{{60, [3]int{40, 40, 80}}, {0, [3]int{0, 0, 0}}})
	c.pop()
	c.pop()
	c.op(opRMoveTo)
	// rlineto to (400,0): plain (300,0)
	c.num(300)
	c.num(0)
	c.op(opRLineTo)
	// rlineto to (250,700): dx = -110 + 0.25*(-40-40-80) = -150, dy = 660 + 40 = 700
	c.blendCall([]bval{{-110, [3]int{-40, -40, -80}}, {660, [3]int{40, 40, 80}}})
	c.pop()
	c.pop()
	c.op(opRLineTo)
	c.op(opClosePath)
	c.op(opEndchar)
	return c.b
}

// glyphB uses OtherSubr 14 (one blended value) and OtherSubr 17 (four blended
// values feeding two rlineto commands).  Default instance: hsbw sbx 50,
// width 480; MoveTo(50,0), LineTo(450,0), LineTo(450,600), LineTo(50,600),
// LineTo(50,300), Close.
func glyphB() []byte {
	c := &cs{}
	c.num(50)
	c.num(480)
	c.op(opHsbw)
	// start subpath at (50,0)
	c.num(0)
	c.num(0)
	c.op(opRMoveTo)
	// hlineto to (450,0)
	c.num(400)
	c.op(opHLineTo)
	// vlineto to (450,600): dy = 560 + 0.25*(40+40+80) = 600  (OtherSubr 14)
	c.blendCall([]bval{{560, [3]int{40, 40, 80}}})
	c.pop()
	c.op(opVLineTo)
	// OtherSubr 17: two rlineto pairs
	//   (dx1,dy1) = (-400, 0)  -> (50,600)
	//   (dx2,dy2) = (0, -300)  -> (50,300)
	c.blendCall([]bval{
		{-360, [3]int{-40, -40, -80}},
		{0, [3]int{0, 0, 0}},
		{0, [3]int{0, 0, 0}},
		{-260, [3]int{-40, -40, -80}},
	})
	c.pop()
	c.pop()
	c.op(opRLineTo)
	c.pop()
	c.pop()
	c.op(opRLineTo)
	c.op(opClosePath)
	c.op(opEndchar)
	return c.b
}

// glyphD combines flex and blend in one charstring.  A blended rmoveto
// (OtherSubr 15) opens the subpath, a flex (OtherSubrs 1, 2, 0) draws two
// cubic segments, and a blended rlineto (OtherSubr 15) follows.  Default
// instance: hsbw sbx 40, width 520; MoveTo(100,100),
// CubeTo((140,110),(180,120),(220,120)),
// CubeTo((260,120),(300,110),(340,100)), LineTo(340,300), Close.
func glyphD() []byte {
	c := &cs{}
	c.num(40)
	c.num(520)
	c.op(opHsbw)
	// blended rmoveto to (100,100)
	c.blendCall([]bval{{20, [3]int{40, 40, 80}}, {60, [3]int{40, 40, 80}}})
	c.pop()
	c.pop()
	c.op(opRMoveTo)

	// flex begin: 0 1 callothersubr
	c.num(0)
	c.num(1)
	c.op(opCallOtherSubr)
	// seven flex points, each an rmoveto followed by 0 2 callothersubr
	flexMoves := [7][2]int{
		{100, 20},  // reference point (200,120)
		{-60, -10}, // (140,110)
		{40, 10},   // (180,120)
		{40, 0},    // (220,120)
		{40, 0},    // (260,120)
		{40, -10},  // (300,110)
		{40, -10},  // (340,100)
	}
	for _, m := range flexMoves {
		c.num(m[0])
		c.num(m[1])
		c.op(opRMoveTo)
		c.num(0)
		c.num(2)
		c.op(opCallOtherSubr)
	}
	// flex end: flexheight endx endy 3 0 callothersubr pop pop setcurrentpoint
	c.num(50)
	c.num(340)
	c.num(100)
	c.num(3)
	c.num(0)
	c.op(opCallOtherSubr)
	c.pop()
	c.pop()
	c.op(opSetCurrentPoint)

	// blended rlineto to (340,300): dx = 0, dy = 160 + 40 = 200
	c.blendCall([]bval{{0, [3]int{0, 0, 0}}, {160, [3]int{40, 40, 80}}})
	c.pop()
	c.pop()
	c.op(opRLineTo)
	c.op(opClosePath)
	c.op(opEndchar)
	return c.b
}

// glyphAcute is a small plain accent mark.  Width 200.
func glyphAcute() []byte {
	c := &cs{}
	c.num(0)
	c.num(200)
	c.op(opHsbw)
	c.num(0)
	c.num(600)
	c.op(opRMoveTo)
	c.num(50)
	c.num(100)
	c.op(opRLineTo)
	c.num(-20)
	c.num(-100)
	c.op(opRLineTo)
	c.op(opClosePath)
	c.op(opEndchar)
	return c.b
}

// glyphAacute is a seac composite of A (base, code 65) and acute (accent,
// code 194) offset by (150,200).  Width taken from the base glyph (500).
func glyphAacute() []byte {
	c := &cs{}
	c.num(0)
	c.num(500)
	c.op(opHsbw)
	// seac: asb adx ady bchar achar
	c.num(0)
	c.num(150)
	c.num(200)
	c.num(65)
	c.num(194)
	c.op(opSeac)
	return c.b
}

// appendCSInt appends v in the Type 1 charstring integer encoding.
func appendCSInt(buf []byte, v int) []byte {
	switch {
	case v >= -107 && v <= 107:
		return append(buf, byte(v+139))
	case v >= 108 && v <= 1131:
		v -= 108
		return append(buf, byte(v/256+247), byte(v%256))
	case v >= -1131 && v <= -108:
		v = -v - 108
		return append(buf, byte(v/256+251), byte(v%256))
	default:
		return append(buf, 255, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}
}

// eexec/charstring encryption constants.
const (
	eexecC1      = 52845
	eexecC2      = 22719
	eexecR0      = 55665
	charstringR0 = 4330
	lenIV        = 4
)

// eexecEncrypt encrypts the private section with the eexec cipher, prepending
// four leading bytes so the ciphertext begins with an ASCII 'X'.
func eexecEncrypt(plain []byte) []byte {
	iv := []byte{'X' ^ byte(eexecR0>>8), 0, 0, 0}
	return encrypt(eexecR0, append(iv, plain...))
}

// charstringEncrypt encrypts a charstring with the charstring cipher,
// prepending lenIV zero bytes.
func charstringEncrypt(plain []byte) []byte {
	buf := make([]byte, lenIV, lenIV+len(plain))
	buf = append(buf, plain...)
	return encrypt(charstringR0, buf)
}

// encrypt applies the Type 1 eexec/charstring stream cipher.
func encrypt(r uint16, data []byte) []byte {
	out := make([]byte, len(data))
	for i, p := range data {
		c := p ^ byte(r>>8)
		out[i] = c
		r = (uint16(c)+r)*eexecC1 + eexecC2
	}
	return out
}

// writeHex writes data as hex, wrapped at 78 characters per line.
func writeHex(buf *bytes.Buffer, data []byte) {
	const hex = "0123456789abcdef"
	col := 0
	for _, b := range data {
		buf.WriteByte(hex[b>>4])
		buf.WriteByte(hex[b&0x0f])
		col += 2
		if col >= 78 {
			buf.WriteByte('\n')
			col = 0
		}
	}
	if col > 0 {
		buf.WriteByte('\n')
	}
}
