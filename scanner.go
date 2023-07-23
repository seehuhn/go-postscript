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
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
)

type Scanner struct {
	Line int // 0-based
	Col  int // 0-based
	DSC  []Comment

	r           io.Reader
	buf         []byte
	pos, used   int
	crSeen      bool
	peek        []byte
	regurgitate bool

	eexec int // 0 = off, 1 = ascii, 2 = binary
	R     uint16

	// Err is the first error returned by r.Read().
	// Once an error has been returned, all subsequent calls to .refill() will
	// return err.
	err error
}

type Comment struct {
	Key   string
	Value string
}

func newScanner(r io.Reader) *Scanner {
	return &Scanner{
		r:   r,
		buf: make([]byte, 512),
	}
}

func (s *Scanner) Read(p []byte) (int, error) {
	for n := range p {
		b, err := s.Next()
		if err != nil {
			return n, err
		}
		p[n] = b
	}
	return len(p), nil
}

func (s *Scanner) ScanToken() (Object, error) {
	err := s.SkipWhiteSpace()
	if err != nil {
		return nil, err
	}
	b, err := s.Peek()
	if err != nil {
		return nil, err
	}
	switch b {
	case '(':
		return s.ReadString()
	case '<':
		bb := s.PeekN(2)
		switch string(bb) {
		case "<<": // dict
			s.SkipByte()
			s.SkipByte()
			return Operator("<<"), nil
		case "<~": // base85-encoded string
			return s.ReadBase85String()
		default: // hex string
			return s.ReadHexString()
		}
	case '>':
		bb := s.PeekN(2)
		switch string(bb) {
		case ">>": // end dict
			s.SkipByte()
			s.SkipByte()
			return Operator(">>"), nil
		default:
			err := s.err
			if err == nil {
				err = &postScriptError{eSyntaxerror, "unexpected '>'"}
			}
			return nil, err
		}
	case '/':
		var name []byte
		s.SkipByte()
		// TODO(voss): implement "immediate names"
		for {
			b, err := s.Peek()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}
			if !isRegular(b) {
				break
			}
			s.SkipByte()
			name = append(name, b)
		}
		return Name(name), nil
	default:
		s.SkipByte()
		opBytes := []byte{b}
		if isRegular(b) {
			for {
				b, err := s.Peek()
				if err == io.EOF {
					break
				} else if err != nil {
					return nil, err
				}
				if !isRegular(b) {
					break
				}
				s.SkipByte()
				opBytes = append(opBytes, b)
			}
		}

		x, err := parseNumber(opBytes)
		if err == nil {
			return x, nil
		}

		return Operator(opBytes), nil
	}
}

func (s *Scanner) ReadString() (String, error) {
	err := s.SkipRequiredByte('(')
	if err != nil {
		return nil, err
	}
	var res []byte
	bracketLevel := 1
	ignoreLF := false
	for {
		b, err := s.Next()
		if err != nil {
			return nil, err
		}
		if ignoreLF && b == 10 {
			continue
		}
		ignoreLF = false
		switch b {
		case '(':
			bracketLevel++
			res = append(res, b)
		case ')':
			bracketLevel--
			if bracketLevel == 0 {
				return String(res), nil
			}
			res = append(res, b)
		case '\\':
			b, err = s.Next()
			if err != nil {
				return nil, err
			}
			switch b {
			case 'n':
				res = append(res, '\n')
			case 'r':
				res = append(res, '\r')
			case 't':
				res = append(res, '\t')
			case 'b':
				res = append(res, '\b')
			case 'f':
				res = append(res, '\f')
			case '(': // literal (
				res = append(res, '(')
			case ')': // literal )
				res = append(res, ')')
			case '\\': // literal \
				res = append(res, '\\')
			case 10: // LF
				// ignore
			case 13: // CR or CR+LF
				// ignore
				ignoreLF = true
			case '0', '1', '2', '3', '4', '5', '6', '7':
				oct := b - '0'
				for i := 0; i < 2; i++ {
					b, err = s.Peek()
					if err == io.EOF {
						break
					} else if err != nil {
						return nil, err
					}
					if b < '0' || b > '7' {
						break
					}
					s.SkipByte()
					oct = oct*8 + (b - '0')
				}
				res = append(res, oct)
			default:
				res = append(res, b)
			}
		case 13: // CR or CR+LF
			res = append(res, '\n')
			ignoreLF = true
		default:
			res = append(res, b)
		}
	}
}

func (s *Scanner) ReadHexString() (String, error) {
	err := s.SkipRequiredByte('<')
	if err != nil {
		return nil, err
	}

	var res []byte
	first := true
	var hi byte
readLoop:
	for {
		b, err := s.Next()
		if err != nil {
			return nil, err
		}
		var lo byte
		switch {
		case b == '>':
			break readLoop
		case b <= 32:
			continue
		case b >= '0' && b <= '9':
			lo = b - '0'
		case b >= 'A' && b <= 'F':
			lo = b - 'A' + 10
		case b >= 'a' && b <= 'f':
			lo = b - 'a' + 10
		default:
			return nil, &postScriptError{eSyntaxerror, fmt.Sprintf("invalid hex digit %q", b)}
		}
		if first {
			hi = lo << 4
			first = false
		} else {
			res = append(res, hi|lo)
			first = true
		}
	}
	if !first {
		res = append(res, hi)
	}

	return String(res), nil
}

func (s *Scanner) ReadBase85String() (String, error) {
	for _, b := range []byte{'<', '~'} {
		err := s.SkipRequiredByte(b)
		if err != nil {
			return nil, err
		}
	}

	var res []byte
	var pos int
	var val uint32
readLoop:
	for {
		b, err := s.Next()
		if err != nil {
			return nil, err
		}
		switch {
		case b == '~':
			break readLoop
		case b <= 32:
			continue
		case b == 'z' && pos == 0:
			res = append(res, 0, 0, 0, 0)
		case b >= '!' && b <= 'u':
			val = val*85 + uint32(b-'!') // TODO(voss): check for overflow?
			pos++
			if pos == 5 {
				res = append(res, byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
				pos = 0
				val = 0
			}
		default:
			return nil, &postScriptError{eSyntaxerror, fmt.Sprintf("invalid base85 digit %q", b)}
		}
	}
	switch pos {
	case 0:
		// pass
	case 1:
		return nil, &postScriptError{eSyntaxerror, "unexpected end of base85 string"}
	default:
		for i := pos; i < 5; i++ {
			val = val*85 + 84
		}
		tail := []byte{byte(val >> 24), byte(val >> 16), byte(val >> 8), byte(val)}
		res = append(res, tail[:pos-1]...)
	}

	err := s.SkipRequiredByte('>')
	if err != nil {
		return nil, err
	}

	return String(res), nil
}

// SkipWhiteSpace skips all input (including comments) until a non-whitespace
// character is found.
func (s *Scanner) SkipWhiteSpace() error {
	for {
		b, err := s.Peek()
		if err != nil {
			return err
		}
		if b <= 32 {
			s.SkipByte()
		} else if b == '%' {
			if s.Col == 0 && s.LookingAt("%%") {
				key, val, err := s.readStructuredComment()
				if err == nil {
					s.DSC = append(s.DSC, Comment{key, val})
					continue
				}
			} else {
				err = s.SkipComment()
				if err != nil {
					return err
				}
			}
		} else {
			return nil
		}
	}
}

// readStructuredComment reads the next structured comment into a key-value pair.
func (s *Scanner) readStructuredComment() (key, value string, err error) {
	if !s.LookingAt("%%") {
		err = errors.New("not a structured comment")
		return
	}
	s.SkipN(2)

	// Read key
	key, err = s.readCommentKey()
	if err != nil {
		s.SkipToEOL()
		return
	}

	// Read value
	value, err = s.readCommentValue()
	return
}

func (s *Scanner) readCommentKey() (string, error) {
	var buf bytes.Buffer
	for {
		b, err := s.Peek()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		if b <= 32 {
			break
		}
		s.SkipByte()
		if b == ':' {
			break
		}
		buf.WriteByte(b)
	}
	if buf.Len() == 0 {
		return "", errors.New("empty DSC key")
	}
	return buf.String(), nil
}

// ReadCommentValue reads the value of a structured comment.
// Multi-line values (using `%%+`) are supported.
// The method consumes the first EOL after the value.
func (s *Scanner) readCommentValue() (string, error) {
	var buf bytes.Buffer

commentLineLoop:
	for {
		for {
			b, err := s.Peek()
			if err == io.EOF {
				break
			} else if err != nil {
				return "", err
			}
			if b == '\n' || b == '\r' || b > 32 {
				break
			}
			s.SkipByte()
		}

		for {
			b, err := s.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return "", err
			} else if b == '\n' { // LF
				break
			} else if b == '\r' { // CR or CR+LF
				s.SkipOptionalByte(10)
				break
			}
			buf.WriteByte(b)
		}

		if s.LookingAt("%%+") {
			s.SkipN(3)
			buf.WriteByte(' ')
			continue commentLineLoop
		}

		break
	}

	return buf.String(), nil
}

// SkipComment skips everything from a % to the end of the line (buth inclusive).
func (s *Scanner) SkipComment() error {
	err := s.SkipRequiredByte('%')
	if err != nil {
		return err
	}
	return s.SkipToEOL()
}

func (s *Scanner) SkipToEOL() error {
	for {
		b, err := s.Next()
		if err != nil {
			return err
		} else if b == 10 { // LF
			return nil
		} else if b == 13 { // CR or CR+LF
			s.SkipOptionalByte(10)
			return nil
		}
	}
}

func (s *Scanner) LookingAt(pat string) bool {
	return string(s.PeekN(len(pat))) == pat
}

// SkipByte skips a single byte of input
func (s *Scanner) SkipByte() {
	s.Next()
}

func (s *Scanner) SkipRequiredByte(expected byte) error {
	seen, err := s.Next()
	if err != nil {
		return err
	}
	if seen != expected {
		return &postScriptError{eSyntaxerror, fmt.Sprintf("expected %q, got %q", expected, seen)}
	}
	return nil
}

func (s *Scanner) SkipOptionalByte(b byte) {
	next, err := s.Peek()
	if err == nil && next == b {
		s.Next()
	}
}

// SkipN skips N bytes which have already been peeked.
func (s *Scanner) SkipN(n int) {
	for i := 0; i < n; i++ {
		s.Next()
	}
}

func (s *Scanner) Peek() (byte, error) {
	for len(s.peek) == 0 {
		b, err := s.readByte()
		if err != nil {
			return 0, err
		}
		s.peek = append(s.peek, b)
	}
	return s.peek[0], nil
}

func (s *Scanner) PeekN(n int) []byte {
	for len(s.peek) < n {
		b, err := s.readByte()
		if err != nil {
			return s.peek
		}
		s.peek = append(s.peek, b)
	}
	return s.peek[:n]
}

func (s *Scanner) Next() (byte, error) {
	var b byte

	if len(s.peek) > 0 && !s.regurgitate {
		b = s.peek[0]
		copy(s.peek, s.peek[1:])
		s.peek = s.peek[:len(s.peek)-1]
	} else {
		var err error
		b, err = s.readByte()
		if err != nil {
			return 0, err
		}
	}

	if s.crSeen && b == 10 {
		// ignore LF after CR
	} else if b == 10 || b == 13 {
		s.Line++
		s.Col = 0
	} else {
		s.Col++
	}
	s.crSeen = (b == 13)

	return b, nil
}

func (s *Scanner) readByte() (byte, error) {
	if s.eexec == 0 {
		return s.readByteRaw()
	}

	b, err := s.readByteEexec()
	if err != nil {
		return 0, err
	}
	return s.eexecDecode(b), nil
}

func (s *Scanner) readByteEexec() (byte, error) {
	if s.eexec == 2 { // binary eexec
		return s.readByteRaw()
	}

	i := 0
	var out byte
readLoop:
	for i < 2 {
		b, err := s.readByteRaw()
		var nibble byte
		switch {
		case err != nil:
			return 0, err
		case b <= 32:
			continue readLoop
		case b >= '0' && b <= '9':
			nibble = b - '0'
		case b >= 'A' && b <= 'F':
			nibble = b - 'A' + 10
		case b >= 'a' && b <= 'f':
			nibble = b - 'a' + 10
		default:
			return 0, fmt.Errorf("invalid hex digit %q", b)
		}
		out = out<<4 | nibble
		i++
	}
	return out, nil
}

func (s *Scanner) readByteRaw() (byte, error) {
	if s.regurgitate && len(s.peek) > 0 {
		b := s.peek[0]
		copy(s.peek, s.peek[1:])
		s.peek = s.peek[:len(s.peek)-1]
		return b, nil
	}

	for s.pos >= s.used {
		err := s.refill()
		if err != nil {
			return 0, err
		}
	}

	b := s.buf[s.pos]
	s.pos++

	return b, nil
}

func (s *Scanner) refill() error {
	if s.err != nil {
		return s.err
	}
	s.used = copy(s.buf, s.buf[s.pos:s.used])
	s.pos = 0

	n, err := s.r.Read(s.buf[s.used:])
	s.used += n
	if err != nil {
		s.err = err
	}
	if n > 0 {
		err = nil
	}
	return err
}

func isRegular(b byte) bool {
	if b <= 32 {
		return false
	}
	switch b {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return false
	default:
		return true
	}
}

func parseNumber(s []byte) (Object, error) {
	x, err := strconv.ParseInt(string(s), 10, 0)
	if err == nil {
		return Integer(x), nil
	}

	y, err := strconv.ParseFloat(string(s), 64)
	// TODO(voss): limitcheck, if err == strconv.ErrRange
	if err == nil && !math.IsInf(y, 0) && !math.IsNaN(y) {
		return Real(y), nil
	}

	mm := radixNumberRe.FindSubmatch(s)
	if mm != nil {
		base, err := strconv.ParseInt(string(mm[1]), 10, 0)
		if err == nil && base >= 2 && base <= 36 {
			z, err := strconv.ParseInt(string(mm[2]), int(base), 0)
			if err == nil {
				return Integer(z), nil
			}
		}
	}

	return nil, &postScriptError{eSyntaxerror, fmt.Sprintf("invalid number %q", s)}
}

var radixNumberRe = regexp.MustCompile(`^([0-9]{1,2})#([0-9a-zA-Z]+)$`)
