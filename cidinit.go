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

import (
	"bytes"
	"sort"
)

var CIDInit = Dict{
	"begincmap": builtin(func(intp *Interpreter) error {
		intp.cmap = &CMapInfo{}
		return nil
	}),
	"endcmap": builtin(func(intp *Interpreter) error {
		if len(intp.DictStack) < 1 || intp.cmap == nil {
			return intp.e(eStackunderflow, "endcmap: cmap dictionary not found")
		}
		sort.Slice(intp.cmap.CodeSpaceRanges, func(i, j int) bool {
			if len(intp.cmap.CodeSpaceRanges[i].Low) != len(intp.cmap.CodeSpaceRanges[j].Low) {
				return len(intp.cmap.CodeSpaceRanges[i].Low) < len(intp.cmap.CodeSpaceRanges[j].Low)
			}
			return bytes.Compare(intp.cmap.CodeSpaceRanges[i].Low, intp.cmap.CodeSpaceRanges[j].Low) < 0
		})
		sort.Slice(intp.cmap.Chars, func(i, j int) bool {
			return bytes.Compare(intp.cmap.Chars[i].Src, intp.cmap.Chars[j].Src) < 0
		})
		sort.Slice(intp.cmap.Ranges, func(i, j int) bool {
			return bytes.Compare(intp.cmap.Ranges[i].Low, intp.cmap.Ranges[j].Low) < 0
		})
		dict := intp.DictStack[len(intp.DictStack)-1]
		dict["CodeMap"] = intp.cmap
		intp.cmap = nil
		return nil
	}),
	"usecmap": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "usecmap: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "usecmap: not enough arguments")
		}
		name, ok := intp.Stack[len(intp.Stack)-1].(Name)
		if !ok {
			return intp.e(eTypecheck, "usecmap: expected name, got %T", intp.Stack[len(intp.Stack)-1])
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.UseCMap = string(name)
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
		intp.cmap.tmpCodeSpaceRanges = make([]CodeSpaceRange, n)
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
			intp.cmap.tmpCodeSpaceRanges[i] = CodeSpaceRange{lo, hi}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.CodeSpaceRanges = append(intp.cmap.CodeSpaceRanges, intp.cmap.tmpCodeSpaceRanges...)
		intp.cmap.tmpCodeSpaceRanges = nil
		return nil
	}),
	"begincidchar": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "begincidchar: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "begincidchar: not enough arguments")
		}
		n, ok := intp.Stack[len(intp.Stack)-1].(Integer)
		if !ok {
			return intp.e(eTypecheck, "begincidchar: expected integer, got %T", intp.Stack[len(intp.Stack)-1])
		} else if n < 0 || n > 100 {
			return intp.e(eRangecheck, "begincidchar: invalid length %d", n)
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.tmpChars = make([]CharMap, n)
		return nil
	}),
	"endcidchar": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "endcidchar: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmap.tmpChars)
		if base < 0 {
			return intp.e(eStackunderflow, "endcidchar: not enough arguments")
		}
		for i := range intp.cmap.tmpChars {
			code, ok := intp.Stack[base+2*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endcidchar: expected string, got %T", intp.Stack[base+2*i])
			}
			val := intp.Stack[base+2*i+1]
			if _, ok := val.(Integer); !ok {
				return intp.e(eTypecheck, "endcidchar: expected integer, got %T", val)
			}
			intp.cmap.tmpChars[i] = CharMap{code, val}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.Chars = append(intp.cmap.Chars, intp.cmap.tmpChars...)
		intp.cmap.tmpChars = nil
		return nil
	}),
	"begincidrange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "begincidrange: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "begincidrange: not enough arguments")
		}
		n, ok := intp.Stack[len(intp.Stack)-1].(Integer)
		if !ok {
			return intp.e(eTypecheck, "begincidrange: expected integer, got %T", intp.Stack[len(intp.Stack)-1])
		} else if n < 0 || n > 100 {
			return intp.e(eRangecheck, "begincidrange: invalid length %d", n)
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.tmpRanges = make([]RangeMap, n)
		return nil
	}),
	"endcidrange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "endcidrange: not in cmap block")
		}
		base := len(intp.Stack) - 3*len(intp.cmap.tmpRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endcidrange: not enough arguments")
		}
		for i := range intp.cmap.tmpRanges {
			lo, ok := intp.Stack[base+3*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endcidrange: expected string, got %T", intp.Stack[base+3*i])
			}
			hi, ok := intp.Stack[base+3*i+1].(String)
			if !ok {
				return intp.e(eTypecheck, "endcidrange: expected string, got %T", intp.Stack[base+3*i+1])
			}
			if len(lo) != len(hi) || bytes.Compare(lo, hi) > 0 {
				return intp.e(eRangecheck, "endcidrange: invalid range <%x> <%x>", lo, hi)
			}
			val := intp.Stack[base+3*i+2]
			if _, ok := val.(Integer); !ok {
				return intp.e(eTypecheck, "endcidrange: expected integer, got %T", val)
			}
			intp.cmap.tmpRanges[i].Low = lo
			intp.cmap.tmpRanges[i].High = hi
			intp.cmap.tmpRanges[i].Dst = val
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.Ranges = append(intp.cmap.Ranges, intp.cmap.tmpRanges...)
		intp.cmap.tmpRanges = nil
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
		intp.cmap.tmpChars = make([]CharMap, n)
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
			intp.cmap.tmpChars[i] = CharMap{code, val}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.Chars = append(intp.cmap.Chars, intp.cmap.tmpChars...)
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
		intp.cmap.tmpRanges = make([]RangeMap, n)
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
			if len(lo) != len(hi) || bytes.Compare(lo, hi) > 0 {
				return intp.e(eRangecheck, "endbfrange: invalid range <%x> <%x>", lo, hi)
			}
			val := intp.Stack[base+3*i+2]
			if !isStringOrArray(val) {
				return intp.e(eTypecheck, "endbfrange: expected string or array of names, got %T", val)
			}
			intp.cmap.tmpRanges[i].Low = lo
			intp.cmap.tmpRanges[i].High = hi
			intp.cmap.tmpRanges[i].Dst = val
		}
		intp.Stack = intp.Stack[:base]
		intp.cmap.Ranges = append(intp.cmap.Ranges, intp.cmap.tmpRanges...)
		intp.cmap.tmpRanges = nil
		return nil
	}),
	"beginnotdefchar": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "beginnotdefchar: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "beginnotdefchar: not enough arguments")
		}
		n, ok := intp.Stack[len(intp.Stack)-1].(Integer)
		if !ok {
			return intp.e(eTypecheck, "beginnotdefchar: expected integer, got %T", intp.Stack[len(intp.Stack)-1])
		} else if n < 0 || n > 100 {
			return intp.e(eRangecheck, "beginnotdefchar: invalid length %d", n)
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.tmpChars = make([]CharMap, n)
		return nil
	}),
	"endnotdefchar": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "endnotdefchar: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmap.tmpChars)
		if base < 0 {
			return intp.e(eStackunderflow, "endnotdefchar: not enough arguments")
		}
		for i := range intp.cmap.tmpChars {
			code, ok := intp.Stack[base+2*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endnotdefchar: expected string, got %T", intp.Stack[base+2*i])
			}
			val := intp.Stack[base+2*i+1]
			if _, ok := val.(Integer); !ok {
				return intp.e(eTypecheck, "endnotdefchar: expected integer, got %T", val)
			}
			intp.cmap.tmpChars[i] = CharMap{code, val}
		}
		intp.Stack = intp.Stack[:base]
		// intp.cmap.Chars = append(intp.cmap.Chars, intp.cmap.tmpChars...)
		intp.cmap.tmpChars = nil
		return nil
	}),
	"beginnotdefrange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "beginnotdefrange: not in cmap block")
		}
		if len(intp.Stack) < 1 {
			return intp.e(eStackunderflow, "beginnotdefrange: not enough arguments")
		}
		n, ok := intp.Stack[len(intp.Stack)-1].(Integer)
		if !ok {
			return intp.e(eTypecheck, "beginnotdefrange: expected integer, got %T", intp.Stack[len(intp.Stack)-1])
		} else if n < 0 || n > 100 {
			return intp.e(eRangecheck, "beginnotdefrange: invalid length %d", n)
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.cmap.tmpRanges = make([]RangeMap, n)
		return nil
	}),
	"endnotdefrange": builtin(func(intp *Interpreter) error {
		if intp.cmap == nil {
			return intp.e(eUndefined, "endnotdefrange: not in cmap block")
		}
		base := len(intp.Stack) - 3*len(intp.cmap.tmpRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endnotdefrange: not enough arguments")
		}
		for i := range intp.cmap.tmpRanges {
			lo, ok := intp.Stack[base+3*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endnotdefrange: expected string, got %T", intp.Stack[base+3*i])
			}
			hi, ok := intp.Stack[base+3*i+1].(String)
			if !ok {
				return intp.e(eTypecheck, "endnotdefrange: expected string, got %T", intp.Stack[base+3*i+1])
			}
			if len(lo) != len(hi) || bytes.Compare(lo, hi) > 0 {
				return intp.e(eRangecheck, "endnotdefrange: invalid range <%x> <%x>", lo, hi)
			}
			val := intp.Stack[base+3*i+2]
			if _, ok := val.(Integer); !ok {
				return intp.e(eTypecheck, "endnotdefrange: expected integer, got %T", val)
			}
			intp.cmap.tmpRanges[i].Low = lo
			intp.cmap.tmpRanges[i].High = hi
			intp.cmap.tmpRanges[i].Dst = val
		}
		intp.Stack = intp.Stack[:base]
		// intp.cmap.Ranges = append(intp.cmap.Ranges, intp.cmap.tmpRanges...)
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

type CMapInfo struct {
	CodeSpaceRanges    []CodeSpaceRange
	tmpCodeSpaceRanges []CodeSpaceRange
	Chars              []CharMap
	tmpChars           []CharMap
	Ranges             []RangeMap
	tmpRanges          []RangeMap
	UseCMap            string
}

type CodeSpaceRange struct {
	Low, High []byte
}

type CharMap struct {
	Src []byte
	Dst Object
}

type RangeMap struct {
	Low, High []byte
	Dst       Object
}
