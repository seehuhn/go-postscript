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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestScanToken(t *testing.T) {
	in := `
	% this is a comment
	123
	-9
	1e6
	-1.
	2#1000
	16#FF
	(ABC)
	ABC
	/ABC
	23A
	23E1
	23#1
	`
	exp := []Object{
		Integer(123),
		Integer(-9),
		Real(1e6),
		Real(-1),
		Integer(0b1000),
		Integer(0xFF),
		String([]byte("ABC")),
		Operator("ABC"),
		Name("ABC"),
		Operator("23A"),
		Real(23e1),
		Integer(1),
	}
	s := newScanner(strings.NewReader(in))
	var oo []Object
	for {
		o, err := s.scanToken()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		oo = append(oo, o)
	}
	if d := cmp.Diff(exp, oo); d != "" {
		t.Errorf("unexpected objects: %s", d)
	}
}

func TestScanString(t *testing.T) {
	exp := "A(BC))\n\r\t\b\f\\DE\n%*!&}^"
	r := strings.NewReader(`(A(BC)\)\
\n\r\t\b\f\\\D\105
%*!&}^)`)
	s := newScanner(r)
	o, err := s.scanString()
	if err != nil {
		t.Fatal(err)
	}
	if string(o) != exp {
		t.Errorf("expected %q, got %q", exp, o)
	}
}

func TestScanString2(t *testing.T) {
	r := strings.NewReader("()")
	s := newScanner(r)
	o, err := s.scanString()
	if err != nil {
		t.Fatal(err)
	}
	if string(o) != "" {
		t.Errorf("expected empty string, got %q", o)
	}
}

func TestScanString3(t *testing.T) {
	for _, nl := range []string{"\n", "\r", "\r\n"} {
		r := strings.NewReader("(A\\" + nl + "B" + nl + "C)")
		s := newScanner(r)
		o, err := s.scanString()
		if err != nil {
			t.Fatal(err)
		}
		if string(o) != "AB\nC" {
			t.Errorf("expected %q, got %q", "AB\nC", o)
		}
	}
}

func TestScanString5(t *testing.T) {
	exp := string([]byte{1, 2, 3, 0, '4', 0o377})
	r := strings.NewReader(`(\1\02\003\0004\777)`)
	s := newScanner(r)
	o, err := s.scanString()
	if err != nil {
		t.Fatal(err)
	}
	if string(o) != exp {
		t.Errorf("expected %q, got %q", exp, o)
	}
}

func TestScanHexString(t *testing.T) {
	in := "<901fa>"
	out := String([]byte{0x90, 0x1f, 0xa0})
	r := strings.NewReader(in)
	s := newScanner(r)
	o, err := s.scanHexString()
	if err != nil {
		t.Fatal(err)
	}
	if string(o) != string(out) {
		t.Errorf("expected %q, got %q", out, o)
	}
}

func TestScanHexString2(t *testing.T) {
	in := "<>"
	out := String([]byte{})
	r := strings.NewReader(in)
	s := newScanner(r)
	o, err := s.scanHexString()
	if err != nil {
		t.Fatal(err)
	}
	if string(o) != string(out) {
		// TODO(voss): syntaxerror
		t.Errorf("expected %q, got %q", out, o)
	}
}

func TestBase85String(t *testing.T) {
	in := `<~z!<N?+"T~>`
	out := String([]byte{0, 0, 0, 0, 1, 2, 3, 4, 5})
	s := newScanner(strings.NewReader(in))
	o, err := s.scanBase85String()
	if err != nil {
		t.Fatal(err)
	}
	if string(o) != string(out) {
		t.Errorf("expected %q, got %q", out, o)
	}
}

func TestLineCol(t *testing.T) {
	r := strings.NewReader("1\n12\r123\r\n\n1\n")
	s := newScanner(r)
	for {
		b, err := s.next()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%d %d %q\n", s.line, s.col, b)
		switch b {
		case '1':
			if s.col != 1 {
				t.Errorf("expected col 1, got %d", s.col)
			}
		case '2':
			if s.col != 2 {
				t.Errorf("expected col 2, got %d", s.col)
			}
		case '3':
			if s.col != 3 {
				t.Errorf("expected col 3, got %d", s.col)
			}
		}
	}
	if s.line != 4 {
		t.Errorf("expected line 4, got %d", s.line)
	}
}
