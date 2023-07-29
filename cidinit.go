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

package postscript

var CIDInit = Dict{
	"begincmap": builtin(func(intp *Interpreter) error {
		intp.cmap = &cmapInfo{}
		return nil
	}),
	"endcmap": builtin(func(intp *Interpreter) error {
		return nil
	}),
	"begincodespacerange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "begincodespacerange: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "begincodespacerange: not enough arguments")
		}
		n, ok := intp.Stack[len(intp.Stack)-1].(Integer)
		if !ok {
			return intp.e(eTypecheck, "begincodespacerange: expected integer, got %T", intp.Stack[len(intp.Stack)-1])
		} else if n < 0 || n > 100 {
			return intp.e(eRangecheck, "begincodespacerange: invalid length %d", n)
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.tmpCodeSpaceRanges = make([]codeSpaceRange, n)
		return nil
	}),
	"endcodespacerange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "endcodespacerange: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmap.tmpCodeSpaceRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endcodespacerange: not enough arguments")
		}
		for i := range intp.cmap.tmpCodeSpaceRanges {
			lo, ok := intp.Stack[base+2*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endcodespacerange: expected string, got %T", intp.Stack[base+2*i])
			}
			hi, ok := intp.Stack[base+2*i+1].(String)
			if !ok {
				return intp.e(eTypecheck, "endcodespacerange: expected string, got %T", intp.Stack[base+2*i+1])
			}
			if len(lo) != len(hi) {
				return intp.e(eRangecheck, "endcodespacerange: expected strings of equal length, got %d and %d", len(lo), len(hi))
			}
			intp.cmap.tmpCodeSpaceRanges[i] = codeSpaceRange{lo, hi}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.codeSpaceRanges = append(intp.cmap.codeSpaceRanges, intp.cmap.tmpCodeSpaceRanges...)
		intp.cmap.tmpCodeSpaceRanges = nil
		return nil
	}),
	"beginbfchar": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "beginbfchar: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "beginbfchar: not enough arguments")
		}
		n, ok := intp.Stack[len(intp.Stack)-1].(Integer)
		if !ok {
			return intp.e(eTypecheck, "beginbfchar: expected integer, got %T", intp.Stack[len(intp.Stack)-1])
		} else if n < 0 || n > 100 {
			return intp.e(eRangecheck, "beginbfchar: invalid length %d", n)
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.tmpChars = make([]bfChar, n)
		return nil
	}),
	"endbfchar": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "endbfchar: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmap.tmpChars)
		if base < 0 {
			return intp.e(eStackunderflow, "endbfchar: not enough arguments")
		}
		for i := range intp.cmap.tmpChars {
			code, ok := intp.Stack[base+2*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endbfchar: expected string, got %T", intp.Stack[base+2*i])
			}
			val := intp.Stack[base+2*i+1]
			if !isStringOrName(val) {
				return intp.e(eTypecheck, "endbfchar: expected string or name, got %T", val)
			}
			intp.cmap.tmpChars[i] = bfChar{code, val}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.chars = append(intp.cmap.chars, intp.cmap.tmpChars...)
		intp.cmap.tmpChars = nil
		return nil
	}),
	"beginbfrange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "beginbfrange: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "beginbfrange: not enough arguments")
		}
		n, ok := intp.Stack[len(intp.Stack)-1].(Integer)
		if !ok {
			return intp.e(eTypecheck, "beginbfrange: expected integer, got %T", intp.Stack[len(intp.Stack)-1])
		} else if n < 0 || n > 100 {
			return intp.e(eRangecheck, "beginbfrange: invalid length %d", n)
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.tmpRanges = make([]bfRange, n)
		return nil
	}),
	"endbfrange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "endbfrange: not in cmap block")
		}
		base := len(intp.Stack) - 3*len(intp.cmap.tmpRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endbfrange: not enough arguments")
		}
		for i := range intp.cmap.tmpRanges {
			lo, ok := intp.Stack[base+3*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endbfrange: expected string, got %T", intp.Stack[base+3*i])
			}
			hi, ok := intp.Stack[base+3*i+1].(String)
			if !ok {
				return intp.e(eTypecheck, "endbfrange: expected string, got %T", intp.Stack[base+3*i+1])
			}
			if len(lo) != len(hi) {
				return intp.e(eRangecheck, "endbfrange: expected strings of equal length, got %d and %d", len(lo), len(hi))
			}
			val := intp.Stack[base+3*i+2]
			if !isStringOrArray(val) {
				return intp.e(eTypecheck, "endbfrange: expected string or array of names, got %T", val)
			}
			intp.cmap.tmpRanges[i].low = lo
			intp.cmap.tmpRanges[i].high = hi
			intp.cmap.tmpRanges[i].value = val
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.ranges = append(intp.cmap.ranges, intp.cmap.tmpRanges...)
		intp.cmap.tmpRanges = nil
		return nil
	}),
}

func isStringOrName(o Object) bool {
	switch o.(type) {
	case String, Name:
		return true
	default:
		return false
	}
}

func isStringOrArray(o Object) bool {
	switch o.(type) {
	case String, Array:
		return true
	default:
		return false
	}
}

type cmapInfo struct {
	codeSpaceRanges    []codeSpaceRange
	tmpCodeSpaceRanges []codeSpaceRange
	chars              []bfChar
	tmpChars           []bfChar
	ranges             []bfRange
	tmpRanges          []bfRange
}

type codeSpaceRange struct {
	low, high []byte
}

type bfChar struct {
	src []byte
	dst Object
}

type bfRange struct {
	low, high []byte
	value     Object
}
