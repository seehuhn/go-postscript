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
	"fmt"
	"io"
	"strings"
)

type Interpreter struct {
	Stack     []Object
	DictStack []Dict
	Fonts     map[Name]Dict

	SystemDict Dict

	scanners []*scanner
	scanOnly int
}

func NewInterpreter() *Interpreter {
	systemDict := makeSystemDict()
	return &Interpreter{
		DictStack: []Dict{
			systemDict,
			systemDict["userdict"].(Dict),
		},
		Fonts:      make(map[Name]Dict),
		SystemDict: systemDict,
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
		// fmt.Println("###", objectString(o))
		err = intp.executeOne(o, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (intp *Interpreter) executeOne(o Object, execProc bool) error {
	// fmt.Println("|-", intp.stackString(), "|", objectString(o))
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

		val, err := intp.load(Name(o))
		if err != nil {
			return err
		}
		err = intp.executeOne(val, true)
		if e2, ok := err.(*postScriptError); ok {
			errordict := intp.SystemDict["errordict"].(Dict)
			proc, ok := errordict[e2.tp]
			if ok {
				err = intp.executeOne(proc, true)
			}
		}
		return err

	case builtin:
		return o(intp)

	case Procedure:
		if execProc {
			for _, token := range o {
				err := intp.executeOne(token, false)
				if err != nil {
					return err
				}
			}
		} else {
			intp.Stack = append(intp.Stack, o)
		}

	default:
		intp.Stack = append(intp.Stack, o)
	}
	return nil
}

func (intp *Interpreter) load(key Object) (Object, error) {
	name, ok := key.(Name)
	if !ok {
		return nil, errTypecheck
	}
	for j := len(intp.DictStack) - 1; j >= 0; j-- {
		d := intp.DictStack[j]
		if val, ok := d[name]; ok {
			return val, nil
		}
	}
	return nil, errUndefined
}

func (intp *Interpreter) stackString() string {
	var ss []string
	for _, o := range intp.Stack {
		ss = append(ss, objectString2(o, true))
	}
	return strings.Join(ss, " ")
}

func equal(a, b Object) (bool, error) {
	normalize := func(obj Object) (Object, error) {
		switch obj := obj.(type) {
		case Real:
			return float64(obj), nil
		case Integer:
			return float64(obj), nil
		case String:
			return string(obj), nil
		case Name:
			return string(obj), nil
		default:
			return nil, fmt.Errorf("equality not implemented for %T", obj)
		}
	}
	a, err := normalize(a)
	if err != nil {
		return false, err
	}
	b, err = normalize(b)
	if err != nil {
		return false, err
	}
	return a == b, nil
}

func objectString(o Object) string {
	return objectString2(o, false)
}

func objectString2(o Object, short bool) string {
	switch o := o.(type) {
	case nil:
		return "currentfile" // TODO(voss)
	case Boolean:
		return fmt.Sprintf("%t", o)
	case Integer:
		return fmt.Sprint(o)
	case Real:
		return fmt.Sprint(o)
	case Name:
		return "/" + string(o)
	case Operator:
		return string(o)
	case Procedure:
		var ss []string
		l := 1
		for _, oi := range o {
			si := objectString2(oi, true)
			l += 1 + len(si)
			if short && l > 8 || l > 40 {
				ss = append(ss, "...")
				break
			}
			ss = append(ss, si)
		}
		return "{" + strings.Join(ss, " ") + "}"
	case Dict:
		return fmt.Sprintf("<Dict %d>", len(o))
	case builtin:
		return "<builtin>"
	case mark:
		return "*"
	default:
		return fmt.Sprintf("<%T>", o)
	}
}

var errExit = errors.New("exit")
