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
)

func makeSystemDict() Dict {
	systemDict := Dict{
		"add": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("array: stack underflow")
			}
			a, ok := intp.Stack[len(intp.Stack)-2].(Real)
			var aIsInt bool
			if !ok {
				ai, ok := intp.Stack[len(intp.Stack)-2].(Integer)
				if !ok {
					return errors.New("add: invalid argument")
				}
				aIsInt = true
				a = Real(ai)
			}
			b, ok := intp.Stack[len(intp.Stack)-1].(Real)
			var bIsInt bool
			if !ok {
				bi, ok := intp.Stack[len(intp.Stack)-1].(Integer)
				if !ok {
					return errors.New("add: invalid argument")
				}
				bIsInt = true
				b = Real(bi)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			c := a + b
			ci := Integer(c)
			if aIsInt && bIsInt && Real(ci) == c {
				intp.Stack = append(intp.Stack, ci)
			} else {
				intp.Stack = append(intp.Stack, c)
			}
			return nil
		}),
		"array": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("array: stack underflow")
			}
			size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return errors.New("array: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			if size < 0 {
				return errors.New("array: invalid argument")
			}
			intp.Stack = append(intp.Stack, make(Array, size))
			return nil
		}),
		"begin": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("begin: stack underflow")
			}
			d, ok := intp.Stack[len(intp.Stack)-1].(Dict)
			if !ok {
				return errors.New("begin: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			intp.DictStack = append(intp.DictStack, d)
			return nil
		}),
		"cleartomark": builtin(func(intp *Interpreter) error {
			for k := len(intp.Stack) - 1; k >= 0; k-- {
				if intp.Stack[k] == theMark {
					intp.Stack = intp.Stack[:k]
					return nil
				}
			}
			return errors.New("cleartomark: no mark found")
		}),
		"closefile": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("closefile: stack underflow")
			}
			if intp.Stack[len(intp.Stack)-1] != nil {
				return errors.New("closefile: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			return io.EOF
		}),
		"count": builtin(func(intp *Interpreter) error {
			intp.Stack = append(intp.Stack, Integer(len(intp.Stack)))
			return nil
		}),
		"currentdict": builtin(func(intp *Interpreter) error {
			intp.Stack = append(intp.Stack, intp.DictStack[len(intp.DictStack)-1])
			return nil
		}),
		"currentfile": builtin(func(intp *Interpreter) error {
			intp.Stack = append(intp.Stack, nil)
			return nil
		}),
		"def": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("def: stack underflow")
			}
			name, ok := intp.Stack[len(intp.Stack)-2].(Name)
			if !ok {
				return errors.New("def: invalid argument")
			}
			intp.DictStack[len(intp.DictStack)-1][name] = intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			return nil
		}),
		"definefont": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("definefont: stack underflow")
			}
			name, ok := intp.Stack[len(intp.Stack)-2].(Name)
			if !ok {
				return errors.New("definefont: invalid argument")
			}
			font, ok := intp.Stack[len(intp.Stack)-1].(Dict)
			if !ok {
				return errors.New("definefont: invalid argument")
			}
			intp.Fonts[name] = font
			intp.Stack = append(intp.Stack[:len(intp.Stack)-2], font)
			return nil
		}),
		"dict": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("dict: stack underflow")
			}
			size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return errors.New("dict: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			if size < 0 {
				return errors.New("dict: invalid argument")
			}
			intp.Stack = append(intp.Stack, make(Dict, size))
			return nil
		}),
		"dup": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("dup: stack underflow")
			}
			intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-1])
			return nil
		}),
		"exec": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errStackunderflow
			}
			obj := intp.Stack[len(intp.Stack)-1]
			switch obj := obj.(type) {
			case builtin:
				return obj(intp)
			default:
				return fmt.Errorf("exec: not implemented for %T", obj)
			}
		}),
		"eexec": builtin(eexec),
		"end": builtin(func(intp *Interpreter) error {
			if len(intp.DictStack) <= 2 {
				return errors.New("end: dict stack underflow")
			}
			intp.DictStack = intp.DictStack[:len(intp.DictStack)-1]
			return nil
		}),
		"eq": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("eq: stack underflow")
			}
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
					return nil, fmt.Errorf("eq: not implemented for %T", obj)
				}
			}
			a, err := normalize(intp.Stack[len(intp.Stack)-2])
			if err != nil {
				return err
			}
			b, err := normalize(intp.Stack[len(intp.Stack)-1])
			if err != nil {
				return err
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			intp.Stack = append(intp.Stack, Boolean(a == b))
			return nil
		}),
		"exch": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("exch: stack underflow")
			}
			intp.Stack[len(intp.Stack)-1], intp.Stack[len(intp.Stack)-2] = intp.Stack[len(intp.Stack)-2], intp.Stack[len(intp.Stack)-1]
			return nil
		}),
		"executeonly": builtin(func(intp *Interpreter) error {
			// not implemented
			return nil
		}),
		"false":         Boolean(false),
		"FontDirectory": Dict{},
		"for": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 4 {
				return errors.New("for: stack underflow")
			}
			// TODO(voss): the spec also allows Real values here
			initial, ok := intp.Stack[len(intp.Stack)-4].(Integer)
			if !ok {
				return errors.New("for: invalid initial argument")
			}
			increment, ok := intp.Stack[len(intp.Stack)-3].(Integer)
			if !ok {
				return errors.New("for: invalid increment argument")
			}
			limit, ok := intp.Stack[len(intp.Stack)-2].(Integer)
			if !ok {
				return errors.New("for: invalid limit argument")
			}
			proc := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-4]
			val := initial
			for {
				if increment > 0 && val > limit || increment < 0 && val < limit {
					break
				}
				intp.Stack = append(intp.Stack, val)
				err := intp.executeOne(proc)
				if err == errExit {
					break
				} else if err != nil {
					return err
				}
				val += increment
			}
			return nil
		}),
		"get": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("get: stack underflow")
			}
			obj := intp.Stack[len(intp.Stack)-2]
			sel := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			switch obj := obj.(type) {
			case Array:
				index, ok := sel.(Integer)
				if !ok {
					return errors.New("get: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return errors.New("get: index out of bounds")
				}
				intp.Stack = append(intp.Stack, obj[index])
			case Dict:
				name, ok := sel.(Name)
				if !ok {
					return errors.New("get: invalid name")
				}
				val, ok := obj[name]
				if !ok {
					return fmt.Errorf("name %q not found", name)
				}
				intp.Stack = append(intp.Stack, val)
			case String:
				index, ok := sel.(Integer)
				if !ok {
					return errors.New("get: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return errors.New("get: index out of bounds")
				}
				intp.Stack = append(intp.Stack, obj[index])
			default:
				return errors.New("get: invalid type")
			}
			return nil
		}),
		"if": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("if: stack underflow")
			}
			cond, ok := intp.Stack[len(intp.Stack)-2].(Boolean)
			if !ok {
				return errors.New("if: invalid condition")
			}
			proc := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			if cond {
				return intp.executeOne(proc)
			}
			return nil
		}),
		"ifelse": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 3 {
				return errors.New("ifelse: stack underflow")
			}
			cond, ok := intp.Stack[len(intp.Stack)-3].(Boolean)
			if !ok {
				return errors.New("ifelse: invalid condition")
			}
			proc1 := intp.Stack[len(intp.Stack)-2]
			proc2 := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-3]
			if cond {
				return intp.executeOne(proc1)
			} else {
				return intp.executeOne(proc2)
			}
		}),
		"index": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("index: stack underflow")
			}
			index, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return errors.New("index: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			if index < 0 || index >= Integer(len(intp.Stack)) {
				return errors.New("index: invalid argument")
			}
			intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-int(index)-1])
			return nil
		}),
		"known": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("known: stack underflow")
			}
			d, ok := intp.Stack[len(intp.Stack)-2].(Dict)
			if !ok {
				return errors.New("known: invalid argument")
			}
			name, ok := intp.Stack[len(intp.Stack)-1].(Name)
			if !ok {
				return errors.New("known: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			_, ok = d[name]
			intp.Stack = append(intp.Stack, Boolean(ok))
			return nil
		}),
		"mark": builtin(func(intp *Interpreter) error {
			intp.Stack = append(intp.Stack, mark{})
			return nil
		}),
		"noaccess": builtin(func(intp *Interpreter) error {
			// not implemented
			return nil
		}),
		"not": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("not: stack underflow")
			}
			obj := intp.Stack[len(intp.Stack)-1]
			switch obj := obj.(type) {
			case Boolean:
				intp.Stack[len(intp.Stack)-1] = !obj
			case Integer:
				intp.Stack[len(intp.Stack)-1] = ^obj
			default:
				return errors.New("not: invalid argument")
			}
			return nil
		}),
		"pop": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("pop: stack underflow")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			return nil
		}),
		"put": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 3 {
				return errors.New("put: stack underflow")
			}
			obj := intp.Stack[len(intp.Stack)-3]
			sel := intp.Stack[len(intp.Stack)-2]
			value := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-3]
			switch obj := obj.(type) {
			case Array:
				index, ok := sel.(Integer)
				if !ok {
					return errors.New("put: invalid index for Array")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return errors.New("put: index out of range for Array")
				}
				obj[index] = value
			case Dict:
				key, ok := sel.(Name)
				if !ok {
					return errors.New("put: invalid key for Dict")
				}
				obj[key] = value
			case String:
				index, ok := sel.(Integer)
				if !ok {
					return errors.New("put: invalid index for String")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return errors.New("put: index out of range for String")
				}
				c, ok := value.(Integer)
				if !ok {
					return errors.New("put: invalid value for String")
				}
				obj[index] = byte(c)
			default:
				return fmt.Errorf("put: invalid argument %T", obj)
			}
			return nil
		}),
		"readonly": builtin(func(intp *Interpreter) error {
			// not implemented
			return nil
		}),
		"readstring": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return errors.New("readstring: stack underflow")
			}
			buf, ok := intp.Stack[len(intp.Stack)-1].(String)
			if !ok {
				return errors.New("readstring: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			s := intp.scanners[len(intp.scanners)-1]
			s.skipByte()
			n, err := s.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			intp.Stack = append(intp.Stack, buf[:n])
			intp.Stack = append(intp.Stack, Boolean(n == len(buf)))
			return nil
		}),
		"StandardEncoding": StandardEncoding,
		"string": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return errors.New("string: stack underflow")
			}
			size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return errors.New("string: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			if size < 0 || size > 1<<16 {
				return errors.New("string: invalid size")
			}
			intp.Stack = append(intp.Stack, make(String, size))
			return nil
		}),
		"true": Boolean(true),
	}
	systemDict["systemdict"] = systemDict
	systemDict["userdict"] = Dict{}

	errorDict := Dict{}
	for _, err := range allErrors {
		err := err
		errorDict[err.tp] = builtin(func(intp *Interpreter) error {
			return errors.New(string(err.tp))
		})
	}
	systemDict["errordict"] = errorDict

	return systemDict
}

var StandardEncoding = Array{
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name("space"),
	Name("exclam"),
	Name("quotedbl"),
	Name("numbersign"),
	Name("dollar"),
	Name("percent"),
	Name("ampersand"),
	Name("quoteright"),
	Name("parenleft"),
	Name("parenright"),
	Name("asterisk"),
	Name("plus"),
	Name("comma"),
	Name("hyphen"),
	Name("period"),
	Name("slash"),
	Name("zero"),
	Name("one"),
	Name("two"),
	Name("three"),
	Name("four"),
	Name("five"),
	Name("six"),
	Name("seven"),
	Name("eight"),
	Name("nine"),
	Name("colon"),
	Name("semicolon"),
	Name("less"),
	Name("equal"),
	Name("greater"),
	Name("question"),
	Name("at"),
	Name("A"),
	Name("B"),
	Name("C"),
	Name("D"),
	Name("E"),
	Name("F"),
	Name("G"),
	Name("H"),
	Name("I"),
	Name("J"),
	Name("K"),
	Name("L"),
	Name("M"),
	Name("N"),
	Name("O"),
	Name("P"),
	Name("Q"),
	Name("R"),
	Name("S"),
	Name("T"),
	Name("U"),
	Name("V"),
	Name("W"),
	Name("X"),
	Name("Y"),
	Name("Z"),
	Name("bracketleft"),
	Name("backslash"),
	Name("bracketright"),
	Name("asciicircum"),
	Name("underscore"),
	Name("quoteleft"),
	Name("a"),
	Name("b"),
	Name("c"),
	Name("d"),
	Name("e"),
	Name("f"),
	Name("g"),
	Name("h"),
	Name("i"),
	Name("j"),
	Name("k"),
	Name("l"),
	Name("m"),
	Name("n"),
	Name("o"),
	Name("p"),
	Name("q"),
	Name("r"),
	Name("s"),
	Name("t"),
	Name("u"),
	Name("v"),
	Name("w"),
	Name("x"),
	Name("y"),
	Name("z"),
	Name("braceleft"),
	Name("bar"),
	Name("braceright"),
	Name("asciitilde"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name("exclamdown"),
	Name("cent"),
	Name("sterling"),
	Name("fraction"),
	Name("yen"),
	Name("florin"),
	Name("section"),
	Name("currency"),
	Name("quotesingle"),
	Name("quotedblleft"),
	Name("guillemotleft"),
	Name("guilsinglleft"),
	Name("guilsinglright"),
	Name("fi"),
	Name("fl"),
	Name(".notdef"),
	Name("endash"),
	Name("dagger"),
	Name("daggerdbl"),
	Name("periodcentered"),
	Name(".notdef"),
	Name("paragraph"),
	Name("bullet"),
	Name("quotesinglbase"),
	Name("quotedblbase"),
	Name("quotedblright"),
	Name("guillemotright"),
	Name("ellipsis"),
	Name("perthousand"),
	Name(".notdef"),
	Name("questiondown"),
	Name(".notdef"),
	Name("grave"),
	Name("acute"),
	Name("circumflex"),
	Name("tilde"),
	Name("macron"),
	Name("breve"),
	Name("dotaccent"),
	Name("dieresis"),
	Name(".notdef"),
	Name("ring"),
	Name("cedilla"),
	Name(".notdef"),
	Name("hungarumlaut"),
	Name("ogonek"),
	Name("caron"),
	Name("emdash"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name("AE"),
	Name(".notdef"),
	Name("ordfeminine"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name("Lslash"),
	Name("Oslash"),
	Name("OE"),
	Name("ordmasculine"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name("ae"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name("dotlessi"),
	Name(".notdef"),
	Name(".notdef"),
	Name("lslash"),
	Name("oslash"),
	Name("oe"),
	Name("germandbls"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
	Name(".notdef"),
}
