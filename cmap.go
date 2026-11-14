// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"bytes"
	"errors"
	"io"
	"slices"
)

// cmapMaxMemory bounds the total memory the interpreter may allocate while
// reading a single CMap.  Real CMaps need only a few MiB; this leaves ample
// headroom for transient allocations while stopping a malformed one well
// before it can exhaust memory.  Each ReadCMap call uses a fresh budget, so a
// document with many CMaps is bounded per CMap, not in aggregate.
const cmapMaxMemory = 64 << 20 // 64 MiB

// cmapMaxOperandStack is the operand stack limit while reading a CMap.  The
// entries of a begin…end block stay on the operand stack until the end
// operator, and some CMap files put thousands of entries into one block.
// The limit keeps the stack below 16 MiB.
const cmapMaxOperandStack = 1 << 20

// ReadCMap reads a CMap from an [io.Reader].
//
// The reader is permissive.  The operators of the CIDInit ProcSet are
// available even if the file does not look them up.  The entry count before
// a begin…end block is ignored: the block holds all entries between the two
// operators, interpreted according to the begin operator.  Malformed entries
// are skipped.  If the file defines no CMap using defineresource, for example
// because it is truncated, the CMap built most recently is returned, with all
// entries read before the problem.
// Execution errors are reported only if no CMap was found.
//
// If the file defines more than one CMap, the result is the last usable one
// whose resource name or CMapName entry equals name.  If name is empty or no
// CMap matches, the result is the last usable CMap defined.  A CMap is usable
// if its dictionary holds a [*CMapInfo] under the CodeMap key.
//
// The returned Dict is the PostScript CMap dictionary, as documented in
// section 5.11.4 (CMap Dictionaries) of the PostScript Language Reference
// Manual.  The returned [*CMapInfo] holds the mapping data; the same value
// is also stored under the "CodeMap" key in the dictionary.
func ReadCMap(r io.Reader, name Name) (Dict, *CMapInfo, error) {
	intp := NewInterpreter()
	intp.MaxOps = 1_000_000 // TODO(voss): measure what is required
	intp.MaxMemory = cmapMaxMemory
	intp.maxStack = cmapMaxOperandStack
	procSet := intp.Resources["ProcSet"].(Dict)["CIDInit"].(Dict)
	intp.DictStack = append(intp.DictStack, procSet)

	execErr := intp.Execute(r)

	cmap, codeMap, err := intp.cmapResult(name)
	if err != nil {
		if execErr != nil {
			return nil, nil, execErr
		}
		return nil, nil, err
	}
	return cmap, codeMap, nil
}

// cmapResult returns the CMap defined by the code executed so far.  An
// unfinished CMap is completed first.  Usable CMaps added to the CMap
// directory take precedence, see [ReadCMap] for the choice between several of
// them.  If there are none, the CMap built most recently is used.
func (intp *Interpreter) cmapResult(name Name) (Dict, *CMapInfo, error) {
	if intp.cmapMappings != nil {
		if b := intp.cmapBlock; b != nil {
			// a truncated file leaves an incomplete entry on top
			mark := min(intp.cmapMark, len(intp.Stack))
			intp.Stack = intp.Stack[:len(intp.Stack)-(len(intp.Stack)-mark)%b.size]
		}
		intp.cmapFinish()
	}

	usable := func(d Dict) bool {
		m, _ := d["CodeMap"].(*CMapInfo)
		return m != nil
	}
	var key Name
	var cmap Dict
	for _, k := range slices.Backward(intp.cmapDefined) {
		d, _ := intp.CMapDirectory[k].(Dict)
		if !usable(d) {
			continue
		}
		if cmap == nil {
			key, cmap = k, d
		}
		if name == "" {
			break
		}
		if n, _ := d["CMapName"].(Name); k == name || n == name {
			key, cmap = k, d
			break
		}
	}
	if cmap == nil && usable(intp.cmapDict) {
		cmap = intp.cmapDict
	}
	if cmap == nil {
		return nil, nil, errors.New("no CMap found")
	}
	codeMap := cmap["CodeMap"].(*CMapInfo)
	if n, _ := cmap["CMapName"].(Name); n == "" && key != "" {
		cmap["CMapName"] = key
	}
	return cmap, codeMap, nil
}

// CMapInfo contains the information for a CMap.
// Each slice lists its entries in the order the CMap file gives them.
//
// The Dst field of a CID or notdef mapping is an [Integer] or a [Real].  For
// a bfchar mapping it is a [String] or a [Name], and for a bfrange mapping
// a [String] or an [Array].
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
// Low and High have the same length.
type CodeSpaceRange struct {
	Low, High []byte
}

// CharMap represents a character mapping for a single code.
type CharMap struct {
	Src []byte
	Dst Object
}

// RangeMap represents a character mapping for a range of codes.
// Low and High have the same length, and Low is not greater than High.
type RangeMap struct {
	Low, High []byte
	Dst       Object
}

// cmapBlock describes one kind of begin…end block of CMap entries.
type cmapBlock struct {
	// size is the number of operands per entry.
	size int

	// add appends an entry to m, unless the entry is malformed.
	add func(m *CMapInfo, entry []Object)
}

var cmapBlocks = map[Name]*cmapBlock{
	"codespacerange": {size: 2, add: addCodeSpaceRange},
	"cidchar":        charBlock(func(m *CMapInfo) *[]CharMap { return &m.CidChars }, isNumber),
	"cidrange":       rangeBlock(func(m *CMapInfo) *[]RangeMap { return &m.CidRanges }, isNumber),
	"bfchar":         charBlock(func(m *CMapInfo) *[]CharMap { return &m.BfChars }, isStringOrName),
	"bfrange":        rangeBlock(func(m *CMapInfo) *[]RangeMap { return &m.BfRanges }, isStringOrArray),
	"notdefchar":     charBlock(func(m *CMapInfo) *[]CharMap { return &m.NotdefChars }, isNumber),
	"notdefrange":    rangeBlock(func(m *CMapInfo) *[]RangeMap { return &m.NotdefRanges }, isNumber),
}

func addCodeSpaceRange(m *CMapInfo, entry []Object) {
	lo, ok1 := entry[0].(String)
	hi, ok2 := entry[1].(String)
	if ok1 && ok2 && len(lo) == len(hi) {
		m.CodeSpaceRanges = append(m.CodeSpaceRanges, CodeSpaceRange{lo, hi})
	}
}

func charBlock(field func(*CMapInfo) *[]CharMap, validDst func(Object) bool) *cmapBlock {
	return &cmapBlock{
		size: 2,
		add: func(m *CMapInfo, entry []Object) {
			code, ok := entry[0].(String)
			if ok && validDst(entry[1]) {
				list := field(m)
				*list = append(*list, CharMap{code, entry[1]})
			}
		},
	}
}

func rangeBlock(field func(*CMapInfo) *[]RangeMap, validDst func(Object) bool) *cmapBlock {
	return &cmapBlock{
		size: 3,
		add: func(m *CMapInfo, entry []Object) {
			lo, ok1 := entry[0].(String)
			hi, ok2 := entry[1].(String)
			if ok1 && ok2 && len(lo) == len(hi) && bytes.Compare(lo, hi) <= 0 &&
				validDst(entry[2]) {
				list := field(m)
				*list = append(*list, RangeMap{lo, hi, entry[2]})
			}
		},
	}
}

// cidInit is the "CIDInit" ProcSet.  This defines functions for creating and
// populating CMAPs.
//
// The functions are explained in section 5.11.4 (CMap Dictionaries) of the
// PostScript Language Reference Manual.
var cidInit = makeCIDInit()

func makeCIDInit() Dict {
	d := Dict{
		"begincmap": &builtin{func(intp *Interpreter) error {
			intp.cmapStart()
			return nil
		}},
		"endcmap": &builtin{func(intp *Interpreter) error {
			intp.cmapStart()
			intp.cmapFinish()
			return nil
		}},
		"usecmap": &builtin{func(intp *Interpreter) error {
			m := intp.cmapStart()
			if n := len(intp.Stack); n > 0 {
				if name, ok := intp.Stack[n-1].(Name); ok {
					intp.Stack = intp.Stack[:n-1]
					m.UseCMap = name
				}
			}
			return nil
		}},
	}
	for name, b := range cmapBlocks {
		d["begin"+name] = &builtin{func(intp *Interpreter) error {
			intp.cmapBegin(b)
			return nil
		}}
		d["end"+name] = &builtin{func(intp *Interpreter) error {
			intp.cmapEnd(b)
			return nil
		}}
	}
	return d
}

// cmapStart returns the CMap of the current dictionary, which receives the
// entries read from now on.  The CMap of a dictionary is the one stored under
// its CodeMap key.  If there is none, a new CMap is stored there straight
// away, so that CMap files without begincmap can be read and defineresource
// accepts the dictionary even if endcmap is missing.  A CMap still being built
// in another dictionary lacks its endcmap, and is completed first.
func (intp *Interpreter) cmapStart() *CMapInfo {
	top := intp.DictStack[len(intp.DictStack)-1]
	m, _ := top["CodeMap"].(*CMapInfo)
	if m != nil && m == intp.cmapMappings {
		return m
	}
	if intp.cmapMappings != nil {
		intp.cmapFinish()
	}
	if m == nil {
		m = &CMapInfo{}
		top["CodeMap"] = m
	}
	intp.cmapMappings = m
	intp.cmapDict = top
	return m
}

// cmapFinish completes the CMap being built.
func (intp *Interpreter) cmapFinish() {
	if intp.cmapBlock != nil {
		intp.cmapFlush()
	}
	intp.cmapMappings = nil
}

// cmapBegin starts a block of entries of kind b.  The entry count before the
// begin operator is discarded, and a block which is still open is completed
// first.
func (intp *Interpreter) cmapBegin(b *cmapBlock) {
	intp.cmapStart()
	if n := len(intp.Stack); n > 0 {
		_, isInt := intp.Stack[n-1].(Integer)
		// in an open block, a count follows the last complete entry
		open := intp.cmapBlock
		if isInt && (open == nil || (n-min(intp.cmapMark, n))%open.size == 1) {
			intp.Stack = intp.Stack[:n-1]
		}
	}
	if intp.cmapBlock != nil {
		intp.cmapFlush()
	}
	intp.cmapBlock = b
	intp.cmapMark = len(intp.Stack)
}

// cmapEnd completes a block of entries.  The kind of the block is given by
// its begin operator; b is used only if no block is open, and then all
// operands are taken as entries.
func (intp *Interpreter) cmapEnd(b *cmapBlock) {
	intp.cmapStart()
	if intp.cmapBlock == nil {
		intp.cmapBlock = b
		intp.cmapMark = 0
	}
	intp.cmapFlush()
}

// cmapFlush adds the entries of the open block to the CMap being built.  The
// entries are all operands pushed since the begin operator.  If their number
// is not a multiple of the entry size, the excess operands at the bottom are
// discarded.
func (intp *Interpreter) cmapFlush() {
	b := intp.cmapBlock
	mark := min(intp.cmapMark, len(intp.Stack))
	operands := intp.Stack[mark:]
	for i := len(operands) % b.size; i < len(operands); i += b.size {
		b.add(intp.cmapMappings, operands[i:i+b.size])
	}
	clear(operands)
	intp.Stack = intp.Stack[:mark]
	intp.cmapBlock = nil
}
