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

type ObjectKind int

const (
	KindInteger ObjectKind = iota
	KindReal
	KindBoolean
	KindString
	KindName
	KindOperator

	KindArray
	KindDict

	KindMark

	KindBuiltIn
)

type Object interface {
	Kind() ObjectKind
}

type Integer int

func (i Integer) Kind() ObjectKind {
	return KindInteger
}

type Real float64

func (r Real) Kind() ObjectKind {
	return KindReal
}

type Boolean bool

func (b Boolean) Kind() ObjectKind {
	return KindBoolean
}

type String []byte

func (s String) Kind() ObjectKind {
	return KindString
}

type Name string

func (n Name) Kind() ObjectKind {
	return KindName
}

type Operator string

func (o Operator) Kind() ObjectKind {
	return KindOperator
}

type Array []Object

func (a Array) Kind() ObjectKind {
	return KindArray
}

type Dict map[Name]Object

func (d Dict) Kind() ObjectKind {
	return KindDict
}

type mark struct{}

func (m mark) Kind() ObjectKind {
	return KindMark
}

var theMark Object = mark{}

type builtin func(*Interpreter) error

func (b builtin) Kind() ObjectKind {
	return KindBuiltIn
}
