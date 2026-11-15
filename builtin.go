// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"bytes"
	"cmp"
	"errors"
	"io"
	"maps"
	"math"
	"reflect"
	"slices"
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
	isoLatin1Encoding := make(Array, 256)
	for i, name := range psenc.ISOLatin1Encoding {
		isoLatin1Encoding[i] = Name(name)
	}

	systemDict := Dict{
		"<<":                &builtin{fn: bDictStart},
		"=":                 &builtin{fn: bPrintObject},
		"==":                &builtin{fn: bPrintObject},
		">>":                &builtin{fn: bDictEnd},
		"[":                 &builtin{fn: bListStart},
		"]":                 &builtin{fn: bListEnd},
		"abs":               &builtin{fn: bAbs},
		"add":               &builtin{fn: bAdd},
		"aload":             &builtin{fn: bAload},
		"anchorsearch":      &builtin{fn: bAnchorsearch},
		"and":               &builtin{fn: bAnd},
		"array":             &builtin{fn: bArray},
		"astore":            &builtin{fn: bAstore},
		"atan":              &builtin{fn: bAtan},
		"begin":             &builtin{fn: bBegin},
		"bind":              &builtin{fn: bBind},
		"bitshift":          &builtin{fn: bBitshift},
		"ceiling":           &builtin{fn: bCeiling},
		"clear":             &builtin{fn: bClear},
		"cleartomark":       &builtin{fn: bCleartomark},
		"closefile":         &builtin{fn: bClosefile},
		"concatmatrix":      &builtin{fn: bConcatmatrix},
		"copy":              &builtin{fn: bCopy},
		"cos":               &builtin{fn: bCos},
		"count":             &builtin{fn: bCount},
		"countdictstack":    &builtin{fn: bCountdictstack},
		"counttomark":       &builtin{fn: bCounttomark},
		"currentdict":       &builtin{fn: bCurrentdict},
		"currentfile":       &builtin{fn: bCurrentfile},
		"currentglobal":     &builtin{fn: bCurrentglobal},
		"currentpacking":    &builtin{fn: bCurrentpacking},
		"cvi":               &builtin{fn: bCvi},
		"cvlit":             &builtin{fn: bCvlit},
		"cvn":               &builtin{fn: bCvn},
		"cvr":               &builtin{fn: bCvr},
		"cvrs":              &builtin{fn: bCvrs},
		"cvs":               &builtin{fn: bCvs},
		"cvx":               &builtin{fn: bCvx},
		"def":               &builtin{fn: bDef},
		"definefont":        &builtin{fn: bDefinefont},
		"defineresource":    &builtin{fn: bDefineresource},
		"dict":              &builtin{fn: bDict},
		"div":               &builtin{fn: bDiv},
		"dup":               &builtin{fn: bDup},
		"eexec":             &builtin{fn: eexec},
		"end":               &builtin{fn: bEnd},
		"eq":                &builtin{fn: bEq},
		"errordict":         errorDict,
		"exch":              &builtin{fn: bExch},
		"exec":              &builtin{fn: bExec},
		"executeonly":       &builtin{fn: bExecuteonly},
		"exit":              &builtin{fn: bExit},
		"exp":               &builtin{fn: bExp},
		"false":             Boolean(false),
		"findfont":          &builtin{fn: bFindfont},
		"findresource":      &builtin{fn: bFindresource},
		"floor":             &builtin{fn: bFloor},
		"flush":             &builtin{fn: bFlush},
		"FontDirectory":     FontDirectory,
		"for":               &builtin{fn: bFor},
		"forall":            &builtin{fn: bForall},
		"ge":                &builtin{fn: bGe},
		"get":               &builtin{fn: bGet},
		"getinterval":       &builtin{fn: bGetinterval},
		"gt":                &builtin{fn: bGt},
		"identmatrix":       &builtin{fn: bIdentmatrix},
		"idiv":              &builtin{fn: bIdiv},
		"if":                &builtin{fn: bIf},
		"ifelse":            &builtin{fn: bIfelse},
		"index":             &builtin{fn: bIndex},
		"internaldict":      &builtin{fn: bInternaldict},
		"ISOLatin1Encoding": isoLatin1Encoding,
		"known":             &builtin{fn: bKnown},
		"languagelevel":     Integer(3),
		"le":                &builtin{fn: bLe},
		"length":            &builtin{fn: bLength},
		"ln":                &builtin{fn: bLn},
		"load":              &builtin{fn: bLoad},
		"log":               &builtin{fn: bLog},
		"loop":              &builtin{fn: bLoop},
		"lt":                &builtin{fn: bLt},
		"mark":              &builtin{fn: bMark},
		"matrix":            &builtin{fn: bMatrix},
		"maxlength":         &builtin{fn: bMaxlength},
		"mod":               &builtin{fn: bMod},
		"mul":               &builtin{fn: bMul},
		"ne":                &builtin{fn: bNe},
		"neg":               &builtin{fn: bNeg},
		"noaccess":          &builtin{fn: bNoaccess},
		"not":               &builtin{fn: bNot},
		"null":              nil,
		"or":                &builtin{fn: bOr},
		"pop":               &builtin{fn: bPop},
		"print":             &builtin{fn: bPrint},
		"put":               &builtin{fn: bPut},
		"putinterval":       &builtin{fn: bPutinterval},
		"rcheck":            &builtin{fn: bRcheck},
		"readonly":          &builtin{fn: bReadonly},
		"readstring":        &builtin{fn: bReadstring},
		"repeat":            &builtin{fn: bRepeat},
		"roll":              &builtin{fn: bRoll},
		"round":             &builtin{fn: bRound},
		"search":            &builtin{fn: bSearch},
		"setglobal":         &builtin{fn: bSetglobal},
		"setpacking":        &builtin{fn: bSetpacking},
		"sin":               &builtin{fn: bSin},
		"sqrt":              &builtin{fn: bSqrt},
		"StandardEncoding":  standardEncoding,
		"stop":              &builtin{fn: bStop},
		"store":             &builtin{fn: bStore},
		"string":            &builtin{fn: bString},
		"sub":               &builtin{fn: bSub},
		"true":              Boolean(true),
		"truncate":          &builtin{fn: bTruncate},
		"type":              &builtin{fn: bType},
		"undef":             &builtin{fn: bUndef},
		"userdict":          userDict,
		"version":           String("3010"),
		"wcheck":            &builtin{fn: bWcheck},
		"where":             &builtin{fn: bWhere},
		"xcheck":            &builtin{fn: bXcheck},
		"xor":               &builtin{fn: bXor},
	}
	for name, val := range systemDict {
		if b, ok := val.(*builtin); ok {
			b.name = string(name)
		}
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
			if err := intp.chargeWork(n - i); err != nil {
				return err
			}
			if err := intp.charge(size * objectSize); err != nil {
				return err
			}
			a := make(Array, size)
			copy(a, intp.Stack[i+1:])
			intp.Stack = append(intp.Stack[:i], a)
			return nil
		}
	}
	if err := intp.chargeWork(n); err != nil {
		return err
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
	if err := intp.chargeWork(n - max(markPos, 0)); err != nil {
		return err
	}
	if markPos < 0 {
		return intp.e(eUnmatchedmark, ">>: missing '<<'")
	} else if (n-markPos)%2 != 1 {
		return intp.e(eRangecheck, "dict literal: odd length")
	}
	size := (n - markPos - 1) / 2
	if err := intp.charge(size * dictEntrySize); err != nil {
		return err
	}
	d := make(Dict, size)
	for i := markPos + 1; i < n; i += 2 {
		name, err := intp.dictKey(intp.Stack[i], ">>")
		if err != nil {
			return err
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
		return intp.pushReal(ar+br, "add")
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

// aload: array aload any0 … anyn-1 array
func bAload(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "aload: not enough arguments")
	}
	arr := intp.Stack[n-1]
	elems, ok := arrayElems(arr)
	if !ok {
		return intp.e(eTypecheck, "aload: needs an array")
	}
	if n+len(elems) > intp.stackLimit() {
		return intp.e(eStackoverflow, "aload: operand stack overflow")
	}
	if err := intp.chargeWork(len(elems)); err != nil {
		return err
	}
	intp.Stack = append(intp.Stack[:n-1], elems...)
	intp.Stack = append(intp.Stack, arr)
	return nil
}

// anchorsearch: string seek anchorsearch post match true
//
//	string seek anchorsearch string false
func bAnchorsearch(intp *Interpreter) error {
	return intp.search("anchorsearch", func(s, seek []byte) int {
		if bytes.HasPrefix(s, seek) {
			return 0
		}
		return -1
	})
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
	if err := intp.chargeWork(int(size)); err != nil {
		return err
	}
	if err := intp.charge(int(size) * objectSize); err != nil {
		return err
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	intp.Stack = append(intp.Stack, make(Array, size))
	return nil
}

// astore: any0 … anyn-1 array astore array
func bAstore(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "astore: not enough arguments")
	}
	arr := intp.Stack[n-1]
	elems, ok := arrayElems(arr)
	if !ok {
		return intp.e(eTypecheck, "astore: needs an array")
	}
	if n-1 < len(elems) {
		return intp.e(eStackunderflow, "astore: not enough arguments")
	}
	if err := intp.chargeWork(len(elems)); err != nil {
		return err
	}
	base := n - 1 - len(elems)
	copy(elems, intp.Stack[base:n-1])
	clear(intp.Stack[base:])
	intp.Stack = append(intp.Stack[:base], arr)
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
	return intp.bindProc(obj)
}

// clear: ⊢ any1 … anyn clear ⊢
func bClear(intp *Interpreter) error {
	if err := intp.chargeWork(len(intp.Stack)); err != nil {
		return err
	}
	clear(intp.Stack)
	intp.Stack = intp.Stack[:0]
	return nil
}

func bCleartomark(intp *Interpreter) error {
	for k, v := range slices.Backward(intp.Stack) {
		if v == theMark {
			if err := intp.chargeWork(len(intp.Stack) - k); err != nil {
				return err
			}
			intp.Stack = intp.Stack[:k]
			return nil
		}
	}
	if err := intp.chargeWork(len(intp.Stack)); err != nil {
		return err
	}
	return intp.e(eUnmatchedmark, "cleartomark: no mark found")
}

func bClosefile(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "closefile: not enough arguments")
	}
	f, ok := intp.Stack[len(intp.Stack)-1].(*file)
	if !ok {
		return intp.e(eTypecheck, "closefile: needs a file")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	f.closed = true
	return nil
}

// concatmatrix: matrix1 matrix2 matrix3 concatmatrix matrix3
func bConcatmatrix(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 3 {
		return intp.e(eStackunderflow, "concatmatrix: not enough arguments")
	}
	a, err := intp.matrixOperand("concatmatrix", intp.Stack[n-3])
	if err != nil {
		return err
	}
	b, err := intp.matrixOperand("concatmatrix", intp.Stack[n-2])
	if err != nil {
		return err
	}
	m, ok := intp.Stack[n-1].(Array)
	if !ok {
		return intp.e(eTypecheck, "concatmatrix: needs an array")
	} else if len(m) != 6 {
		return intp.e(eRangecheck, "concatmatrix: needs 6 elements, not %d", len(m))
	}
	res := [6]float64{
		a[0]*b[0] + a[1]*b[2],
		a[0]*b[1] + a[1]*b[3],
		a[2]*b[0] + a[3]*b[2],
		a[2]*b[1] + a[3]*b[3],
		a[4]*b[0] + a[5]*b[2] + b[4],
		a[4]*b[1] + a[5]*b[3] + b[5],
	}
	for _, x := range res {
		if math.IsInf(x, 0) || math.IsNaN(x) {
			return intp.e(eUndefinedresult, "concatmatrix: undefined result")
		}
	}
	for i, x := range res {
		m[i] = Real(x)
	}
	intp.Stack = append(intp.Stack[:n-3], m)
	return nil
}

func bCopy(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "copy: not enough arguments")
	}
	if n, ok := intp.Stack[len(intp.Stack)-1].(Integer); ok {
		if n < 0 {
			return intp.e(eRangecheck, "copy: invalid count %d", n)
		}
		if n > Integer(len(intp.Stack)-1) {
			return intp.e(eStackunderflow, "copy: not enough arguments")
		}
		if err := intp.chargeWork(int(n)); err != nil {
			return err
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
		if err := intp.chargeWork(len(a)); err != nil {
			return err
		}
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
		if err := intp.chargeWork(len(a)); err != nil {
			return err
		}
		if err := intp.charge(len(a) * dictEntrySize); err != nil {
			return err
		}
		maps.Copy(b, a)
		res = b
	case String:
		if err := intp.chargeWork(len(a)); err != nil {
			return err
		}
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

// countdictstack: – countdictstack int
func bCountdictstack(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, Integer(len(intp.DictStack)))
	return nil
}

// counttomark: mark obj1 … objn counttomark mark obj1 … objn n
func bCounttomark(intp *Interpreter) error {
	n := len(intp.Stack)
	for i := n - 1; i >= 0; i-- {
		if intp.Stack[i] == theMark {
			if err := intp.chargeWork(n - i); err != nil {
				return err
			}
			intp.Stack = append(intp.Stack, Integer(n-1-i))
			return nil
		}
	}
	if err := intp.chargeWork(n); err != nil {
		return err
	}
	return intp.e(eUnmatchedmark, "counttomark: no mark")
}

func bCurrentdict(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, intp.DictStack[len(intp.DictStack)-1])
	return nil
}

func bCurrentfile(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, intp.files[len(intp.files)-1])
	return nil
}

// currentglobal: – currentglobal bool
func bCurrentglobal(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, Boolean(intp.global))
	return nil
}

// currentpacking: – currentpacking bool
func bCurrentpacking(intp *Interpreter) error {
	intp.Stack = append(intp.Stack, Boolean(intp.packing))
	return nil
}

// cvlit: any cvlit any
func bCvlit(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "cvlit: not enough arguments")
	}
	switch o := intp.Stack[n-1].(type) {
	case Procedure:
		// the array shares the value of the procedure
		intp.Stack[n-1] = Array(o)
	case Operator:
		intp.Stack[n-1] = Name(o)
	}
	return nil
}

// cvn: string cvn name
func bCvn(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "cvn: not enough arguments")
	}
	s, ok := intp.Stack[n-1].(String)
	if !ok {
		return intp.e(eTypecheck, "cvn: needs a string")
	}
	if err := intp.chargeWork(len(s)); err != nil {
		return err
	}
	if err := intp.charge(len(s)); err != nil {
		return err
	}
	intp.Stack[n-1] = Name(s)
	return nil
}

// cvrs: num radix string cvrs substring
func bCvrs(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 3 {
		return intp.e(eStackunderflow, "cvrs: not enough arguments")
	}
	num := intp.Stack[n-3]
	radix, ok1 := intp.Stack[n-2].(Integer)
	buf, ok2 := intp.Stack[n-1].(String)
	if !isNumber(num) || !ok1 || !ok2 {
		return intp.e(eTypecheck, "cvrs: invalid arguments")
	}
	if radix < 2 || radix > 36 {
		return intp.e(eRangecheck, "cvrs: invalid radix %d", radix)
	}

	var text []byte
	switch num := num.(type) {
	case Integer:
		if radix == 10 {
			text = strconv.AppendInt(nil, int64(num), 10)
		} else {
			text = formatRadix(num, int(radix))
		}
	case Real:
		if radix == 10 {
			text = formatReal(num)
			break
		}
		truncated := math.Trunc(float64(num))
		// The limits are powers of two and so exact as float64 values.
		if !(truncated >= math.MinInt && truncated < -math.MinInt) {
			return intp.e(eRangecheck, "cvrs: number too large to convert to integer")
		}
		text = formatRadix(Integer(truncated), int(radix))
	}
	return intp.storeText("cvrs", 3, text, buf)
}

// cvs: any string cvs substring
func bCvs(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 2 {
		return intp.e(eStackunderflow, "cvs: not enough arguments")
	}
	buf, ok := intp.Stack[n-1].(String)
	if !ok {
		return intp.e(eTypecheck, "cvs: needs a string")
	}
	text := []byte("--nostringval--")
	switch o := intp.Stack[n-2].(type) {
	case Integer:
		text = strconv.AppendInt(nil, int64(o), 10)
	case Real:
		text = formatReal(o)
	case Boolean:
		text = strconv.AppendBool(nil, bool(o))
	case String:
		text = o
	case Name:
		text = []byte(o)
	case Operator:
		text = []byte(o)
	case *builtin:
		text = []byte(o.name)
	}
	return intp.storeText("cvs", 2, text, buf)
}

func bCvx(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "cvx: not enough arguments")
	}
	// strings stay literal
	switch o := intp.Stack[len(intp.Stack)-1].(type) {
	case Array:
		// the procedure shares the value of the array
		intp.Stack[len(intp.Stack)-1] = Procedure(o)
	case Name:
		intp.Stack[len(intp.Stack)-1] = Operator(o)
	}
	return nil
}

func bDef(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "def: not enough arguments")
	}
	name, err := intp.dictKey(intp.Stack[len(intp.Stack)-2], "def")
	if err != nil {
		return err
	}
	intp.DictStack[len(intp.DictStack)-1][name] = intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	return nil
}

func bDefinefont(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "definefont: not enough arguments")
	}
	name, err := intp.dictKey(intp.Stack[len(intp.Stack)-2], "definefont")
	if err != nil {
		return err
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
	if class == "CMap" {
		intp.cmapDefined = append(intp.cmapDefined, key)
	}
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
	if err := intp.chargeWork(int(size)); err != nil {
		return err
	}
	if err := intp.charge(int(size) * dictEntrySize); err != nil {
		return err
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	// no size hint: iterating a map costs time in proportion to its capacity
	intp.Stack = append(intp.Stack, make(Dict))
	return nil
}

func bDup(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "dup: not enough arguments")
	}
	intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-1])
	return nil
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
	if err := intp.chargeWork(operandWork(a) + operandWork(b)); err != nil {
		return err
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-2], Boolean(equal(a, b)))
	return nil
}

func bExch(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "exch: not enough arguments")
	}
	intp.Stack[len(intp.Stack)-1], intp.Stack[len(intp.Stack)-2] = intp.Stack[len(intp.Stack)-2], intp.Stack[len(intp.Stack)-1]
	return nil
}

func bExec(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "exec: not enough arguments")
	}
	obj := intp.Stack[len(intp.Stack)-1]
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	// a literal object is pushed back
	return intp.executeOne(obj, true)
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
		return intp.e(eTypecheck, "findresource: needs a name, not %T", intp.Stack[len(intp.Stack)-1])
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
		if err := intp.chargeWork(len(keyObj)); err != nil {
			return err
		}
		key = Name(keyObj)
	default:
		return intp.e(eUndefinedresource, "findresource: needs a name or string, not %T", keyObj)
	}
	catDict, ok := cat.(Dict)
	if !ok {
		return intp.e(eUndefined, "resource category %q is not a dictionary", catName)
	}
	obj, ok := catDict[key]
	if !ok {
		return intp.e(eUndefinedresource, "resource %q not found in category %q", key, catName)
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-2], obj)
	return nil
}

// flush: – flush –
//
// There is no standard output, so there is nothing to do.
func bFlush(intp *Interpreter) error {
	return nil
}

func bFor(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 4 {
		return intp.e(eStackunderflow, "for: not enough arguments")
	}
	initial, increment, limit := intp.Stack[n-4], intp.Stack[n-3], intp.Stack[n-2]
	if !isNumber(initial) || !isNumber(increment) || !isNumber(limit) {
		return intp.e(eTypecheck, "for: needs numbers")
	}
	proc, ok := intp.Stack[n-1].(Procedure)
	if !ok {
		return intp.e(eTypecheck, "for: needs a procedure")
	}
	intp.Stack = intp.Stack[:n-4]

	// The control variable is an integer if initial and increment both
	// are, and a real otherwise.
	_, initialIsInt := initial.(Integer)
	inc, isInt := increment.(Integer)
	isInt = isInt && initialIsInt
	val := initial
	if !isInt {
		val = toReal(initial)
	}

	direction, _ := compareNumbers(increment, Integer(0))
	for {
		if c, _ := compareNumbers(val, limit); c != 0 && c == direction {
			break
		}
		intp.Stack = append(intp.Stack, val)
		err := intp.executeOne(proc, true)
		if err == errExit {
			break
		} else if err != nil {
			return err
		}

		if isInt {
			v := val.(Integer)
			next := v + inc
			if (inc > 0) != (next > v) && inc != 0 {
				// the next value is beyond every integer limit
				break
			}
			val = next
		} else {
			val = val.(Real) + toReal(increment)
		}
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
		name, err := intp.dictKey(sel, "get")
		if err != nil {
			return err
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

// identmatrix: matrix identmatrix matrix
func bIdentmatrix(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "identmatrix: not enough arguments")
	}
	m, ok := intp.Stack[n-1].(Array)
	if !ok {
		return intp.e(eTypecheck, "identmatrix: needs an array")
	} else if len(m) != 6 {
		return intp.e(eRangecheck, "identmatrix: needs 6 elements, not %d", len(m))
	}
	for i, x := range [6]Real{1, 0, 0, 1, 0, 0} {
		m[i] = x
	}
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
	proc, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
	if !ok {
		return intp.e(eTypecheck, "if: needs a procedure")
	}
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
	proc1, ok1 := intp.Stack[len(intp.Stack)-2].(Procedure)
	proc2, ok2 := intp.Stack[len(intp.Stack)-1].(Procedure)
	if !ok1 || !ok2 {
		return intp.e(eTypecheck, "ifelse: needs procedures")
	}
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
	name, err := intp.dictKey(intp.Stack[len(intp.Stack)-1], "known")
	if err != nil {
		return err
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
	name, err := intp.dictKey(intp.Stack[len(intp.Stack)-1], "load")
	if err != nil {
		return err
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	val, err := intp.loadName(name)
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
	proc, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
	if !ok {
		return intp.e(eTypecheck, "loop: needs a procedure")
	}
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
	if err := intp.charge(6 * objectSize); err != nil {
		return err
	}
	intp.Stack = append(intp.Stack, Array{Real(1), Real(0), Real(0), Real(1), Real(0), Real(0)})
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
		return intp.pushReal(ar*br, "mul")
	} else {
		ci := ai * bi
		// check for integer overflow
		if (ai == -1 && bi == math.MinInt) || (ai != 0 && ci/ai != bi) {
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
	if err := intp.chargeWork(operandWork(a) + operandWork(b)); err != nil {
		return err
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-2], Boolean(!equal(a, b)))
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

// print: string print –
//
// There is no standard output, so the string is discarded.
func bPrint(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "print: not enough arguments")
	}
	if _, ok := intp.Stack[n-1].(String); !ok {
		return intp.e(eTypecheck, "print: needs a string")
	}
	intp.Stack = intp.Stack[:n-1]
	return nil
}

// bPrintObject implements the operators = and ==: any = –
//
// There is no standard output, so the object is discarded.
func bPrintObject(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "=: not enough arguments")
	}
	intp.Stack[n-1] = nil
	intp.Stack = intp.Stack[:n-1]
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
		key, err := intp.dictKey(sel, "put")
		if err != nil {
			return err
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
		if Integer(len(src)) > Integer(len(dst))-index {
			return intp.e(eRangecheck, "putinterval: index out of range")
		}
		if err := intp.chargeWork(len(src)); err != nil {
			return err
		}
		copy(dst[index:], src)
	case String:
		src, ok := src.(String)
		if !ok {
			return intp.e(eTypecheck, "putinterval: mismatched argument types")
		}
		if Integer(len(src)) > Integer(len(dst))-index {
			return intp.e(eRangecheck, "putinterval: index out of range")
		}
		if err := intp.chargeWork(len(src)); err != nil {
			return err
		}
		copy(dst[index:], src)
	default:
		return intp.e(eTypecheck, "putinterval: invalid argument type %T", dst)
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-3]
	return nil
}

// rcheck: array|dict|file|string rcheck bool
func bRcheck(intp *Interpreter) error {
	return intp.accessCheck("rcheck", true)
}

func bReadonly(intp *Interpreter) error {
	// not implemented
	return nil
}

func bReadstring(intp *Interpreter) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "readstring: not enough arguments")
	}
	f, ok := intp.Stack[len(intp.Stack)-2].(*file)
	if !ok {
		return intp.e(eTypecheck, "readstring: needs a file")
	}
	buf, ok := intp.Stack[len(intp.Stack)-1].(String)
	if !ok {
		return intp.e(eTypecheck, "readstring: invalid argument")
	}
	if len(buf) == 0 {
		return intp.e(eRangecheck, "readstring: empty string")
	}
	if err := intp.chargeWork(len(buf)); err != nil {
		return err
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	n, err := f.Read(buf)
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
	for range count {
		err := intp.executeOne(proc, true)
		if err == errExit {
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
		return intp.e(eTypecheck, "roll: invalid count")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-2]
	if n == 0 {
		return nil
	}
	j %= n
	if j < 0 {
		j += n
	}
	if j == 0 {
		return nil
	}

	// Move the top j elements below the other n-j of the top n, by three
	// reversals in place.
	ji := int(j)
	ni := int(n)
	if err := intp.chargeWork(2 * ni); err != nil {
		return err
	}
	data := intp.Stack[len(intp.Stack)-ni:]
	slices.Reverse(data)
	slices.Reverse(data[:ji])
	slices.Reverse(data[ji:])

	return nil
}

// search: string seek search post match pre true
//
//	string seek search string false
func bSearch(intp *Interpreter) error {
	return intp.search("search", bytes.Index)
}

// setglobal: bool setglobal –
func bSetglobal(intp *Interpreter) error {
	return intp.popBoolean("setglobal", &intp.global)
}

// setpacking: bool setpacking –
func bSetpacking(intp *Interpreter) error {
	return intp.popBoolean("setpacking", &intp.packing)
}

func bStop(intp *Interpreter) error {
	return errStop
}

// store: key value store –
func bStore(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 2 {
		return intp.e(eStackunderflow, "store: not enough arguments")
	}
	key, err := intp.dictKey(intp.Stack[n-2], "store")
	if err != nil {
		return err
	}
	target := intp.DictStack[len(intp.DictStack)-1]
	for _, d := range slices.Backward(intp.DictStack) {
		if _, ok := d[key]; ok {
			target = d
			break
		}
	}
	target[key] = intp.Stack[n-1]
	intp.Stack = intp.Stack[:n-2]
	return nil
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
	if err := intp.chargeWork(int(size)); err != nil {
		return err
	}
	if err := intp.charge(int(size)); err != nil {
		return err
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
		return intp.pushReal(ar-br, "sub")
	} else {
		ci := ai - bi
		// check for integer overflow
		if (ai < 0 && bi > 0 && ci >= 0) || (ai >= 0 && bi < 0 && ci < 0) {
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
	var tp Operator
	switch obj.(type) {
	case Array, Procedure:
		tp = "arraytype"
	case Boolean:
		tp = "booleantype"
	case Dict:
		tp = "dicttype"
	case *file:
		tp = "filetype"
	// fonttype
	// gstatetype (LanguageLevel 2)
	case Integer:
		tp = "integertype"
	case Name, Operator:
		tp = "nametype"
	case nil:
		tp = "nulltype"
	case *builtin:
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
	intp.Stack[len(intp.Stack)-1] = tp
	return nil
}

// undef: dict key undef –
func bUndef(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 2 {
		return intp.e(eStackunderflow, "undef: not enough arguments")
	}
	d, ok := intp.Stack[n-2].(Dict)
	if !ok {
		return intp.e(eTypecheck, "undef: needs a dictionary")
	}
	key, err := intp.dictKey(intp.Stack[n-1], "undef")
	if err != nil {
		return err
	}
	delete(d, key)
	intp.Stack = intp.Stack[:n-2]
	return nil
}

// wcheck: array|dict|file|string wcheck bool
func bWcheck(intp *Interpreter) error {
	// all files are input files
	return intp.accessCheck("wcheck", false)
}

func bWhere(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return intp.e(eStackunderflow, "where: not enough arguments")
	}
	key, err := intp.dictKey(intp.Stack[len(intp.Stack)-1], "where")
	if err != nil {
		return err
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	for _, d := range slices.Backward(intp.DictStack) {
		if _, ok := d[key]; ok {
			intp.Stack = append(intp.Stack, d, Boolean(true))
			return nil
		}
	}
	intp.Stack = append(intp.Stack, Boolean(false))
	return nil
}

// xcheck: any xcheck bool
func bXcheck(intp *Interpreter) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "xcheck: not enough arguments")
	}
	var res bool
	switch intp.Stack[n-1].(type) {
	case Procedure, Operator, *builtin:
		res = true
	}
	intp.Stack[n-1] = Boolean(res)
	return nil
}

// operandWork returns the number of bytes inspected when o is compared.
func operandWork(o Object) int {
	switch o := o.(type) {
	case String:
		return len(o)
	case Name:
		return len(o)
	case Operator:
		return len(o)
	}
	return 0
}

// equal reports whether a and b are equal in the sense of the eq operator.
// Numbers compare by mathematical value, and strings and names by their
// characters.  Arrays, dictionaries and operators are equal only if they are
// the same object.  The literal/executable attribute is ignored.
func equal(a, b Object) bool {
	switch a := a.(type) {
	case Integer, Real:
		c, ok := compareNumbers(a, b)
		return ok && c == 0
	case String:
		switch b := b.(type) {
		case String:
			return bytes.Equal(a, b)
		default:
			t, ok := nameText(b)
			return ok && string(a) == t
		}
	case Name, Operator:
		s, _ := nameText(a)
		switch b := b.(type) {
		case String:
			return string(b) == s
		default:
			t, ok := nameText(b)
			return ok && s == t
		}
	case Boolean:
		b, ok := b.(Boolean)
		return ok && a == b
	case mark:
		_, ok := b.(mark)
		return ok
	case Dict:
		b, ok := b.(Dict)
		return ok && sameDict(a, b)
	case Array:
		return sameArray(a, b)
	case Procedure:
		return sameArray(a, b)
	case *builtin:
		b, ok := b.(*builtin)
		return ok && a == b
	case *file:
		b, ok := b.(*file)
		return ok && a == b
	case nil:
		return b == nil
	}
	return false
}

// nameText returns the characters of a literal or executable name.
func nameText(o Object) (string, bool) {
	switch o := o.(type) {
	case Name:
		return string(o), true
	case Operator:
		return string(o), true
	}
	return "", false
}

// sameDict reports whether a and b are the same dictionary.
func sameDict(a, b Dict) bool {
	return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(b).UnsafePointer()
}

// sameArray reports whether a and b are arrays or procedures which share the
// same value, i.e. the same elements of the same storage.  Empty arrays have
// no storage to share, and all compare equal.
func sameArray(a, b Object) bool {
	x, ok := arrayElems(a)
	if !ok {
		return false
	}
	y, ok := arrayElems(b)
	if !ok || len(x) != len(y) {
		return false
	}
	return len(x) == 0 || &x[0] == &y[0]
}

// arrayElems returns the elements of an array or procedure.
func arrayElems(o Object) ([]Object, bool) {
	switch o := o.(type) {
	case Array:
		return o, true
	case Procedure:
		return o, true
	}
	return nil, false
}

// compareNumbers compares a and b by their mathematical values and returns
// -1, 0 or +1.  The result ok is false if a or b is not a number, or is NaN.
func compareNumbers(a, b Object) (c int, ok bool) {
	switch a := a.(type) {
	case Integer:
		switch b := b.(type) {
		case Integer:
			return cmp.Compare(a, b), true
		case Real:
			if math.IsNaN(float64(b)) {
				return 0, false
			}
			return compareIntReal(int64(a), float64(b)), true
		}
	case Real:
		if math.IsNaN(float64(a)) {
			return 0, false
		}
		switch b := b.(type) {
		case Integer:
			return -compareIntReal(int64(b), float64(a)), true
		case Real:
			if math.IsNaN(float64(b)) {
				return 0, false
			}
			return cmp.Compare(a, b), true
		}
	}
	return 0, false
}

// compareIntReal compares the integer i to the real number r, which must not
// be NaN.  Converting i to float64 could round, so the comparison is done on
// the integer part of r first.
func compareIntReal(i int64, r float64) int {
	switch {
	case r >= 0x1p63:
		return -1
	case r < -0x1p63:
		return 1
	}
	t := math.Trunc(r)
	if c := cmp.Compare(i, int64(t)); c != 0 {
		return c
	}
	return cmp.Compare(t, r)
}

// compare implements the relational operators ge, gt, le and lt, which
// apply to two numbers or two strings.  The function test receives the
// result of comparing the first operand to the second: -1, 0 or +1.
func (intp *Interpreter) compare(op string, test func(int) bool) error {
	if len(intp.Stack) < 2 {
		return intp.e(eStackunderflow, "%s: not enough arguments", op)
	}
	a := intp.Stack[len(intp.Stack)-2]
	b := intp.Stack[len(intp.Stack)-1]

	var result bool
	s1, ok1 := a.(String)
	s2, ok2 := b.(String)
	switch {
	case isNumber(a) && isNumber(b):
		c, ok := compareNumbers(a, b)
		result = ok && test(c)
	case ok1 && ok2:
		if err := intp.chargeWork(len(s1) + len(s2)); err != nil {
			return err
		}
		result = test(bytes.Compare(s1, s2))
	default:
		return intp.e(eTypecheck, "%s: invalid argument types", op)
	}
	intp.Stack = append(intp.Stack[:len(intp.Stack)-2], Boolean(result))
	return nil
}

// bindProc implements bind for root and the procedures nested in it.  A
// procedure may occur many times in the tree, or contain itself, so each
// distinct procedure is visited only once.
func (intp *Interpreter) bindProc(root Procedure) error {
	type procKey struct {
		first *Object
		n     int
	}
	seen := make(map[procKey]bool)
	todo := []Procedure{root}
	for len(todo) > 0 {
		proc := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if len(proc) == 0 {
			continue
		}
		key := procKey{&proc[0], len(proc)}
		if seen[key] {
			continue
		}
		seen[key] = true

		// at least one step per procedure, for the bookkeeping in seen
		if err := intp.chargeWork(workPerStep + len(proc)); err != nil {
			return err
		}
		for i, elem := range proc {
			switch obj := elem.(type) {
			case Operator:
				// Only executable names are substituted, and only where the
				// value found is an operator.  A literal name, e.g. the "/add"
				// in "{/add}", is data and stays unchanged.
				val, err := intp.loadName(Name(obj))
				if err != nil {
					continue
				}
				if _, ok := val.(*builtin); ok {
					proc[i] = val
				}
			case Procedure:
				todo = append(todo, obj)
			}
		}
	}
	return nil
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
		return intp.pushRealAsInt(op)
	case String:
		x, err := intp.parseNumberString(op, "cvi")
		if err != nil {
			if psErr, ok := errors.AsType[*postScriptError](err); ok && psErr.tp == eLimitcheck {
				return intp.e(eRangecheck, "cvi: number too large to convert to integer")
			}
			return err
		}
		if r, isReal := x.(Real); isReal {
			return intp.pushRealAsInt(r)
		}
		intp.Stack = append(intp.Stack, x)
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
		x, err := intp.parseNumberString(op, "cvr")
		if err != nil {
			return err
		}
		if i, isInt := x.(Integer); isInt {
			x = Real(i)
		}
		intp.Stack = append(intp.Stack, x)
	default:
		return intp.e(eTypecheck, "cvr: invalid argument type")
	}

	return nil
}

// pushRealAsInt truncates r toward zero and pushes the result onto the
// operand stack as an integer.
func (intp *Interpreter) pushRealAsInt(r Real) error {
	truncated := math.Trunc(float64(r))
	// The limits are powers of two and so exact as float64 values.
	if !(truncated >= math.MinInt && truncated < -math.MinInt) {
		return intp.e(eRangecheck, "cvi: number too large to convert to integer")
	}
	intp.Stack = append(intp.Stack, Integer(truncated))
	return nil
}

// parseNumberString interprets s, ignoring surrounding white space, as a
// PostScript number, for the operator op.
func (intp *Interpreter) parseNumberString(s String, op string) (Object, error) {
	if err := intp.chargeWork(len(s)); err != nil {
		return nil, err
	}
	x, err := ParseNumber(trimSpace(s))
	if psErr, ok := errors.AsType[*postScriptError](err); ok {
		return nil, intp.e(psErr.tp, "%s: %s", op, psErr.msg)
	} else if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, intp.e(eSyntaxerror, "%s: invalid number in string", op)
	}
	return x, nil
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

	return intp.pushReal(Real(val1/val2), "div")
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

	return intp.pushReal(Real(math.Pow(baseVal, expVal)), "exp")
}

// pushReal pushes r onto the operand stack, or returns an undefinedresult
// error for the operator op where r is not a finite number.
func (intp *Interpreter) pushReal(r Real, op string) error {
	if math.IsInf(float64(r), 0) || math.IsNaN(float64(r)) {
		return intp.e(eUndefinedresult, "%s: undefined result", op)
	}
	intp.Stack = append(intp.Stack, r)
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
	return intp.compare("ge", func(c int) bool { return c >= 0 })
}

// gt: num1 num2 gt bool OR string1 string2 gt bool
func bGt(intp *Interpreter) error {
	return intp.compare("gt", func(c int) bool { return c > 0 })
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
	if int1 == math.MinInt && int2 == -1 {
		return intp.e(eUndefinedresult, "idiv: quotient too large")
	}

	// Integer division with truncation toward zero
	result := int1 / int2
	intp.Stack = append(intp.Stack, result)
	return nil
}

// le: num1 num2 le bool OR string1 string2 le bool
func bLe(intp *Interpreter) error {
	return intp.compare("le", func(c int) bool { return c <= 0 })
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
	return intp.compare("lt", func(c int) bool { return c < 0 })
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
		if n == math.MinInt {
			// the negation does not fit into an integer
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
		// Ties go to the greater integer.  The subtraction is exact, unlike
		// val+0.5, which can round up before the floor is taken.
		val := float64(n)
		f := math.Floor(val)
		if val-f >= 0.5 {
			f++
		}
		intp.Stack = append(intp.Stack, Real(f))
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

// toReal returns the value of a number as a real.
func toReal(obj Object) Real {
	if i, ok := obj.(Integer); ok {
		return Real(i)
	}
	r, _ := obj.(Real)
	return r
}

// dictKey returns the dictionary key which o stands for, for the operator
// op.  A string is converted to a name, and the literal/executable attribute
// of a name is ignored.
func (intp *Interpreter) dictKey(o Object, op string) (Name, error) {
	switch o := o.(type) {
	case Name:
		return o, nil
	case Operator:
		return Name(o), nil
	case String:
		if err := intp.chargeWork(len(o)); err != nil {
			return "", err
		}
		return Name(o), nil
	}
	return "", intp.e(eTypecheck, "%s: invalid dictionary key %T", op, o)
}

func isNumber(obj Object) bool {
	_, isInt := obj.(Integer)
	_, isReal := obj.(Real)
	return isInt || isReal
}

// accessCheck implements rcheck and wcheck.  Access attributes are not
// enforced, so every composite object can be read and written.
func (intp *Interpreter) accessCheck(op string, fileAccess bool) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "%s: not enough arguments", op)
	}
	res := true
	switch intp.Stack[n-1].(type) {
	case Array, Procedure, Dict, String:
		// pass
	case *file:
		res = fileAccess
	default:
		return intp.e(eTypecheck, "%s: invalid argument type %T", op, intp.Stack[n-1])
	}
	intp.Stack[n-1] = Boolean(res)
	return nil
}

// storeText implements the common tail of cvs and cvrs: the nargs topmost
// operands are replaced by the initial part of buf, overwritten with text.
func (intp *Interpreter) storeText(op string, nargs int, text []byte, buf String) error {
	if len(text) > len(buf) {
		return intp.e(eRangecheck, "%s: string too short", op)
	}
	if err := intp.chargeWork(len(text)); err != nil {
		return err
	}
	// a string may be converted into itself
	copy(buf, text)
	base := len(intp.Stack) - nargs
	clear(intp.Stack[base:])
	intp.Stack = append(intp.Stack[:base], buf[:len(text)])
	return nil
}

// formatReal returns the text of a real number.  The text always has
// a decimal point or an exponent, so that it reads back as a real.
func formatReal(r Real) []byte {
	text := strconv.AppendFloat(nil, float64(r), 'g', -1, 64)
	if !bytes.ContainsAny(text, ".e") {
		text = append(text, '.', '0')
	}
	return text
}

// formatRadix returns the digits of an integer for a radix other than 10.
// A negative number is shown as its two's complement, in 32 bits where the
// number fits.
func formatRadix(i Integer, radix int) []byte {
	u := uint64(i)
	if i < 0 && i >= math.MinInt32 {
		u = uint64(uint32(i))
	}
	return bytes.ToUpper(strconv.AppendUint(nil, u, radix))
}

// search implements search and anchorsearch.  The function find returns the
// position of the match, or -1.  The results share the value of the string
// searched.
func (intp *Interpreter) search(op string, find func(s, seek []byte) int) error {
	n := len(intp.Stack)
	if n < 2 {
		return intp.e(eStackunderflow, "%s: not enough arguments", op)
	}
	s, ok1 := intp.Stack[n-2].(String)
	seek, ok2 := intp.Stack[n-1].(String)
	if !ok1 || !ok2 {
		return intp.e(eTypecheck, "%s: needs strings", op)
	}
	if err := intp.chargeWork(len(s) + len(seek)); err != nil {
		return err
	}
	pos := find(s, seek)
	if pos < 0 {
		intp.Stack[n-1] = Boolean(false)
		return nil
	}
	end := pos + len(seek)
	post, match, pre := s[end:], s[pos:end], s[:pos]
	intp.Stack = intp.Stack[:n-2]
	intp.Stack = append(intp.Stack, post, match)
	if op == "search" {
		intp.Stack = append(intp.Stack, pre)
	}
	intp.Stack = append(intp.Stack, Boolean(true))
	return nil
}

// popBoolean removes a boolean operand and stores it in dst.
func (intp *Interpreter) popBoolean(op string, dst *bool) error {
	n := len(intp.Stack)
	if n < 1 {
		return intp.e(eStackunderflow, "%s: not enough arguments", op)
	}
	b, ok := intp.Stack[n-1].(Boolean)
	if !ok {
		return intp.e(eTypecheck, "%s: needs a boolean", op)
	}
	*dst = bool(b)
	intp.Stack = intp.Stack[:n-1]
	return nil
}

// matrixOperand returns the six numbers of a matrix operand.
func (intp *Interpreter) matrixOperand(op string, o Object) (m [6]float64, err error) {
	elems, ok := arrayElems(o)
	if !ok {
		return m, intp.e(eTypecheck, "%s: needs an array", op)
	} else if len(elems) != 6 {
		return m, intp.e(eRangecheck, "%s: needs 6 elements, not %d", op, len(elems))
	}
	for i, x := range elems {
		if !isNumber(x) {
			return m, intp.e(eTypecheck, "%s: needs numbers", op)
		}
		m[i] = float64(toReal(x))
	}
	return m, nil
}

var errExit = errors.New("exit")
var errStop = errors.New("stop")
