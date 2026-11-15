// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
	if !sameDict(intp.Stack[0].(Dict), intp.DictStack[len(intp.DictStack)-1]) {
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
	case *builtin:
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
	case *builtin:
		// pass
	case Operator:
		t.Fatalf("p[1] is an Operator: %v", obj)
	default:
		t.Fatal("test is broken")
	}
}

// TestCmdBindLiteralName tests that bind leaves literal names alone.  Only
// executable names stand for operators; "/add" is data.
func TestCmdBindLiteralName(t *testing.T) {
	intp, err := run("{/add} bind", 1)
	if err != nil {
		t.Fatal(err)
	}
	proc := intp.Stack[0].(Procedure)
	if len(proc) != 1 {
		t.Fatalf("len(p): %d != 1", len(proc))
	}
	if name, ok := proc[0].(Name); !ok || name != "add" {
		t.Errorf("p[0] is %T(%v), want Name(add)", proc[0], proc[0])
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

// TestCmdBindShared tests bind on a procedure tree in which one procedure
// occurs 2^60 times.  Visiting every occurrence would not terminate in
// practice.
func TestCmdBindShared(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("/p {1 add} def\n")
	for range 60 {
		sb.WriteString("/p [ /p load dup ] cvx def\n")
	}
	sb.WriteString("/p load bind")
	intp, err := run(sb.String(), 1)
	if err != nil {
		t.Fatal(err)
	}
	proc := intp.Stack[0].(Procedure)
	for range 60 {
		proc = proc[0].(Procedure)
	}
	if _, ok := proc[1].(*builtin); !ok {
		t.Errorf("innermost p[1] is %T, want a builtin", proc[1])
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

// TestCmdCopyOverflow checks that a malicious count near math.MaxInt64
// returns an error instead of panicking from an integer overflow in the
// stack-bounds check.
func TestCmdCopyOverflow(t *testing.T) {
	intp := NewInterpreter()
	if err := intp.ExecuteString("9223372036854775807 copy\n"); err == nil {
		t.Fatal("expected error, got nil")
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
	cases := []struct {
		operands string
		want     bool
	}{
		{"1 2", false},
		{"4.0 4", true},
		{"4 4.5", false},
		{"(abc) (abc)", true},
		{"(abc) (abd)", false},
		{"(abc) /abc", true},
		{"/abc (abc)", true},
		{"/abc /abd", false},
		{"{abc} 0 get /abc", true},
		{"(abc) {abc} 0 get", true},
		{"1 (1)", false},
		{"true true", true},
		{"true false", false},
		{"true 1", false},
		{"mark mark", true},
		{"mark 0", false},
		{"[1 2 3] dup", true},
		{"[1 2 3] [1 2 3]", false},
		{"[1 2 3] dup 0 2 getinterval exch 0 2 getinterval", true},
		{"[1 2 3] dup 0 2 getinterval exch 1 2 getinterval", false},
		{"[1 2 3] dup 0 2 getinterval", false},
		{"[] []", true},
		{"{1 2} dup", true},
		{"{1 2} {1 2}", false},
		{"[1 2] dup cvx", true},
		{"1 dict dup", true},
		{"1 dict 1 dict", false},
		{"1 dict 1", false},
		{"1 dict [1]", false},
		{"/add load dup", true},
		{"/add load /add load", true},
		{"/add load /sub load", false},
		{"/add load /add", false},
		{"/CIDInit /ProcSet findresource begin /begincidrange load /begincidchar load", false},
	}
	for _, c := range cases {
		for _, op := range []string{"eq", "ne"} {
			intp, err := run(c.operands+" "+op, 1)
			if err != nil {
				t.Errorf("%s %s: %v", c.operands, op, err)
				continue
			}
			want := Boolean(c.want == (op == "eq"))
			if got := intp.Stack[0]; got != want {
				t.Errorf("%s %s: got %v, want %v", c.operands, op, got, want)
			}
		}
	}
}

func TestEqualNull(t *testing.T) {
	if !equal(nil, nil) {
		t.Error("null is not equal to null")
	}
	if equal(nil, Integer(0)) || equal(Integer(0), nil) {
		t.Error("null is equal to 0")
	}
}

func TestEqualNaN(t *testing.T) {
	nan := Real(math.NaN())
	for _, b := range []Object{nan, Real(1), Integer(1)} {
		if equal(nan, b) || equal(b, nan) {
			t.Errorf("NaN is equal to %v", b)
		}
	}
}

// TestEqLargeIntegers checks that integers which round to the same float64
// still compare unequal.
func TestEqLargeIntegers(t *testing.T) {
	if math.MaxInt < 1<<62 {
		t.Skip("integers are 32 bits wide")
	}
	cases := []struct {
		code string
		want bool
	}{
		{"9007199254740993 9007199254740992 eq", false},
		{"9007199254740993 9007199254740992 gt", true},
		{"9007199254740993 9007199254740992.0 eq", false},
		{"9007199254740992 9007199254740992.0 eq", true},
		{"9223372036854775807 9223372036854775808.0 lt", true},
	}
	for _, c := range cases {
		intp, err := run(c.code, 1)
		if err != nil {
			t.Errorf("%s: %v", c.code, err)
			continue
		}
		if got := intp.Stack[0]; got != Boolean(c.want) {
			t.Errorf("%s: got %v, want %v", c.code, got, c.want)
		}
	}
}

func TestCompareIntReal(t *testing.T) {
	cases := []struct {
		i    int64
		r    float64
		want int
	}{
		{0, 0, 0},
		{0, math.Copysign(0, -1), 0},
		{1, 0.5, 1},
		{0, 0.5, -1},
		{-2, -2.5, 1},
		{-3, -2.5, -1},
		{1<<53 + 1, 1 << 53, 1},
		{1 << 53, 1<<53 + 2, -1},
		{math.MaxInt64, 0x1p63, -1},
		{math.MinInt64, -0x1p63, 0},
		{math.MinInt64, -0x1p64, 1},
		{0, math.Inf(1), -1},
		{0, math.Inf(-1), 1},
	}
	for _, c := range cases {
		if got := compareIntReal(c.i, c.r); got != c.want {
			t.Errorf("compareIntReal(%d, %g) = %d, want %d", c.i, c.r, got, c.want)
		}
	}
}

func TestCmdRelational(t *testing.T) {
	// results for ge, gt, le, lt
	cases := []struct {
		operands string
		want     [4]bool
	}{
		{"1 2", [4]bool{false, false, true, true}},
		{"2 2", [4]bool{true, false, true, false}},
		{"4.2 4", [4]bool{true, true, false, false}},
		{"4 4.0", [4]bool{true, false, true, false}},
		{"(abc) (d)", [4]bool{false, false, true, true}},
		{"(aba) (ab)", [4]bool{true, true, false, false}},
		{"(aba) (aba)", [4]bool{true, false, true, false}},
		{"<ff> <01>", [4]bool{true, true, false, false}},
		{"() ()", [4]bool{true, false, true, false}},
	}
	ops := [4]string{"ge", "gt", "le", "lt"}
	for _, c := range cases {
		for k, op := range ops {
			intp, err := run(c.operands+" "+op, 1)
			if err != nil {
				t.Errorf("%s %s: %v", c.operands, op, err)
				continue
			}
			if got := intp.Stack[0]; got != Boolean(c.want[k]) {
				t.Errorf("%s %s: got %v, want %v", c.operands, op, got, c.want[k])
			}
		}
	}

	for _, operands := range []string{"1 (1)", "(a) /a", "/a /b", "true false", "[] []", "1"} {
		for _, op := range ops {
			_, err := run(operands+" "+op, 0)
			if err == nil {
				t.Errorf("%s %s: no error", operands, op)
			}
		}
	}
}

// TestCmdRelationalKeepsOperands checks that the operands stay on the stack
// when a relational operator fails.
func TestCmdRelationalKeepsOperands(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("1 (1) lt")
	if err == nil {
		t.Fatal("no error")
	}
	if d := cmp.Diff([]Object{Integer(1), String("1")}, intp.Stack); d != "" {
		t.Errorf("wrong stack (-want +got):\n%s", d)
	}
}

func TestCmdCvx(t *testing.T) {
	intp, err := run("[1 2] dup cvx exch 0 3 put", 1)
	if err != nil {
		t.Fatal(err)
	}
	want := Procedure{Integer(3), Integer(2)}
	if d := cmp.Diff(want, intp.Stack[0]); d != "" {
		t.Errorf("cvx does not share the array value (-want +got):\n%s", d)
	}

	intp, err = run("/foo cvx", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := intp.Stack[0]; got != Operator("foo") {
		t.Errorf("/foo cvx: got %#v, want Operator(\"foo\")", got)
	}
}

func TestStackUnderflow(t *testing.T) {
	for _, code := range []string{
		"cvx", "1 eq", "1 ne", "closefile", "eexec", "1 readstring", "type",
		"{} repeat",
	} {
		if _, err := run(code, 0); err == nil {
			t.Errorf("%s: no error", code)
		}
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
	if !sameDict(intp.Stack[0].(Dict), intp.Resources["ProcSet"].(Dict)["CIDInit"].(Dict)) {
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

// TestCmdPutIntervalOverflow checks that a malicious index near
// math.MaxInt64 returns an error instead of panicking from an integer
// overflow in the destination-bounds check.
func TestCmdPutIntervalOverflow(t *testing.T) {
	intp := NewInterpreter()
	if err := intp.ExecuteString("10 string 9223372036854775807 (X) putinterval\n"); err == nil {
		t.Fatal("expected error, got nil")
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
		{
			"1 2 3 4 5 5 2 roll 3 -1 roll",
			[]Object{Integer(4), Integer(5), Integer(2), Integer(3), Integer(1)},
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

func TestCmdAtan(t *testing.T) {
	type testCase struct {
		num, den Object
		out      Object
	}
	cases := []testCase{
		{Integer(0), Integer(1), Real(0.0)},
		{Integer(1), Integer(0), Real(90.0)},
		{Integer(-100), Integer(0), Real(270.0)},
		{Integer(4), Integer(4), Real(45.0)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.num, c.den}
		err := intp.ExecuteString("atan")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if result, ok := intp.Stack[0].(Real); !ok {
			t.Fatalf("result is not Real: %T", intp.Stack[0])
		} else if math.Abs(float64(result-c.out.(Real))) > 1e-10 {
			t.Fatalf("atan(%v, %v): %v != %v", c.num, c.den, result, c.out)
		}
	}
}

func TestCmdBitshift(t *testing.T) {
	type testCase struct {
		val, shift Object
		out        Object
	}
	cases := []testCase{
		{Integer(7), Integer(3), Integer(56)},
		{Integer(142), Integer(-3), Integer(17)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.val, c.shift}
		err := intp.ExecuteString("bitshift")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("bitshift(%v, %v): %v != %v", c.val, c.shift, intp.Stack[0], c.out)
		}
	}
}

func TestCmdCeiling(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Real(3.2), Real(4.0)},
		{Real(-4.8), Real(-4.0)},
		{Integer(99), Integer(99)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("ceiling")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("ceiling(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdCos(t *testing.T) {
	type testCase struct {
		angle Object
		out   Object
	}
	cases := []testCase{
		{Integer(0), Real(1.0)},
		{Integer(90), Real(0.0)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.angle}
		err := intp.ExecuteString("cos")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if result, ok := intp.Stack[0].(Real); !ok {
			t.Fatalf("result is not Real: %T", intp.Stack[0])
		} else if math.Abs(float64(result-c.out.(Real))) > 1e-10 {
			t.Fatalf("cos(%v): %v != %v", c.angle, result, c.out)
		}
	}
}

func TestCmdCvi(t *testing.T) {
	type testCase struct {
		in  string
		out Object
	}
	cases := []testCase{
		{"(3.3E1) cvi", Integer(33)},
		{"-47.8 cvi", Integer(-47)},
		{"520.9 cvi", Integer(520)},
		{"( 12\n) cvi", Integer(12)},
		{"(16#ff) cvi", Integer(255)},
		{"(-7.9) cvi", Integer(-7)},
		{fmt.Sprintf("%d.0 cvi", math.MinInt), Integer(math.MinInt)},
	}
	for _, c := range cases {
		intp, err := run(c.in, 1)
		if err != nil {
			t.Fatal(err)
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("%s: %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdCvr(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Integer(42), Real(42.0)},
		{Real(3.14), Real(3.14)},
		{String("2"), Real(2)},
		{String(" 1.5\n"), Real(1.5)},
		{String("8#17"), Real(15)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("cvr")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("cvr(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestNumberConversionErrors(t *testing.T) {
	type testCase struct {
		prog string
		tp   Name
	}
	cases := []testCase{
		{"(Inf) cvr", eSyntaxerror},
		{"(NaN) cvi", eSyntaxerror},
		{"(0x1p4) cvr", eSyntaxerror},
		{"(1_0) cvi", eSyntaxerror},
		{"(1 2) cvi", eSyntaxerror},
		{"() cvr", eSyntaxerror},
		{"(1e999) cvr", eLimitcheck},
		{fmt.Sprintf("%.1f cvi", float64(-math.MinInt)), eRangecheck},
		{fmt.Sprintf("(%.1f) cvi", float64(-math.MinInt)), eRangecheck},
		{"(1e999) cvi", eRangecheck},
	}
	for _, c := range cases {
		t.Run(c.prog, func(t *testing.T) {
			intp := NewInterpreter()
			err := intp.ExecuteString(c.prog)
			if psErr, ok := errors.AsType[*postScriptError](err); !ok || psErr.tp != c.tp {
				t.Fatalf("got %v, want %s", err, c.tp)
			}
		})
	}
}

func TestIntegerOverflowToReal(t *testing.T) {
	minInt := fmt.Sprint(math.MinInt)
	want := Real(-float64(math.MinInt))
	for _, prog := range []string{
		"-1 " + minInt + " mul",
		minInt + " -1 mul",
		"0 " + minInt + " sub",
		minInt + " neg",
		minInt + " abs",
	} {
		t.Run(prog, func(t *testing.T) {
			intp, err := run(prog, 1)
			if err != nil {
				t.Fatal(err)
			}
			if intp.Stack[0] != want {
				t.Errorf("got %v, want %v", intp.Stack[0], want)
			}
		})
	}
}

func TestCmdDiv(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(3), Integer(2), Real(1.5)},
		{Integer(4), Integer(2), Real(2.0)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("div")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("div(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestCmdExp(t *testing.T) {
	type testCase struct {
		base, exp Object
		out       Object
	}
	cases := []testCase{
		{Integer(9), Real(0.5), Real(3.0)},
		{Integer(-9), Integer(-1), Real(-1.0 / 9.0)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.base, c.exp}
		err := intp.ExecuteString("exp")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if result, ok := intp.Stack[0].(Real); !ok {
			t.Fatalf("result is not Real: %T", intp.Stack[0])
		} else if math.Abs(float64(result-c.out.(Real))) > 1e-5 {
			t.Fatalf("exp(%v, %v): %v != %v", c.base, c.exp, result, c.out)
		}
	}
}

func TestCmdFloor(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Real(3.2), Real(3.0)},
		{Real(-4.8), Real(-5.0)},
		{Integer(99), Integer(99)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("floor")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("floor(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdGe(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Real(4.2), Integer(4), Boolean(true)},
		{String("abc"), String("d"), Boolean(false)},
		{String("aba"), String("ab"), Boolean(true)},
		{String("aba"), String("aba"), Boolean(true)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("ge")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("ge(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestCmdGt(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(5), Integer(4), Boolean(true)},
		{Integer(3), Integer(4), Boolean(false)},
		{String("abc"), String("ab"), Boolean(true)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("gt")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("gt(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestCmdIdiv(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(3), Integer(2), Integer(1)},
		{Integer(4), Integer(2), Integer(2)},
		{Integer(-5), Integer(2), Integer(-2)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("idiv")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("idiv(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestCmdLe(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(3), Integer(4), Boolean(true)},
		{Integer(5), Integer(4), Boolean(false)},
		{String("abc"), String("abd"), Boolean(true)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("le")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("le(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestCmdLn(t *testing.T) {
	type testCase struct {
		in  Object
		out Real
	}
	cases := []testCase{
		{Integer(10), Real(2.30259)},
		{Integer(100), Real(4.60517)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("ln")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if result, ok := intp.Stack[0].(Real); !ok {
			t.Fatalf("result is not Real: %T", intp.Stack[0])
		} else if math.Abs(float64(result-c.out)) > 1e-4 {
			t.Fatalf("ln(%v): %v != %v", c.in, result, c.out)
		}
	}
}

func TestCmdLog(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Integer(10), Real(1.0)},
		{Integer(100), Real(2.0)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("log")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("log(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdLt(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(3), Integer(4), Boolean(true)},
		{Integer(5), Integer(4), Boolean(false)},
		{String("abc"), String("abd"), Boolean(true)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("lt")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("lt(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestCmdMod(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Integer(5), Integer(3), Integer(2)},
		{Integer(5), Integer(2), Integer(1)},
		{Integer(-5), Integer(3), Integer(-2)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("mod")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("mod(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestCmdNeg(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Real(4.5), Real(-4.5)},
		{Integer(-3), Integer(3)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("neg")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("neg(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdRound(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Real(3.2), Real(3.0)},
		{Real(6.5), Real(7.0)},
		{Real(-4.8), Real(-5.0)},
		{Real(-6.5), Real(-6.0)},
		{Real(-0.5), Real(0.0)},
		{Real(0.49999999999999994), Real(0.0)},
		{Real(-0.49999999999999994), Real(0.0)},
		{Real(4503599627370497), Real(4503599627370497)},
		{Real(-4503599627370497), Real(-4503599627370497)},
		{Integer(99), Integer(99)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("round")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("round(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdSin(t *testing.T) {
	type testCase struct {
		angle Object
		out   Real
	}
	cases := []testCase{
		{Integer(0), Real(0.0)},
		{Integer(90), Real(1.0)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.angle}
		err := intp.ExecuteString("sin")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if result, ok := intp.Stack[0].(Real); !ok {
			t.Fatalf("result is not Real: %T", intp.Stack[0])
		} else if math.Abs(float64(result-c.out)) > 1e-10 {
			t.Fatalf("sin(%v): %v != %v", c.angle, result, c.out)
		}
	}
}

func TestCmdSqrt(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Integer(9), Real(3.0)},
		{Integer(16), Real(4.0)},
		{Real(2.25), Real(1.5)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("sqrt")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("sqrt(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdTruncate(t *testing.T) {
	type testCase struct {
		in  Object
		out Object
	}
	cases := []testCase{
		{Real(3.2), Real(3.0)},
		{Real(-4.8), Real(-4.0)},
		{Integer(99), Integer(99)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.in}
		err := intp.ExecuteString("truncate")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("truncate(%v): %v != %v", c.in, intp.Stack[0], c.out)
		}
	}
}

func TestCmdXor(t *testing.T) {
	type testCase struct {
		a, b Object
		out  Object
	}
	cases := []testCase{
		{Boolean(true), Boolean(true), Boolean(false)},
		{Boolean(true), Boolean(false), Boolean(true)},
		{Boolean(false), Boolean(true), Boolean(true)},
		{Boolean(false), Boolean(false), Boolean(false)},
		{Integer(7), Integer(3), Integer(4)},
		{Integer(12), Integer(3), Integer(15)},
	}
	for _, c := range cases {
		intp := NewInterpreter()
		intp.Stack = []Object{c.a, c.b}
		err := intp.ExecuteString("xor")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if intp.Stack[0] != c.out {
			t.Fatalf("xor(%v, %v): %v != %v", c.a, c.b, intp.Stack[0], c.out)
		}
	}
}

func TestArithmeticUndefinedResult(t *testing.T) {
	programs := []string{
		"10 400 exp",
		"1e200 dup mul",
		"1e308 1e308 add",
		"1e308 neg 1e308 sub",
		"1e300 1e-300 div",
		fmt.Sprint(math.MinInt) + " -1 idiv",
	}
	for _, prog := range programs {
		t.Run(prog, func(t *testing.T) {
			intp := NewInterpreter()
			err := intp.ExecuteString(prog)
			if psErr, ok := errors.AsType[*postScriptError](err); !ok || psErr.tp != eUndefinedresult {
				t.Fatalf("got %v, want undefinedresult", err)
			}
		})
	}
}

func TestNull(t *testing.T) {
	cases := []struct {
		code string
		want Object
	}{
		{"null", nil},
		{"/null load", nil},
		{"null type", Operator("nulltype")},
		{"systemdict /null known", Boolean(true)},
		{"null null eq", Boolean(true)},
		{"null 0 eq", Boolean(false)},
		{"currentfile null eq", Boolean(false)},
		{"1 array 0 get null eq", Boolean(true)},
	}
	for _, c := range cases {
		intp, err := run(c.code, 1)
		if err != nil {
			t.Errorf("%s: %v", c.code, err)
			continue
		}
		if got := intp.Stack[0]; got != c.want {
			t.Errorf("%s: got %v, want %v", c.code, got, c.want)
		}
	}
}

func TestCurrentfile(t *testing.T) {
	intp, err := run("currentfile currentfile eq currentfile type", 2)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Boolean(true), Operator("filetype")}); d != "" {
		t.Error(d)
	}
}

func TestClosefile(t *testing.T) {
	cases := []struct {
		code string
		want []Object
	}{
		// tokens after closefile are not read
		{"1 currentfile closefile 2", []Object{Integer(1)}},
		// the running procedure continues
		{"{ currentfile closefile 1 } exec 2", []Object{Integer(1)}},
		// closing twice has no effect
		{"{ currentfile dup closefile closefile 1 } exec", []Object{Integer(1)}},
		// a closed file reads as end-of-file
		{"{ currentfile dup closefile 3 string readstring } exec",
			[]Object{String{}, Boolean(false)}},
		{"{ currentfile dup closefile eexec 1 } exec", []Object{Integer(1)}},
	}
	for _, c := range cases {
		intp, err := run(c.code, len(c.want))
		if err != nil {
			t.Errorf("%s: %v", c.code, err)
			continue
		}
		if d := cmp.Diff(intp.Stack, c.want); d != "" {
			t.Errorf("%s: %s", c.code, d)
		}
	}
}

func TestFileClosedAfterExecute(t *testing.T) {
	intp := NewInterpreter()
	if err := intp.ExecuteString("/f currentfile def"); err != nil {
		t.Fatal(err)
	}
	if err := intp.ExecuteString("f 3 string readstring"); err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{String{}, Boolean(false)}); d != "" {
		t.Error(d)
	}
}

func TestFileOperandErrors(t *testing.T) {
	for _, code := range []string{
		"null closefile",
		"null eexec",
		"null 3 string readstring",
		"currentfile () readstring",
	} {
		if _, err := run(code, 0); err == nil {
			t.Errorf("%s: no error", code)
		}
	}
}

// eexecEncrypter encrypts data for eexec.  Successive calls to hex continue
// the same encrypted stream.
type eexecEncrypter struct {
	r uint16
}

// hex encrypts plain and returns the ciphertext in hexadecimal.
func (e *eexecEncrypter) hex(plain string) string {
	var out strings.Builder
	for _, p := range []byte(plain) {
		c := p ^ byte(e.r>>8)
		e.r = (uint16(c)+e.r)*eexecC1 + eexecC2
		fmt.Fprintf(&out, "%02x", c)
	}
	return out.String()
}

// eexecHex encrypts the initialisation vector "abcd", followed by plain, and
// returns the ciphertext in hexadecimal.
func eexecHex(plain string) string {
	e := &eexecEncrypter{r: eexecR}
	return e.hex("abcd" + plain)
}

func TestEexecFile(t *testing.T) {
	inner := "currentfile outer eq currentfile type currentdict systemdict eq " +
		"currentfile closefile "
	code := "/outer currentfile def currentfile eexec\n" + eexecHex(inner) +
		"\ncurrentdict userdict eq"
	intp, err := run(code, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []Object{Boolean(false), Operator("filetype"), Boolean(true), Boolean(true)}
	if d := cmp.Diff(intp.Stack, want); d != "" {
		t.Error(d)
	}
}

func TestEexecSourceClosed(t *testing.T) {
	cases := []struct {
		plain string
		want  []Object
	}{
		// closing the source ends the encrypted section, too
		{"outer closefile 7 ", []Object{}},
		// the filter reads as end-of-file once its source is closed
		{"{ outer closefile currentfile 3 string readstring } exec ",
			[]Object{String{}, Boolean(false)}},
	}
	for _, c := range cases {
		code := "/outer currentfile def currentfile eexec\n" + eexecHex(c.plain) +
			"\n8"
		intp, err := run(code, len(c.want))
		if err != nil {
			t.Errorf("%q: %v", c.plain, err)
			continue
		}
		if d := cmp.Diff(intp.Stack, c.want, cmpopts.EquateEmpty()); d != "" {
			t.Errorf("%q: %s", c.plain, d)
		}
	}
}

func TestEexecReadSource(t *testing.T) {
	// Reading from the source of an active filter returns the raw bytes,
	// and decryption continues after them.
	e := &eexecEncrypter{r: eexecR}
	code := "/outer currentfile def currentfile eexec\n" +
		e.hex("abcd outer 3 string readstring ") + "XYZ" +
		e.hex(" currentfile closefile ") + "\n8"
	intp, err := run(code, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []Object{String("XYZ"), Boolean(true), Integer(8)}
	if d := cmp.Diff(intp.Stack, want); d != "" {
		t.Error(d)
	}
}

func TestEexecCRLFClose(t *testing.T) {
	// The look-ahead past "closefile" must not leak into the cleartext.
	code := "currentfile eexec\n" + eexecHex("currentfile closefile\r") +
		"\n0000000000000000 8"
	intp, err := run(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(0), Integer(8)}); d != "" {
		t.Error(d)
	}
}

func TestReadstringPosition(t *testing.T) {
	cases := []struct {
		code string
		want []Object
	}{
		{"{currentfile 2 string readstring pop currentfile 2 string readstring pop} exec abcd",
			[]Object{String("ab"), String("cd")}},
		// only the first white-space character ends the token
		{"currentfile 1 string readstring  5",
			[]Object{String(" "), Boolean(true), Integer(5)}},
		{"currentfile 1 string readstring\n\n5",
			[]Object{String("\n"), Boolean(true), Integer(5)}},
		// CR LF counts as one character
		{"currentfile 1 string readstring\r\n5", []Object{String("5"), Boolean(true)}},
		{"currentfile 1 string readstring\r5", []Object{String("5"), Boolean(true)}},
		// a delimiter is not consumed
		{"currentfile 1 string readstring(5",
			[]Object{String("("), Boolean(true), Integer(5)}},
		{"/a /readstring load def currentfile 1 string a\n5",
			[]Object{String("5"), Boolean(true)}},
		{"currentfile 3 string readstring", []Object{String{}, Boolean(false)}},
	}
	for _, c := range cases {
		intp, err := run(c.code, len(c.want))
		if err != nil {
			t.Errorf("%q: %v", c.code, err)
			continue
		}
		if d := cmp.Diff(intp.Stack, c.want); d != "" {
			t.Errorf("%q: %s", c.code, d)
		}
	}
}

func TestRepeatExitStop(t *testing.T) {
	// stop ends the program, not just the loop
	if _, err := run("3 { stop } repeat 1", 0); err != nil {
		t.Error(err)
	}
	intp, err := run("0 3 { 1 add exit } repeat", 1)
	if err != nil {
		t.Fatal(err)
	}
	if intp.Stack[0] != Integer(1) {
		t.Errorf("got %v, want 1", intp.Stack[0])
	}
}

func TestCmdType(t *testing.T) {
	cases := []struct {
		code string
		want Operator
	}{
		{"[] type", "arraytype"},
		{"{} type", "arraytype"},
		{"true type", "booleantype"},
		{"1 dict type", "dicttype"},
		{"currentfile type", "filetype"},
		{"1 type", "integertype"},
		{"/a type", "nametype"},
		{"/a cvx type", "nametype"},
		{"null type", "nulltype"},
		{"/add load type", "operatortype"},
		{"1.5 type", "realtype"},
		{"() type", "stringtype"},
		{"mark type", "marktype"},
	}
	for _, c := range cases {
		intp, err := run(c.code, 1)
		if err != nil {
			t.Errorf("%s: %v", c.code, err)
			continue
		}
		if got := intp.Stack[0]; got != c.want {
			t.Errorf("%s: got %v, want %v", c.code, got, c.want)
		}
	}
}

func TestEexecEmpty(t *testing.T) {
	intp, err := run("1 currentfile eexec", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.DictStack) != 2 {
		t.Errorf("dict stack has %d entries, want 2", len(intp.DictStack))
	}
}

func TestEexecNested(t *testing.T) {
	code := "currentfile eexec\n" + eexecHex("currentfile eexec\n")
	if _, err := run(code, 0); err == nil {
		t.Error("nested eexec: no error")
	}
}

// errType returns the PostScript error name of err, or "" if err is not
// a PostScript error.
func errType(err error) Name {
	if e, ok := errors.AsType[*postScriptError](err); ok {
		return e.tp
	}
	return ""
}

// stackTest runs code and checks the resulting operand stack, or the type
// of the resulting error.
type stackTest struct {
	code string
	want []Object
	err  Name
}

func runStackTests(t *testing.T, tests []stackTest) {
	t.Helper()
	for _, tc := range tests {
		intp := NewInterpreter()
		err := intp.ExecuteString(tc.code)
		if tc.err != "" {
			if got := errType(err); got != tc.err {
				t.Errorf("%q: got error %v, want %s", tc.code, err, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: %v", tc.code, err)
			continue
		}
		if d := cmp.Diff(tc.want, intp.Stack, cmpopts.EquateEmpty()); d != "" {
			t.Errorf("%q: wrong stack (-want +got):\n%s", tc.code, d)
		}
	}
}

func TestStringDictKeys(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "(a) 1 def a", want: []Object{Integer(1)}},
		{code: "/a cvx 1 def a", want: []Object{Integer(1)}},
		{code: "1 dict dup (k) 5 put /k get", want: []Object{Integer(5)}},
		{code: "<< /a 1 >> (a) get", want: []Object{Integer(1)}},
		{code: "<< (a) 1 >> /a get", want: []Object{Integer(1)}},
		{code: "<< /a 1 >> (a) known", want: []Object{Boolean(true)}},
		{code: "<< /a 1 >> (b) known", want: []Object{Boolean(false)}},
		{code: "/a 7 def (a) load", want: []Object{Integer(7)}},
		{code: "/a 7 def (a) where exch pop", want: []Object{Boolean(true)}},
		{code: "(zz) where", want: []Object{Boolean(false)}},
		{code: "(F) 1 dict definefont pop FontDirectory /F known", want: []Object{Boolean(true)}},
		{code: "1 2 def", err: eTypecheck},
		{code: "1 dict 1 2 put", err: eTypecheck},
		{code: "<< 1 2 >>", err: eTypecheck},
		{code: "1 dict [] known", err: eTypecheck},
	})
}

func TestCmdExec(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "1 exec", want: []Object{Integer(1)}},
		{code: "(abc) exec", want: []Object{String("abc")}},
		{code: "/a exec", want: []Object{Name("a")}},
		{code: "[1 2] exec", want: []Object{Array{Integer(1), Integer(2)}}},
		{code: "1 2 /add cvx exec", want: []Object{Integer(3)}},
		{code: "1 2 /add load exec", want: []Object{Integer(3)}},
		{code: "/p {7} def /p cvx exec", want: []Object{Integer(7)}},
		{code: "{1 2 add} exec", want: []Object{Integer(3)}},
		{code: "/zz cvx exec", err: eUndefined},
		{code: "1 0 /div load exec", err: eUndefinedresult},
		{code: "exec", err: eStackunderflow},
	})
}

func TestProcedureOperands(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "true 5 if", err: eTypecheck},
		{code: "false 5 if", err: eTypecheck},
		{code: "true /add if", err: eTypecheck},
		{code: "true {1} 2 ifelse", err: eTypecheck},
		{code: "true 1 {2} ifelse", err: eTypecheck},
		{code: "5 loop", err: eTypecheck},
		{code: "0 1 2 5 for", err: eTypecheck},
		{code: "true {1} if", want: []Object{Integer(1)}},
		{code: "{exit} loop", want: nil},
	})
}

func TestCmdForReal(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "0 0.5 1 {} for", want: []Object{Real(0), Real(0.5), Real(1)}},
		{code: "0.0 1 2 {} for", want: []Object{Real(0), Real(1), Real(2)}},
		{code: "1 1 2.5 {} for", want: []Object{Integer(1), Integer(2)}},
		{code: "3 -1 1.5 {} for", want: []Object{Integer(3), Integer(2)}},
		{code: "1 -0.5 0 {} for", want: []Object{Real(1), Real(0.5), Real(0)}},
		{code: "3 1 1 {} for", want: nil},
		{code: "1 -1 3 {} for", want: nil},
		{code: "0 1 3 {dup 1 eq {exit} if} for", want: []Object{Integer(0), Integer(1)}},
		{code: "(a) 1 2 {} for", err: eTypecheck},
		{code: "0 (a) 2 {} for", err: eTypecheck},
		{code: "0 1 (a) {} for", err: eTypecheck},
	})
}

// TestCmdForOverflow checks that the loop ends when the control variable
// leaves the range of integers.
func TestCmdForOverflow(t *testing.T) {
	for _, tc := range []struct {
		initial, increment, limit Integer
		want                      int
	}{
		{math.MaxInt - 2, 2, math.MaxInt, 2},
		{math.MinInt + 2, -2, math.MinInt, 2},
	} {
		intp := NewInterpreter()
		intp.MaxOps = 1000
		intp.Stack = []Object{tc.initial, tc.increment, tc.limit}
		err := intp.ExecuteString("{} for")
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != tc.want {
			t.Errorf("got %d values, want %d", len(intp.Stack), tc.want)
		}
	}
}

func TestStackOperators(t *testing.T) {
	arr := Array{Integer(1), Integer(2), Integer(3)}
	runStackTests(t, []stackTest{
		{code: "1 2 3 clear", want: nil},
		{code: "clear", want: nil},
		{code: "mark 1 2 counttomark", want: []Object{theMark, Integer(1), Integer(2), Integer(2)}},
		{code: "mark counttomark", want: []Object{theMark, Integer(0)}},
		{code: "1 2 counttomark", err: eUnmatchedmark},
		{code: "[1 2 3] aload", want: []Object{Integer(1), Integer(2), Integer(3), arr}},
		{code: "{1 2 3} aload pop", want: []Object{Integer(1), Integer(2), Integer(3)}},
		{code: "[] aload", want: []Object{Array{}}},
		{code: "1 aload", err: eTypecheck},
		{code: "aload", err: eStackunderflow},
		{code: "600 array aload", err: eStackoverflow},
		{code: "0 1 2 3 3 array astore", want: []Object{Integer(0), arr}},
		{code: "0 array astore", want: []Object{Array{}}},
		{code: "1 2 3 array astore", err: eStackunderflow},
		{code: "1 2 3 astore", err: eTypecheck},
		{code: "astore", err: eStackunderflow},
		{code: "countdictstack", want: []Object{Integer(2)}},
		{code: "1 dict begin countdictstack end", want: []Object{Integer(3)}},
	})
}

// TestCmdAstoreShared checks that astore fills the array it is given.
func TestCmdAstoreShared(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "/a 2 array def 5 6 a astore pop a 1 get", want: []Object{Integer(6)}},
	})
}

func TestCmdStoreUndef(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "/x 1 def /x 2 store x", want: []Object{Integer(2)}},
		{code: "/x 1 def 1 dict begin /x 2 store end x", want: []Object{Integer(2)}},
		{code: "1 dict begin /y 2 store currentdict /y known end", want: []Object{Boolean(true)}},
		{code: "1 dict begin /y 2 store end userdict /y known", want: []Object{Boolean(false)}},
		{code: "(x) 3 store x", want: []Object{Integer(3)}},
		{code: "1 2 store", err: eTypecheck},
		{code: "/x store", err: eStackunderflow},
		{code: "<< /a 1 >> dup /a undef /a known", want: []Object{Boolean(false)}},
		{code: "<< /a 1 >> dup (a) undef length", want: []Object{Integer(0)}},
		{code: "<< /a 1 >> /b undef", want: nil},
		{code: "1 /a undef", err: eTypecheck},
		{code: "1 dict 1 undef", err: eTypecheck},
		{code: "/a undef", err: eStackunderflow},
	})
}

func TestConversionOperators(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "(abc) cvn", want: []Object{Name("abc")}},
		{code: "() cvn", want: []Object{Name("")}},
		{code: "/a cvn", err: eTypecheck},
		{code: "cvn", err: eStackunderflow},

		{code: "/abc cvx cvlit", want: []Object{Name("abc")}},
		{code: "{1 2} cvlit", want: []Object{Array{Integer(1), Integer(2)}}},
		{code: "5 cvlit", want: []Object{Integer(5)}},
		{code: "cvlit", err: eStackunderflow},

		{code: "{1} xcheck [1] xcheck", want: []Object{Boolean(true), Boolean(false)}},
		{code: "/a xcheck /a cvx xcheck", want: []Object{Boolean(false), Boolean(true)}},
		{code: "/add load xcheck 1 xcheck", want: []Object{Boolean(true), Boolean(false)}},
		{code: "xcheck", err: eStackunderflow},

		{code: "(a) rcheck [1] rcheck {1} rcheck 1 dict rcheck currentfile rcheck",
			want: []Object{Boolean(true), Boolean(true), Boolean(true), Boolean(true), Boolean(true)}},
		{code: "(a) wcheck [1] wcheck 1 dict wcheck currentfile wcheck",
			want: []Object{Boolean(true), Boolean(true), Boolean(true), Boolean(false)}},
		{code: "1 rcheck", err: eTypecheck},
		{code: "/a wcheck", err: eTypecheck},
		{code: "rcheck", err: eStackunderflow},

		{code: "123 10 string cvs", want: []Object{String("123")}},
		{code: "-7 10 string cvs", want: []Object{String("-7")}},
		{code: "1.5 10 string cvs", want: []Object{String("1.5")}},
		{code: "2.0 10 string cvs", want: []Object{String("2.0")}},
		{code: "1e20 10 string cvs", want: []Object{String("1e+20")}},
		{code: "true 10 string cvs", want: []Object{String("true")}},
		{code: "(ab) 10 string cvs", want: []Object{String("ab")}},
		{code: "() 10 string cvs", want: []Object{String("")}},
		{code: "/abc 10 string cvs", want: []Object{String("abc")}},
		{code: "/abc cvx 10 string cvs", want: []Object{String("abc")}},
		{code: "/add load 10 string cvs", want: []Object{String("add")}},
		{code: "[1] 20 string cvs", want: []Object{String("--nostringval--")}},
		{code: "null 20 string cvs", want: []Object{String("--nostringval--")}},
		{code: "123 2 string cvs", err: eRangecheck},
		{code: "123 /a cvs", err: eTypecheck},
		{code: "10 string cvs", err: eStackunderflow},

		{code: "255 16 10 string cvrs", want: []Object{String("FF")}},
		{code: "5 2 10 string cvrs", want: []Object{String("101")}},
		{code: "-1 16 10 string cvrs", want: []Object{String("FFFFFFFF")}},
		{code: "-12 10 10 string cvrs", want: []Object{String("-12")}},
		{code: "1.5 10 10 string cvrs", want: []Object{String("1.5")}},
		{code: "17.9 16 10 string cvrs", want: []Object{String("11")}},
		{code: "1e30 16 40 string cvrs", err: eRangecheck},
		{code: "255 16 1 string cvrs", err: eRangecheck},
		{code: "255 1 10 string cvrs", err: eRangecheck},
		{code: "255 37 10 string cvrs", err: eRangecheck},
		{code: "(a) 16 10 string cvrs", err: eTypecheck},
		{code: "255 16.0 10 string cvrs", err: eTypecheck},
		{code: "255 16 /a cvrs", err: eTypecheck},
		{code: "16 10 string cvrs", err: eStackunderflow},
	})
}

// TestCmdCvsShared checks that cvs writes into the string it is given.
func TestCmdCvsShared(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "/s (xxxxx) def 42 s cvs pop s", want: []Object{String("42xxx")}},
		{code: "/s (xxxxx) def 255 16 s cvrs pop s", want: []Object{String("FFxxx")}},
		{code: "/s (abc) def s s cvs", want: []Object{String("abc")}},
	})
}

func TestCmdSearch(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "(abcdef) (cd) search",
			want: []Object{String("ef"), String("cd"), String("ab"), Boolean(true)}},
		{code: "(abc) (a) search",
			want: []Object{String("bc"), String("a"), String(""), Boolean(true)}},
		{code: "(abc) (x) search", want: []Object{String("abc"), Boolean(false)}},
		{code: "(abc) /a search", err: eTypecheck},
		{code: "1 (a) search", err: eTypecheck},
		{code: "(a) search", err: eStackunderflow},
		{code: "(abc) (ab) anchorsearch",
			want: []Object{String("c"), String("ab"), Boolean(true)}},
		{code: "(abc) (bc) anchorsearch", want: []Object{String("abc"), Boolean(false)}},
		{code: "(abc) 1 anchorsearch", err: eTypecheck},
		{code: "(a) anchorsearch", err: eStackunderflow},
		// the results are substrings of the string searched
		{code: "/s (abcdef) def s (cd) search pop pop pop 0 88 put s",
			want: []Object{String("abcdXf")}},
	})
}

func TestModeOperators(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "currentpacking", want: []Object{Boolean(false)}},
		{code: "true setpacking currentpacking", want: []Object{Boolean(true)}},
		{code: "currentpacking true setpacking setpacking currentpacking",
			want: []Object{Boolean(false)}},
		{code: "1 setpacking", err: eTypecheck},
		{code: "setpacking", err: eStackunderflow},
		{code: "currentglobal", want: []Object{Boolean(false)}},
		{code: "true setglobal currentglobal", want: []Object{Boolean(true)}},
		{code: "1 setglobal", err: eTypecheck},
		{code: "setglobal", err: eStackunderflow},
		{code: "systemdict /currentpacking known", want: []Object{Boolean(true)}},
	})
}

func TestOutputOperators(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "1 (x) print flush", want: []Object{Integer(1)}},
		{code: "1 print", err: eTypecheck},
		{code: "print", err: eStackunderflow},
		{code: "1 2 = [3] ==", want: []Object{Integer(1)}},
		{code: "=", err: eStackunderflow},
		{code: "==", err: eStackunderflow},
	})
}

func TestSystemdictConstants(t *testing.T) {
	runStackTests(t, []stackTest{
		{code: "languagelevel", want: []Object{Integer(3)}},
		{code: "version type", want: []Object{Operator("stringtype")}},
		{code: "ISOLatin1Encoding length", want: []Object{Integer(256)}},
		{code: "ISOLatin1Encoding 65 get", want: []Object{Name("A")}},
		{code: "ISOLatin1Encoding 39 get", want: []Object{Name("quoteright")}},
		{code: "ISOLatin1Encoding 0 get", want: []Object{Name(".notdef")}},
	})
}

func TestMatrixOperators(t *testing.T) {
	ident := Array{Real(1), Real(0), Real(0), Real(1), Real(0), Real(0)}
	runStackTests(t, []stackTest{
		{code: "matrix", want: []Object{ident}},
		{code: "6 array identmatrix", want: []Object{ident}},
		{code: "/m 6 array def m identmatrix pop m", want: []Object{ident}},
		{code: "5 array identmatrix", err: eRangecheck},
		{code: "1 identmatrix", err: eTypecheck},
		{code: "identmatrix", err: eStackunderflow},
		{code: "[2 0 0 2 0 0] [1 0 0 1 5 5] matrix concatmatrix",
			want: []Object{Array{Real(2), Real(0), Real(0), Real(2), Real(5), Real(5)}}},
		{code: "[1 0 0 1 5 5] [2 0 0 3 1 1] matrix concatmatrix",
			want: []Object{Array{Real(2), Real(0), Real(0), Real(3), Real(11), Real(16)}}},
		{code: "[0 1 -1 0 0 0] [1 2 3 4 5 6] 6 array concatmatrix",
			want: []Object{Array{Real(3), Real(4), Real(-1), Real(-2), Real(5), Real(6)}}},
		{code: "/m 6 array def matrix matrix m concatmatrix pop m", want: []Object{ident}},
		{code: "[1e200 0 0 1 0 0] [1e200 0 0 1 0 0] matrix concatmatrix", err: eUndefinedresult},
		{code: "[1 0 0 1 0] matrix matrix concatmatrix", err: eRangecheck},
		{code: "matrix [1 0 0 1 0 0 0] matrix concatmatrix", err: eRangecheck},
		{code: "matrix matrix 5 array concatmatrix", err: eRangecheck},
		{code: "[1 0 0 1 0 (a)] matrix matrix concatmatrix", err: eTypecheck},
		{code: "1 matrix matrix concatmatrix", err: eTypecheck},
		{code: "matrix matrix 1 concatmatrix", err: eTypecheck},
		{code: "matrix matrix concatmatrix", err: eStackunderflow},
	})
}

// TestBulkOperatorsCharged checks that the new operators which touch many
// elements count against MaxOps in proportion.
func TestBulkOperatorsCharged(t *testing.T) {
	for _, code := range []string{
		"/a 400 array def {a aload pop clear} loop",
		"/s 60000 string def {s (x) search pop} loop",
		"/s 60000 string def {s cvn pop} loop",
	} {
		intp := NewInterpreter()
		intp.MaxOps = 100_000
		err := intp.ExecuteString(code)
		if err != ErrExecutionLimitExceeded {
			t.Errorf("%q: got %v, want execution limit", code, err)
		}
		if intp.NumOps > 2*intp.MaxOps {
			t.Errorf("%q: %d operations counted", code, intp.NumOps)
		}
	}
}

// TestCmdCvnMemory checks that names created by cvn count against MaxMemory.
func TestCmdCvnMemory(t *testing.T) {
	intp := NewInterpreter()
	intp.MaxMemory = 1 << 20
	err := intp.ExecuteString("/s 60000 string def {s cvn pop} loop")
	if errType(err) != eLimitcheck {
		t.Errorf("got %v, want limitcheck", err)
	}
}

// TestBulkOperatorLimit checks that a single bulk operator can exhaust
// MaxOps.
func TestBulkOperatorLimit(t *testing.T) {
	big := make([]Object, 400)
	for i := range big {
		big[i] = Integer(i)
	}
	long := String(strings.Repeat("x", 400))
	withMark := append([]Object{theMark}, big...)

	for _, tc := range []struct {
		code  string
		stack []Object
	}{
		{"clear", big},
		{"counttomark", withMark},
		{"counttomark", big},
		{"aload", []Object{Array(big)}},
		{"astore", append(slices.Clone(big), make(Array, 400))},
		{"cvs", []Object{long, make(String, 400)}},
		{"def", []Object{long, Integer(1)}},
	} {
		intp := NewInterpreter()
		intp.MaxOps = 3
		intp.Stack = slices.Clone(tc.stack)
		err := intp.ExecuteString(tc.code)
		if err != ErrExecutionLimitExceeded {
			t.Errorf("%s: got %v, want execution limit", tc.code, err)
		}
	}
}

// TestCmdMatrixMemory checks that matrix counts against MaxMemory.
func TestCmdMatrixMemory(t *testing.T) {
	intp := NewInterpreter()
	intp.MaxMemory = 1000
	err := intp.ExecuteString("{matrix pop} loop")
	if errType(err) != eLimitcheck {
		t.Errorf("got %v, want limitcheck", err)
	}
}
