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
	"strings"
)

type Object interface{}

type Integer int

type Real float64

type Boolean bool

type String []byte

func (s String) String() string {
	return fmt.Sprintf("%q", string(s))
}

type Name string

func (n Name) String() string {
	return "/" + string(n)
}

type Operator string

type Array []Object

type Procedure []Object

func (p Procedure) String() string {
	var ss []string
	ss = append(ss, "{")
	for i, o := range p {
		if i > 0 {
			ss = append(ss, " ")
		}
		ss = append(ss, fmt.Sprint(o))
	}
	ss = append(ss, "}")
	return strings.Join(ss, "")
}

type Dict map[Name]Object

func (d Dict) String() string {
	return fmt.Sprintf("<Dict %d>", len(d))
}

type mark struct{}

var theMark Object = mark{}

type builtin func(*Interpreter) error
