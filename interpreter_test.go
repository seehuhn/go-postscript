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
	"os"
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

func TestNestedProcedures(t *testing.T) {
	intp := NewInterpreter()
	err := intp.ExecuteString("/a { {[1 2]} {3} ifelse } def true a false a")
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(intp.Stack, []Object{Array{Integer(1), Integer(2)}, Integer(3)}); d != "" {
		t.Fatal(d)
	}
}

func TestXXX(t *testing.T) {
	intp := NewInterpreter()

	fd, err := os.Open("../type1/NimbusRoman-Regular.pfa")
	// fd, err := os.Open("../type1/cour.pfa")
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	err = intp.Execute(fd)
	if err != nil {
		t.Fatal(err)
	}

	for key, font := range intp.Fonts {
		fmt.Printf("# %s\n\n", key)
		for key, val := range font {
			if key == "Private" || key == "FontInfo" {
				fmt.Println(string(key) + ":")
				for k2, v2 := range val.(Dict) {
					valString := fmt.Sprint(v2)
					if len(valString) > 70 {
						valString = fmt.Sprintf("<%T>", val)
					}
					fmt.Println("  "+string(k2)+":", valString)
				}
				continue
			}
			valString := fmt.Sprint(val)
			if len(valString) > 70 {
				valString = fmt.Sprintf("<%T>", val)
			}
			fmt.Println(string(key)+":", valString)
		}
	}
}
