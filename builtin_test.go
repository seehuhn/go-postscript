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
	"fmt"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func run(s string, stackLen int) (*Interpreter, error) {
	intp := NewInterpreter()
	err := intp.ExecuteString(s)
	if err == nil && len(intp.Stack) != stackLen {
		err = fmt.Errorf("stack length is %d, expected %d", len(intp.Stack), stackLen)
	}
	return intp, err
}

func TestArrayLiteral(t *testing.T) {
	intp, err := run("[1 [] 2]", 1)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack[0], Array{Integer(1), Array{}, Integer(2)}); d != "" {
		t.Fatal(d)
	}
}

func TestDictLiteral(t *testing.T) {
	intp, err := run("<< /a 1 /b 2 >>", 1)
	if err != nil {
		t.Fatal(err)
	}
	expected := Dict{
		Name("a"): Integer(1),
		Name("b"): Integer(2),
	}
	if d := cmp.Diff(intp.Stack[0], expected); d != "" {
		t.Fatal(d)
	}
}

func TestCmdAbs(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Integer(0), Integer(0)},
		{Integer(1), Integer(1)},
		{Integer(-1), Integer(1)},
		{Integer(-100), Integer(100)},
		{Integer(math.MinInt), -Real(math.MinInt)},
		{Real(0), Real(0)},
		{Real(1), Real(1)},
		{Real(-1), Real(1)},
		{Real(-100), Real(100)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("abs")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("intp.Stack[0]: %v != %v", intp.Stack[0], c.out)
		}
	}
}

func TestCmdAdd(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(1), Integer(2), Integer(3)},
		{Integer(1), Real(2), Real(3)},
		{Real(1), Integer(2), Real(3)},
		{Real(1), Real(2), Real(3)},
		{Integer(math.MaxInt), Integer(1), Real(math.MaxInt + 1)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("add")
		if err != nil {
			t.Error(err)
			continue
		}
		if len(intp.Stack) != 1 {
			t.Errorf("len(intp.Stack): %d != 1", len(intp.Stack))
			continue
		}
		if intp.Stack[0] != c.out {
			t.Errorf("intp.Stack[0]: %v (%T) != %v",
				intp.Stack[0], intp.Stack[0], c.out)
		}
	}
}

func TestCmdAnd(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Boolean(false), Boolean(false), Boolean(false)},
		{Boolean(false), Boolean(true), Boolean(false)},
		{Boolean(true), Boolean(false), Boolean(false)},
		{Boolean(true), Boolean(true), Boolean(true)},
		{Integer(0), Integer(0), Integer(0)},
		{Integer(99), Integer(1), Integer(1)},
		{Integer(52), Integer(7), Integer(4)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("and")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("intp.Stack[0]: %v (%T) != %v",
				intp.Stack[0], intp.Stack[0], c.out)
		}
	}
}

func TestCmdArray(t *testing.T) {
	intp, err := run("3 array", 1)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack[0], Array{nil, nil, nil}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdBegin(t *testing.T) {
	intp, err := run("1 dict dup begin", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !isSameDict(intp.Stack[0].(Dict), intp.DictStack[len(intp.DictStack)-1]) {
		t.Fatal("wrong dict on stack")
	}
}

func TestCmdBind(t *testing.T) {
	intp, err := run("{1 add} bind", 1)
	if err != nil {
		t.Fatal(err)
	}
	proc := intp.Stack[0].(Procedure)
	if len(proc) != 2 {
		t.Fatalf("len(p): %d != 2", len(proc))
	}
	switch obj := proc[1].(type) {
	case builtin:
		// pass
	case Operator:
		t.Fatalf("p[1] is an Operator: %v", obj)
	default:
		t.Fatal("test is broken")
	}
}

// TestCmdBind2 tests whether bind works recursively.
func TestCmdBind2(t *testing.T) {
	intp, err := run("{{1 add}} bind", 1)
	if err != nil {
		t.Fatal(err)
	}
	outer := intp.Stack[0].(Procedure)
	if len(outer) != 1 {
		t.Fatalf("len(p): %d != 1", len(outer))
	}
	inner := outer[0].(Procedure)
	if len(inner) != 2 {
		t.Fatalf("len(p): %d != 2", len(inner))
	}
	switch obj := inner[1].(type) {
	case builtin:
		// pass
	case Operator:
		t.Fatalf("p[1] is an Operator: %v", obj)
	default:
		t.Fatal("test is broken")
	}
}

// TestCmdBind3 tests whether bind can be trapped in an infinite loop.
func TestCmdBind3(t *testing.T) {
	intp, err := run("{{}} dup dup 0 exch put bind", 1)
	if err != nil {
		t.Fatal(err)
	}
	proc := intp.Stack[0].(Procedure)
	if len(proc) != 1 {
		t.Fatalf("len(p): %d != 1", len(proc))
	}
	if _, ok := proc[0].(Procedure); !ok {
		t.Fatalf("p[0] is not a Procedure: %v", proc[0])
	}
}

func TestCmdCopy(t *testing.T) {
	intp, err := run(`1 2 3 4 2 copy`, 6)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{
		Integer(1), Integer(2), Integer(3), Integer(4),
		Integer(3), Integer(4)}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdCopy2(t *testing.T) {
	intp, err := run(`
		/a [1 2] def
		/b [3 4 5] def
		/c a b copy def
		c 1 6 put
		a b c`, 3)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{
		Array{Integer(1), Integer(2)},
		Array{Integer(1), Integer(6), Integer(5)},
		Array{Integer(1), Integer(6)},
	}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdCurrentDict(t *testing.T) {
	a := Dict{"test": Name("testint 1 2 3")}
	intp := NewInterpreter()
	intp.DictStack = append(intp.DictStack, a)
	err := intp.ExecuteString("currentdict")
	if err != nil {
		t.Fatal(err)
	}

	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	if len(intp.DictStack) != 3 {
		t.Fatalf("len(intp.DictStack): %d != 3", len(intp.DictStack))
	}

	// make sure the same dict is on the top of each stack
	b := intp.DictStack[2]
	c := intp.Stack[0].(Dict)
	if d := cmp.Diff(a, b); d != "" {
		t.Fatal(d)
	}
	if d := cmp.Diff(a, c); d != "" {
		t.Fatal(d)
	}
}

func TestCmdDef(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/a 1 def /b 2 def")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 0 {
		t.Fatalf("len(intp.Stack): %d != 0", len(intp.Stack))
	}
	if len(intp.DictStack) != 2 {
		t.Fatalf("len(intp.DictStack): %d != 2", len(intp.DictStack))
	}

	// make sure the same dict is on the top of each stack
	a := Dict{
		"a": Integer(1),
		"b": Integer(2),
	}
	b := intp.DictStack[1]
	if d := cmp.Diff(a, b); d != "" {
		t.Fatal(d)
	}
}

func TestCmdDef2(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/inc {1 add} def 2 inc")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}

	if intp.Stack[0] != Integer(3) {
		t.Fatalf("intp.Stack[0]: %v != 3", intp.Stack[0])
	}
}

func TestCmdDict(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("12 dict")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	if len(intp.DictStack) != 2 {
		t.Fatalf("len(intp.DictStack): %d != 2", len(intp.DictStack))
	}
	d, ok := intp.Stack[0].(Dict)
	if !ok || len(d) != 0 {
		t.Fatalf("intp.Stack[0] is not a Dict")
	}
}

func TestCmdDup(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("1 2 3 dup")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 4 {
		t.Fatalf("len(intp.Stack): %d != 4", len(intp.Stack))
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(1), Integer(2), Integer(3), Integer(3)}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdEnd(t *testing.T) {
	intp := NewInterpreter()
	intp.DictStack = append(intp.DictStack, Dict{})
	err := intp.ExecuteString("end")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 0 {
		t.Fatalf("len(intp.Stack): %d != 0", len(intp.Stack))
	}
	if len(intp.DictStack) != 2 {
		t.Fatalf("len(intp.DictStack): %d != 2", len(intp.DictStack))
	}
}

func TestCmdEq(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString(`
		1 2 eq
		4.0 4 eq
		(abc) (abc) eq
		(abc) /abc eq
	`)
	if err != nil {
		t.Fatal(err)
	}
	T := Boolean(true)
	F := Boolean(false)
	if d := cmp.Diff(intp.Stack, []Object{F, T, T, T}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdExch(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("1 2 3 exch")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 3 {
		t.Fatalf("len(intp.Stack): %d != 3", len(intp.Stack))
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(1), Integer(3), Integer(2)}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdFalse(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("false")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	if intp.Stack[0] != Boolean(false) {
		t.Fatalf("intp.Stack[0]: %v != false", intp.Stack[0])
	}
}

func TestFindResource(t *testing.T) {
	intp, err := run("/CIDInit /ProcSet findresource", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !isSameDict(intp.Stack[0].(Dict), CIDInit) {
		t.Fatalf("intp.Stack[0]: %v != CIDInit", intp.Stack[0])
	}
}

func TestCmdFor(t *testing.T) {
	intp, err := run("1 1 3 {} for", 3)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(1), Integer(2), Integer(3)}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdFor2(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("0 1 1 4 {add} for")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	if intp.Stack[0] != Integer(10) {
		t.Fatalf("intp.Stack[0]: %v != 6", intp.Stack[0])
	}
}

func TestCmdForall(t *testing.T) {
	type testCase struct {
		in  string
		out []Object
	}
	cases := []testCase{
		{
			"[3 2 1] {} forall",
			[]Object{Integer(3), Integer(2), Integer(1)},
		},
		{
			"(abc) {} forall",
			[]Object{Integer(97), Integer(98), Integer(99)},
		},
		{
			"<< /a 1 >> {} forall",
			[]Object{Name("a"), Integer(1)},
		},
		{
			"0 [1 2 3] {add} forall",
			[]Object{Integer(6)},
		},
		{
			"[3 2 1] {dup 2 eq {exit} if} forall",
			[]Object{Integer(3), Integer(2)},
		},
	}
	for _, c := range cases {
		intp, err := run(c.in, len(c.out))
		if err != nil {
			t.Fatal(err)
		}
		if d := cmp.Diff(intp.Stack, c.out); d != "" {
			t.Fatal(d)
		}
	}
}

func TestCmdGetInterval(t *testing.T) {
	intp, err := run(`
		/s (abcdef) def
		s dup 1 3 getinterval
		dup 1 67 put`, 2)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{String("abCdef"), String("bCd")}); d != "" {
		t.Error(d)
	}
}

func TestCmdGetInterval2(t *testing.T) {
	intp, err := run(`
		[1 2 0 4 5]
		dup 2 1 getinterval
		0 3 put`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack[0], Array{Integer(1), Integer(2), Integer(3), Integer(4), Integer(5)}); d != "" {
		t.Error(d)
	}
}

func TestCmdIfElse(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("true {1} {2} ifelse")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	if intp.Stack[0] != Integer(1) {
		t.Fatalf("intp.Stack[0]: %v != 1", intp.Stack[0])
	}
	intp.Stack = intp.Stack[:0]

	err = intp.ExecuteString("false {1} {2} ifelse")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	if intp.Stack[0] != Integer(2) {
		t.Fatalf("intp.Stack[0]: %v != 1", intp.Stack[0])
	}
}

func TestCmdIndex(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("(a) (b) (c) (d) 1 index")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 5 {
		t.Fatalf("len(intp.Stack): %d != 5", len(intp.Stack))
	}
	if d := cmp.Diff(intp.Stack, []Object{String("a"), String("b"), String("c"), String("d"), String("c")}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdIndex2(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("(a) (b) (c) (d) 3 index")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 5 {
		t.Fatalf("len(intp.Stack): %d != 5", len(intp.Stack))
	}
	if d := cmp.Diff(intp.Stack, []Object{String("a"), String("b"), String("c"), String("d"), String("a")}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdLength(t *testing.T) {
	type testCases struct {
		in  string
		out int
	}
	cases := []testCases{
		{"[1 2 4]", 3},
		{"[]", 0},
		{"{1 2 [3 4 5]}", 7},
		{"/ar 20 array def ar", 20},
		{"/mydict 5 dict def mydict", 0},
		{"/mydict 5 dict def mydict /firstkey (firstvalue) put mydict", 1},
		{"(abc)", 3},
		{`(abc\n)`, 4},
		{"()", 0},
		{"/foo", 3},
	}

	for i, c := range cases {
		intp := NewInterpreter()
		err := intp.ExecuteString(c.in)
		if err != nil || len(intp.Stack) != 1 {
			t.Fatal("error executing test case: ", i, err, len(intp.Stack))
		}
		err = intp.ExecuteString("length")
		if err != nil {
			t.Error(err)
			continue
		}
		if len(intp.Stack) != 1 {
			t.Errorf("%d: len(intp.Stack): %d != 1", i, len(intp.Stack))
			continue
		}
		if intp.Stack[0] != Integer(c.out) {
			t.Errorf("%d: intp.Stack[0]: %v != %d", i, intp.Stack[0], c.out)
		}
	}
}

func TestCmdMul(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(3), Integer(2), Integer(6)},
		{Integer(1), Integer(-2), Integer(-2)},
		{Integer(1), Real(-2), Real(-2)},
		{Real(1), Integer(-2), Real(-2)},
		{Real(1), Real(-2), Real(-2)},
		{Integer(math.MaxInt), Integer(2), Real(math.MaxInt) * 2},
	}
	for i, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("mul")
		if err != nil {
			t.Error(err)
			continue
		}
		if len(intp.Stack) != 1 {
			t.Errorf("len(intp.Stack): %d != 1", len(intp.Stack))
			continue
		}
		if intp.Stack[0] != c.out {
			t.Errorf("%d: intp.Stack[0]: %v (%T) != %v",
				i, intp.Stack[0], intp.Stack[0], c.out)
		}
	}
}

func TestCmdOr(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Boolean(false), Boolean(false), Boolean(false)},
		{Boolean(false), Boolean(true), Boolean(true)},
		{Boolean(true), Boolean(false), Boolean(true)},
		{Boolean(true), Boolean(true), Boolean(true)},
		{Integer(0), Integer(0), Integer(0)},
		{Integer(17), Integer(5), Integer(21)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("or")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("intp.Stack[0]: %v (%T) != %v",
				intp.Stack[0], intp.Stack[0], c.out)
		}
	}
}

func TestCmdPop(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("1 2 3 pop")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(1), Integer(2)}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdPut(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/ar [5 17 3 8] def ar 2 (abcd) put ar")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	ar := intp.Stack[0].(Array)
	if d := cmp.Diff(ar, Array{Integer(5), Integer(17), String("abcd"), Integer(8)}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdPut2(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/d 5 dict def d /abc 123 put d")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	d := intp.Stack[0].(Dict)
	if d := cmp.Diff(d, Dict{"abc": Integer(123)}); d != "" {
		t.Fatal(d)
	}
}

func TestCmdPut3(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/st (abc) def st 0 65 put st")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
	}
	st := intp.Stack[0].(String)
	if d := cmp.Diff(st, String("Abc")); d != "" {
		t.Fatal(d)
	}
}

func TestCmdPutInterval(t *testing.T) {
	intp, err := run(`
		/s (abcdef) def
		s 1 (xyz) putinterval
		s`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack[0], String("axyzef")); d != "" {
		t.Error(d)
	}
}

func TestCmdRoll(t *testing.T) {
	type testCase struct {
		in  string
		out []Object
	}
	cases := []testCase{
		{
			"(a)(b)(c) 3 -1 roll",
			[]Object{String("b"), String("c"), String("a")},
		},
		{
			"(a)(b)(c) 3 1 roll",
			[]Object{String("c"), String("a"), String("b")},
		},
		{
			"(a)(b)(c) 3 0 roll",
			[]Object{String("a"), String("b"), String("c")},
		},
		{
			"1 2 3 4 5 4 5 roll",
			[]Object{Integer(1), Integer(5), Integer(2), Integer(3), Integer(4)},
		},
	}
	for _, c := range cases {
		intp, err := run(c.in, len(c.out))
		if err != nil {
			t.Error(err)
			continue
		}
		if d := cmp.Diff(intp.Stack, c.out); d != "" {
			t.Error(d)
		}
	}
}

func TestCmdSub(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(3), Integer(2), Integer(1)},
		{Integer(1), Integer(-2), Integer(3)},
		{Integer(1), Real(-2), Real(3)},
		{Real(1), Integer(-2), Real(3)},
		{Real(1), Real(-2), Real(3)},
		{Integer(math.MaxInt), Integer(-1), Real(math.MaxInt + 1)},
	}
	for i, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("sub")
		if err != nil {
			t.Error(err)
			continue
		}
		if len(intp.Stack) != 1 {
			t.Errorf("len(intp.Stack): %d != 1", len(intp.Stack))
			continue
		}
		if intp.Stack[0] != c.out {
			t.Errorf("%d: intp.Stack[0]: %v (%T) != %v",
				i, intp.Stack[0], intp.Stack[0], c.out)
		}
	}
}

func TestReadstring(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("currentfile 3 string readstring A B")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{String("A B"), Boolean(true)}); d != "" {
		t.Fatal(d)
	}
}
