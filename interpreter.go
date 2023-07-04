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
	"errors"
	"io"
	"strings"
)

type Interpreter struct {
	Stack     []Object
	DictStack []Dict

	scanners []*scanner
	scanOnly int
}

func NewInterpreter() *Interpreter {
	return &Interpreter{
		DictStack: []Dict{
			systemDict,
			make(Dict), // userdict
		},
	}
}

func (intp *Interpreter) ExecuteString(code string) error {
	return intp.Execute(strings.NewReader(code))
}

func (intp *Interpreter) Execute(r io.Reader) error {
	s := newScanner(r)
	return intp.executeScanner(s)
}

func (intp *Interpreter) executeScanner(s *scanner) error {
	intp.scanners = append(intp.scanners, s)
	defer func() {
		intp.scanners = intp.scanners[:len(intp.scanners)-1]
	}()

	for {
		o, err := s.scanToken()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		err = intp.executeOne(o)
		if err != nil {
			return err
		}
	}
	return nil
}

func (intp *Interpreter) executeOne(o Object) error {
	switch o := o.(type) {
	case Operator:
		op := o
		switch op {
		case "{":
			intp.scanOnly++
			fallthrough
		case "[", "<<":
			intp.Stack = append(intp.Stack, theMark)
			return nil
		case "}":
			intp.scanOnly--
			n := len(intp.Stack)
			for i := n - 1; i >= 0; i-- {
				if intp.Stack[i] == theMark {
					size := n - i - 1
					a := make(Procedure, size)
					copy(a, intp.Stack[i+1:])
					intp.Stack = append(intp.Stack[:i], a)
					return nil
				}
			}
			return errors.New("unmatched '}'")
		case "]":
			n := len(intp.Stack)
			for i := n - 1; i >= 0; i-- {
				if intp.Stack[i] == theMark {
					size := n - i - 1
					a := make(Array, size)
					copy(a, intp.Stack[i+1:])
					intp.Stack = append(intp.Stack[:i], a)
					return nil
				}
			}
			return errors.New("unmatched ']'")
		}

		if intp.scanOnly > 0 {
			intp.Stack = append(intp.Stack, o)
			return nil
		}

		for j := len(intp.DictStack) - 1; j >= 0; j-- {
			d := intp.DictStack[j]
			if o, ok := d[Name(op)]; ok {
				return intp.executeOne(o)
			}
		}
		return errors.New("unknown operator '" + string(o) + "'")

	case builtin:
		return o(intp)

	case Procedure:
		for _, token := range o {
			err := intp.executeOne(token)
			if err != nil {
				return err
			}
		}

	default:
		intp.Stack = append(intp.Stack, o)
	}
	return nil
}

var errExit = errors.New("exit")
