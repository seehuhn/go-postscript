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
	"fmt"
	"io"
	"slices"
	"sort"

	"golang.org/x/exp/maps"
)

// ReadCMap reads a CMap File from an [io.Reader].
//
// This is a thin wrapper around the [Interpreter.Execute] method.
//
// The returned Dict is a PostScript CMap dictionaries, as documented in
// section 5.11.4 (CMap Dictionaries) of the PostScript Language Reference
// Manual.  The "CodeMap" field of the CMAP dictionary can be cast to a
// [*CMapInfo] object, which contains the mapping data.
func ReadCMap(r io.Reader) (Dict, error) {
	intp := NewInterpreter()
	intp.MaxOps = 1_000_000 // TODO(voss): measure what is required
	err := intp.Execute(r)
	if err != nil {
		return nil, err
	}

	// make the function deterministic
	names := maps.Keys(intp.CMapDirectory)
	slices.Sort(names)

	for _, name := range names {
		val := intp.CMapDirectory[name]
		cmap, ok := val.(Dict)
		if !ok {
			continue
		}

		// If there is more than one CMap in the file, we return the first one.

		if n, _ := cmap["CMapName"].(Name); n == "" {
			cmap["CMapName"] = Name(name)
		}
		return cmap, nil
	}
	return nil, fmt.Errorf("no valid CMap found")
}

// CMapInfo contains the information for a CMap.
type CMapInfo struct {
	UseCMap         Name
	CodeSpaceRanges []CodeSpaceRange
	CidChars        []CharMap
	CidRanges       []RangeMap
	BfChars         []CharMap
	BfRanges        []RangeMap
	NotdefChars     []CharMap
	NotdefRanges    []RangeMap
}

// CodeSpaceRange represents a range of character codes.
type CodeSpaceRange struct {
	Low, High []byte
}

// CharMap represents a character mapping for a single code.
type CharMap struct {
	Src []byte
	Dst Object
}

// RangeMap represents a character mapping for a range of codes.
type RangeMap struct {
	Low, High []byte
	Dst       Object
}

// cidInit is the "CIDInit" ProcSet.  This defines functions for creating and
// populating CMAPs.
//
// The functions are explained in section 5.11.4 (CMap Dictionaries) of the
// PostScript Language Reference Manual.
var cidInit = Dict{
	"begincmap": builtin(func(intp *Interpreter) error {
		intp.cmapMappings = &CMapInfo{}
		return nil
	}),
	"endcmap": builtin(func(intp *Interpreter) error {
		if len(intp.DictStack) < 1 || intp.cmapMappings == nil {
			return intp.e(eStackunderflow, "endcmap: cmap dictionary not found")
		}

		// TODO(voss): is the sorting necessary?
		sort.Slice(intp.cmapMappings.CodeSpaceRanges, func(i, j int) bool {
			if len(intp.cmapMappings.CodeSpaceRanges[i].Low) != len(intp.cmapMappings.CodeSpaceRanges[j].Low) {
				return len(intp.cmapMappings.CodeSpaceRanges[i].Low) < len(intp.cmapMappings.CodeSpaceRanges[j].Low)
			}
			return bytes.Compare(intp.cmapMappings.CodeSpaceRanges[i].Low, intp.cmapMappings.CodeSpaceRanges[j].Low) < 0
		})
		sort.Slice(intp.cmapMappings.CidChars, func(i, j int) bool {
			return bytes.Compare(intp.cmapMappings.CidChars[i].Src, intp.cmapMappings.CidChars[j].Src) < 0
		})
		sort.Slice(intp.cmapMappings.CidRanges, func(i, j int) bool {
			return bytes.Compare(intp.cmapMappings.CidRanges[i].Low, intp.cmapMappings.CidRanges[j].Low) < 0
		})
		sort.Slice(intp.cmapMappings.BfChars, func(i, j int) bool {
			return bytes.Compare(intp.cmapMappings.BfChars[i].Src, intp.cmapMappings.BfChars[j].Src) < 0
		})
		sort.Slice(intp.cmapMappings.BfRanges, func(i, j int) bool {
			return bytes.Compare(intp.cmapMappings.BfRanges[i].Low, intp.cmapMappings.BfRanges[j].Low) < 0
		})
		sort.Slice(intp.cmapMappings.NotdefChars, func(i, j int) bool {
			return bytes.Compare(intp.cmapMappings.NotdefChars[i].Src, intp.cmapMappings.NotdefChars[j].Src) < 0
		})
		sort.Slice(intp.cmapMappings.NotdefRanges, func(i, j int) bool {
			return bytes.Compare(intp.cmapMappings.NotdefRanges[i].Low, intp.cmapMappings.NotdefRanges[j].Low) < 0
		})

		dict := intp.DictStack[len(intp.DictStack)-1]
		dict["CodeMap"] = intp.cmapMappings
		intp.cmapMappings = nil
		return nil
	}),
	"usecmap": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapMappings.UseCMap = name
		return nil
	}),
	"begincodespacerange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapCodeSpaceRanges = make([]CodeSpaceRange, n)
		return nil
	}),
	"endcodespacerange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
			return intp.e(eUndefined, "endcodespacerange: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmapCodeSpaceRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endcodespacerange: not enough arguments")
		}
		for i := range intp.cmapCodeSpaceRanges {
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
			intp.cmapCodeSpaceRanges[i] = CodeSpaceRange{lo, hi}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmapMappings.CodeSpaceRanges = append(intp.cmapMappings.CodeSpaceRanges, intp.cmapCodeSpaceRanges...)
		intp.cmapCodeSpaceRanges = nil
		return nil
	}),
	"begincidchar": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapChars = slices.Grow(intp.cmapChars, int(n))[:n]
		return nil
	}),
	"endcidchar": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
			return intp.e(eUndefined, "endcidchar: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmapChars)
		if base < 0 {
			return intp.e(eStackunderflow, "endcidchar: not enough arguments")
		}
		for i := range intp.cmapChars {
			code, ok := intp.Stack[base+2*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endcidchar: expected string, got %T", intp.Stack[base+2*i])
			}
			val := intp.Stack[base+2*i+1]
			if _, ok := val.(Integer); !ok {
				return intp.e(eTypecheck, "endcidchar: expected integer, got %T", val)
			}
			intp.cmapChars[i] = CharMap{code, val}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmapMappings.CidChars = append(intp.cmapMappings.CidChars, intp.cmapChars...)
		intp.cmapChars = intp.cmapChars[:0]
		return nil
	}),
	"begincidrange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapRanges = slices.Grow(intp.cmapRanges, int(n))[:n]
		return nil
	}),
	"endcidrange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
			return intp.e(eUndefined, "endcidrange: not in cmap block")
		}
		base := len(intp.Stack) - 3*len(intp.cmapRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endcidrange: not enough arguments")
		}
		for i := range intp.cmapRanges {
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
			intp.cmapRanges[i].Low = lo
			intp.cmapRanges[i].High = hi
			intp.cmapRanges[i].Dst = val
		}
		intp.Stack = intp.Stack[:base]
		intp.cmapMappings.CidRanges = append(intp.cmapMappings.CidRanges, intp.cmapRanges...)
		intp.cmapRanges = intp.cmapRanges[:0]
		return nil
	}),
	"beginbfchar": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapChars = slices.Grow(intp.cmapChars, int(n))[:n]
		return nil
	}),
	"endbfchar": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
			return intp.e(eUndefined, "endbfchar: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmapChars)
		if base < 0 {
			return intp.e(eStackunderflow, "endbfchar: not enough arguments")
		}
		for i := range intp.cmapChars {
			code, ok := intp.Stack[base+2*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endbfchar: expected string, got %T", intp.Stack[base+2*i])
			}
			val := intp.Stack[base+2*i+1]
			if !isStringOrName(val) {
				return intp.e(eTypecheck, "endbfchar: expected string or name, got %T", val)
			}
			intp.cmapChars[i] = CharMap{code, val}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmapMappings.BfChars = append(intp.cmapMappings.BfChars, intp.cmapChars...)
		intp.cmapChars = intp.cmapChars[:0]
		return nil
	}),
	"beginbfrange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapRanges = slices.Grow(intp.cmapRanges, int(n))[:n]
		return nil
	}),
	"endbfrange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
			return intp.e(eUndefined, "endbfrange: not in cmap block")
		}
		base := len(intp.Stack) - 3*len(intp.cmapRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endbfrange: not enough arguments")
		}
		for i := range intp.cmapRanges {
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
			intp.cmapRanges[i].Low = lo
			intp.cmapRanges[i].High = hi
			intp.cmapRanges[i].Dst = val
		}
		intp.Stack = intp.Stack[:base]
		intp.cmapMappings.BfRanges = append(intp.cmapMappings.BfRanges, intp.cmapRanges...)
		intp.cmapRanges = intp.cmapRanges[:0]
		return nil
	}),
	"beginnotdefchar": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapChars = slices.Grow(intp.cmapChars, int(n))[:n]
		return nil
	}),
	"endnotdefchar": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
			return intp.e(eUndefined, "endnotdefchar: not in cmap block")
		}
		base := len(intp.Stack) - 2*len(intp.cmapChars)
		if base < 0 {
			return intp.e(eStackunderflow, "endnotdefchar: not enough arguments")
		}
		for i := range intp.cmapChars {
			code, ok := intp.Stack[base+2*i].(String)
			if !ok {
				return intp.e(eTypecheck, "endnotdefchar: expected string, got %T", intp.Stack[base+2*i])
			}
			val := intp.Stack[base+2*i+1]
			if _, ok := val.(Integer); !ok {
				return intp.e(eTypecheck, "endnotdefchar: expected integer, got %T", val)
			}
			intp.cmapChars[i] = CharMap{code, val}
		}
		intp.Stack = intp.Stack[:base]
		intp.cmapMappings.NotdefChars = append(intp.cmapMappings.NotdefChars, intp.cmapChars...)
		intp.cmapChars = intp.cmapChars[:0]
		return nil
	}),
	"beginnotdefrange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
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
		intp.cmapRanges = slices.Grow(intp.cmapRanges, int(n))[:n]
		return nil
	}),
	"endnotdefrange": builtin(func(intp *Interpreter) error {
		if intp.cmapMappings == nil {
			return intp.e(eUndefined, "endnotdefrange: not in cmap block")
		}
		base := len(intp.Stack) - 3*len(intp.cmapRanges)
		if base < 0 {
			return intp.e(eStackunderflow, "endnotdefrange: not enough arguments")
		}
		for i := range intp.cmapRanges {
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
			intp.cmapRanges[i].Low = lo
			intp.cmapRanges[i].High = hi
			intp.cmapRanges[i].Dst = val
		}
		intp.Stack = intp.Stack[:base]
		intp.cmapMappings.NotdefRanges = append(intp.cmapMappings.NotdefRanges, intp.cmapRanges...)
		intp.cmapRanges = intp.cmapRanges[:0]
		return nil
	}),
}
