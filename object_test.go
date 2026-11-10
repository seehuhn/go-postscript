// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"testing"
)

func FuzzStrings(f *testing.F) {
	f.Add("hello world")
	f.Add("hello\nworld")
	f.Add("hello\rworld")
	f.Add("hello\r\nworld")
	f.Add("hello\n\rworld")
	f.Add("hello\\world")
	f.Add("hello(world(")
	f.Add("hello(world)")
	f.Add("hello)world(")
	f.Add("hello)world)")
	f.Fuzz(func(t *testing.T, a string) {
		s := String(a)
		ps := s.PS()
		intp := NewInterpreter()
		err := intp.ExecuteString(ps)
		if err != nil {
			t.Fatal(err)
		}
		if len(intp.Stack) != 1 {
			t.Fatalf("len(intp.Stack): %d != 1", len(intp.Stack))
		}
		if ss, ok := intp.Stack[0].(String); !ok || string(ss) != a {
			t.Fatalf("intp.Stack[0]: %q != %q", ss, a)
		}
	})
}
