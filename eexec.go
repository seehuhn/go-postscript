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
)

func eexec(intp *Interpreter) error {
	k := len(intp.DictStack)
	if len(intp.Stack) < 1 {
		return fmt.Errorf("stack underflow")
	}
	if intp.Stack[len(intp.Stack)-1] != nil {
		return fmt.Errorf("invalid argument")
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	intp.DictStack = append(intp.DictStack, intp.SystemDict)
	r, err := eexecDecode(intp.scanners[len(intp.scanners)-1])
	if err != nil {
		return err
	}
	err = intp.Execute(r)
	if err != nil && err != io.EOF {
		return err
	}
	intp.DictStack = intp.DictStack[:k]
	return nil
}

func eexecDecode(s *scanner) (io.Reader, error) {
	for {
		b, err := s.peek()
		if err != nil {
			return nil, err
		}
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			break
		}
		s.dropByte()
	}

	bb, err := s.peekN(4)
	if err != nil {
		return nil, err
	}
	isHex := true
	for _, b := range bb {
		if !('0' <= b && b <= '9' || 'a' <= b && b <= 'f' || 'A' <= b && b <= 'F') {
			isHex = false
			break
		}
	}

	return &eexecReader{
		s:        s,
		n:        4,
		R:        55665,
		c1:       52845,
		c2:       22719,
		isBinary: !isHex,
	}, err
}

type eexecReader struct {
	s         *scanner
	n         int
	R, c1, c2 uint16
	isBinary  bool
}

func (r *eexecReader) Read(p []byte) (int, error) {
	for r.n > 0 {
		_, err := r.nextPlain()
		if err != nil {
			return 0, err
		}
		r.n--
	}
	for i := range p {
		b, err := r.nextPlain()
		// os.Stdout.Write([]byte{b})
		if err != nil {
			return i, err
		}
		p[i] = b
	}
	return len(p), nil
}

func (r *eexecReader) nextPlain() (byte, error) {
	cipher, err := r.nextCipher()
	if err != nil {
		return 0, err
	}
	plain := cipher ^ byte(r.R>>8)
	r.R = (uint16(cipher)+r.R)*r.c1 + r.c2
	return plain, nil
}

func (r *eexecReader) nextCipher() (byte, error) {
	if r.isBinary {
		return r.s.next()
	}

	i := 0
	var out byte
readLoop:
	for i < 2 {
		b, err := r.s.next()
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

const (
	n = 4
	R = 55665
)
