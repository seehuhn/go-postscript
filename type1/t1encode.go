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
	"math"

	"seehuhn.de/go/geom/path"
)

func (g *Glyph) encodeCharString(wx, wy int32) []byte {
	var buf []byte

	if wy == 0 {
		buf = appendInt(buf, 0)
		buf = appendInt(buf, wx)
		buf = appendOp(buf, t1hsbw)
	} else {
		buf = appendInt(buf, 0)
		buf = appendInt(buf, 0)
		buf = appendInt(buf, wx)
		buf = appendInt(buf, wy)
		buf = appendOp(buf, t1sbw)
	}

	// TODO(voss): emit hstem3 and vstem3 operators where possible.
	for i := 0; i+1 < len(g.HStem); i += 2 {
		buf = appendInt(buf, int32(g.HStem[i]))
		buf = appendInt(buf, int32(g.HStem[i+1])-int32(g.HStem[i]))
		buf = appendOp(buf, t1hstem)
	}
	for i := 0; i+1 < len(g.VStem); i += 2 {
		buf = appendInt(buf, int32(g.VStem[i]))
		buf = appendInt(buf, int32(g.VStem[i+1])-int32(g.VStem[i]))
		buf = appendOp(buf, t1vstem)
	}

	posX := 0.0
	posY := 0.0
	var dx, dy float64
	if g.Outline != nil {
		i := 0 // coordinate index
		for _, cmd := range g.Outline.Cmds {
			switch cmd {
			case path.CmdMoveTo:
				x, y := g.Outline.Coords[i].X, g.Outline.Coords[i].Y
				i++
				if math.Abs(y-posY) < 1e-6 {
					buf, dx = appendNumber(buf, x-posX)
					buf = appendOp(buf, t1hmoveto)
					posX += dx
				} else if math.Abs(x-posX) < 1e-6 {
					buf, dy = appendNumber(buf, y-posY)
					buf = appendOp(buf, t1vmoveto)
					posY += dy
				} else {
					buf, dx = appendNumber(buf, x-posX)
					buf, dy = appendNumber(buf, y-posY)
					buf = appendOp(buf, t1rmoveto)
					posX += dx
					posY += dy
				}
			case path.CmdLineTo:
				x, y := g.Outline.Coords[i].X, g.Outline.Coords[i].Y
				i++
				if math.Abs(y-posY) < 1e-6 {
					buf, dx = appendNumber(buf, x-posX)
					buf = appendOp(buf, t1hlineto)
					posX += dx
				} else if math.Abs(x-posX) < 1e-6 {
					buf, dy = appendNumber(buf, y-posY)
					buf = appendOp(buf, t1vlineto)
					posY += dy
				} else {
					buf, dx = appendNumber(buf, x-posX)
					buf, dy = appendNumber(buf, y-posY)
					buf = appendOp(buf, t1rlineto)
					posX += dx
					posY += dy
				}
			case path.CmdCubeTo:
				x1, y1 := g.Outline.Coords[i].X, g.Outline.Coords[i].Y
				x2, y2 := g.Outline.Coords[i+1].X, g.Outline.Coords[i+1].Y
				x3, y3 := g.Outline.Coords[i+2].X, g.Outline.Coords[i+2].Y
				i += 3
				if math.Abs(y1-posY) < 1e-6 && math.Abs(x3-x2) < 1e-6 {
					var dxa, dxb, dyb, dyc float64
					buf, dxa = appendNumber(buf, x1-posX)
					buf, dxb = appendNumber(buf, x2-posX-dxa)
					buf, dyb = appendNumber(buf, y2-posY)
					buf, dyc = appendNumber(buf, y3-posY-dyb)
					buf = appendOp(buf, t1hvcurveto)
					posX += dxa + dxb
					posY += dyb + dyc
				} else if math.Abs(x1-posX) < 1e-6 && math.Abs(y3-y2) < 1e-6 {
					var dya, dxb, dyb, dxc float64
					buf, dya = appendNumber(buf, y1-posY)
					buf, dxb = appendNumber(buf, x2-posX)
					buf, dyb = appendNumber(buf, y2-posY-dya)
					buf, dxc = appendNumber(buf, x3-posX-dxb)
					buf = appendOp(buf, t1vhcurveto)
					posX += dxb + dxc
					posY += dya + dyb
				} else {
					var dxa, dxb, dxc, dya, dyb, dyc float64
					buf, dxa = appendNumber(buf, x1-posX)
					buf, dya = appendNumber(buf, y1-posY)
					buf, dxb = appendNumber(buf, x2-posX-dxa)
					buf, dyb = appendNumber(buf, y2-posY-dya)
					buf, dxc = appendNumber(buf, x3-posX-dxa-dxb)
					buf, dyc = appendNumber(buf, y3-posY-dya-dyb)
					buf = appendOp(buf, t1rrcurveto)
					posX += dxa + dxb + dxc
					posY += dya + dyb + dyc
				}
			case path.CmdClose:
				buf = appendOp(buf, t1closepath)
			}
		}
	}
	buf = appendOp(buf, t1endchar)
	return buf
}

func appendOp(buf []byte, op t1op) []byte {
	if op < 256 {
		return append(buf, byte(op))
	}
	return append(buf, byte(op>>8), byte(op))
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

func appendNumber(buf []byte, x float64) ([]byte, float64) {
	xInt := int32(x)
	if float64(xInt) == x {
		return appendInt(buf, xInt), x
	}

	var bestP, bestQ int32
	bestDelta := math.Inf(1)
	for q := int32(1); q <= 107; q++ {
		pf := math.Round(x * float64(q))
		if pf > math.MaxInt32 {
			pf = math.MaxInt32
		} else if pf < math.MinInt32 {
			pf = math.MinInt32
		}
		p := int32(pf)
		delta := math.Abs(pf/float64(q) - x)
		if delta <= bestDelta {
			bestDelta = delta
			bestP = p
			bestQ = q
		}
	}
	buf = appendInt(buf, bestP)
	buf = appendInt(buf, bestQ)
	buf = appendOp(buf, t1div)
	return buf, float64(bestP) / float64(bestQ)
}
