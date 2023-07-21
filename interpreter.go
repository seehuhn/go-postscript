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

// TODO(voss): prevent infinite loops

type Interpreter struct {
	Stack      []Object
	DictStack  []Dict
	Fonts      Dict
	DSC        []Comment
	CheckStart bool

	SystemDict Dict
	UserDict   Dict

	scanners  []*scanner
	procStart []int
}

func NewInterpreter() *Interpreter {
	systemDict := makeSystemDict()
	userDict := systemDict["userdict"].(Dict)
	return &Interpreter{
		DictStack: []Dict{
			systemDict,
			userDict,
		},
		Fonts:      systemDict["FontDirectory"].(Dict),
		SystemDict: systemDict,
		UserDict:   userDict,
	}
}

func (intp *Interpreter) ExecuteString(code string) error {
	return intp.Execute(strings.NewReader(code))
}

func (intp *Interpreter) Execute(r io.Reader) error {
	s := newScanner(r)
	err := intp.executeScanner(s)
	if err != nil {
		return err
	}

	intp.DSC = append(intp.DSC, s.DSC...)
	return nil
}

func (intp *Interpreter) executeScanner(s *scanner) error {
	if intp.CheckStart {
		head, err := s.peekN(2)
		if err != nil {
			return err
		}
		if string(head) != "%!" {
			return errors.New("not a PostScript file")
		}
		intp.CheckStart = false
	}

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
	// if !execProc {
	// 	fmt.Println("|-", intp.stackString(), "|", intp.objectString(o))
	// }

	if len(intp.Stack) > maxOperandStackDepth {
		return errStackoverflow
	}

	if o == Operator("}") {
		if len(intp.procStart) == 0 {
			return errors.New("unmatched '}'")
		}
		a := intp.procStart[len(intp.procStart)-1]
		intp.procStart = intp.procStart[:len(intp.procStart)-1]
		b := len(intp.Stack)
		proc := make(Procedure, b-a)
		copy(proc, intp.Stack[a:])
		intp.Stack = append(intp.Stack[:a], proc)
		return nil
	} else if o == Operator("{") {
		intp.procStart = append(intp.procStart, len(intp.Stack))
		return nil
	} else if len(intp.procStart) > 0 {
		intp.Stack = append(intp.Stack, o)
		return nil
	}

	switch o := o.(type) {
	case Operator:
		val, err := intp.load(o)
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
	var name Name
	switch key := key.(type) {
	case Name:
		name = key
	case Operator:
		name = Name(key)
	default:
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
		ss = append(ss, intp.objectString2(o, true))
	}
	return strings.Join(ss, " ")
}

func (intp *Interpreter) objectString(o Object) string {
	return intp.objectString2(o, false)
}

func (intp *Interpreter) objectString2(o Object, short bool) string {
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
	case Array:
		var ss []string
		l := 1
		for _, oi := range o {
			si := intp.objectString2(oi, true)
			l += 1 + len(si)
			if short && l > 8 || l > 40 {
				ss = append(ss, "...")
				break
			}
			ss = append(ss, si)
		}
		return "[" + strings.Join(ss, " ") + "]"
	case Procedure:
		var ss []string
		l := 1
		for i, oi := range o {
			o[i] = nil // protect against infinite loops
			si := intp.objectString2(oi, true)
			o[i] = oi
			l += 1 + len(si)
			if short && l > 8 || l > 40 {
				ss = append(ss, "...")
				break
			}
			ss = append(ss, si)
		}
		return "{" + strings.Join(ss, " ") + "}"
	case Dict:
		if isSameDict(o, intp.SystemDict) {
			return "*systemdict*"
		} else if isSameDict(o, intp.UserDict) {
			return "*userdict*"
		}
		return fmt.Sprintf("<Dict %d>", len(o))
	case builtin:
		return "<builtin>"
	case mark:
		return "*"
	default:
		return fmt.Sprintf("<%T>", o)
	}
}

const (
	maxArraySize            = 65536
	maxDictionaryStackDepth = 20
	maxOperandStackDepth    = 500
)
