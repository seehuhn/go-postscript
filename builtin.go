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
	"math"
	"strconv"
)

func makeSystemDict() Dict {
	FontDirectory := Dict{}
	userDict := Dict{}
	errorDict := Dict{}

	systemDict := Dict{
		"[": builtin(func(intp *Interpreter) error {
			intp.Stack = append(intp.Stack, theMark)
			return nil
		}),
		"]": builtin(func(intp *Interpreter) error {
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
			return intp.e(eUnmatchedmark, "]: missing '['")
		}),
		"<<": builtin(func(intp *Interpreter) error {
			intp.Stack = append(intp.Stack, theMark)
			return nil
		}),
		">>": builtin(func(intp *Interpreter) error {
			n := len(intp.Stack)
			markPos := -1
			for i := n - 1; i >= 0; i-- {
				if intp.Stack[i] == theMark {
					markPos = i
					break
				}
			}
			if markPos < 0 {
				return intp.e(eUnmatchedmark, ">>: missing '<<'")
			} else if (n-markPos)%2 != 1 {
				return intp.e(eRangecheck, "dict literal: odd length")
			}
			d := make(Dict, (n-markPos-1)/2)
			for i := markPos + 1; i < n; i += 2 {
				name, ok := intp.Stack[i].(Name)
				if !ok {
					return intp.e(eTypecheck, "dict literal: keys must be Name, not %T", intp.Stack[i])
				}
				d[name] = intp.Stack[i+1]
			}
			intp.Stack = append(intp.Stack[:markPos], d)
			return nil
		}),
		"abs": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "abs: not enough arguments")
			}
			x := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			switch x := x.(type) {
			case Integer:
				if x == math.MinInt {
					intp.Stack = append(intp.Stack, -Real(x))
				} else if x < 0 {
					intp.Stack = append(intp.Stack, -x)
				} else {
					intp.Stack = append(intp.Stack, x)
				}
			case Real:
				if x < 0 {
					intp.Stack = append(intp.Stack, -x)
				} else {
					intp.Stack = append(intp.Stack, x)
				}
			default:
				return intp.e(eTypecheck, "abs: needs a number")
			}
			return nil
		}),
		"add": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "add: not enough arguments")
			}
			ar, aIsReal := intp.Stack[len(intp.Stack)-2].(Real)
			ai, aIsInt := intp.Stack[len(intp.Stack)-2].(Integer)
			br, bIsReal := intp.Stack[len(intp.Stack)-1].(Real)
			bi, bIsInt := intp.Stack[len(intp.Stack)-1].(Integer)
			if !(aIsReal || aIsInt) || !(bIsReal || bIsInt) {
				return intp.e(eTypecheck, "add: needs numbers")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			if aIsReal || bIsReal {
				if aIsInt {
					ar = Real(ai)
				}
				if bIsInt {
					br = Real(bi)
				}
				intp.Stack = append(intp.Stack, ar+br)
			} else {
				ci := ai + bi
				// check for integer overflow
				if (ai < 0 && bi < 0 && ci >= 0) || (ai > 0 && bi > 0 && ci <= 0) {
					intp.Stack = append(intp.Stack, Real(ai)+Real(bi))
				} else {
					intp.Stack = append(intp.Stack, ci)
				}
			}
			return nil
		}),
		"and": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "and: not enough arguments")
			}
			a := intp.Stack[len(intp.Stack)-2]
			b := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			switch a := a.(type) {
			case Boolean:
				b, ok := b.(Boolean)
				if !ok {
					return intp.e(eTypecheck, "and: mismatched argument types")
				}
				intp.Stack = append(intp.Stack, a && b)
			case Integer:
				b, ok := b.(Integer)
				if !ok {
					return intp.e(eTypecheck, "and: mismatched argument types")
				}
				intp.Stack = append(intp.Stack, a&b)
			default:
				return intp.e(eTypecheck, "and: invalid argument type %T", a)
			}
			return nil
		}),
		"array": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "array: not enough arguments")
			}
			size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return intp.e(eTypecheck, "array: need an integer")
			} else if size < 0 {
				return intp.e(eRangecheck, "array: invalid size %d", size)
			} else if size > maxArraySize {
				return intp.e(eLimitcheck, "array: invalid size %d", size)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			intp.Stack = append(intp.Stack, make(Array, size))
			return nil
		}),
		"begin": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "begin: not enough arguments")
			}
			if len(intp.DictStack) >= maxDictStackDepth {
				return intp.e(eDictstackoverflow, "begin")
			}
			d, ok := intp.Stack[len(intp.Stack)-1].(Dict)
			if !ok {
				return intp.e(eTypecheck, "begin: needs a dictionary")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			intp.DictStack = append(intp.DictStack, d)
			return nil
		}),
		"bind": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "bind: not enough arguments")
			}
			obj, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
			if !ok {
				return intp.e(eTypecheck, "bind: needs a procedure, not %T", obj)
			}
			intp.bindProc(obj)
			return nil
		}),
		"cleartomark": builtin(func(intp *Interpreter) error {
			for k := len(intp.Stack) - 1; k >= 0; k-- {
				if intp.Stack[k] == theMark {
					intp.Stack = intp.Stack[:k]
					return nil
				}
			}
			return intp.e(eUnmatchedmark, "cleartomark: no mark found")
		}),
		"closefile": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "closefile: not enough arguments")
			}
			if x := intp.Stack[len(intp.Stack)-1]; x != nil {
				return intp.e(eTypecheck, "closefile: needs a file, not %T", x)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			return io.EOF
		}),
		"copy": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "copy: not enough arguments")
			}
			if n, ok := intp.Stack[len(intp.Stack)-1].(Integer); ok {
				if n < 0 {
					return intp.e(eRangecheck, "copy: invalid count %d", n)
				}
				if len(intp.Stack) < int(n)+1 {
					return intp.e(eStackunderflow, "copy: not enough arguments")
				}
				intp.Stack = intp.Stack[:len(intp.Stack)-1]
				intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-int(n):]...)
				return nil
			}
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "copy: not enough arguments")
			}
			a := intp.Stack[len(intp.Stack)-2]
			b := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			var res Object
			switch a := a.(type) {
			case Array:
				b, ok := b.(Array)
				if !ok {
					return intp.e(eTypecheck, "copy: mismatched argument types")
				} else if len(b) < len(a) {
					return intp.e(eRangecheck, "copy: not enough space in destination")
				}
				n := copy(b, a)
				res = b[:n]
			case Dict:
				b, ok := b.(Dict)
				if !ok {
					return intp.e(eTypecheck, "copy: mismatched argument types")
				}
				for k, v := range a {
					b[k] = v
				}
				res = b
			case String:
				b, ok := b.(String)
				if !ok {
					return intp.e(eTypecheck, "copy: mismatched argument types")
				} else if len(b) < len(a) {
					return intp.e(eRangecheck, "copy: not enough space in destination")
				}
				n := copy(b, a)
				res = b[:n]
			default:
				return intp.e(eTypecheck, "copy: invalid type %T", a)
			}
			intp.Stack = append(intp.Stack, res)
			return nil
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
		"cvx": builtin(func(intp *Interpreter) error {
			// not implemented
			return nil
		}),
		"def": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "def: not enough arguments")
			}
			name, ok := intp.Stack[len(intp.Stack)-2].(Name)
			if !ok {
				return intp.e(eTypecheck, "def: needs name, not %T", intp.Stack[len(intp.Stack)-2])
			}
			intp.DictStack[len(intp.DictStack)-1][name] = intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			return nil
		}),
		"definefont": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "definefont: not enough arguments")
			}
			name, ok := intp.Stack[len(intp.Stack)-2].(Name)
			if !ok {
				return intp.e(eTypecheck, "definefont: needs name, not %T", intp.Stack[len(intp.Stack)-2])
			}
			font, ok := intp.Stack[len(intp.Stack)-1].(Dict)
			if !ok {
				return intp.e(eTypecheck, "definefont: needs font, not %T", intp.Stack[len(intp.Stack)-1])
			}
			intp.FontDirectory[name] = font
			intp.Stack = append(intp.Stack[:len(intp.Stack)-2], font)
			return nil
		}),
		"dict": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "dict: not enough arguments")
			}
			size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return intp.e(eTypecheck, "dict: needs an integer, not %T", intp.Stack[len(intp.Stack)-1])
			} else if size < 0 {
				return intp.e(eRangecheck, "dict: invalid size %d", size)
			} else if size > maxDictSize {
				return intp.e(eLimitcheck, "dict: invalid size %d", size)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			intp.Stack = append(intp.Stack, make(Dict, size))
			return nil
		}),
		"dup": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "dup: not enough arguments")
			}
			intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-1])
			return nil
		}),
		"exec": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "exec: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-1]

			switch obj := obj.(type) {
			case builtin:
				return obj(intp)
			case Procedure:
				return intp.executeOne(obj, true)
			default:
				return intp.e(eTypecheck, "exec: not implemented for %T", obj)
			}
		}),
		"eexec": builtin(eexec),
		"end": builtin(func(intp *Interpreter) error {
			if len(intp.DictStack) <= 2 {
				return intp.e(eDictstackunderflow, "end: dictionary stack is empty")
			}
			intp.DictStack = intp.DictStack[:len(intp.DictStack)-1]
			return nil
		}),
		"eq": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "eq: not enough arguments")
			}
			a := intp.Stack[len(intp.Stack)-2]
			b := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]

			isEqual, err := equal(a, b)
			if err != nil {
				return err
			}

			intp.Stack = append(intp.Stack, Boolean(isEqual))
			return nil
		}),
		"errordict": errorDict,
		"exch": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "exch: not enough arguments")
			}
			intp.Stack[len(intp.Stack)-1], intp.Stack[len(intp.Stack)-2] = intp.Stack[len(intp.Stack)-2], intp.Stack[len(intp.Stack)-1]
			return nil
		}),
		"executeonly": builtin(func(intp *Interpreter) error {
			// not implemented
			return nil
		}),
		"exit": builtin(func(intp *Interpreter) error {
			return errExit
		}),
		"false": Boolean(false),
		"findfont": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "findfont: not enough arguments")
			}
			name, ok := intp.Stack[len(intp.Stack)-1].(Name)
			if !ok {
				return intp.e(eTypecheck, "findfont: needs a name, not %T", intp.Stack[len(intp.Stack)-1])
			}
			font, ok := intp.FontDirectory[name]
			if !ok {
				return intp.e(eInvalidfont, "font %q not found", name)
			}
			intp.Stack = append(intp.Stack[:len(intp.Stack)-1], font)
			return nil
		}),
		"FontDirectory": FontDirectory,
		"for": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 4 {
				return intp.e(eStackunderflow, "for: not enough arguments")
			}
			// TODO(voss): the spec also allows Real values here
			initial, ok := intp.Stack[len(intp.Stack)-4].(Integer)
			if !ok {
				return intp.e(eTypecheck, "for: invalid start")
			}
			increment, ok := intp.Stack[len(intp.Stack)-3].(Integer)
			if !ok {
				return intp.e(eTypecheck, "for: invalid increment")
			}
			limit, ok := intp.Stack[len(intp.Stack)-2].(Integer)
			if !ok {
				return intp.e(eTypecheck, "for: invalid limit")
			}
			proc := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-4]
			val := initial
			for {
				if increment > 0 && val > limit || increment < 0 && val < limit {
					break
				}
				intp.Stack = append(intp.Stack, val)
				err := intp.executeOne(proc, true)
				if err == errExit {
					break
				} else if err != nil {
					return err
				}
				val += increment
			}
			return nil
		}),
		"forall": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "forall: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-2]
			proc, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
			if !ok {
				return intp.e(eTypecheck, "forall: invalid argument")
			}
			switch obj := obj.(type) {
			case Array:
				intp.Stack = intp.Stack[:len(intp.Stack)-2]
				for _, val := range obj {
					intp.Stack = append(intp.Stack, val)
					err := intp.executeOne(proc, true)
					if err == errExit {
						break
					} else if err != nil {
						return err
					}
				}
			case String:
				intp.Stack = intp.Stack[:len(intp.Stack)-2]
				for _, c := range obj {
					intp.Stack = append(intp.Stack, Integer(c))
					err := intp.executeOne(proc, true)
					if err == errExit {
						break
					} else if err != nil {
						return err
					}
				}
			case Dict:
				intp.Stack = intp.Stack[:len(intp.Stack)-2]
				for key, val := range obj {
					intp.Stack = append(intp.Stack, key, val)
					err := intp.executeOne(proc, true)
					if err == errExit {
						break
					} else if err != nil {
						return err
					}
				}
			default:
				return intp.e(eTypecheck, "forall: invalid type %T", obj)
			}
			return nil
		}),
		"get": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "get: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-2]
			sel := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			switch obj := obj.(type) {
			case Array:
				index, ok := sel.(Integer)
				if !ok {
					return intp.e(eTypecheck, "get: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return intp.e(eRangecheck, "get: index out of bounds")
				}
				intp.Stack = append(intp.Stack, obj[index])
			case Procedure:
				index, ok := sel.(Integer)
				if !ok {
					return intp.e(eTypecheck, "get: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return intp.e(eRangecheck, "get: index out of bounds")
				}
				intp.Stack = append(intp.Stack, obj[index])
			case Dict:
				name, ok := sel.(Name)
				if !ok {
					return intp.e(eTypecheck, "get: invalid dict key")
				}
				val, ok := obj[name]
				if !ok {
					return intp.e(eUndefined, "get: missing dict key %q", name)
				}
				intp.Stack = append(intp.Stack, val)
			case String:
				index, ok := sel.(Integer)
				if !ok {
					return intp.e(eTypecheck, "get: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return intp.e(eRangecheck, "get: index out of bounds")
				}
				intp.Stack = append(intp.Stack, obj[index])
			default:
				return intp.e(eTypecheck, "get: invalid argument type %T", obj)
			}
			return nil
		}),
		"getinterval": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 3 {
				return intp.e(eStackunderflow, "getinterval: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-3]
			var n int
			switch obj := obj.(type) {
			case Array:
				n = len(obj)
			case String:
				n = len(obj)
			default:
				return intp.e(eTypecheck, "getinterval: invalid argument type %T", obj)
			}
			index, ok := intp.Stack[len(intp.Stack)-2].(Integer)
			if !ok {
				return intp.e(eTypecheck, "getinterval: invalid index")
			} else if index < 0 || index >= Integer(n) {
				return intp.e(eRangecheck, "getinterval: index %d out of bounds", index)
			}
			count, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return intp.e(eTypecheck, "getinterval: invalid count")
			} else if count < 0 || count > Integer(n)-index {
				return intp.e(eRangecheck, "getinterval: count %d out of bounds", count)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-3]
			var res Object
			switch obj := obj.(type) {
			case Array:
				res = obj[index : index+count]
			case String:
				res = obj[index : index+count]
			}
			intp.Stack = append(intp.Stack, res)
			return nil
		}),
		"if": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "if: not enough arguments")
			}
			cond, ok := intp.Stack[len(intp.Stack)-2].(Boolean)
			if !ok {
				return intp.e(eTypecheck, "if: invalid condition")
			}
			proc := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			if cond {
				return intp.executeOne(proc, true)
			}
			return nil
		}),
		"ifelse": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 3 {
				return intp.e(eStackunderflow, "ifelse: not enough arguments")
			}
			cond, ok := intp.Stack[len(intp.Stack)-3].(Boolean)
			if !ok {
				return intp.e(eTypecheck, "ifelse: invalid condition")
			}
			proc1 := intp.Stack[len(intp.Stack)-2]
			proc2 := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-3]
			if cond {
				return intp.executeOne(proc1, true)
			} else {
				return intp.executeOne(proc2, true)
			}
		}),
		"index": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "index: not enough arguments")
			}
			index, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return intp.e(eTypecheck, "index: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			if index < 0 || index >= Integer(len(intp.Stack)) {
				return intp.e(eRangecheck, "index: index out of bounds")
			}
			intp.Stack = append(intp.Stack, intp.Stack[len(intp.Stack)-int(index)-1])
			return nil
		}),
		"known": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "known: not enough arguments")
			}
			d, ok := intp.Stack[len(intp.Stack)-2].(Dict)
			if !ok {
				return intp.e(eTypecheck, "known: invalid argument")
			}
			name, ok := intp.Stack[len(intp.Stack)-1].(Name)
			if !ok {
				return intp.e(eTypecheck, "known: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			_, ok = d[name]
			intp.Stack = append(intp.Stack, Boolean(ok))
			return nil
		}),
		"length": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "length: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			var res int
			switch obj := obj.(type) {
			case Array:
				res = len(obj)
			case Procedure:
				res = len(obj)
			case Dict:
				res = len(obj)
			case String:
				res = len(obj)
			case Name:
				res = len(obj)
			case Operator:
				res = len(obj)
			default:
				return intp.e(eTypecheck, "length: invalid argument type %T", obj)
			}
			intp.Stack = append(intp.Stack, Integer(res))
			return nil
		}),
		"load": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "load: not enough arguments")
			}
			name, ok := intp.Stack[len(intp.Stack)-1].(Name)
			if !ok {
				return intp.e(eTypecheck, "load: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			val, err := intp.load(name)
			if err != nil {
				return err
			}
			intp.Stack = append(intp.Stack, val)
			return nil
		}),
		"loop": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "loop: not enough arguments")
			}
			proc := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			for {
				err := intp.executeOne(proc, true)
				if err == errExit {
					return nil
				} else if err != nil {
					return err
				}
			}
		}),
		"mark": builtin(func(intp *Interpreter) error {
			intp.Stack = append(intp.Stack, mark{})
			return nil
		}),
		"matrix": builtin(func(intp *Interpreter) error {
			m := Array{Integer(1), Integer(0), Integer(0), Integer(1), Integer(0), Integer(0)}
			intp.Stack = append(intp.Stack, m)
			return nil
		}),
		"mul": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "mul: not enough arguments")
			}
			ar, aIsReal := intp.Stack[len(intp.Stack)-2].(Real)
			ai, aIsInt := intp.Stack[len(intp.Stack)-2].(Integer)
			br, bIsReal := intp.Stack[len(intp.Stack)-1].(Real)
			bi, bIsInt := intp.Stack[len(intp.Stack)-1].(Integer)
			if !(aIsReal || aIsInt) || !(bIsReal || bIsInt) {
				return intp.e(eTypecheck, "mul: needs numbers as arguments")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			if aIsReal || bIsReal {
				if aIsInt {
					ar = Real(ai)
				}
				if bIsInt {
					br = Real(bi)
				}
				intp.Stack = append(intp.Stack, ar*br)
			} else {
				ci := ai * bi
				// check for integer overflow
				if ai != 0 && ci/ai != bi {
					intp.Stack = append(intp.Stack, Real(ai)*Real(bi))
				} else {
					intp.Stack = append(intp.Stack, ci)
				}
			}
			return nil
		}),
		"ne": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "ne: not enough arguments")
			}
			a := intp.Stack[len(intp.Stack)-2]
			b := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]

			isEqual, err := equal(a, b)
			if err != nil {
				return err
			}

			intp.Stack = append(intp.Stack, Boolean(!isEqual))
			return nil
		}),
		"noaccess": builtin(func(intp *Interpreter) error {
			// not implemented
			return nil
		}),
		"not": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "not: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-1]
			switch obj := obj.(type) {
			case Boolean:
				intp.Stack[len(intp.Stack)-1] = !obj
			case Integer:
				intp.Stack[len(intp.Stack)-1] = ^obj
			default:
				return intp.e(eTypecheck, "not: invalid argument type %T", obj)
			}
			return nil
		}),
		"or": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "or: not enough arguments")
			}
			a := intp.Stack[len(intp.Stack)-2]
			b := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			switch a := a.(type) {
			case Boolean:
				b, ok := b.(Boolean)
				if !ok {
					return intp.e(eTypecheck, "or: mismatched argument types")
				}
				intp.Stack = append(intp.Stack, a || b)
			case Integer:
				b, ok := b.(Integer)
				if !ok {
					return intp.e(eTypecheck, "or: mismatched argument types")
				}
				intp.Stack = append(intp.Stack, a|b)
			default:
				return intp.e(eTypecheck, "or: invalid argument type %T", a)
			}
			return nil
		}),
		"pop": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "pop: not enough arguments")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			return nil
		}),
		"put": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 3 {
				return intp.e(eStackunderflow, "put: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-3]
			sel := intp.Stack[len(intp.Stack)-2]
			value := intp.Stack[len(intp.Stack)-1]
			intp.Stack = intp.Stack[:len(intp.Stack)-3]
			switch obj := obj.(type) {
			case Array:
				index, ok := sel.(Integer)
				if !ok {
					return intp.e(eTypecheck, "put: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return intp.e(eRangecheck, "put: index %d out of range", index)
				}
				obj[index] = value
			case Procedure:
				index, ok := sel.(Integer)
				if !ok {
					return intp.e(eTypecheck, "put: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return intp.e(eRangecheck, "put: index %d out of range", index)
				}
				obj[index] = value
			case Dict:
				key, ok := sel.(Name)
				if !ok {
					return intp.e(eTypecheck, "put: invalid dict key")
				}
				obj[key] = value
			case String:
				index, ok := sel.(Integer)
				if !ok {
					return intp.e(eTypecheck, "put: invalid index")
				}
				if index < 0 || index >= Integer(len(obj)) {
					return intp.e(eRangecheck, "put: index %d out of range", index)
				}
				c, ok := value.(Integer)
				if !ok {
					return intp.e(eTypecheck, "put: invalid value")
				}
				obj[index] = byte(c)
			default:
				return intp.e(eTypecheck, "put: invalid argument type %T", obj)
			}
			return nil
		}),
		"putinterval": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 3 {
				return intp.e(eStackunderflow, "putinterval: not enough arguments")
			}
			dst := intp.Stack[len(intp.Stack)-3]
			index, ok := intp.Stack[len(intp.Stack)-2].(Integer)
			if !ok {
				return intp.e(eTypecheck, "putinterval: invalid index")
			} else if index < 0 {
				return intp.e(eRangecheck, "putinterval: index out of range")
			}
			src := intp.Stack[len(intp.Stack)-1]

			switch dst := dst.(type) {
			case Array:
				src, ok := src.(Array)
				if !ok {
					return intp.e(eTypecheck, "putinterval: mismatched argument types")
				}
				if int(index)+len(src) > len(dst) {
					return intp.e(eRangecheck, "putinterval: index out of range")
				}
				copy(dst[index:], src)
			case String:
				src, ok := src.(String)
				if !ok {
					return intp.e(eTypecheck, "putinterval: mismatched argument types")
				}
				if int(index)+len(src) > len(dst) {
					return intp.e(eRangecheck, "putinterval: index out of range")
				}
				copy(dst[index:], src)
			default:
				return intp.e(eTypecheck, "putinterval: invalid argument type %T", dst)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-3]
			return nil
		}),
		"readonly": builtin(func(intp *Interpreter) error {
			// not implemented
			return nil
		}),
		"readstring": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "readstring: not enough arguments")
			}
			buf, ok := intp.Stack[len(intp.Stack)-1].(String)
			if !ok {
				return intp.e(eTypecheck, "readstring: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			s := intp.scanners[len(intp.scanners)-1]
			_, err := s.Next()
			if err != nil && err != io.EOF {
				return err
			}
			n, err := s.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			intp.Stack = append(intp.Stack, buf[:n])
			intp.Stack = append(intp.Stack, Boolean(n == len(buf)))
			return nil
		}),
		"repeat": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "repeat: not enough arguments")
			}
			count, ok := intp.Stack[len(intp.Stack)-2].(Integer)
			if !ok {
				return intp.e(eTypecheck, "repeat: invalid argument")
			} else if count < 0 {
				return intp.e(eRangecheck, "repeat: negative count")
			}
			proc, ok := intp.Stack[len(intp.Stack)-1].(Procedure)
			if !ok {
				return intp.e(eTypecheck, "repeat: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			for i := Integer(0); i < count; i++ {
				err := intp.executeOne(proc, true)
				if err == errStop {
					break
				} else if err != nil {
					return err
				}
			}
			return nil
		}),
		"roll": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "roll: not enough arguments")
			}
			n, ok := intp.Stack[len(intp.Stack)-2].(Integer)
			if !ok {
				return intp.e(eTypecheck, "roll: invalid argument")
			}
			if n < 0 || n > Integer(len(intp.Stack)-2) {
				return intp.e(eRangecheck, "roll: length %d out of bounds", n)
			}
			j, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return intp.e(eTypecheck, "roll: count %d out of bounds", j)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			if n == 0 {
				return nil
			}
			j %= n
			if j < 0 {
				j += n
			}

			// Remove j elements from the top of the stack, and insert these
			// between the intp.Stack[len(intp.Stack)-n:] and the rest of the
			// stack.
			ji := int(j)
			ni := int(n)
			data := intp.Stack[len(intp.Stack)-ni:]
			tmp := make([]Object, j)
			copy(tmp, data[ni-ji:])
			copy(data[ji:], data[:ni-ji])
			copy(data, tmp)

			return nil
		}),
		"StandardEncoding": StandardEncoding,
		"stop": builtin(func(intp *Interpreter) error {
			return errStop
		}),
		"string": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "string: not enough arguments")
			}
			size, ok := intp.Stack[len(intp.Stack)-1].(Integer)
			if !ok {
				return intp.e(eTypecheck, "string: invalid argument")
			} else if size < 0 {
				return intp.e(eRangecheck, "string: invalid size %d", size)
			} else if size > maxStringSize {
				return intp.e(eLimitcheck, "string: invalid size %d", size)
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			intp.Stack = append(intp.Stack, make(String, size))
			return nil
		}),
		"sub": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 2 {
				return intp.e(eStackunderflow, "sub: not enough arguments")
			}
			ar, aIsReal := intp.Stack[len(intp.Stack)-2].(Real)
			ai, aIsInt := intp.Stack[len(intp.Stack)-2].(Integer)
			br, bIsReal := intp.Stack[len(intp.Stack)-1].(Real)
			bi, bIsInt := intp.Stack[len(intp.Stack)-1].(Integer)
			if !(aIsReal || aIsInt) || !(bIsReal || bIsInt) {
				return intp.e(eTypecheck, "sub: needs numbers as arguments")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-2]
			if aIsReal || bIsReal {
				if aIsInt {
					ar = Real(ai)
				}
				if bIsInt {
					br = Real(bi)
				}
				intp.Stack = append(intp.Stack, ar-br)
			} else {
				ci := ai - bi
				// check for integer overflow
				if (ai < 0 && bi > 0 && ci >= 0) || (ai > 0 && bi < 0 && ci <= 0) {
					intp.Stack = append(intp.Stack, Real(ai)-Real(bi))
				} else {
					intp.Stack = append(intp.Stack, ci)
				}
			}
			return nil
		}),
		"true": Boolean(true),
		"type": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "type: not enough arguments")
			}
			obj := intp.Stack[len(intp.Stack)-1]
			var tp Name
			switch obj.(type) {
			case Array, Procedure:
				tp = "arraytype"
			case Boolean:
				tp = "booleantype"
			case Dict:
				tp = "dicttype"
			case nil: // TODO(voss)
				tp = "filetype"
			// fonttype
			// gstatetype (LanguageLevel 2)
			case Integer:
				tp = "integertype"
			case Name, Operator:
				tp = "nametype"
			// tp = "nulltype"
			case builtin:
				tp = "operatortype"
			// tp = "packedarraytype" (LanguageLevel 2)
			case Real:
				tp = "realtype"
			// tp = "savetype"
			case String:
				tp = "stringtype"
			case mark:
				tp = "marktype"
			default:
				return intp.e(eTypecheck, "type: not implemented for %T", obj)
			}
			intp.Stack = append(intp.Stack, tp)
			return nil
		}),
		"userdict": userDict,
		"where": builtin(func(intp *Interpreter) error {
			if len(intp.Stack) < 1 {
				return intp.e(eStackunderflow, "where: not enough arguments")
			}
			key, ok := intp.Stack[len(intp.Stack)-1].(Name)
			if !ok {
				return intp.e(eTypecheck, "where: invalid argument")
			}
			intp.Stack = intp.Stack[:len(intp.Stack)-1]
			for j := len(intp.DictStack) - 1; j >= 0; j-- {
				d := intp.DictStack[j]
				if _, ok := d[key]; ok {
					intp.Stack = append(intp.Stack, d, Boolean(true))
					return nil
				}
			}
			intp.Stack = append(intp.Stack, Boolean(false))
			return nil
		}),
	}
	systemDict["systemdict"] = systemDict

	return systemDict
}

func equal(a, b Object) (bool, error) {
	_, aIsDict := a.(Dict)
	_, bIsDict := b.(Dict)
	if aIsDict && bIsDict {
		return isSameDict(a.(Dict), b.(Dict)), nil
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
			return nil, &postScriptError{eTypecheck, fmt.Sprintf("equality not implemented for %T", obj)}
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

func (intp *Interpreter) bindProc(proc Procedure) {
	for i, elem := range proc {
		switch obj := elem.(type) {
		case Name:
			val, err := intp.load(obj)
			if err != nil {
				continue
			}
			_, ok := val.(builtin)
			if ok {
				proc[i] = val
			}
		case Operator:
			val, err := intp.load(obj)
			if err != nil {
				continue
			}
			_, ok := val.(builtin)
			if ok {
				proc[i] = val
			}
		case Procedure:
			// be careful to avoid infinite loops
			proc[i] = nil
			intp.bindProc(obj)
			proc[i] = obj
		}
	}
}

// don't look!
func isSameDict(a, b Dict) bool {
	if len(a) != len(b) {
		return false
	}

	testKeyInt := 0
	var testKey Name
	for {
		testKey = Name(strconv.Itoa(testKeyInt))
		_, inA := a[testKey]
		if !inA {
			break
		}
		testKeyInt++
	}

	if _, inB := b[testKey]; inB {
		return false
	}

	a[testKey] = true
	_, isSame := b[testKey]
	delete(a, testKey)
	return isSame
}

// TODO(voss): try to carry around fewer copies of the standard encoding
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

var errExit = errors.New("exit")
var errStop = errors.New("stop")
