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
	"fmt"
	"math"
	"slices"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/membudget"
	"seehuhn.de/go/postscript/funit"
)

// decodeInfo holds the state shared by all glyphs of one font while their
// charstrings are decoded.  Buffers are reused from one glyph to the next, so
// the glyphs of a font must be decoded one after another, not concurrently.
type decodeInfo struct {
	subrs [][]byte
	seacs []seacInfo

	// weightVector holds the multiple master blend weights, or nil for a
	// non-MM font.  The OtherSubr 14-18 blend operators consume it.
	weightVector []float64

	// budget bounds the total charstring bytes processed across all glyphs of
	// the font.  Each charstring body is charged its length before execution,
	// so amplification via repeated subroutine calls trips the budget.  Shared
	// by every glyph, so it bounds total decode work, not per-glyph work.
	// Must not be nil.
	budget *membudget.Budget

	// scratch collects the outline of the glyph currently being decoded.  Its
	// capacity is reused between glyphs, and the finished outline is copied
	// into fresh slices before it is returned.
	scratch path.Data

	// flexData collects the coordinates of the flex sequence currently being
	// decoded.  A charstring can name any number of them, so this grows with
	// the input; like scratch, its capacity is reused between glyphs.
	flexData []float64

	// stackBuf backs the charstring operand stack.  The depth check runs at
	// the start of each command, after the preceding command has pushed its
	// value, so one slot beyond the largest limit is needed to hold a value
	// which is about to be rejected.
	stackBuf [stackLimitBlend + 1]float64

	// psStackBuf backs the PostScript operand stack which passes values
	// between callothersubr and the following pop commands.  It holds the
	// argN operands taken off the charstring stack, so the charstring stack
	// limit bounds it too.
	psStackBuf [stackLimitBlend]float64
}

// Operand-stack depth limits.  The specification limits the charstring
// operand stack to 24 entries, and a multiple master blend can use only 22
// of them, because callothersubr takes the argument count and the othersubr
// number on top of the values being blended.  Blending 6 values across 16
// masters needs 98 entries and is therefore not permitted; such charstrings
// are accepted anyway rather than dropping the glyph.
const (
	stackLimit      = 24
	stackLimitBlend = 98
)

type seacInfo struct {
	name         string
	base, accent int
	dx, dy       float64
}

func (info *decodeInfo) decodeCharString(code []byte, name string) *Glyph {
	// operand-stack depth limit.  MM fonts get the raised limit so that
	// over-long blend argument lists still decode; ordinary fonts keep the
	// limit from the specification.
	maxStack := stackLimit
	if info.weightVector != nil {
		maxStack = stackLimitBlend
	}
	stack := info.stackBuf[:0]
	clearStack := func() {
		stack = stack[:0]
	}

	postscriptStack := info.psStackBuf[:0]
	var flexStart vec.Vec2 // current point when the flex sequence started
	inFlex := false

	res := &Glyph{}

	outline := &info.scratch
	outline.Cmds = outline.Cmds[:0]
	outline.Coords = outline.Coords[:0]

	// Reset the flex buffer here rather than only at the start of a flex
	// sequence, so that a glyph ending inside one cannot leak its
	// coordinates into a glyph which ends a flex it never started.
	info.flexData = info.flexData[:0]

	// On a malformed charstring, abandon the partial decode and
	// return a blank stub carrying only WidthX/WidthY (zero before
	// hsbw/sbw set them).  The caller substitutes this for the bad
	// glyph so layout metrics survive without leaking partial hint
	// or path state.
	bail := func() *Glyph {
		return &Glyph{
			Outline: &path.Data{},
			WidthX:  res.WidthX,
			WidthY:  res.WidthY,
		}
	}

	// charge the top-level charstring body
	if err := info.budget.Charge(len(code)); err != nil {
		return bail()
	}

	var posX, posY float64
	var LsbX funit.Int16 // TODO(voss): use float64
	var LsbY funit.Int16
	isClosed := true   // no sub-path is open
	haveStart := false // a MoveTo has been emitted for the current sub-path
	// A closepath with no sub-path open is a no-op.  Charstrings in the wild
	// contain redundant closepath commands, and emitting a Close for them
	// would leave the outline unreadable by the normal path rules.
	rClosePath := func() {
		if haveStart {
			outline.Close()
		}
		isClosed = true
		haveStart = false
	}
	rMoveTo := func(dx, dy float64) {
		posX += dx
		posY += dy
		if inFlex {
			// during flex, rmoveto is used only to track coordinates;
			// the actual path commands are emitted by othersubr 0
			return
		}
		if !isClosed {
			rClosePath()
		}
		outline.MoveTo(vec.Vec2{X: posX, Y: posY})
		haveStart = true
	}
	// Unlike the PostScript operator, the Type 1 closepath command leaves the
	// current point where it is.  A drawing command which follows a closepath
	// therefore starts a new sub-path at the current point, and we make this
	// explicit so that the outline can be read using the normal path rules.
	startSubPath := func(p vec.Vec2) {
		if !haveStart {
			outline.MoveTo(p)
			haveStart = true
		}
	}
	rLineTo := func(dx, dy float64) {
		startSubPath(vec.Vec2{X: posX, Y: posY})
		posX += dx
		posY += dy
		outline.LineTo(vec.Vec2{X: posX, Y: posY})
		isClosed = false
	}
	rCurveTo := func(dxa, dya, dxb, dyb, dxc, dyc float64) {
		startSubPath(vec.Vec2{X: posX, Y: posY})
		xa := posX + dxa
		ya := posY + dya
		xb := xa + dxb
		yb := ya + dyb
		posX = xb + dxc
		posY = yb + dyc
		outline.CubeTo(
			vec.Vec2{X: xa, Y: ya},
			vec.Vec2{X: xb, Y: yb},
			vec.Vec2{X: posX, Y: posY},
		)
		isClosed = false
	}

	cmdStack := [][]byte{code}
glyphLoop:
	for len(cmdStack) > 0 {
		cmdStack, code = cmdStack[:len(cmdStack)-1], cmdStack[len(cmdStack)-1]

	opLoop:
		for len(code) > 0 {
			if len(stack) > maxStack {
				return bail()
			}

			op := t1op(code[0])
			if op >= 32 && op <= 246 {
				stack = append(stack, float64(op)-139)
				code = code[1:]
				// fmt.Println("# push", stack[len(stack)-1])
				continue
			} else if op >= 247 && op <= 250 {
				if len(code) < 2 {
					return bail()
				}
				val := (float64(op)-247)*256 + float64(code[1]) + 108
				stack = append(stack, val)
				code = code[2:]
				// fmt.Println("# push", stack[len(stack)-1])
				continue
			} else if op >= 251 && op <= 254 {
				if len(code) < 2 {
					return bail()
				}
				val := (251-float64(op))*256 - float64(code[1]) - 108
				stack = append(stack, val)
				code = code[2:]
				// fmt.Println("# push", stack[len(stack)-1])
				continue
			} else if op == 255 {
				if len(code) < 5 {
					return bail()
				}
				val := int32(code[1])<<24 | int32(code[2])<<16 |
					int32(code[3])<<8 | int32(code[4])
				stack = append(stack, float64(val))
				code = code[5:]
				// fmt.Println("# push", stack[len(stack)-1])
				continue
			}

			if op == 12 {
				if len(code) < 2 {
					return bail()
				}
				op = op<<8 | t1op(code[1])
				code = code[2:]
			} else {
				code = code[1:]
			}

		opSwitch:
			switch op {
			case t1endchar:
				// fmt.Println("endchar")
				break glyphLoop
			case t1hsbw:
				if len(stack) < 2 {
					return bail()
				}
				// fmt.Printf("hsbw(%g, %g)\n", stack[0], stack[1])
				posX = stack[0]
				posY = 0
				LsbX = funit.Int16(math.Round(stack[0]))
				LsbY = 0
				res.WidthX = stack[1]
				res.WidthY = 0
				clearStack()
			case t1seac:
				if len(stack) < 5 {
					return bail()
				}
				// fmt.Printf("seac(%g, %g, %g, %g, %g)\n", stack[0], stack[1], stack[2], stack[3], stack[4])
				// asb := stack[0]
				adX := stack[1]
				adY := stack[2]
				bchar, err := getInt(stack[3])
				if err != nil {
					return bail()
				}
				achar, err := getInt(stack[4])
				if err != nil {
					return bail()
				}
				info.seacs = append(info.seacs, seacInfo{
					name:   name,
					base:   bchar,
					accent: achar,
					dx:     adX,
					dy:     adY,
				})
				// clearStack()
				break glyphLoop
			case t1sbw:
				if len(stack) < 4 {
					return bail()
				}
				// fmt.Printf("sbw(%g, %g, %g, %g)\n", stack[0], stack[1], stack[2], stack[3])
				posX = stack[0]
				posY = stack[1]
				LsbX = funit.Int16(math.Round(stack[0]))
				LsbY = funit.Int16(math.Round(stack[1]))
				res.WidthX = stack[2]
				res.WidthY = stack[3]
				clearStack()

			case t1closepath:
				// fmt.Println("closepath")
				rClosePath()
			case t1hlineto:
				if len(stack) < 1 {
					return bail()
				}
				// fmt.Printf("hlineto(%g)\n", stack[0])
				rLineTo(stack[0], 0)
				clearStack()
			case t1hmoveto:
				if len(stack) < 1 {
					return bail()
				}
				// fmt.Printf("hmoveto(%g)\n", stack[0])
				rMoveTo(stack[0], 0)
				clearStack()
			case t1hvcurveto:
				if len(stack) < 4 {
					return bail()
				}
				// fmt.Printf("hvcurveto(%g, %g, %g, %g)\n", stack[0], stack[1], stack[2], stack[3])
				rCurveTo(stack[0], 0, stack[1], stack[2], 0, stack[3])
				clearStack()
			case t1rlineto:
				if len(stack) < 2 {
					return bail()
				}
				// fmt.Printf("rlineto(%g, %g)\n", stack[0], stack[1])
				rLineTo(stack[0], stack[1])
				clearStack()
			case t1rmoveto:
				if len(stack) < 2 {
					return bail()
				}
				// fmt.Printf("rmoveto(%g, %g)\n", stack[0], stack[1])
				rMoveTo(stack[0], stack[1])
				clearStack()
			case t1rrcurveto:
				if len(stack) < 6 {
					return bail()
				}
				// fmt.Printf("rrcurveto(%g, %g, %g, %g, %g, %g)\n", stack[0], stack[1], stack[2], stack[3], stack[4], stack[5])
				rCurveTo(stack[0], stack[1], stack[2], stack[3], stack[4], stack[5])
				clearStack()
			case t1vhcurveto:
				if len(stack) < 4 {
					return bail()
				}
				// fmt.Printf("vhcurveto(%g, %g, %g, %g)\n", stack[0], stack[1], stack[2], stack[3])
				rCurveTo(0, stack[0], stack[1], stack[2], stack[3], 0)
				clearStack()
			case t1vlineto:
				if len(stack) < 1 {
					return bail()
				}
				// fmt.Printf("vlineto(%g)\n", stack[0])
				rLineTo(0, stack[0])
				clearStack()
			case t1vmoveto:
				if len(stack) < 1 {
					return bail()
				}
				// fmt.Printf("vmoveto(%g)\n", stack[0])
				rMoveTo(0, stack[0])
				clearStack()

			case t1dotsection:
				// fmt.Println("dotsection")
				clearStack()
			case t1hstem:
				if len(stack) < 2 {
					return bail()
				}
				// fmt.Printf("hstem(%g, %g)\n", stack[0], stack[1])
				a := LsbY + funit.Int16(math.Round(stack[0]))
				b := a + funit.Int16(math.Round(stack[1]))
				res.HStem = append(res.HStem, a, b)
				clearStack()
			case t1hstem3:
				if len(stack) < 6 {
					return bail()
				}
				// fmt.Printf("hstem3(%g, %g, %g, %g, %g, %g)\n", stack[0], stack[1], stack[2], stack[3], stack[4], stack[5])
				a := LsbY + funit.Int16(math.Round(stack[0]))
				b := a + funit.Int16(math.Round(stack[1]))
				c := LsbY + funit.Int16(math.Round(stack[2]))
				d := c + funit.Int16(math.Round(stack[3]))
				e := LsbY + funit.Int16(math.Round(stack[4]))
				f := e + funit.Int16(math.Round(stack[5]))
				res.HStem = append(res.HStem[:0], a, b, c, d, e, f)
				clearStack()
			case t1vstem:
				if len(stack) < 2 {
					return bail()
				}
				// fmt.Printf("vstem(%g, %g)\n", stack[0], stack[1])
				a := LsbX + funit.Int16(math.Round(stack[0]))
				b := a + funit.Int16(math.Round(stack[1]))
				res.VStem = append(res.VStem, a, b)
				clearStack()
			case t1vstem3:
				if len(stack) < 6 {
					return bail()
				}
				// fmt.Printf("vstem3(%g, %g, %g, %g, %g, %g)\n", stack[0], stack[1], stack[2], stack[3], stack[4], stack[5])
				a := LsbX + funit.Int16(math.Round(stack[0]))
				b := a + funit.Int16(math.Round(stack[1]))
				c := LsbX + funit.Int16(math.Round(stack[2]))
				d := c + funit.Int16(math.Round(stack[3]))
				e := LsbX + funit.Int16(math.Round(stack[4]))
				f := e + funit.Int16(math.Round(stack[5]))
				res.VStem = append(res.VStem[:0], a, b, c, d, e, f)
				clearStack()
			case t1div:
				if len(stack) < 2 {
					return bail()
				}
				// fmt.Printf("div(%g, %g)\n", stack[0], stack[1])
				stack = append(stack[:len(stack)-2], stack[len(stack)-2]/stack[len(stack)-1])

			case t1callsubr:
				if len(stack) < 1 {
					return bail()
				}
				idx, err := getInt(stack[len(stack)-1])
				if err != nil {
					return bail()
				}
				stack = stack[:len(stack)-1]
				switch idx { // pre-defined subroutines
				case 3:
					// Entry 3 in the Subrs array is a charstring that does nothing.
					break opSwitch
				}

				if idx < 0 || idx >= len(info.subrs) {
					return bail()
				}
				// fmt.Printf("callsubr(%d)\n", idx)

				cmdStack = append(cmdStack, code)
				if len(cmdStack) > 10 {
					return bail()
				}
				code = info.subrs[idx]
				// charge the subr body to bound fan-out amplification
				if err := info.budget.Charge(len(code)); err != nil {
					return bail()
				}
			case t1callothersubr:
				if len(stack) < 2 {
					return bail()
				}
				idx, err := getInt(stack[len(stack)-1])
				if err != nil {
					return bail()
				}
				argN, err := getInt(stack[len(stack)-2])
				if err != nil {
					return bail()
				}
				// argN args must fit in the main stack, so no legitimate
				// charstring can have argN > maxStack; the upper bound
				// also keeps argN+2 below from overflowing on 32-bit
				// platforms.
				if argN < 0 || argN > maxStack {
					return bail()
				}
				if len(stack) < argN+2 {
					return bail()
				}
				// fmt.Println("callothersubr", idx, args)
				stack = stack[:len(stack)-2]
				postscriptStack = postscriptStack[:0]
				for range argN {
					val := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					postscriptStack = append(postscriptStack, val)
				}
				// horizontal flex:
				// starting point: x0 A
				// reference point: x3 A
				//     x1 y1  x2  B  x3 B curveto
				//     x4  B  x5 y5  x6 A curveto
				// seven coordinate pairs:
				//     x3-x0 0     rmoveto   reference point relative to starting point
				//     x1-x3 y1-A  rmoveto   first rrcurveto pair relative to reference point
				//     x2-x1 B-y1  rmoveto
				//     x3-x2 0     rmoveto
				//     x4-x3 0     rmoveto   second rrcurveto
				//     x5-x4 y5-B  rmoveto
				//     x6-x5 A-y5  rmoveto

				switch idx {
				case 0: // flex end (3 args, 2 returns)
					if argN != 3 {
						return bail()
					}
					inFlex = false
					if flexData := info.flexData; len(flexData) == 14 {
						startSubPath(flexStart)
						outline.CubeTo(
							vec.Vec2{X: flexData[2], Y: flexData[3]},
							vec.Vec2{X: flexData[4], Y: flexData[5]},
							vec.Vec2{X: flexData[6], Y: flexData[7]},
						)
						outline.CubeTo(
							vec.Vec2{X: flexData[8], Y: flexData[9]},
							vec.Vec2{X: flexData[10], Y: flexData[11]},
							vec.Vec2{X: flexData[12], Y: flexData[13]},
						)
						isClosed = false
					}
					postscriptStack = postscriptStack[:len(postscriptStack)-1]
				case 1: // flex start (0 args)
					if argN != 0 {
						return bail()
					}
					inFlex = true
					flexStart = vec.Vec2{X: posX, Y: posY}
					info.flexData = info.flexData[:0]
				case 2: // flex coordinate pair (0 args)
					if argN != 0 {
						return bail()
					}
					info.flexData = append(info.flexData, posX, posY)
				case 3: // hint replacement (1 arg)
					if argN != 1 {
						return bail()
					}
					postscriptStack = append(postscriptStack[:0], 3)
				case 14, 15, 16, 17, 18: // multiple master blend
					if info.weightVector == nil {
						// a blend othersubr in a non-MM font is malformed;
						// fall through to the unknown-othersubr contract below
						break
					}
					k := len(info.weightVector)
					m := idx - 13 // number of blended values
					if idx == 18 {
						m = 6
					}
					if argN != m*k {
						return bail()
					}
					// postscriptStack holds the argN operands reversed: the
					// forward operand p (0-indexed) is postscriptStack[argN-1-p].
					// layout: m base values, then for each value its k-1 deltas
					// for masters 2..k.
					// m is at most 6, the widest blend othersubr 18 performs
					var results [6]float64
					for i := range m {
						v := postscriptStack[argN-1-i] // base value i
						for mm := 1; mm < k; mm++ {
							d := postscriptStack[argN-1-(m+i*(k-1)+(mm-1))]
							v += d * info.weightVector[mm]
						}
						results[i] = v
					}
					// push results reversed so the following m pops return
					// them in the base-value order
					postscriptStack = postscriptStack[:0]
					for i := m - 1; i >= 0; i-- {
						postscriptStack = append(postscriptStack, results[i])
					}
				default:
					// unknown (font-private) othersubr: ignored, its operands
					// remain on the postscript stack for the following pop
					// operators (permissive reading)
				}
			case t1pop:
				if len(postscriptStack) < 1 {
					return bail()
				}
				// fmt.Println("pop")
				val := postscriptStack[len(postscriptStack)-1]
				postscriptStack = postscriptStack[:len(postscriptStack)-1]
				stack = append(stack, val)

			case t1return:
				// fmt.Println("return")
				break opLoop
			case t1setcurrentpoint:
				if len(stack) < 2 {
					return bail()
				}
				// fmt.Printf("setcurrentpoint(%g, %g)\n", stack[0], stack[1])
				posX = stack[0]
				posY = stack[1]
				clearStack()

			default:
				return bail()
			}
		}
	}
	if !isClosed {
		rClosePath()
	}

	// copy the outline out of the scratch buffer, using nil instead of empty
	// slices so that a blank glyph looks the same as one from [bail]
	res.Outline = &path.Data{}
	if len(outline.Cmds) > 0 {
		res.Outline.Cmds = slices.Clone(outline.Cmds)
		res.Outline.Coords = slices.Clone(outline.Coords)
	}
	return res
}

func getInt(x float64) (int, error) {
	i := int(x)
	if float64(i) != x {
		return 0, invalidSince("invalid operand type")
	}
	return i, nil
}

type t1op uint16

func (op t1op) Bytes() []byte {
	if op > 255 {
		return []byte{byte(op >> 8), byte(op)}
	}
	return []byte{byte(op)}
}

func (op t1op) String() string {
	switch op {
	case t1hstem:
		return "hstem"
	case t1vstem:
		return "vstem"
	case t1vmoveto:
		return "vmoveto"
	case t1rlineto:
		return "rlineto"
	case t1hlineto:
		return "hlineto"
	case t1vlineto:
		return "vlineto"
	case t1rrcurveto:
		return "rrcurveto"
	case t1callsubr:
		return "callsubr"
	case t1return:
		return "return"
	case t1hsbw:
		return "hsbw"
	case t1endchar:
		return "endchar"
	case t1rmoveto:
		return "rmoveto"
	case t1hmoveto:
		return "hmoveto"
	case t1vhcurveto:
		return "vhcurveto"
	case t1hvcurveto:
		return "hvcurveto"
	case t1dotsection:
		return "dotsection"
	case t1div:
		return "div"
	case 255:
		return "int32"
	}
	if 32 <= op && op <= 246 {
		return fmt.Sprintf("int1(%d)", op)
	}
	if 247 <= op && op <= 254 {
		return fmt.Sprintf("int2(%d)", op)
	}
	return fmt.Sprintf("t1op(%d)", op)
}

const (
	t1hstem     t1op = 0x0001
	t1vstem     t1op = 0x0003
	t1vmoveto   t1op = 0x0004
	t1rlineto   t1op = 0x0005
	t1hlineto   t1op = 0x0006
	t1vlineto   t1op = 0x0007
	t1rrcurveto t1op = 0x0008
	t1closepath t1op = 0x0009
	t1callsubr  t1op = 0x000a
	t1return    t1op = 0x000b
	t1hsbw      t1op = 0x000d
	t1endchar   t1op = 0x000e
	t1rmoveto   t1op = 0x0015
	t1hmoveto   t1op = 0x0016
	t1vhcurveto t1op = 0x001e
	t1hvcurveto t1op = 0x001f

	t1dotsection      t1op = 0x0c00
	t1vstem3          t1op = 0x0c01
	t1hstem3          t1op = 0x0c02
	t1seac            t1op = 0x0c06
	t1sbw             t1op = 0x0c07
	t1div             t1op = 0x0c0c
	t1callothersubr   t1op = 0x0c10
	t1pop             t1op = 0x0c11
	t1setcurrentpoint t1op = 0x0c21
)
