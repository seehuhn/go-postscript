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

func TestScanReadStringBomb(t *testing.T) {
	const limit = 100
	s := newScanner(strings.NewReader("(" + strings.Repeat("a", limit+1) + ")"))
	s.maxStringBytes = limit
	if _, err := s.ReadString(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScanReadHexStringBomb(t *testing.T) {
	const limit = 100
	s := newScanner(strings.NewReader("<" + strings.Repeat("00", limit+1) + ">"))
	s.maxStringBytes = limit
	if _, err := s.ReadHexString(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScanReadBase85StringBomb(t *testing.T) {
	const limit = 100
	// "z" in base85 expands to 4 NUL bytes per token.  limit/4 + 1 zs
	// produce limit + 4 output bytes.
	s := newScanner(strings.NewReader("<~" + strings.Repeat("z", limit/4+1) + "~>"))
	s.maxStringBytes = limit
	if _, err := s.ReadBase85String(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScanReadHexStringTailBomb(t *testing.T) {
	const limit = 100
	// 2*limit hex digits fill res to exactly limit, then one trailing
	// digit triggers the odd-tail append after the loop.
	s := newScanner(strings.NewReader("<" + strings.Repeat("00", limit) + "0>"))
	s.maxStringBytes = limit
	if _, err := s.ReadHexString(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScanReadBase85StringTailBomb(t *testing.T) {
	const limit = 100
	// limit/4 "z" tokens fill res to exactly limit, then a partial final
	// block ("!!!" => pos=3) triggers the tail-append overshoot check
	// after the loop.
	s := newScanner(strings.NewReader("<~" + strings.Repeat("z", limit/4) + "!!!~>"))
	s.maxStringBytes = limit
	if _, err := s.ReadBase85String(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScanNameBomb(t *testing.T) {
	const limit = 100
	s := newScanner(strings.NewReader("/" + strings.Repeat("a", limit+1) + " "))
	s.maxNameBytes = limit
	if _, err := s.ScanToken(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScanOperatorBomb(t *testing.T) {
	const limit = 100
	s := newScanner(strings.NewReader(strings.Repeat("a", limit+1) + " "))
	s.maxNameBytes = limit
	if _, err := s.ScanToken(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScanDSCKeyBomb(t *testing.T) {
	const limit = 100
	// A DSC key longer than maxNameBytes is rejected inside
	// readStructuredComment; the line is skipped and parsing continues, so
	// no DSC entry is recorded and the following token still scans.
	body := "%%" + strings.Repeat("A", limit+1000) + ": value\n42"
	s := newScanner(strings.NewReader(body))
	s.maxNameBytes = limit
	tok, err := s.ScanToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != Integer(42) {
		t.Errorf("got %v, want 42", tok)
	}
	if len(s.DSC) != 0 {
		t.Errorf("expected no DSC entries, got %d", len(s.DSC))
	}
}

func TestScanDSCBomb(t *testing.T) {
	const limit = 100
	// Single DSC comment with a value much longer than the cap.  Parsing
	// must complete normally, but the stored value must be truncated to
	// at most maxDSCBytes bytes.
	body := "%%X: " + strings.Repeat("a", limit+1000) + "\n"
	s := newScanner(strings.NewReader(body))
	s.maxDSCBytes = limit
	if _, err := s.ScanToken(); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if len(s.DSC) != 1 {
		t.Fatalf("expected 1 DSC entry, got %d", len(s.DSC))
	}
	if got := len(s.DSC[0].Value); got > limit {
		t.Errorf("DSC value length %d exceeds bound %d", got, limit)
	}
}

func TestScanDSCMultiLineBomb(t *testing.T) {
	const limit = 100
	// Multi-line DSC value via %%+ continuation must also be capped.
	var b strings.Builder
	b.WriteString("%%X: short\n")
	for range 10 {
		b.WriteString("%%+ " + strings.Repeat("b", 200) + "\n")
	}
	s := newScanner(strings.NewReader(b.String()))
	s.maxDSCBytes = limit
	if _, err := s.ScanToken(); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if len(s.DSC) != 1 {
		t.Fatalf("expected 1 DSC entry, got %d", len(s.DSC))
	}
	if got := len(s.DSC[0].Value); got > limit {
		t.Errorf("DSC value length %d exceeds bound %d", got, limit)
	}
}

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
		token, err := s.ScanToken()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		oo = append(oo, token)
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
	o, err := s.ReadString()
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
	o, err := s.ReadString()
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
		o, err := s.ReadString()
		if err != nil {
			t.Fatal(err)
		}
		if string(o) != "AB\nC" {
			t.Errorf("expected %q, got %q", "AB\nC", o)
		}
	}
}

func TestScanStringMultiEOL(t *testing.T) {
	// PostScript Language Reference: an unescaped end-of-line marker inside
	// a literal string is normalised to a single LF, whether the marker is
	// CR, LF, or CR-LF. Consecutive markers therefore yield consecutive LFs.
	cases := []struct {
		in   string
		want string
	}{
		{"(a\r\rb)", "a\n\nb"},
		{"(a\r\n\nb)", "a\n\nb"},
		{"(a\r\n\r\nb)", "a\n\nb"},
	}
	for _, c := range cases {
		s := newScanner(strings.NewReader(c.in))
		o, err := s.ReadString()
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if string(o) != c.want {
			t.Errorf("%q: got %q, want %q", c.in, string(o), c.want)
		}
	}
}

func TestScanString5(t *testing.T) {
	exp := string([]byte{1, 2, 3, 0, '4', 0o377})
	r := strings.NewReader(`(\1\02\003\0004\777)`)
	s := newScanner(r)
	o, err := s.ReadString()
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
	o, err := s.ReadHexString()
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
	o, err := s.ReadHexString()
	if err != nil {
		t.Fatal(err)
	}
	if string(o) != string(out) {
		t.Errorf("expected %q, got %q", out, o)
	}
}

func TestBase85String(t *testing.T) {
	in := `<~z!<N?+"T~>`
	out := String([]byte{0, 0, 0, 0, 1, 2, 3, 4, 5})
	s := newScanner(strings.NewReader(in))
	o, err := s.ReadBase85String()
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
		b, err := s.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		// fmt.Printf("%d %d %q\n", s.line, s.col, b)
		switch b {
		case '1':
			if s.Col != 1 {
				t.Errorf("expected col 1, got %d", s.Col)
			}
		case '2':
			if s.Col != 2 {
				t.Errorf("expected col 2, got %d", s.Col)
			}
		case '3':
			if s.Col != 3 {
				t.Errorf("expected col 3, got %d", s.Col)
			}
		}
	}
	if s.Line != 5 {
		t.Errorf("expected line 5, got %d", s.Line)
	}
}

func TestLineCol2(t *testing.T) {
	type testCase struct {
		in   string
		line int
		col  int
	}
	cases := []testCase{
		{"1", 0, 1},
		{" 1", 0, 2},
		{"\n1", 1, 1},
		{"\n\n\n\n1", 4, 1},
	}
	for _, c := range cases {
		s := newScanner(strings.NewReader(c.in))
		token, err := s.ScanToken()
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if token != Integer(1) {
			t.Errorf("expected %q, got %q", Integer(1), token)
		}
		if s.Line != c.line || s.Col != c.col {
			t.Errorf("expected line %d col %d, got %d %d", c.line, c.col, s.Line, s.Col)
		}
	}
}

func TestDSC(t *testing.T) {
	in := `%!PS-Adobe-3.0
%%Creator: (seehuhn.de/go/pdf)
%%CreationDate: today
%%+ or tomorrow
%%EOF`
	s := newScanner(strings.NewReader(in))
	token, err := s.ScanToken()
	if err != io.EOF {
		t.Errorf("expected EOF, got %q", token)
	}
	if len(s.DSC) != 3 {
		fmt.Println(s.DSC)
		t.Fatalf("expected 3 comments, got %d", len(s.DSC))
	}
	if s.DSC[0].Key != "Creator" || s.DSC[0].Value != "(seehuhn.de/go/pdf)" {
		t.Errorf("expected Creator, got %q", s.DSC[0])
	}
	if s.DSC[1].Key != "CreationDate" || s.DSC[1].Value != "today or tomorrow" {
		t.Errorf("expected CreationDate, got %q", s.DSC[1])
	}
	if s.DSC[2].Key != "EOF" || s.DSC[2].Value != "" {
		t.Errorf("expected EOF, got %q", s.DSC[2])
	}
}

func TestDSC2(t *testing.T) {
	in := `%%EndComments
A
/B
`
	s := newScanner(strings.NewReader(in))
	token, err := s.ScanToken()
	if err != nil {
		t.Errorf("expected nil, got %q", err)
	}
	if token != Operator("A") {
		t.Errorf("expected A, got %q", token)
	}
}

func TestDSC3(t *testing.T) {
	in := `%% Some text here
/A
`
	s := newScanner(strings.NewReader(in))
	token, err := s.ScanToken()
	if err != nil {
		t.Errorf("expected nil, got %q", err)
	}
	if token != Name("A") {
		t.Errorf("expected A, got %q", token)
	}
}
