// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"fmt"
	"strings"
)

type Object any

type Integer int

type Real float64

type Boolean bool

type String []byte

func (s String) String() string {
	return fmt.Sprintf("%q", string(s))
}

// PS returns the string as it would be written in a PostScript file.
func (s String) PS() string {
	var out []byte

	l := []byte(s)

	level := 0
	for _, c := range l {
		if c == '(' {
			level++
		} else if c == ')' {
			level--
			if level < 0 {
				break
			}
		}
	}
	balanced := level == 0

	out = append(out, '(')
	for _, c := range l {
		switch c {
		case '\\':
			out = append(out, '\\', '\\')
		case '(':
			if balanced {
				out = append(out, '(')
			} else {
				out = append(out, '\\', '(')
			}
		case ')':
			if balanced {
				out = append(out, ')')
			} else {
				out = append(out, '\\', ')')
			}
		case '\r':
			out = append(out, '\\', 'r')
		default:
			out = append(out, c)
		}
	}
	out = append(out, ')')

	return string(out)
}

type Name string

func (n Name) String() string {
	return "/" + string(n)
}

func (n Name) PS() string {
	for _, c := range []byte(n) {
		if class[c] != regular {
			panic(fmt.Sprintf("invalid character in name: %q", n))
		}
	}
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
