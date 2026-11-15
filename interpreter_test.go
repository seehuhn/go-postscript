// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestArray(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("[ 1 2 3 ]")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 {
		t.Fatal("len(intp.Stack) != 1")
	}
	if d := cmp.Diff(intp.Stack[0], Array{Integer(1), Integer(2), Integer(3)}); d != "" {
		t.Fatal(d)
	}
}

func TestProcedures(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/x {0 eq {9} if} def 0 x 1 2 x")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(9), Integer(1)}); d != "" {
		t.Fatal(d)
	}
}

func TestIfElse(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("[ 1 false {2 ]} {] 2} ifelse")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Array{Integer(1)}, Integer(2)}); d != "" {
		t.Error(d)
	}
}

// TestProcedures2 tests live-patching of procedures.
func TestProcedures2(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString(`
	/test
	{0 eq {1 2} {3 4} ifelse}
	dup 2 get 1 2 dict put
	def
	0 test
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 2 {
		t.Fatal("len(intp.Stack) != 2")
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(1), Dict{}}); d != "" {
		t.Fatal(d)
	}
}

func TestNestedProcedures(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/a {{1 2} {3 4} ifelse} def false a")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Integer(3), Integer(4)}); d != "" {
		t.Fatal(d)
	}
}

func TestNestedProcedures2(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/a { {[1 2]} {3} ifelse } def true a false a")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Array{Integer(1), Integer(2)}, Integer(3)}); d != "" {
		t.Fatal(d)
	}
}

// TestTrailingProcedure checks that a procedure in the last position of
// an executed procedure is pushed, not executed.
func TestTrailingProcedure(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("{ 1 {2} } exec true { {undefinedname} } if")
	if err != nil {
		t.Fatal(err)
	}
	want := []Object{Integer(1), Procedure{Integer(2)}, Procedure{Operator("undefinedname")}}
	if d := cmp.Diff(want, intp.Stack); d != "" {
		t.Fatalf("stack mismatch (-want +got):\n%s", d)
	}
}

// TestBraceNames checks that braces only delimit procedures when read from
// a file: inside an executed procedure, "{" and "}" are ordinary names.
func TestBraceNames(t *testing.T) {
	for _, name := range []Operator{"{", "}"} {
		for _, proc := range []Procedure{
			{name, Integer(1)}, // early position
			{Integer(1), name}, // tail position
		} {
			intp := NewInterpreter()
			intp.UserDict["p"] = proc
			err := intp.ExecuteString("p")
			if psErr, ok := errors.AsType[*postScriptError](err); !ok || psErr.tp != eUndefined {
				t.Errorf("%v: got error %v, want undefined", proc, err)
			}
		}
	}
}

// TestUnbalancedBraces checks that procedures do not extend past the end
// of the file which contains them, and that an unfinished procedure leaves
// the operand stack unchanged.
func TestUnbalancedBraces(t *testing.T) {
	for _, code := range []string{"5 { 1", "5 1 }", "5 { { } 2", "5 { 1 (abc"} {
		intp := NewInterpreter()
		err := intp.ExecuteString(code)
		if psErr, ok := errors.AsType[*postScriptError](err); !ok || psErr.tp != eSyntaxerror {
			t.Errorf("%q: got error %v, want syntaxerror", code, err)
		}
		if code != "5 1 }" {
			if d := cmp.Diff([]Object{Integer(5)}, intp.Stack); d != "" {
				t.Errorf("%q: stack mismatch (-want +got):\n%s", code, d)
			}
		}
	}

	intp := NewInterpreter()
	if err := intp.ExecuteString("{ 1"); err == nil {
		t.Error("unterminated procedure: got no error")
	}
	err := intp.ExecuteString("2 }")
	if psErr, ok := errors.AsType[*postScriptError](err); !ok || psErr.tp != eSyntaxerror {
		t.Errorf("procedure spans two files: got error %v, want syntaxerror", err)
	}
}

// TestLongProcedure checks that the length of a procedure is limited by the
// maximum array size, not by the operand stack depth.
func TestLongProcedure(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("{ " + strings.Repeat("1 ", maxArraySize) + "}")
	if err != nil {
		t.Fatal(err)
	}
	if len(intp.Stack) != 1 || len(intp.Stack[0].(Procedure)) != maxArraySize {
		t.Errorf("wrong result for a procedure of %d tokens", maxArraySize)
	}

	intp = NewInterpreter()
	err = intp.ExecuteString("{ " + strings.Repeat("1 ", maxArraySize+1) + "}")
	if psErr, ok := errors.AsType[*postScriptError](err); !ok || psErr.tp != eLimitcheck {
		t.Errorf("got error %v, want limitcheck", err)
	}
}

func FuzzInterpreter(f *testing.F) {
	f.Add("1 2 add")
	f.Add("12 array dup 5 6 put")
	f.Add("/a {{1 2} {3 4} ifelse} def false a")
	f.Add("0 1 1 4 {add} for")
	f.Add("0 1 1 4 {add} for")
	f.Add("[1 2 3] 1 get")
	f.Add("[1 2 3] dup 1 9 put")
	f.Add("<< /a 1 >> {} forall")
	f.Add("/a {a} def a")
	f.Add("/a {{}} def userdict /a get dup 0 exch put a")
	f.Add("/a {{} {}} def userdict /a get dup 0 exch put userdict /a get dup 1 exch put a")
	builtins := slices.Sorted(maps.Keys(makeSystemDict()))
	for _, name := range builtins {
		f.Add("1 [2 3] (four) " + string(name))
	}

	f.Fuzz(func(t *testing.T, a string) {
		// Just check that the interpreter does not crash or hang.
		intp := NewInterpreter()
		intp.MaxOps = 1000000
		err := intp.ExecuteString(a)
		if err == nil || err == io.EOF {
			return
		}
		_, ok := err.(*postScriptError)
		if ok {
			return
		}
		t.Errorf("%#v", err)
	})
}

// workProgram pushes n zeros and then runs op rolls times.
func workProgram(n, rolls int, op string) string {
	return fmt.Sprintf("%d { 0 } repeat %d { %s } repeat", n, rolls, op)
}

func numOps(t *testing.T, code string) (int, error) {
	t.Helper()
	intp := NewInterpreter()
	intp.maxStack = 1 << 20
	intp.MaxOps = 1_000_000
	err := intp.ExecuteString(code)
	return intp.NumOps, err
}

// TestWorkChargedRoll checks that the cost of roll grows with the number
// of elements moved, so that a deep stack cannot be rolled for free.
func TestWorkChargedRoll(t *testing.T) {
	small, err := numOps(t, workProgram(100, 1000, "100 1 roll"))
	if err != nil {
		t.Fatal(err)
	}
	large, err := numOps(t, workProgram(10000, 1000, "10000 1 roll"))
	if err != nil {
		t.Fatal(err)
	}
	if large < 10*small {
		t.Errorf("roll cost does not grow with depth: %d vs %d", small, large)
	}

	// beyond-budget canary: each roll touches 10^5 elements
	_, err = numOps(t, workProgram(100000, 10000, "100000 1 roll"))
	if !errors.Is(err, ErrExecutionLimitExceeded) {
		t.Errorf("got %v, want ErrExecutionLimitExceeded", err)
	}
}

// TestWorkChargedOthers checks that operators touching many elements or
// bytes are charged at least one step per workPerStep units of work.
func TestWorkChargedOthers(t *testing.T) {
	const twoStrings = "/s 60000 string def /u 60000 string def s u "
	cases := []struct {
		name string
		pre  string // setup
		op   string // the code whose cost is measured
		work int    // units of work done by op
	}{
		{"copy", "10000 { 0 } repeat", "10000 copy", 10000},
		{"cleartomark", "mark 10000 { 0 } repeat", "cleartomark", 10000},
		{"eq", twoStrings, "eq", 120000},
		{"ge", twoStrings, "ge", 120000},
		{"gt", twoStrings, "gt", 120000},
		{"le", twoStrings, "le", 120000},
		{"lt", twoStrings, "lt", 120000},
		{"string", "", "60000 string", 60000},
		{"array", "", "60000 array", 60000},
		{"putinterval", "/s 60000 string def /d 60000 string def d 0 s", "putinterval", 60000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			intp := NewInterpreter()
			intp.maxStack = 1 << 20
			if err := intp.ExecuteString(c.pre); err != nil {
				t.Fatal(err)
			}
			before := intp.NumOps
			if err := intp.ExecuteString(c.op); err != nil {
				t.Fatal(err)
			}
			if got := intp.NumOps - before; got < c.work/workPerStep {
				t.Errorf("cost %d is too low for %d units of work", got, c.work)
			}
		})
	}
}

// TestWorkLimit checks that a program doing little but moving large amounts
// of data is stopped by MaxOps.
func TestWorkLimit(t *testing.T) {
	intp := NewInterpreter()
	intp.MaxOps = 10_000
	err := intp.ExecuteString("/s 60000 string def 1000 { s s eq pop } repeat")
	if !errors.Is(err, ErrExecutionLimitExceeded) {
		t.Errorf("got %v, want ErrExecutionLimitExceeded", err)
	}
}

// TestWorkLimitBind checks that bind is charged for every distinct procedure
// it visits, even for procedures too short to be charged by length.
func TestWorkLimitBind(t *testing.T) {
	intp := NewInterpreter()
	intp.MaxOps = 2_000_000
	err := intp.ExecuteString("/p {} def 100000 { /p [ /p load ] cvx def } repeat")
	if err != nil {
		t.Fatal(err)
	}
	err = intp.ExecuteString("1000 { /p load bind pop } repeat")
	if !errors.Is(err, ErrExecutionLimitExceeded) {
		t.Errorf("got %v, want ErrExecutionLimitExceeded", err)
	}
}

// TestDSCAfterError checks that the DSC comments seen before an error are
// kept.
func TestDSCAfterError(t *testing.T) {
	intp := NewInterpreter()
	if err := intp.ExecuteString("%%Title: test\nundefinedname"); err == nil {
		t.Fatal("got no error")
	}
	if len(intp.DSC) != 1 || intp.DSC[0].Key != "Title" {
		t.Errorf("wrong DSC comments: %v", intp.DSC)
	}
}
