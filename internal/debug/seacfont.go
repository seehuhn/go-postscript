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

package debug

import "bytes"

// MakeSeacFont returns a deterministic non-MM Type 1 font program in PFA form
// which exercises seac composites.
//
// The glyphs are .notdef, A (left side bearing 30), acute (left side bearing
// 60), Aacute (seac of A and acute with asb 60, adx 120, ady 200), Bacute
// (seac whose base glyph B is not in the font) and Xacute (seac whose base
// code 0 is unused in StandardEncoding) and Awide (like Aacute, but with
// width 600 where the base glyph has width 500).  The built-in encoding is
// custom: code 65 is acute, code 194 is A and code 97 is Aacute, so seac
// components cannot be found through the font's encoding.
func MakeSeacFont() []byte {
	buf := &bytes.Buffer{}
	buf.WriteString(seacClearText)
	buf.WriteString("currentfile eexec\n")
	writeHex(buf, eexecEncrypt(buildPrivate(seacGlyphs())))
	buf.WriteString(trailer)
	return buf.Bytes()
}

const seacClearText = `%!FontType1-1.1: QuireSeacTest 001.000
10 dict begin
/FontInfo 16 dict dup begin
/version (001.000) def
/FullName (Quire Seac Test) def
/FamilyName (Quire Seac Test) def
/Weight (Regular) def
/ItalicAngle 0 def
/isFixedPitch false def
/UnderlinePosition -100 def
/UnderlineThickness 50 def
end def
/FontName /QuireSeacTest def
/Encoding 256 array
0 1 255 {1 index exch /.notdef put} for
dup 65 /acute put
dup 194 /A put
dup 97 /Aacute put
dup 98 /Bacute put
dup 99 /Xacute put
dup 100 /Awide put
readonly def
/PaintType 0 def
/FontType 1 def
/FontMatrix [0.001 0 0 0.001 0 0] def
/FontBBox [0 0 400 900] def
currentdict end
`

func seacGlyphs() map[string][]byte {
	return map[string][]byte{
		".notdef": glyphNotdef(),
		"A":       seacGlyphA(),
		"acute":   seacGlyphAcute(),
		"Aacute":  seacComposite(30, 500, 60, 120, 200, 'A', 194),
		"Awide":   seacComposite(30, 600, 60, 120, 200, 'A', 194),
		"Bacute":  seacComposite(0, 500, 0, 0, 0, 'B', 194),
		"Xacute":  seacComposite(0, 500, 0, 0, 0, 0, 194),
	}
}

// seacGlyphA is a triangle with left side bearing 30 and width 500.
func seacGlyphA() []byte {
	c := &cs{}
	c.num(30)
	c.num(500)
	c.op(opHsbw)
	c.num(0)
	c.num(0)
	c.op(opRMoveTo)
	c.num(300)
	c.op(opHLineTo)
	c.num(700)
	c.op(opVLineTo)
	c.op(opClosePath)
	c.op(opEndchar)
	return c.b
}

// seacGlyphAcute is a square with left side bearing 60 and width 300.
func seacGlyphAcute() []byte {
	c := &cs{}
	c.num(60)
	c.num(300)
	c.op(opHsbw)
	c.num(0)
	c.num(600)
	c.op(opRMoveTo)
	c.num(100)
	c.op(opHLineTo)
	c.num(100)
	c.op(opVLineTo)
	c.op(opClosePath)
	c.op(opEndchar)
	return c.b
}

// seacComposite is a seac glyph with left side bearing sbx and width wx.
func seacComposite(sbx, wx, asb, adx, ady, bchar, achar int) []byte {
	c := &cs{}
	c.num(sbx)
	c.num(wx)
	c.op(opHsbw)
	c.num(asb)
	c.num(adx)
	c.num(ady)
	c.num(bchar)
	c.num(achar)
	c.op(opSeac)
	return c.b
}
