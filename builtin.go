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
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"

	"seehuhn.de/go/postscript/psenc"
)

func makeSystemDict() Dict {
	FontDirectory := Dict{}
	userDict := Dict{}
	errorDict := Dict{}

	standardEncoding := make(Array, 256)
	for i, name := range psenc.StandardEncoding {
		standardEncoding[i] = Name(name)
	}

	systemDict := Dict{
		"[":                builtin(bListStart),
		"]":                builtin(bListEnd),
		"<<":               builtin(bDictStart),
		">>":               builtin(bDictEnd),
		"abs":              builtin(bAbs),
		"add":              builtin(bAdd),
		"and":              builtin(bAnd),
		"atan":             builtin(bAtan),
		"array":            builtin(bArray),
		"begin":            builtin(bBegin),
		"bind":             builtin(bBind),
		"bitshift":         builtin(bBitshift),
		"ceiling":          builtin(bCeiling),
		"cleartomark":      builtin(bCleartomark),
		"closefile":        builtin(bClosefile),
		"cos":              builtin(bCos),
		"copy":             builtin(bCopy),
		"count":            builtin(bCount),
		"currentdict":      builtin(bCurrentdict),
		"currentfile":      builtin(bCurrentfile),
		"cvi":              builtin(bCvi),
		"cvr":              builtin(bCvr),
		"cvx":              builtin(bCvx),
		"def":              builtin(bDef),
		"definefont":       builtin(bDefinefont),
		"defineresource":   builtin(bDefineresource),
		"dict":             builtin(bDict),
		"div":              builtin(bDiv),
		"dup":              builtin(bDup),
		"exec":             builtin(bExec),
		"eexec":            builtin(eexec),
		"end":              builtin(bEnd),
		"eq":               builtin(bEq),
		"errordict":        errorDict,
		"exch":             builtin(bExch),
		"executeonly":      builtin(bExecuteonly),
		"exp":              builtin(bExp),
		"exit":             builtin(bExit),
		"false":            Boolean(false),
		"findfont":         builtin(bFindfont),
		"findresource":     builtin(bFindresource),
		"floor":            builtin(bFloor),
		"FontDirectory":    FontDirectory,
		"for":              builtin(bFor),
		"forall":           builtin(bForall),
		"ge":               builtin(bGe),
		"get":              builtin(bGet),
		"getinterval":      builtin(bGetinterval),
		"gt":               builtin(bGt),
		"idiv":             builtin(bIdiv),
		"if":               builtin(bIf),
		"ifelse":           builtin(bIfelse),
		"index":            builtin(bIndex),
		"internaldict":     builtin(bInternaldict),
		"known":            builtin(bKnown),
		"le":               builtin(bLe),
		"length":           builtin(bLength),
		"ln":               builtin(bLn),
		"load":             builtin(bLoad),
		"log":              builtin(bLog),
		"loop":             builtin(bLoop),
		"lt":               builtin(bLt),
		"mark":             builtin(bMark),
		"matrix":           builtin(bMatrix),
		"maxlength":        builtin(bMaxlength),
		"mod":              builtin(bMod),
		"mul":              builtin(bMul),
		"neg":              builtin(bNeg),
		"ne":               builtin(bNe),
		"noaccess":         builtin(bNoaccess),
		"not":              builtin(bNot),
		"or":               builtin(bOr),
		"pop":              builtin(bPop),
		"put":              builtin(bPut),
		"putinterval":      builtin(bPutinterval),
		"readonly":         builtin(bReadonly),
		"readstring":       builtin(bReadstring),
		"repeat":           builtin(bRepeat),
		"roll":             builtin(bRoll),
		"round":            builtin(bRound),
		"sin":              builtin(bSin),
		"sqrt":             builtin(bSqrt),
		"StandardEncoding": standardEncoding,
		"stop":             builtin(bStop),
		"string":           builtin(bString),
		"sub":              builtin(bSub),
		"true":             Boolean(true),
		"truncate":         builtin(bTruncate),
		"type":             builtin(bType),
		"userdict":         userDict,
		"where":            builtin(bWhere),
		"xor":              builtin(bXor),
	}
	systemDict["systemdict"] = systemDict

	return systemDict
}

func bListStart(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, theMark)
	return nil
}

func bListEnd(intp *Interpreter) error {
	n := len(intp.Stack)
	for i := n - 1; i >= 0; i-- {
		if intp.Stack[i] == theMark {
			size := n - i - 1
			a := make(Array, size)
			copy(a, intp.Stack[i+1:])
			intp.Stack = append(intp.Stack[:i], a)
			return nil
		}
	}
	return intp.e(eUnmatchedmark, "]: missing '['")
}

func bDictStart(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, theMark)
	return nil
}

func bDictEnd(intp *Interpreter) error {
	n := len(intp.Stack)
	markPos := -1
	for i := n - 1; i >= 0; i-- {
		if intp.Stack[i] == theMark {
			markPos = i
			break
		}
	}
	if markPos < 0 {
		return intp.e(eUnmatchedmark, ">>: missing '<<'")
	} else if (n-markPos)%2 != 1 {
		return intp.e(eRangecheck, "dict literal: odd length")
	}
	d := make(Dict, (n-markPos-1)/2)
	for i := markPos + 1; i < n; i += 2 {
		name, ok := intp.Stack[i].(Name)
		if !ok {
			return intp.e(eTypecheck, "dict literal: keys must be Name, not %T", intp.Stack[i])
		}
		d[name] = intp.Stack[i+1]
	}
	intp.Stack = append(intp.Stack[:markPos], d)
	return nil
}

func bAbs(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "abs: not enough arguments")
	}
	x := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	switch x := x.(type) {
	case Integer:
		if x == math.MinInt {
			intp.Stack = append(intp.Stack, -Real(x))
		} else if x < 0 {
			intp.Stack = append(intp.Stack, -x)
		} else {
			intp.Stack = append(intp.Stack, x)
		}
	case Real:
		if x < 0 {
			intp.Stack = append(intp.Stack, -x)
		} else {
			intp.Stack = append(intp.Stack, x)
		}
	default:
		return intp.e(eTypecheck, "abs: needs a number")
	}
	return nil
}

func bAdd(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "add: not enough arguments")
	}
	ar, aIsReal := intp.Stack[len(intp.Stack)-2].(Real)
	ai, aIsInt := intp.Stack[len(intp.Stack)-2].(Integer)
	br, bIsReal := intp.Stack[len(intp.Stack)-1].(Real)
	bi, bIsInt := intp.Stack[len(intp.Stack)-1].(Integer)
	if !(aIsReal || aIsInt) || !(bIsReal || bIsInt) {
		return intp.e(eTypecheck, "add: needs numbers")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	if aIsReal || bIsReal {
		if aIsInt {
			ar = Real(ai)
		}
		if bIsInt {
			br = Real(bi)
		}
		intp.Stack = append(intp.Stack, ar+br)
	} else {
		ci := ai + bi
		// check for integer overflow
		if (ai < 0 && bi < 0 && ci >= 0) || (ai > 0 && bi > 0 && ci <= 0) {
			intp.Stack = append(intp.Stack, Real(ai)+Real(bi))
		} else {
			intp.Stack = append(intp.Stack, ci)
		}
	}
	return nil
}

func bAnd(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "and: not enough arguments")
	}
	a := intp.Stack[len(intp.Stack)-2]
	b := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	switch a := a.(type) {
	case Boolean:
		b, ok := b.(Boolean)
		if !ok {
			return intp.e(eTypecheck, "and: mismatched argument types")
		}
		intp.Stack = append(intp.Stack, a && b)
	case Integer:
		b, ok := b.(Integer)
		if !ok {
			return intp.e(eTypecheck, "and: mismatched argument types")
		}
		intp.Stack = append(intp.Stack, a&b)
	default:
		return intp.e(eTypecheck, "and: invalid argument type %T", a)
	}
	return nil
}

func bArray(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "array: not enough arguments")
	}
	size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
	if !ok {
		return intp.e(eTypecheck, "array: need an integer")
	} else if size < 0 {
		return intp.e(eRangecheck, "array: invalid size %d", size)
	} else if size > maxArraySize {
		return intp.e(eLimitcheck, "array: invalid size %d", size)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	intp.Stack = append(intp.Stack, make(Array, size))
	return nil
}

func bBegin(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "begin: not enough arguments")
	}
	if len(intp.DictStack) >= maxDictStackDepth {
		return intp.e(eDictstackoverflow, "begin")
	}
	d, ok := intp.Stack[len(intp.Stack)-1].(Dict)
	if !ok {
		return intp.e(eTypecheck, "begin: needs a dictionary")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	intp.DictStack = append(intp.DictStack, d)
	return nil
}

func bBind(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "bind: not enough arguments")
	}
	obj, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
	if !ok {
		return intp.e(eTypecheck, "bind: needs a procedure, not %T", obj)
	}
	intp.bindProc(obj)
	return nil
}

func bCleartomark(intp *Interpreter) error {
	for k := len(intp.Stack) - 1; k >= 0; k-- {
		if intp.Stack[k] == theMark {
			intp.Stack = intp.Stack[:k]
			return nil
		}
	}
	return intp.e(eUnmatchedmark, "cleartomark: no mark found")
}

func bClosefile(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "closefile: not enough arguments")
	}
	if x := intp.Stack[len(intp.Stack)-1]; x != nil {
		return intp.e(eTypecheck, "closefile: needs a file, not %T", x)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	return io.EOF
}

func bCopy(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "copy: not enough arguments")
	}
	if n, ok := intp.Stack[len(intp.Stack)-1].(Integer); ok {
		if n < 0 {
			return intp.e(eRangecheck, "copy: invalid count %d", n)
		}
		if len(intp.Stack) < int(n)+1 {
			return intp.e(eStackunderflow, "copy: not enough arguments")
		}
		intp.Stack = intp.Stack[:len(intp.Stack)-1]
		intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-int(n):]...)
		return nil
	}
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "copy: not enough arguments")
	}
	a := intp.Stack[len(intp.Stack)-2]
	b := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	var res Object
	switch a := a.(type) {
	case Array:
		b, ok := b.(Array)
		if !ok {
			return intp.e(eTypecheck, "copy: mismatched argument types")
		} else if len(b) < len(a) {
			return intp.e(eRangecheck, "copy: not enough space in destination")
		}
		n := copy(b, a)
		res = b[:n]
	case Dict:
		b, ok := b.(Dict)
		if !ok {
			return intp.e(eTypecheck, "copy: mismatched argument types")
		}
		for k, v := range a {
			b[k] = v
		}
		res = b
	case String:
		b, ok := b.(String)
		if !ok {
			return intp.e(eTypecheck, "copy: mismatched argument types")
		} else if len(b) < len(a) {
			return intp.e(eRangecheck, "copy: not enough space in destination")
		}
		n := copy(b, a)
		res = b[:n]
	default:
		return intp.e(eTypecheck, "copy: invalid type %T", a)
	}
	intp.Stack = append(intp.Stack, res)
	return nil
}

func bCount(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, Integer(len(intp.Stack)))
	return nil
}

func bCurrentdict(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, intp.DictStack[len(intp.DictStack)-1])
	return nil
}

func bCurrentfile(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, nil)
	return nil
}

func bCvx(intp *Interpreter) error {
	// nearly not implemented
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "cvx: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-1]
	if a, ok := obj.(Array); ok {
		b := make(Procedure, len(a))
		copy(b, a)
		intp.Stack[len(intp.Stack)-1] = b
	}
	return nil
}

func bDef(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "def: not enough arguments")
	}
	name, ok := intp.Stack[len(intp.Stack)-2].(Name)
	if !ok {
		return intp.e(eTypecheck, "def: needs name, not %T", intp.Stack[len(intp.Stack)-2])
	}
	intp.DictStack[len(intp.DictStack)-1][name] = intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	return nil
}

func bDefinefont(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "definefont: not enough arguments")
	}
	name, ok := intp.Stack[len(intp.Stack)-2].(Name)
	if !ok {
		return intp.e(eTypecheck, "definefont: needs name, not %T", intp.Stack[len(intp.Stack)-2])
	}
	font, ok := intp.Stack[len(intp.Stack)-1].(Dict)
	if !ok {
		return intp.e(eTypecheck, "definefont: needs font, not %T", intp.Stack[len(intp.Stack)-1])
	}
	intp.FontDirectory[name] = font
	intp.Stack = append(intp.Stack[:len(intp.Stack)-2], font)
	return nil
}

func bDefineresource(intp *Interpreter) error {
	// TODO(voss): implement the behaviour described in section 3.9 of PLRM.
	if len(intp.Stack) < 3 {
		return intp.e(eStackunderflow, "defineresource: not enough arguments")
	}
	key, ok := intp.Stack[len(intp.Stack)-3].(Name)
	if !ok {
		return intp.e(eTypecheck, "defineresource: needs name, not %T", intp.Stack[len(intp.Stack)-3])
	}
	instance := intp.Stack[len(intp.Stack)-2]
	class, ok := intp.Stack[len(intp.Stack)-1].(Name)
	if !ok {
		return intp.e(eTypecheck, "defineresource: needs name, not %T", intp.Stack[len(intp.Stack)-1])
	}
	classDict, ok := intp.Resources[class].(Dict)
	if !ok {
		return intp.e(eUndefined, "defineresource: undefined resource class %q", class)
	}

	switch class {
	case "CMap":
		if d, ok := instance.(Dict); !ok {
			return intp.e(eTypecheck, "defineresource: needs dict, not %T", instance)
		} else if _, ok := d["CodeMap"].(*CMapInfo); !ok {
			return intp.e(eTypecheck, "defineresource: not a CMap")
		}
	}

	classDict[key] = instance
	intp.Stack = append(intp.Stack[:len(intp.Stack)-3], instance)
	return nil
}

func bDict(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "dict: not enough arguments")
	}
	size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
	if !ok {
		return intp.e(eTypecheck, "dict: needs an integer, not %T", intp.Stack[len(intp.Stack)-1])
	} else if size < 0 {
		return intp.e(eRangecheck, "dict: invalid size %d", size)
	} else if size > maxDictSize {
		return intp.e(eLimitcheck, "dict: invalid size %d", size)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	intp.Stack = append(intp.Stack, make(Dict, size))
	return nil
}

func bDup(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "dup: not enough arguments")
	}
	intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-1])
	return nil
}

func bExec(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "exec: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch obj := obj.(type) {
	case builtin:
		return obj(intp)
	case Procedure:
		return intp.executeOne(obj, true)
	default:
		return intp.e(eTypecheck, "exec: not implemented for %T", obj)
	}
}

func bEnd(intp *Interpreter) error {
	if len(intp.DictStack) <= 2 {
		return intp.e(eDictstackunderflow, "end: dictionary stack is empty")
	}
	intp.DictStack = intp.DictStack[:len(intp.DictStack)-1]
	return nil
}

func bEq(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "eq: not enough arguments")
	}
	a := intp.Stack[len(intp.Stack)-2]
	b := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	isEqual, err := equal(a, b)
	if err != nil {
		return err
	}

	intp.Stack = append(intp.Stack, Boolean(isEqual))
	return nil
}

func bExch(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "exch: not enough arguments")
	}
	intp.Stack[len(intp.Stack)-1], intp.Stack[len(intp.Stack)-2] = intp.Stack[len(intp.Stack)-2], intp.Stack[len(intp.Stack)-1]
	return nil
}

func bExecuteonly(intp *Interpreter) error {
	// not implemented
	return nil
}

func bExit(intp *Interpreter) error {
	return errExit
}

func bFindfont(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "findfont: not enough arguments")
	}
	name, ok := intp.Stack[len(intp.Stack)-1].(Name)
	if !ok {
		return intp.e(eTypecheck, "findfont: needs a name, not %T", intp.Stack[len(intp.Stack)-1])
	}
	font, ok := intp.FontDirectory[name]
	if !ok {
		return intp.e(eInvalidfont, "font %q not found", name)
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-1], font)
	return nil
}

func bFindresource(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "findresource: not enough arguments")
	}
	catName, ok := intp.Stack[len(intp.Stack)-1].(Name)
	if !ok {
		return intp.e(eTypecheck, "findresource: needs a name, not %T", intp.Stack[len(intp.Stack)-2])
	}
	cat, ok := intp.Resources[catName]
	if !ok {
		return intp.e(eUndefined, "resource category %q not found", catName)
	}
	keyObj := intp.Stack[len(intp.Stack)-2]
	var key Name
	switch keyObj := keyObj.(type) {
	case Name:
		key = keyObj
	case String:
		key = Name(keyObj)
	default:
		return intp.e(eUndefinedresource, "findresource: needs a name or string, not %T", keyObj)
	}
	catDict := cat.(Dict)
	obj, ok := catDict[key]
	if !ok {
		return intp.e(eUndefinedresource, "resource %q not found in category %q", key, catName)
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-2], obj)
	return nil
}

func bFor(intp *Interpreter) error {
	if len(intp.Stack) < 4 {
		return intp.e(eStackunderflow, "for: not enough arguments")
	}
	// TODO(voss): the spec also allows Real values here
	initial, ok := intp.Stack[len(intp.Stack)-4].(Integer)
	if !ok {
		return intp.e(eTypecheck, "for: invalid start")
	}
	increment, ok := intp.Stack[len(intp.Stack)-3].(Integer)
	if !ok {
		return intp.e(eTypecheck, "for: invalid increment")
	}
	limit, ok := intp.Stack[len(intp.Stack)-2].(Integer)
	if !ok {
		return intp.e(eTypecheck, "for: invalid limit")
	}
	proc := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-4]
	val := initial
	for {
		if increment > 0 && val > limit || increment < 0 && val < limit {
			break
		}
		intp.Stack = append(intp.Stack, val)
		err := intp.executeOne(proc, true)
		if err == errExit {
			break
		} else if err != nil {
			return err
		}
		val += increment
	}
	return nil
}

func bForall(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "forall: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-2]
	proc, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
	if !ok {
		return intp.e(eTypecheck, "forall: invalid argument")
	}
	switch obj := obj.(type) {
	case Array:
		intp.Stack = intp.Stack[:len(intp.Stack)-2]
		for _, val := range obj {
			intp.Stack = append(intp.Stack, val)
			err := intp.executeOne(proc, true)
			if err == errExit {
				break
			} else if err != nil {
				return err
			}
		}
	case String:
		intp.Stack = intp.Stack[:len(intp.Stack)-2]
		for _, c := range obj {
			intp.Stack = append(intp.Stack, Integer(c))
			err := intp.executeOne(proc, true)
			if err == errExit {
				break
			} else if err != nil {
				return err
			}
		}
	case Dict:
		intp.Stack = intp.Stack[:len(intp.Stack)-2]
		for key, val := range obj {
			intp.Stack = append(intp.Stack, key, val)
			err := intp.executeOne(proc, true)
			if err == errExit {
				break
			} else if err != nil {
				return err
			}
		}
	default:
		return intp.e(eTypecheck, "forall: invalid type %T", obj)
	}
	return nil
}

func bGet(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "get: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-2]
	sel := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	switch obj := obj.(type) {
	case Array:
		index, ok := sel.(Integer)
		if !ok {
			return intp.e(eTypecheck, "get: invalid index")
		}
		if index < 0 || index >= Integer(len(obj)) {
			return intp.e(eRangecheck, "get: index out of bounds")
		}
		intp.Stack = append(intp.Stack, obj[index])
	case Procedure:
		index, ok := sel.(Integer)
		if !ok {
			return intp.e(eTypecheck, "get: invalid index")
		}
		if index < 0 || index >= Integer(len(obj)) {
			return intp.e(eRangecheck, "get: index out of bounds")
		}
		intp.Stack = append(intp.Stack, obj[index])
	case Dict:
		name, ok := sel.(Name)
		if !ok {
			return intp.e(eTypecheck, "get: invalid dict key")
		}
		val, ok := obj[name]
		if !ok {
			return intp.e(eUndefined, "get: missing dict key %q", name)
		}
		intp.Stack = append(intp.Stack, val)
	case String:
		index, ok := sel.(Integer)
		if !ok {
			return intp.e(eTypecheck, "get: invalid index")
		}
		if index < 0 || index >= Integer(len(obj)) {
			return intp.e(eRangecheck, "get: index out of bounds")
		}
		intp.Stack = append(intp.Stack, obj[index])
	default:
		return intp.e(eTypecheck, "get: invalid argument type %T", obj)
	}
	return nil
}

func bGetinterval(intp *Interpreter) error {
	if len(intp.Stack) < 3 {
		return intp.e(eStackunderflow, "getinterval: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-3]
	var n int
	switch obj := obj.(type) {
	case Array:
		n = len(obj)
	case String:
		n = len(obj)
	default:
		return intp.e(eTypecheck, "getinterval: invalid argument type %T", obj)
	}
	index, ok := intp.Stack[len(intp.Stack)-2].(Integer)
	if !ok {
		return intp.e(eTypecheck, "getinterval: invalid index")
	} else if index < 0 || index >= Integer(n) {
		return intp.e(eRangecheck, "getinterval: index %d out of bounds", index)
	}
	count, ok := intp.Stack[len(intp.Stack)-1].(Integer)
	if !ok {
		return intp.e(eTypecheck, "getinterval: invalid count")
	} else if count < 0 || count > Integer(n)-index {
		return intp.e(eRangecheck, "getinterval: count %d out of bounds", count)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-3]
	var res Object
	switch obj := obj.(type) {
	case Array:
		res = obj[index : index+count]
	case String:
		res = obj[index : index+count]
	}
	intp.Stack = append(intp.Stack, res)
	return nil
}

func bIf(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "if: not enough arguments")
	}
	cond, ok := intp.Stack[len(intp.Stack)-2].(Boolean)
	if !ok {
		return intp.e(eTypecheck, "if: invalid condition")
	}
	proc := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	if cond {
		return intp.executeOne(proc, true)
	}
	return nil
}

func bIfelse(intp *Interpreter) error {
	if len(intp.Stack) < 3 {
		return intp.e(eStackunderflow, "ifelse: not enough arguments")
	}
	cond, ok := intp.Stack[len(intp.Stack)-3].(Boolean)
	if !ok {
		return intp.e(eTypecheck, "ifelse: invalid condition")
	}
	proc1 := intp.Stack[len(intp.Stack)-2]
	proc2 := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-3]
	if cond {
		return intp.executeOne(proc1, true)
	} else {
		return intp.executeOne(proc2, true)
	}
}

func bIndex(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "index: not enough arguments")
	}
	index, ok := intp.Stack[len(intp.Stack)-1].(Integer)
	if !ok {
		return intp.e(eTypecheck, "index: invalid argument")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	if index < 0 || index >= Integer(len(intp.Stack)) {
		return intp.e(eRangecheck, "index: index out of bounds")
	}
	intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-int(index)-1])
	return nil
}

func bInternaldict(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "internaldict: not enough arguments")
	}
	index, ok := intp.Stack[len(intp.Stack)-1].(Integer)
	if !ok {
		return intp.e(eTypecheck, "internaldict: invalid argument")
	}
	if index != 1183615869 {
		return intp.e(eInvalidaccess, "internaldict: wrong passcode")
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-1], intp.InternalDict)
	return nil
}

func bKnown(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "known: not enough arguments")
	}
	d, ok := intp.Stack[len(intp.Stack)-2].(Dict)
	if !ok {
		return intp.e(eTypecheck, "known: invalid argument")
	}
	name, ok := intp.Stack[len(intp.Stack)-1].(Name)
	if !ok {
		return intp.e(eTypecheck, "known: invalid argument")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	_, ok = d[name]
	intp.Stack = append(intp.Stack, Boolean(ok))
	return nil
}

func bLength(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "length: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	var res int
	switch obj := obj.(type) {
	case Array:
		res = len(obj)
	case Procedure:
		res = len(obj)
	case Dict:
		res = len(obj)
	case String:
		res = len(obj)
	case Name:
		res = len(obj)
	case Operator:
		res = len(obj)
	default:
		return intp.e(eTypecheck, "length: invalid argument type %T", obj)
	}
	intp.Stack = append(intp.Stack, Integer(res))
	return nil
}

func bLoad(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "load: not enough arguments")
	}
	name, ok := intp.Stack[len(intp.Stack)-1].(Name)
	if !ok {
		return intp.e(eTypecheck, "load: invalid argument")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	val, err := intp.load(name)
	if err != nil {
		return err
	}
	intp.Stack = append(intp.Stack, val)
	return nil
}

func bLoop(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "loop: not enough arguments")
	}
	proc := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	for {
		err := intp.executeOne(proc, true)
		if err == errExit {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func bMark(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, mark{})
	return nil
}

func bMatrix(intp *Interpreter) error {
	m := Array{Integer(1), Integer(0), Integer(0), Integer(1), Integer(0), Integer(0)}
	intp.Stack = append(intp.Stack, m)
	return nil
}

func bMaxlength(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "maxlength: not enough arguments")
	}
	dict, ok := intp.Stack[len(intp.Stack)-1].(Dict)
	if !ok {
		return intp.e(eTypecheck, "maxlength: invalid argument")
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-1], Integer(len(dict)+1))
	return nil
}

func bMul(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "mul: not enough arguments")
	}
	ar, aIsReal := intp.Stack[len(intp.Stack)-2].(Real)
	ai, aIsInt := intp.Stack[len(intp.Stack)-2].(Integer)
	br, bIsReal := intp.Stack[len(intp.Stack)-1].(Real)
	bi, bIsInt := intp.Stack[len(intp.Stack)-1].(Integer)
	if !(aIsReal || aIsInt) || !(bIsReal || bIsInt) {
		return intp.e(eTypecheck, "mul: needs numbers as arguments")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	if aIsReal || bIsReal {
		if aIsInt {
			ar = Real(ai)
		}
		if bIsInt {
			br = Real(bi)
		}
		intp.Stack = append(intp.Stack, ar*br)
	} else {
		ci := ai * bi
		// check for integer overflow
		if ai != 0 && ci/ai != bi {
			intp.Stack = append(intp.Stack, Real(ai)*Real(bi))
		} else {
			intp.Stack = append(intp.Stack, ci)
		}
	}
	return nil
}

func bNe(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "ne: not enough arguments")
	}
	a := intp.Stack[len(intp.Stack)-2]
	b := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	isEqual, err := equal(a, b)
	if err != nil {
		return err
	}

	intp.Stack = append(intp.Stack, Boolean(!isEqual))
	return nil
}

func bNoaccess(intp *Interpreter) error {
	// not implemented
	return nil
}

func bNot(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "not: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-1]
	switch obj := obj.(type) {
	case Boolean:
		intp.Stack[len(intp.Stack)-1] = !obj
	case Integer:
		intp.Stack[len(intp.Stack)-1] = ^obj
	default:
		return intp.e(eTypecheck, "not: invalid argument type %T", obj)
	}
	return nil
}

func bOr(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "or: not enough arguments")
	}
	a := intp.Stack[len(intp.Stack)-2]
	b := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	switch a := a.(type) {
	case Boolean:
		b, ok := b.(Boolean)
		if !ok {
			return intp.e(eTypecheck, "or: mismatched argument types")
		}
		intp.Stack = append(intp.Stack, a || b)
	case Integer:
		b, ok := b.(Integer)
		if !ok {
			return intp.e(eTypecheck, "or: mismatched argument types")
		}
		intp.Stack = append(intp.Stack, a|b)
	default:
		return intp.e(eTypecheck, "or: invalid argument type %T", a)
	}
	return nil
}

func bPop(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "pop: not enough arguments")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	return nil
}

func bPut(intp *Interpreter) error {
	if len(intp.Stack) < 3 {
		return intp.e(eStackunderflow, "put: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-3]
	sel := intp.Stack[len(intp.Stack)-2]
	value := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-3]
	switch obj := obj.(type) {
	case Array:
		index, ok := sel.(Integer)
		if !ok {
			return intp.e(eTypecheck, "put: invalid index")
		}
		if index < 0 || index >= Integer(len(obj)) {
			return intp.e(eRangecheck, "put: index %d out of range", index)
		}
		obj[index] = value
	case Procedure:
		index, ok := sel.(Integer)
		if !ok {
			return intp.e(eTypecheck, "put: invalid index")
		}
		if index < 0 || index >= Integer(len(obj)) {
			return intp.e(eRangecheck, "put: index %d out of range", index)
		}
		obj[index] = value
	case Dict:
		key, ok := sel.(Name)
		if !ok {
			return intp.e(eTypecheck, "put: invalid dict key")
		}
		obj[key] = value
	case String:
		index, ok := sel.(Integer)
		if !ok {
			return intp.e(eTypecheck, "put: invalid index")
		}
		if index < 0 || index >= Integer(len(obj)) {
			return intp.e(eRangecheck, "put: index %d out of range", index)
		}
		c, ok := value.(Integer)
		if !ok {
			return intp.e(eTypecheck, "put: invalid value")
		}
		obj[index] = byte(c)
	default:
		return intp.e(eTypecheck, "put: invalid argument type %T", obj)
	}
	return nil
}

func bPutinterval(intp *Interpreter) error {
	if len(intp.Stack) < 3 {
		return intp.e(eStackunderflow, "putinterval: not enough arguments")
	}
	dst := intp.Stack[len(intp.Stack)-3]
	index, ok := intp.Stack[len(intp.Stack)-2].(Integer)
	if !ok {
		return intp.e(eTypecheck, "putinterval: invalid index")
	} else if index < 0 {
		return intp.e(eRangecheck, "putinterval: index out of range")
	}
	src := intp.Stack[len(intp.Stack)-1]

	switch dst := dst.(type) {
	case Array:
		src, ok := src.(Array)
		if !ok {
			return intp.e(eTypecheck, "putinterval: mismatched argument types")
		}
		if int(index)+len(src) > len(dst) {
			return intp.e(eRangecheck, "putinterval: index out of range")
		}
		copy(dst[index:], src)
	case String:
		src, ok := src.(String)
		if !ok {
			return intp.e(eTypecheck, "putinterval: mismatched argument types")
		}
		if int(index)+len(src) > len(dst) {
			return intp.e(eRangecheck, "putinterval: index out of range")
		}
		copy(dst[index:], src)
	default:
		return intp.e(eTypecheck, "putinterval: invalid argument type %T", dst)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-3]
	return nil
}

func bReadonly(intp *Interpreter) error {
	// not implemented
	return nil
}

func bReadstring(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "readstring: not enough arguments")
	}
	buf, ok := intp.Stack[len(intp.Stack)-1].(String)
	if !ok {
		return intp.e(eTypecheck, "readstring: invalid argument")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	s := intp.scanners[len(intp.scanners)-1]
	_, err := s.Next()
	if err != nil && err != io.EOF {
		return err
	}
	n, err := s.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	intp.Stack = append(intp.Stack, buf[:n])
	intp.Stack = append(intp.Stack, Boolean(n == len(buf)))
	return nil
}

func bRepeat(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "repeat: not enough arguments")
	}
	count, ok := intp.Stack[len(intp.Stack)-2].(Integer)
	if !ok {
		return intp.e(eTypecheck, "repeat: invalid argument")
	} else if count < 0 {
		return intp.e(eRangecheck, "repeat: negative count")
	}
	proc, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
	if !ok {
		return intp.e(eTypecheck, "repeat: invalid argument")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	for i := Integer(0); i < count; i++ {
		err := intp.executeOne(proc, true)
		if err == errStop {
			break
		} else if err != nil {
			return err
		}
	}
	return nil
}

func bRoll(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "roll: not enough arguments")
	}
	n, ok := intp.Stack[len(intp.Stack)-2].(Integer)
	if !ok {
		return intp.e(eTypecheck, "roll: invalid argument")
	}
	if n < 0 || n > Integer(len(intp.Stack)-2) {
		return intp.e(eRangecheck, "roll: length %d out of bounds", n)
	}
	j, ok := intp.Stack[len(intp.Stack)-1].(Integer)
	if !ok {
		return intp.e(eTypecheck, "roll: count %d out of bounds", j)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	if n == 0 {
		return nil
	}
	j %= n
	if j < 0 {
		j += n
	}

	// Remove j elements from the top of the stack, and insert these
	// between the intp.Stack[len(intp.Stack)-n:] and the rest of the
	// stack.
	ji := int(j)
	ni := int(n)
	data := intp.Stack[len(intp.Stack)-ni:]
	tmp := make([]Object, j)
	copy(tmp, data[ni-ji:])
	copy(data[ji:], data[:ni-ji])
	copy(data, tmp)

	return nil
}

func bStop(intp *Interpreter) error {
	return errStop
}

func bString(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "string: not enough arguments")
	}
	size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
	if !ok {
		return intp.e(eTypecheck, "string: invalid argument")
	} else if size < 0 {
		return intp.e(eRangecheck, "string: invalid size %d", size)
	} else if size > maxStringSize {
		return intp.e(eLimitcheck, "string: invalid size %d", size)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	intp.Stack = append(intp.Stack, make(String, size))
	return nil
}

func bSub(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "sub: not enough arguments")
	}
	ar, aIsReal := intp.Stack[len(intp.Stack)-2].(Real)
	ai, aIsInt := intp.Stack[len(intp.Stack)-2].(Integer)
	br, bIsReal := intp.Stack[len(intp.Stack)-1].(Real)
	bi, bIsInt := intp.Stack[len(intp.Stack)-1].(Integer)
	if !(aIsReal || aIsInt) || !(bIsReal || bIsInt) {
		return intp.e(eTypecheck, "sub: needs numbers as arguments")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	if aIsReal || bIsReal {
		if aIsInt {
			ar = Real(ai)
		}
		if bIsInt {
			br = Real(bi)
		}
		intp.Stack = append(intp.Stack, ar-br)
	} else {
		ci := ai - bi
		// check for integer overflow
		if (ai < 0 && bi > 0 && ci >= 0) || (ai > 0 && bi < 0 && ci <= 0) {
			intp.Stack = append(intp.Stack, Real(ai)-Real(bi))
		} else {
			intp.Stack = append(intp.Stack, ci)
		}
	}
	return nil
}

func bType(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "type: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-1]
	var tp Name
	switch obj.(type) {
	case Array, Procedure:
		tp = "arraytype"
	case Boolean:
		tp = "booleantype"
	case Dict:
		tp = "dicttype"
	case nil: // TODO(voss)
		tp = "filetype"
	// fonttype
	// gstatetype (LanguageLevel 2)
	case Integer:
		tp = "integertype"
	case Name, Operator:
		tp = "nametype"
	// tp = "nulltype"
	case builtin:
		tp = "operatortype"
	// tp = "packedarraytype" (LanguageLevel 2)
	case Real:
		tp = "realtype"
	// tp = "savetype"
	case String:
		tp = "stringtype"
	case mark:
		tp = "marktype"
	default:
		return intp.e(eTypecheck, "type: not implemented for %T", obj)
	}
	intp.Stack = append(intp.Stack, tp)
	return nil
}

func bWhere(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "where: not enough arguments")
	}
	key, ok := intp.Stack[len(intp.Stack)-1].(Name)
	if !ok {
		return intp.e(eTypecheck, "where: invalid argument")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	for j := len(intp.DictStack) - 1; j >= 0; j-- {
		d := intp.DictStack[j]
		if _, ok := d[key]; ok {
			intp.Stack = append(intp.Stack, d, Boolean(true))
			return nil
		}
	}
	intp.Stack = append(intp.Stack, Boolean(false))
	return nil
}

func equal(a, b Object) (bool, error) {
	_, aIsDict := a.(Dict)
	_, bIsDict := b.(Dict)
	if aIsDict && bIsDict {
		return isSameDict(a.(Dict), b.(Dict)), nil
	}

	normalize := func(obj Object) (Object, error) {
		switch obj := obj.(type) {
		case Real:
			return float64(obj), nil
		case Integer:
			return float64(obj), nil
		case String:
			return string(obj), nil
		case Name:
			return string(obj), nil
		default:
			return nil, &postScriptError{eTypecheck, fmt.Sprintf("equality not implemented for %T", obj)}
		}
	}
	a, err := normalize(a)
	if err != nil {
		return false, err
	}
	b, err = normalize(b)
	if err != nil {
		return false, err
	}
	return a == b, nil
}

func (intp *Interpreter) bindProc(proc Procedure) {
	for i, elem := range proc {
		switch obj := elem.(type) {
		case Name:
			val, err := intp.load(obj)
			if err != nil {
				continue
			}
			_, ok := val.(builtin)
			if ok {
				proc[i] = val
			}
		case Operator:
			val, err := intp.load(obj)
			if err != nil {
				continue
			}
			_, ok := val.(builtin)
			if ok {
				proc[i] = val
			}
		case Procedure:
			// be careful to avoid infinite loops
			proc[i] = nil
			intp.bindProc(obj)
			proc[i] = obj
		}
	}
}

// don't look!
func isSameDict(a, b Dict) bool {
	if len(a) != len(b) {
		return false
	}

	testKey := Name(rand.Text())

	if _, inB := b[testKey]; inB {
		return false
	}

	a[testKey] = true
	_, isSame := b[testKey]
	delete(a, testKey)
	return isSame
}

// atan: num den atan angle
func bAtan(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "atan: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-2]
	den := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	var numVal, denVal float64

	// Convert num to float64
	switch n := num.(type) {
	case Integer:
		numVal = float64(n)
	case Real:
		numVal = float64(n)
	default:
		return intp.e(eTypecheck, "atan: num must be a number")
	}

	// Convert den to float64
	switch d := den.(type) {
	case Integer:
		denVal = float64(d)
	case Real:
		denVal = float64(d)
	default:
		return intp.e(eTypecheck, "atan: den must be a number")
	}

	// Check for undefinedresult (both zero)
	if numVal == 0 && denVal == 0 {
		return intp.e(eUndefinedresult, "atan: both arguments are zero")
	}

	// Calculate atan2 and convert from radians to degrees
	radians := math.Atan2(numVal, denVal)
	degrees := radians * 180.0 / math.Pi

	// Ensure result is in [0, 360) range
	if degrees < 0 {
		degrees += 360
	}

	intp.Stack = append(intp.Stack, Real(degrees))
	return nil
}

// bitshift: int1 shift bitshift int2
func bBitshift(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "bitshift: not enough arguments")
	}

	int1, ok1 := intp.Stack[len(intp.Stack)-2].(Integer)
	shift, ok2 := intp.Stack[len(intp.Stack)-1].(Integer)

	if !ok1 || !ok2 {
		return intp.e(eTypecheck, "bitshift: both arguments must be integers")
	}

	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	var result Integer
	if shift >= 0 {
		// Left shift
		result = int1 << uint(shift)
	} else {
		// Right shift
		result = int1 >> uint(-shift)
	}

	intp.Stack = append(intp.Stack, result)
	return nil
}

// ceiling: num1 ceiling num2
func bCeiling(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "ceiling: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch n := num.(type) {
	case Integer:
		intp.Stack = append(intp.Stack, n)
	case Real:
		intp.Stack = append(intp.Stack, Real(math.Ceil(float64(n))))
	default:
		return intp.e(eTypecheck, "ceiling: argument must be a number")
	}

	return nil
}

// cos: angle cos real
func bCos(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "cos: not enough arguments")
	}

	angle := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	var angleVal float64
	switch a := angle.(type) {
	case Integer:
		angleVal = float64(a)
	case Real:
		angleVal = float64(a)
	default:
		return intp.e(eTypecheck, "cos: argument must be a number")
	}

	// Convert degrees to radians
	radians := angleVal * math.Pi / 180.0
	result := math.Cos(radians)

	intp.Stack = append(intp.Stack, Real(result))
	return nil
}

// cvi: num cvi int OR string cvi int
func bCvi(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "cvi: not enough arguments")
	}

	operand := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch op := operand.(type) {
	case Integer:
		intp.Stack = append(intp.Stack, op)
	case Real:
		// Truncate toward zero
		truncated := math.Trunc(float64(op))
		// Check for overflow
		if truncated > math.MaxInt64 || truncated < math.MinInt64 {
			return intp.e(eRangecheck, "cvi: number too large to convert to integer")
		}
		intp.Stack = append(intp.Stack, Integer(truncated))
	case String:
		// Parse string as number - simplified implementation
		str := string(op)
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			truncated := math.Trunc(val)
			if truncated > math.MaxInt64 || truncated < math.MinInt64 {
				return intp.e(eRangecheck, "cvi: number too large to convert to integer")
			}
			intp.Stack = append(intp.Stack, Integer(truncated))
		} else if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			intp.Stack = append(intp.Stack, Integer(val))
		} else {
			return intp.e(eSyntaxerror, "cvi: invalid number in string")
		}
	default:
		return intp.e(eTypecheck, "cvi: invalid argument type")
	}

	return nil
}

// cvr: num cvr real OR string cvr real
func bCvr(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "cvr: not enough arguments")
	}

	operand := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch op := operand.(type) {
	case Integer:
		intp.Stack = append(intp.Stack, Real(op))
	case Real:
		intp.Stack = append(intp.Stack, op)
	case String:
		// Parse string as number - simplified implementation
		str := string(op)
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			intp.Stack = append(intp.Stack, Real(val))
		} else {
			return intp.e(eSyntaxerror, "cvr: invalid number in string")
		}
	default:
		return intp.e(eTypecheck, "cvr: invalid argument type")
	}

	return nil
}

// div: num1 num2 div quotient
func bDiv(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "div: not enough arguments")
	}

	num1 := intp.Stack[len(intp.Stack)-2]
	num2 := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	var val1, val2 float64

	// Convert num1 to float64
	switch n := num1.(type) {
	case Integer:
		val1 = float64(n)
	case Real:
		val1 = float64(n)
	default:
		return intp.e(eTypecheck, "div: first argument must be a number")
	}

	// Convert num2 to float64
	switch n := num2.(type) {
	case Integer:
		val2 = float64(n)
	case Real:
		val2 = float64(n)
	default:
		return intp.e(eTypecheck, "div: second argument must be a number")
	}

	if val2 == 0 {
		return intp.e(eUndefinedresult, "div: division by zero")
	}

	result := val1 / val2
	intp.Stack = append(intp.Stack, Real(result))
	return nil
}

// exp: base exponent exp real
func bExp(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "exp: not enough arguments")
	}

	base := intp.Stack[len(intp.Stack)-2]
	exponent := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	var baseVal, expVal float64

	// Convert base to float64
	switch b := base.(type) {
	case Integer:
		baseVal = float64(b)
	case Real:
		baseVal = float64(b)
	default:
		return intp.e(eTypecheck, "exp: base must be a number")
	}

	// Convert exponent to float64
	switch e := exponent.(type) {
	case Integer:
		expVal = float64(e)
	case Real:
		expVal = float64(e)
	default:
		return intp.e(eTypecheck, "exp: exponent must be a number")
	}

	// Check for meaningful result with fractional exponent
	if expVal != math.Trunc(expVal) && baseVal < 0 {
		return intp.e(eUndefinedresult, "exp: negative base with fractional exponent")
	}

	result := math.Pow(baseVal, expVal)

	// Check for infinite or NaN result
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return intp.e(eUndefinedresult, "exp: undefined result")
	}

	intp.Stack = append(intp.Stack, Real(result))
	return nil
}

// floor: num1 floor num2
func bFloor(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "floor: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch n := num.(type) {
	case Integer:
		intp.Stack = append(intp.Stack, n)
	case Real:
		intp.Stack = append(intp.Stack, Real(math.Floor(float64(n))))
	default:
		return intp.e(eTypecheck, "floor: argument must be a number")
	}

	return nil
}

// ge: num1 num2 ge bool OR string1 string2 ge bool
func bGe(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "ge: not enough arguments")
	}

	obj1 := intp.Stack[len(intp.Stack)-2]
	obj2 := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	// Handle number comparison
	if isNumber(obj1) && isNumber(obj2) {
		val1 := numberToFloat(obj1)
		val2 := numberToFloat(obj2)
		intp.Stack = append(intp.Stack, Boolean(val1 >= val2))
		return nil
	}

	// Handle string comparison
	if str1, ok1 := obj1.(String); ok1 {
		if str2, ok2 := obj2.(String); ok2 {
			// Lexical comparison
			result := string(str1) >= string(str2)
			intp.Stack = append(intp.Stack, Boolean(result))
			return nil
		}
	}

	return intp.e(eTypecheck, "ge: invalid argument types")
}

// gt: num1 num2 gt bool OR string1 string2 gt bool
func bGt(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "gt: not enough arguments")
	}

	obj1 := intp.Stack[len(intp.Stack)-2]
	obj2 := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	// Handle number comparison
	if isNumber(obj1) && isNumber(obj2) {
		val1 := numberToFloat(obj1)
		val2 := numberToFloat(obj2)
		intp.Stack = append(intp.Stack, Boolean(val1 > val2))
		return nil
	}

	// Handle string comparison
	if str1, ok1 := obj1.(String); ok1 {
		if str2, ok2 := obj2.(String); ok2 {
			// Lexical comparison
			result := string(str1) > string(str2)
			intp.Stack = append(intp.Stack, Boolean(result))
			return nil
		}
	}

	return intp.e(eTypecheck, "gt: invalid argument types")
}

// idiv: int1 int2 idiv quotient
func bIdiv(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "idiv: not enough arguments")
	}

	int1, ok1 := intp.Stack[len(intp.Stack)-2].(Integer)
	int2, ok2 := intp.Stack[len(intp.Stack)-1].(Integer)

	if !ok1 || !ok2 {
		return intp.e(eTypecheck, "idiv: both arguments must be integers")
	}

	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	if int2 == 0 {
		return intp.e(eUndefinedresult, "idiv: division by zero")
	}

	// Integer division with truncation toward zero
	result := int1 / int2
	intp.Stack = append(intp.Stack, result)
	return nil
}

// le: num1 num2 le bool OR string1 string2 le bool
func bLe(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "le: not enough arguments")
	}

	obj1 := intp.Stack[len(intp.Stack)-2]
	obj2 := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	// Handle number comparison
	if isNumber(obj1) && isNumber(obj2) {
		val1 := numberToFloat(obj1)
		val2 := numberToFloat(obj2)
		intp.Stack = append(intp.Stack, Boolean(val1 <= val2))
		return nil
	}

	// Handle string comparison
	if str1, ok1 := obj1.(String); ok1 {
		if str2, ok2 := obj2.(String); ok2 {
			// Lexical comparison
			result := string(str1) <= string(str2)
			intp.Stack = append(intp.Stack, Boolean(result))
			return nil
		}
	}

	return intp.e(eTypecheck, "le: invalid argument types")
}

// ln: num ln real
func bLn(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "ln: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	var val float64
	switch n := num.(type) {
	case Integer:
		val = float64(n)
	case Real:
		val = float64(n)
	default:
		return intp.e(eTypecheck, "ln: argument must be a number")
	}

	if val <= 0 {
		return intp.e(eRangecheck, "ln: argument must be positive")
	}

	result := math.Log(val)
	intp.Stack = append(intp.Stack, Real(result))
	return nil
}

// log: num log real
func bLog(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "log: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	var val float64
	switch n := num.(type) {
	case Integer:
		val = float64(n)
	case Real:
		val = float64(n)
	default:
		return intp.e(eTypecheck, "log: argument must be a number")
	}

	if val <= 0 {
		return intp.e(eRangecheck, "log: argument must be positive")
	}

	result := math.Log10(val)
	intp.Stack = append(intp.Stack, Real(result))
	return nil
}

// lt: num1 num2 lt bool OR string1 string2 lt bool
func bLt(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "lt: not enough arguments")
	}

	obj1 := intp.Stack[len(intp.Stack)-2]
	obj2 := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	// Handle number comparison
	if isNumber(obj1) && isNumber(obj2) {
		val1 := numberToFloat(obj1)
		val2 := numberToFloat(obj2)
		intp.Stack = append(intp.Stack, Boolean(val1 < val2))
		return nil
	}

	// Handle string comparison
	if str1, ok1 := obj1.(String); ok1 {
		if str2, ok2 := obj2.(String); ok2 {
			// Lexical comparison
			result := string(str1) < string(str2)
			intp.Stack = append(intp.Stack, Boolean(result))
			return nil
		}
	}

	return intp.e(eTypecheck, "lt: invalid argument types")
}

// mod: int1 int2 mod remainder
func bMod(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "mod: not enough arguments")
	}

	int1, ok1 := intp.Stack[len(intp.Stack)-2].(Integer)
	int2, ok2 := intp.Stack[len(intp.Stack)-1].(Integer)

	if !ok1 || !ok2 {
		return intp.e(eTypecheck, "mod: both arguments must be integers")
	}

	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	if int2 == 0 {
		return intp.e(eUndefinedresult, "mod: modulo by zero")
	}

	// Remainder operation with sign of dividend
	result := int1 % int2
	intp.Stack = append(intp.Stack, result)
	return nil
}

// neg: num1 neg num2
func bNeg(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "neg: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch n := num.(type) {
	case Integer:
		if n == math.MinInt64 {
			// Handle most negative integer overflow
			intp.Stack = append(intp.Stack, Real(-float64(n)))
		} else {
			intp.Stack = append(intp.Stack, -n)
		}
	case Real:
		intp.Stack = append(intp.Stack, -n)
	default:
		return intp.e(eTypecheck, "neg: argument must be a number")
	}

	return nil
}

// round: num1 round num2
func bRound(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "round: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch n := num.(type) {
	case Integer:
		intp.Stack = append(intp.Stack, n)
	case Real:
		// PostScript spec: if equally close to two integers, return the greater
		val := float64(n)
		if val >= 0 {
			intp.Stack = append(intp.Stack, Real(math.Floor(val+0.5)))
		} else {
			// For negative numbers: -6.5 should round to -6 (the greater)
			// Check if it's exactly halfway
			floor := math.Floor(val)
			ceil := math.Ceil(val)
			if math.Abs(val-floor) == math.Abs(val-ceil) {
				// Exactly halfway, return the greater (closer to zero)
				intp.Stack = append(intp.Stack, Real(ceil))
			} else {
				// Normal rounding
				intp.Stack = append(intp.Stack, Real(math.Round(val)))
			}
		}
	default:
		return intp.e(eTypecheck, "round: argument must be a number")
	}

	return nil
}

// sin: angle sin real
func bSin(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "sin: not enough arguments")
	}

	angle := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	var angleVal float64
	switch a := angle.(type) {
	case Integer:
		angleVal = float64(a)
	case Real:
		angleVal = float64(a)
	default:
		return intp.e(eTypecheck, "sin: argument must be a number")
	}

	// Convert degrees to radians
	radians := angleVal * math.Pi / 180.0
	result := math.Sin(radians)

	intp.Stack = append(intp.Stack, Real(result))
	return nil
}

// sqrt: num sqrt real
func bSqrt(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "sqrt: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	var val float64
	switch n := num.(type) {
	case Integer:
		val = float64(n)
	case Real:
		val = float64(n)
	default:
		return intp.e(eTypecheck, "sqrt: argument must be a number")
	}

	if val < 0 {
		return intp.e(eRangecheck, "sqrt: argument must be non-negative")
	}

	result := math.Sqrt(val)
	intp.Stack = append(intp.Stack, Real(result))
	return nil
}

// truncate: num1 truncate num2
func bTruncate(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "truncate: not enough arguments")
	}

	num := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	switch n := num.(type) {
	case Integer:
		intp.Stack = append(intp.Stack, n)
	case Real:
		intp.Stack = append(intp.Stack, Real(math.Trunc(float64(n))))
	default:
		return intp.e(eTypecheck, "truncate: argument must be a number")
	}

	return nil
}

// xor: bool1 bool2 xor bool3 OR int1 int2 xor int3
func bXor(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "xor: not enough arguments")
	}

	obj1 := intp.Stack[len(intp.Stack)-2]
	obj2 := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]

	// Handle boolean XOR
	if bool1, ok1 := obj1.(Boolean); ok1 {
		if bool2, ok2 := obj2.(Boolean); ok2 {
			result := bool(bool1) != bool(bool2)
			intp.Stack = append(intp.Stack, Boolean(result))
			return nil
		}
	}

	// Handle integer bitwise XOR
	if int1, ok1 := obj1.(Integer); ok1 {
		if int2, ok2 := obj2.(Integer); ok2 {
			result := int1 ^ int2
			intp.Stack = append(intp.Stack, result)
			return nil
		}
	}

	return intp.e(eTypecheck, "xor: arguments must be both booleans or both integers")
}

// Helper functions
func isNumber(obj Object) bool {
	_, isInt := obj.(Integer)
	_, isReal := obj.(Real)
	return isInt || isReal
}

func numberToFloat(obj Object) float64 {
	switch n := obj.(type) {
	case Integer:
		return float64(n)
	case Real:
		return float64(n)
	default:
		return 0 // Should not happen if isNumber was checked first
	}
}

var errExit = errors.New("exit")
var errStop = errors.New("stop")
