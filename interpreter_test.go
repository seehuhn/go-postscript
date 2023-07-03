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
