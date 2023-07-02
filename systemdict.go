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

import "errors"

var systemDict = Dict{
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
	"currentdict": builtin(func(intp *Interpreter) error {
		intp.Stack = append(intp.Stack, intp.DictStack[len(intp.DictStack)-1])
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
	"end": builtin(func(intp *Interpreter) error {
		if len(intp.DictStack) < 2 {
			return errors.New("end: dict stack underflow")
		}
		intp.DictStack = intp.DictStack[:len(intp.DictStack)-1]
		return nil
	}),
	"false": Boolean(false),
	"readonly": builtin(func(intp *Interpreter) error {
		// not implemented
		return nil
	}),
	"StandardEncoding": Array{
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
	},
	"true": Boolean(true),
}
