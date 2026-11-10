// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

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
	if s.eexecMode != 0 {
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
		s.eexecMode = 2 // binary
	} else {
		s.eexecMode = 1 // hex
	}
	s.eexecState = eexecR

	// skip the IV
	s.regurgitate = true
	for range eexecN {
		_, err := s.ReadByte()
		if err != nil {
			return err
		}
	}
	s.regurgitate = false

	return nil
}

func (s *scanner) EndEexec() {
	s.eexecMode = 0
}

func (s *scanner) eexecDecode(b byte) byte {
	out := b ^ byte(s.eexecState>>8)
	s.eexecState = (uint16(b)+s.eexecState)*eexecC1 + eexecC2
	return out
}

const (
	eexecN  = 4
	eexecR  = 55665
	eexecC1 = 52845
	eexecC2 = 22719
)
