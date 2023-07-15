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

type scanner struct {
	r io.Reader

	buf       []byte
	pos, used int

	Line   int // 0-based
	Col    int // 0-based
	crSeen bool

	DSC        []Comment
	headerDone bool

	err error
}

type Comment struct {
	Key   string
	Value string
}

func newScanner(r io.Reader) *scanner {
	return &scanner{
		r:   r,
		buf: make([]byte, 256),
	}
}

func (s *scanner) Read(p []byte) (int, error) {
	for n := range p {
		b, err := s.next()
		if err != nil {
			return n, err
		}
		p[n] = b
	}
	return len(p), nil
}

func (s *scanner) scanToken() (Object, error) {
	err := s.skipWhiteSpace()
	if err != nil {
		return nil, err
	}
	s.headerDone = true
	b, err := s.peek()
	if err != nil {
		return nil, err
	}
	switch b {
	case '(':
		return s.readString()
	case '<':
		bb, err := s.peekN(2)
		if err != nil {
			return nil, err
		}
		switch bb[1] {
		case '<': // dict
			s.skipByte()
			s.skipByte()
			return Operator("<<"), nil
		case '~': // base85-encoded string
			return s.readBase85String()
		default: // hex string
			return s.readHexString()
		}
	case '/':
		var name []byte
		s.skipByte()
		// TODO(voss): implement "immediate names"
		for {
			b, err := s.peek()
			if err != nil {
				return nil, err
			}
			if !isRegular(b) {
				break
			}
			s.skipByte()
			name = append(name, b)
		}
		return Name(name), nil
	default:
		s.skipByte()
		opBytes := []byte{b}
		if isRegular(b) {
			for {
				b, err := s.peek()
				if err == io.EOF {
					break
				} else if err != nil {
					return nil, err
				}
				if !isRegular(b) {
					break
				}
				s.skipByte()
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

	return nil, fmt.Errorf("invalid number %q", s)
}

func (s *scanner) readString() (String, error) {
	err := s.skipRequiredByte('(')
	if err != nil {
		return nil, err
	}
	var res []byte
	bracketLevel := 1
	ignoreLF := false
	for {
		b, err := s.next()
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
			b, err = s.next()
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
					b, err = s.peek()
					if err == io.EOF {
						break
					} else if err != nil {
						return nil, err
					}
					if b < '0' || b > '7' {
						break
					}
					s.skipByte()
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

func (s *scanner) readHexString() (String, error) {
	err := s.skipRequiredByte('<')
	if err != nil {
		return nil, err
	}

	var res []byte
	first := true
	var hi byte
readLoop:
	for {
		b, err := s.next()
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
			return nil, fmt.Errorf("invalid hex digit %q", b)
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

func (s *scanner) readBase85String() (String, error) {
	for _, b := range []byte{'<', '~'} {
		err := s.skipRequiredByte(b)
		if err != nil {
			return nil, err
		}
	}

	var res []byte
	var pos int
	var val uint32
readLoop:
	for {
		b, err := s.next()
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
			// TODO(voss): syntaxerror
			return nil, fmt.Errorf("invalid base85 digit %q", b)
		}
	}
	switch pos {
	case 0:
		// pass
	case 1:
		// TODO(voss): syntaxerror
		return nil, fmt.Errorf("unexpected end marker in base85 stream")
	default:
		for i := pos; i < 5; i++ {
			val = val*85 + 84
		}
		tail := []byte{byte(val >> 24), byte(val >> 16), byte(val >> 8), byte(val)}
		res = append(res, tail[:pos-1]...)
	}

	err := s.skipRequiredByte('>')
	if err != nil {
		return nil, err
	}

	return String(res), nil
}

// skipWhiteSpace skips all whitespace and comments.
func (s *scanner) skipWhiteSpace() error {
	for {
		if s.Col == 0 && s.lookingAt("%%") {
			key, val, err := s.readStructuredComment()
			if err == nil {
				s.DSC = append(s.DSC, Comment{key, val})
				continue
			}
		}

		b, err := s.peek()
		if err != nil {
			return err
		}
		if b <= 32 {
			s.skipByte()
		} else if b == '%' {
			err = s.skipComment()
			if err != nil {
				return err
			}
		} else {
			return nil
		}
	}
}

// ReadStructuredComment reads the next structured comment into a key-value pair.
func (s *scanner) readStructuredComment() (key, value string, err error) {
	if !s.lookingAt("%%") {
		err = errors.New("not a structured comment")
		return
	}
	s.skipN(2)

	// Read key
	key, err = s.readCommentKey()
	if err != nil {
		return
	}

	// Read value
	value, err = s.readCommentValue()
	return
}

func (s *scanner) readCommentKey() (string, error) {
	var buf bytes.Buffer
	for {
		b, err := s.next()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		if b == ':' || b <= 32 {
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
func (s *scanner) readCommentValue() (string, error) {
	var buf bytes.Buffer

	for {
		for {
			b, err := s.peek()
			if err == io.EOF {
				break
			} else if err != nil {
				return "", err
			}
			if b == '\n' || b == '\r' || b > 32 {
				break
			}
			s.skipByte()
		}

		for {
			b, err := s.next()
			if err == io.EOF {
				break
			} else if err != nil {
				return "", err
			} else if b == 10 { // LF
				break
			} else if b == 13 { // CR or CR+LF
				s.skipOptionalByte(10)
				break
			}
			buf.WriteByte(b)
		}

		if !s.lookingAt("%%+") {
			break
		}
		s.skipN(3)
		buf.WriteByte(' ')
	}

	return buf.String(), nil
}

// skipComment skips everything from a % to the end of the line (buth inclusive).
func (s *scanner) skipComment() error {
	err := s.skipRequiredByte('%')
	if err != nil {
		return err
	}
	for {
		b, err := s.next()
		if err != nil {
			return err
		} else if b == 10 { // LF
			return nil
		} else if b == 13 { // CR or CR+LF
			s.skipOptionalByte(10)
			return nil
		}
	}
}

func (s *scanner) skipRequiredByte(b byte) error {
	next, err := s.next()
	if err != nil {
		return err
	}
	if next != b {
		return fmt.Errorf("expected %c, got %c", b, next)
	}
	return nil
}

func (s *scanner) skipOptionalByte(b byte) {
	next, err := s.peek()
	if err == nil && next == b {
		s.skipByte()
	}
}

func (s *scanner) refill() error {
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

func (s *scanner) lookingAt(pat string) bool {
	ahead, err := s.peekN(len(pat))
	if err != nil {
		return false
	}
	return string(ahead) == pat
}

func (s *scanner) peek() (byte, error) {
	for s.pos >= s.used {
		err := s.refill()
		if err != nil {
			return 0, err
		}
	}
	return s.buf[s.pos], nil
}

func (s *scanner) peekN(n int) ([]byte, error) {
	for s.pos+n > s.used {
		err := s.refill()
		if err != nil {
			return nil, err
		}
	}
	return s.buf[s.pos : s.pos+n], nil
}

func (s *scanner) next() (byte, error) {
	b, err := s.peek()
	if err != nil {
		return 0, err
	}
	s.skipByte()
	return b, nil
}

// skipByte skips a single byte which has already been peeked.
func (s *scanner) skipByte() {
	if s.pos >= s.used {
		panic("unreachable")
	}

	b := s.buf[s.pos]
	s.pos++

	if s.crSeen && b == 10 {
		// ignore LF after CR
	} else if b == 10 || b == 13 {
		s.Line++
		s.Col = 0
	} else {
		s.Col++
	}
	s.crSeen = (b == 13)
}

// skipN skips N bytes which have already been peeked.
func (s *scanner) skipN(n int) {
	for i := 0; i < n; i++ {
		s.skipByte()
	}
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

var radixNumberRe = regexp.MustCompile(`^([0-9]{1,2})#([0-9a-zA-Z]+)$`)
