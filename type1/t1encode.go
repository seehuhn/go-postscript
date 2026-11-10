// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

// encodeCharString returns the charstring for the glyph, using a zero
// sidebearing.  All values must lie on the grid (see [snap]) and pass the
// checks in [Font.encodeCharstrings], so that every operand is written
// exactly.
func (g *Glyph) encodeCharString() []byte {
	var buf []byte

	if g.WidthY == 0 {
		buf = appendInt(buf, 0)
		buf = appendNumber(buf, g.WidthX)
		buf = appendOp(buf, t1hsbw)
	} else {
		buf = appendInt(buf, 0)
		buf = appendInt(buf, 0)
		buf = appendNumber(buf, g.WidthX)
		buf = appendNumber(buf, g.WidthY)
		buf = appendOp(buf, t1sbw)
	}

	// TODO(voss): emit hstem3 and vstem3 operators where possible.
	buf = appendStemOps(buf, g.HStem, t1hstem)
	buf = appendStemOps(buf, g.VStem, t1vstem)

	if g.Outline == nil {
		return appendOp(buf, t1endchar)
	}

	var pos vec.Vec2 // current point
	i := 0           // coordinate index
	for _, cmd := range g.Outline.Cmds {
		switch cmd {
		case path.CmdMoveTo, path.CmdLineTo:
			p := g.Outline.Coords[i]
			i++
			d := p.Sub(pos)
			hOp, vOp, rOp := t1hmoveto, t1vmoveto, t1rmoveto
			if cmd == path.CmdLineTo {
				hOp, vOp, rOp = t1hlineto, t1vlineto, t1rlineto
			}
			switch {
			case d.Y == 0:
				buf = appendNumber(buf, d.X)
				buf = appendOp(buf, hOp)
			case d.X == 0:
				buf = appendNumber(buf, d.Y)
				buf = appendOp(buf, vOp)
			default:
				buf = appendNumber(buf, d.X)
				buf = appendNumber(buf, d.Y)
				buf = appendOp(buf, rOp)
			}
			pos = p
		case path.CmdCubeTo:
			p1, p2, p3 := g.Outline.Coords[i], g.Outline.Coords[i+1], g.Outline.Coords[i+2]
			i += 3
			da, db, dc := p1.Sub(pos), p2.Sub(p1), p3.Sub(p2)
			switch {
			case da.Y == 0 && dc.X == 0:
				buf = appendNumber(buf, da.X)
				buf = appendNumber(buf, db.X)
				buf = appendNumber(buf, db.Y)
				buf = appendNumber(buf, dc.Y)
				buf = appendOp(buf, t1hvcurveto)
			case da.X == 0 && dc.Y == 0:
				buf = appendNumber(buf, da.Y)
				buf = appendNumber(buf, db.X)
				buf = appendNumber(buf, db.Y)
				buf = appendNumber(buf, dc.X)
				buf = appendOp(buf, t1vhcurveto)
			default:
				buf = appendNumber(buf, da.X)
				buf = appendNumber(buf, da.Y)
				buf = appendNumber(buf, db.X)
				buf = appendNumber(buf, db.Y)
				buf = appendNumber(buf, dc.X)
				buf = appendNumber(buf, dc.Y)
				buf = appendOp(buf, t1rrcurveto)
			}
			pos = p3
		case path.CmdClose:
			buf = appendOp(buf, t1closepath)
		}
	}
	return appendOp(buf, t1endchar)
}

func appendOp(buf []byte, op t1op) []byte {
	if op < 256 {
		return append(buf, byte(op))
	}
	return append(buf, byte(op>>8), byte(op))
}

// appendStemOps appends one stem operator for each pair of edges in stems.
// The edges are given relative to the zero sidebearing which
// encodeCharString writes.
func appendStemOps(buf []byte, stems []float64, op t1op) []byte {
	for i := 0; i+1 < len(stems); i += 2 {
		buf = appendNumber(buf, stems[i])
		buf = appendNumber(buf, stems[i+1]-stems[i])
		buf = appendOp(buf, op)
	}
	return buf
}

func appendInt(buf []byte, x int32) []byte {
	switch {
	case x >= -107 && x <= 107:
		return append(buf, byte(x+139))
	case x >= 108 && x <= 1131:
		x -= 108
		return append(buf, byte(x/256+247), byte(x%256))
	case x >= -1131 && x <= -108:
		x = -x - 108
		return append(buf, byte(x/256+251), byte(x%256))
	default:
		return append(buf, 255, byte(x>>24), byte(x>>16), byte(x>>8), byte(x))
	}
}

// appendNumber appends the charstring encoding of x, which must lie on the
// grid and satisfy [numberUsable].  An integer is written directly, any
// other value as p/q with q a power of two, using the div operator.
func appendNumber(buf []byte, x float64) []byte {
	p := int32(x * gridScale)
	q := int32(gridScale)
	for q > 1 && p%2 == 0 {
		p /= 2
		q /= 2
	}
	buf = appendInt(buf, p)
	if q == 1 {
		return buf
	}
	buf = appendInt(buf, q)
	return appendOp(buf, t1div)
}
