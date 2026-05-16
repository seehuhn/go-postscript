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

// Production size bounds for scanner-produced objects.  Set well above
// any value seen in legitimate PostScript input; each scanner copies
// these into mutable fields so tests can shrink the bounds without
// touching package state.
const (
	defaultMaxStringBytes = 16 * 1024 * 1024
	defaultMaxNameBytes   = 4096
	defaultMaxDSCBytes    = 1 << 20
)

// A scanner breaks up a PostScript input stream into tokens.
//
// Scanners are not safe for concurrent use.
type scanner struct {
	Line int // 0-based
	Col  int // 0-based
	DSC  []Comment

	src         io.Reader
	buf         []byte
	pos, used   int
	crSeen      bool
	peek        []byte
	regurgitate bool

	eexecMode  int // 0 = off, 1 = ascii, 2 = binary
	eexecState uint16

	// Per-scanner size bounds.  Initialised from the package-level
	// defaults; tests override these directly.
	//
	// maxStringBytes caps the byte length of string, hex-string, and
	// base85-string objects.
	maxStringBytes int
	// maxNameBytes caps the byte length of name and operator tokens,
	// and also of DSC comment keys.
	maxNameBytes int
	// maxDSCBytes caps the byte length stored for a single DSC comment
	// value.  Bytes past the cap are read and discarded so that parsing
	// continues normally.
	maxDSCBytes int

	// err is the first error returned by src.Read().
	// Once an error has been recorded, refill returns it on every call.
	err error
}

type Comment struct {
	Key   string
	Value string
}

func newScanner(r io.Reader) *scanner {
	return &scanner{
		src:            r,
		buf:            make([]byte, 512),
		maxStringBytes: defaultMaxStringBytes,
		maxNameBytes:   defaultMaxNameBytes,
		maxDSCBytes:    defaultMaxDSCBytes,
	}
}

func (s *scanner) Read(p []byte) (int, error) {
	for n := range p {
		b, err := s.ReadByte()
		if err != nil {
			return n, err
		}
		p[n] = b
	}
	return len(p), nil
}

func (s *scanner) ScanToken() (Object, error) {
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
			if class[b] != regular {
				break
			}
			if len(name) >= s.maxNameBytes {
				return nil, &postScriptError{eLimitcheck, "name too long"}
			}
			s.SkipByte()
			name = append(name, b)
		}
		return Name(name), nil
	default:
		s.SkipByte()
		opBytes := []byte{b}
		if class[b] == regular {
			for {
				b, err := s.Peek()
				if err == io.EOF {
					break
				} else if err != nil {
					return nil, err
				}
				if class[b] != regular {
					break
				}
				if len(opBytes) >= s.maxNameBytes {
					return nil, &postScriptError{eLimitcheck, "operator too long"}
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

func (s *scanner) ReadString() (String, error) {
	err := s.SkipRequiredByte('(')
	if err != nil {
		return nil, err
	}
	var res []byte
	bracketLevel := 1
	ignoreLF := false
	for {
		b, err := s.ReadByte()
		if err != nil {
			return nil, err
		}
		if ignoreLF {
			ignoreLF = false
			if b == 10 {
				continue
			}
		}
		var out byte
		switch b {
		case '(':
			bracketLevel++
			out = b
		case ')':
			bracketLevel--
			if bracketLevel == 0 {
				return String(res), nil
			}
			out = b
		case '\\':
			b, err = s.ReadByte()
			if err != nil {
				return nil, err
			}
			switch b {
			case 'n':
				out = '\n'
			case 'r':
				out = '\r'
			case 't':
				out = '\t'
			case 'b':
				out = '\b'
			case 'f':
				out = '\f'
			case 10: // LF
				// line continuation
				continue
			case 13: // CR or CR+LF
				// line continuation; ignore an immediately following LF
				ignoreLF = true
				continue
			case '0', '1', '2', '3', '4', '5', '6', '7':
				oct := b - '0'
				for range 2 {
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
				out = oct
			default:
				out = b
			}
		case 13: // CR or CR+LF
			out = '\n'
			ignoreLF = true
		default:
			out = b
		}
		if len(res) >= s.maxStringBytes {
			return nil, &postScriptError{eLimitcheck, "string too long"}
		}
		res = append(res, out)
	}
}

func (s *scanner) ReadHexString() (String, error) {
	err := s.SkipRequiredByte('<')
	if err != nil {
		return nil, err
	}

	var res []byte
	first := true
	var hi byte
readLoop:
	for {
		b, err := s.ReadByte()
		if err != nil {
			return nil, err
		}
		var lo byte
		switch {
		case b == '>':
			break readLoop
		case class[b] == space:
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
			if len(res) >= s.maxStringBytes {
				return nil, &postScriptError{eLimitcheck, "hex string too long"}
			}
			res = append(res, hi|lo)
			first = true
		}
	}
	if !first {
		if len(res) >= s.maxStringBytes {
			return nil, &postScriptError{eLimitcheck, "hex string too long"}
		}
		res = append(res, hi)
	}

	return String(res), nil
}

func (s *scanner) ReadBase85String() (String, error) {
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
		b, err := s.ReadByte()
		if err != nil {
			return nil, err
		}
		switch {
		case b == '~':
			break readLoop
		case class[b] == space:
			continue
		case b == 'z' && pos == 0:
			if len(res)+4 > s.maxStringBytes {
				return nil, &postScriptError{eLimitcheck, "base85 string too long"}
			}
			res = append(res, 0, 0, 0, 0)
		case b >= '!' && b <= 'u':
			val = val*85 + uint32(b-'!')
			pos++
			if pos == 5 {
				if len(res)+4 > s.maxStringBytes {
					return nil, &postScriptError{eLimitcheck, "base85 string too long"}
				}
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
		if len(res)+pos-1 > s.maxStringBytes {
			return nil, &postScriptError{eLimitcheck, "base85 string too long"}
		}
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
func (s *scanner) SkipWhiteSpace() error {
	for {
		b, err := s.Peek()
		if err != nil {
			return err
		}
		if class[b] == space {
			s.SkipByte()
		} else if b == '%' {
			if s.Col == 0 && s.LookingAt("%%") {
				key, val, err := s.readStructuredComment()
				if err == nil {
					s.DSC = append(s.DSC, Comment{key, val})
					continue
				}
			} else {
				s.SkipComment()
			}
		} else {
			return nil
		}
	}
}

// readStructuredComment reads the next structured comment into a key-value pair.
func (s *scanner) readStructuredComment() (key, value string, err error) {
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

func (s *scanner) readCommentKey() (string, error) {
	var buf bytes.Buffer
	for {
		b, err := s.Peek()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		if class[b] == space {
			break
		}
		s.SkipByte()
		if b == ':' {
			break
		}
		if buf.Len() >= s.maxNameBytes {
			return "", &postScriptError{eLimitcheck, "DSC key too long"}
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
// Values longer than s.maxDSCBytes are truncated; the remaining input is
// still consumed so the surrounding parse continues normally.
func (s *scanner) readCommentValue() (string, error) {
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
			if b == '\n' || b == '\r' || class[b] != space {
				break
			}
			s.SkipByte()
		}

		for {
			b, err := s.ReadByte()
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
			if buf.Len() < s.maxDSCBytes {
				buf.WriteByte(b)
			}
		}

		if s.LookingAt("%%+") {
			s.SkipN(3)
			if buf.Len() < s.maxDSCBytes {
				buf.WriteByte(' ')
			}
			continue commentLineLoop
		}

		break
	}

	return buf.String(), nil
}

// SkipComment skips everything from a % to the end of the line (both inclusive).
func (s *scanner) SkipComment() {
	err := s.SkipRequiredByte('%')
	if err == nil {
		s.SkipToEOL()
	}
}

func (s *scanner) SkipToEOL() {
	for {
		b, err := s.ReadByte()
		if err != nil {
			return
		} else if b == 10 { // LF
			return
		} else if b == 13 { // CR or CR+LF
			s.SkipOptionalByte(10)
			return
		}
	}
}

func (s *scanner) LookingAt(pat string) bool {
	return string(s.PeekN(len(pat))) == pat
}

// SkipByte skips a single byte of input
func (s *scanner) SkipByte() {
	s.ReadByte()
}

func (s *scanner) SkipRequiredByte(expected byte) error {
	seen, err := s.ReadByte()
	if err != nil {
		return err
	}
	if seen != expected {
		return &postScriptError{eSyntaxerror, fmt.Sprintf("expected %q, got %q", expected, seen)}
	}
	return nil
}

func (s *scanner) SkipOptionalByte(b byte) {
	next, err := s.Peek()
	if err == nil && next == b {
		s.ReadByte()
	}
}

// SkipN skips N bytes which have already been peeked.
func (s *scanner) SkipN(n int) {
	for range n {
		s.ReadByte()
	}
}

func (s *scanner) Peek() (byte, error) {
	for len(s.peek) == 0 {
		b, err := s.readByte()
		if err != nil {
			return 0, err
		}
		s.peek = append(s.peek, b)
	}
	return s.peek[0], nil
}

func (s *scanner) PeekN(n int) []byte {
	for len(s.peek) < n {
		b, err := s.readByte()
		if err != nil {
			return s.peek
		}
		s.peek = append(s.peek, b)
	}
	return s.peek[:n]
}

func (s *scanner) ReadByte() (byte, error) {
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

func (s *scanner) readByte() (byte, error) {
	if s.eexecMode == 0 {
		return s.readByteRaw()
	}

	b, err := s.readByteEexec()
	if err != nil {
		return 0, err
	}
	return s.eexecDecode(b), nil
}

func (s *scanner) readByteEexec() (byte, error) {
	if s.eexecMode == 2 { // binary eexec
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
		case class[b] == space:
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

func (s *scanner) readByteRaw() (byte, error) {
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

func (s *scanner) refill() error {
	if s.err != nil {
		return s.err
	}
	s.used = copy(s.buf, s.buf[s.pos:s.used])
	s.pos = 0

	n, err := s.src.Read(s.buf[s.used:])
	s.used += n
	if err != nil {
		s.err = err
	}
	if n > 0 {
		err = nil
	}
	return err
}

type characterClass byte

const (
	regular characterClass = iota
	space
	delimiter
)

// class classifies each byte per PostScript PLRM 3.1 (whitespace + special
// characters). regular is the zero value, so unlisted bytes are regular
// by default.
var class = [256]characterClass{
	0:   space,
	9:   space,
	10:  space,
	12:  space,
	13:  space,
	32:  space,
	'(': delimiter,
	')': delimiter,
	'<': delimiter,
	'>': delimiter,
	'[': delimiter,
	']': delimiter,
	'{': delimiter,
	'}': delimiter,
	'/': delimiter,
	'%': delimiter,
}

func parseNumber(s []byte) (Object, error) {
	x, err := strconv.ParseInt(string(s), 10, 0)
	if err == nil {
		return Integer(x), nil
	}

	y, err := strconv.ParseFloat(string(s), 64)
	if errors.Is(err, strconv.ErrRange) {
		return nil, &postScriptError{eLimitcheck, fmt.Sprintf("number %q out of range", s)}
	} else if err == nil && !math.IsInf(y, 0) && !math.IsNaN(y) {
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
