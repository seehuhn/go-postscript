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

import "io"

func eexec(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return &postScriptError{eStackunderflow, "eexec"}
	}
	if intp.Stack[len(intp.Stack)-1] != nil {
		return &postScriptError{eTypecheck, "eexec"}
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]

	k := len(intp.DictStack)
	intp.DictStack = append(intp.DictStack, intp.SystemDict)

	s := intp.scanners[len(intp.scanners)-1]
	err := s.BeginEexec(eexecN)
	if err != nil {
		return err
	}
	err = intp.executeScanner(s)
	if err != nil && err != io.EOF {
		return err
	}
	s.EndEexec()

	intp.DictStack = intp.DictStack[:k]
	return nil
}

func (s *scanner) BeginEexec(ivLen int) error {
	if s.eexec != 0 {
		return &postScriptError{eInvalidaccess, "nested eexec not supported"}
	}

	for {
		b, err := s.Peek()
		if err != nil {
			return err
		}
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			break
		}
		s.SkipByte()
	}
	bb := s.PeekN(ivLen)
	if len(bb) < ivLen {
		return s.err
	}
	isBinary := false
	for _, b := range bb {
		if !('0' <= b && b <= '9' || 'a' <= b && b <= 'f' || 'A' <= b && b <= 'F') {
			isBinary = true
			break
		}
	}

	if isBinary {
		s.eexec = 2 // binary
	} else {
		s.eexec = 1 // hex
	}
	s.r = eexecR

	// skip the IV
	s.regurgitate = true
	for range eexecN {
		_, err := s.Next()
		if err != nil {
			return err
		}
	}
	s.regurgitate = false

	return nil
}

func (s *scanner) EndEexec() {
	s.eexec = 0
}

func (s *scanner) eexecDecode(b byte) byte {
	out := b ^ byte(s.r>>8)
	s.r = (uint16(b)+s.r)*eexecC1 + eexecC2
	return out
}

const (
	eexecN  = 4
	eexecR  = 55665
	eexecC1 = 52845
	eexecC2 = 22719
)
