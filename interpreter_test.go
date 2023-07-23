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
	"io"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/maps"
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
	builtins := maps.Keys(makeSystemDict())
	sort.Slice(builtins, func(i, j int) bool { return builtins[i] < builtins[j] })
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
