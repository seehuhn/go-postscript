// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"fmt"
	"io"
	"strings"
)

// Object is a PostScript object.  The null object is represented by nil.
type Object any

// Integer is a PostScript integer object.  Integers are 32 bits wide.
type Integer int32

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

// builtin is an operator implemented in Go.  Built-in operators are held
// as *builtin, so that eq can compare them by identity.
type builtin struct {
	fn func(*Interpreter) error

	// name is the key of the operator in systemdict, if any.
	name string
}

// file is a file object for reading program input.  Files are held as *file,
// so that eq can compare them by identity.  A file is open while the
// interpreter executes it.
//
// An eexec decryption filter shares the scanner of the file it decrypts,
// which is recorded in src.
type file struct {
	s      *scanner
	src    *file
	closed bool
}

// isClosed reports whether f or the file it reads from has been closed.
// A filter whose source is closed reads as end-of-file.
func (f *file) isClosed() bool {
	for ; f != nil; f = f.src {
		if f.closed {
			return true
		}
	}
	return false
}

// Read reads data from the file.  While an eexec filter is active on
// the scanner, reading from the filter's source bypasses the decryption.
func (f *file) Read(p []byte) (int, error) {
	if f.isClosed() {
		return 0, io.EOF
	}
	if f.src == nil && f.s.eexecMode != 0 {
		return f.s.readRaw(p)
	}
	return f.s.Read(p)
}
