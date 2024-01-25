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
	"io"
	"strings"
)

// Interpreter represents one instance of the PostScript interpreter.
type Interpreter struct {
	// CheckStart can be set to true to make the interpreter check that the
	// first two input bytes are "%!"; otherwise, ErrNoPostScript is returned.
	// This flag must be set before the first call to Execute.
	CheckStart bool

	// MaxOps can be set to a positive value to limit the number of executed
	// operations.  If this limit is exceeded, ErrExecutionLimitExceeded is
	// returned.
	MaxOps int

	// Stack is the PostScript operand stack.
	Stack []Object

	// DictStack is the PostScript dictionary stack.
	DictStack []Dict

	// NumOps is the number of executed operations so far.
	NumOps int

	// SystemDict is the PostScript system dictionary.
	SystemDict Dict

	// UserDict is the PostScript user dictionary.
	UserDict Dict

	// ErrorDict is the PostScript error dictionary.
	ErrorDict Dict

	// Resources is a Dict of Dicts, which contains the resources
	// for each category.
	Resources Dict

	// FontDirectory is the PostScript font directory.
	// The `definefont` PostScript operator adds fonts to this dictionary.
	FontDirectory Dict

	CMapDirectory Dict

	// DSC contains all DSC comments found in the input so far.
	// These are comments of the form "%%key: value" or "%%key".
	DSC []Comment

	errors    []*postScriptError
	scanners  []*scanner
	procStart []int

	execStackDepth int

	// cmap holds data for the CMap dictionary being constructed, while
	// a `begincmap` ... `endcmap` block is being executed.
	cmap *CMapInfo
}

// NewInterpreter creates a new instance of the PostScript interpreter.
func NewInterpreter() *Interpreter {
	systemDict := makeSystemDict()
	userDict := systemDict["userdict"].(Dict)

	fontDirectory := systemDict["FontDirectory"].(Dict)
	cmapDirectory := Dict{}
	resources := Dict{
		"Font":    fontDirectory,
		"CIDFont": Dict{},
		"CMap":    cmapDirectory,
		"ProcSet": Dict{
			"CIDInit": CIDInit,
		},
	}

	intp := &Interpreter{
		DictStack: []Dict{
			systemDict,
			userDict,
		},
		FontDirectory: fontDirectory,
		CMapDirectory: cmapDirectory,
		Resources:     resources,
		SystemDict:    systemDict,
		UserDict:      userDict,
		ErrorDict:     systemDict["errordict"].(Dict),
	}

	for _, name := range allErrors {
		intp.ErrorDict[name] = defaultErrorHandler
	}

	return intp
}

// ExecuteString executes the PostScript code in the given string.
func (intp *Interpreter) ExecuteString(code string) error {
	return intp.Execute(strings.NewReader(code))
}

// Execute executes the PostScript code in the given reader.
func (intp *Interpreter) Execute(r io.Reader) error {
	s := newScanner(r)
	err := intp.executeScanner(s)
	if err == errExit {
		err = intp.e(eInvalidexit, "exit outside loop")
	} else if err == errStop {
		err = nil
	}
	if err != nil {
		return err
	}

	intp.DSC = append(intp.DSC, s.DSC...)

	return nil
}

func (intp *Interpreter) executeScanner(s *scanner) error {
	if intp.CheckStart {
		head := s.PeekN(2)
		if string(head) != "%!" {
			err := s.err
			if err == nil || err == io.EOF {
				err = ErrNoPostScript
			}
			return err
		}
		intp.CheckStart = false
	}

	intp.scanners = append(intp.scanners, s)
	defer func() {
		intp.scanners = intp.scanners[:len(intp.scanners)-1]
	}()

	for {
		o, err := s.ScanToken()
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

func (intp *Interpreter) executeOne(obj Object, execProc bool) error {
	// if !execProc {
	//  // TODO(voss): also print the commands reached by tail recursion
	// 	fmt.Println("|-", intp.stackString(), "|", intp.objectString(obj))
	// }

	if execProc {
		if intp.execStackDepth >= 100 {
			return intp.e(eExecstackoverflow, "exec stack overflow")
		}
		intp.execStackDepth++
		defer func() { intp.execStackDepth-- }()
	}

	if len(intp.Stack) > maxOperandStackDepth {
		return intp.e(eStackoverflow, "operand stack overflow")
	}

	if obj == Operator("}") {
		if len(intp.procStart) == 0 {
			return intp.e(eSyntaxerror, "unmatched '}'")
		}
		a := intp.procStart[len(intp.procStart)-1]
		intp.procStart = intp.procStart[:len(intp.procStart)-1]
		b := len(intp.Stack)
		proc := make(Procedure, b-a)
		copy(proc, intp.Stack[a:])
		intp.Stack = append(intp.Stack[:a], proc)
		return nil
	} else if obj == Operator("{") {
		intp.procStart = append(intp.procStart, len(intp.Stack))
		return nil
	} else if len(intp.procStart) > 0 {
		intp.Stack = append(intp.Stack, obj)
		return nil
	}

recurseTail:
	intp.NumOps++
	if intp.MaxOps > 0 && intp.NumOps > intp.MaxOps {
		return ErrExecutionLimitExceeded
	}

	switch o := obj.(type) {
	case Operator:
		val, err := intp.load(o)
		if err != nil {
			return err
		}
		obj = val
		execProc = true
		goto recurseTail

	case builtin:
		err := o(intp)
		if e2, ok := err.(*postScriptError); ok {
			level := len(intp.errors)
			if level < 5 {
				intp.errors = append(intp.errors, e2)
				if proc, ok := intp.ErrorDict[e2.tp]; ok {
					err = intp.executeOne(proc, true)
				}
				intp.errors = intp.errors[:level]
			}
		}
		return err

	case Procedure:
		if execProc {
			if len(o) == 0 {
				return nil
			}

			// use tail recursion
			for _, token := range o[:len(o)-1] {
				err := intp.executeOne(token, false)
				if err != nil {
					return err
				}
			}
			obj = o[len(o)-1]
			goto recurseTail
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
		return nil, intp.e(eTypecheck, "load: expected name or operator, got %T", key)
	}
	for j := len(intp.DictStack) - 1; j >= 0; j-- {
		d := intp.DictStack[j]
		if val, ok := d[name]; ok {
			return val, nil
		}
	}
	return nil, intp.e(eUndefined, "load: %s not defined", name)
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
	case String:
		s := o.PS()
		if short && len(s) > 13 {
			s = s[:5] + "..." + s[len(s)-5:]
		}
		return s
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
	maxArraySize         = 65536
	maxDictSize          = 65536
	maxDictStackDepth    = 20
	maxOperandStackDepth = 500
	maxStringSize        = 65536
)
