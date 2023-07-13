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
