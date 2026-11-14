// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import "io"

func eexec(intp *Interpreter) error {
	if len(intp.Stack) < 1 {
		return &postScriptError{eStackunderflow, "eexec"}
	}
	f, ok := intp.Stack[len(intp.Stack)-1].(*file)
	if !ok {
		return &postScriptError{eTypecheck, "eexec"}
	}
	intp.Stack = intp.Stack[:len(intp.Stack)-1]
	if f.isClosed() {
		// a closed file reads as end-of-file
		return nil
	}

	s := f.s
	if s.eexecMode != 0 {
		return &postScriptError{eInvalidaccess, "nested eexec not supported"}
	}

	k := len(intp.DictStack)
	intp.DictStack = append(intp.DictStack, intp.SystemDict)
	defer func() {
		s.EndEexec()
		intp.DictStack = intp.DictStack[:k]
	}()

	err := s.BeginEexec(eexecN)
	if err == io.EOF {
		// the encrypted section is empty
		return nil
	} else if err != nil {
		return err
	}
	return intp.executeFile(&file{s: s, src: f})
}

func (s *scanner) BeginEexec(ivLen int) error {
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

// EndEexec switches the scanner back to reading the input directly.
// Look-ahead bytes belong to the decryption filter, so they are discarded.
func (s *scanner) EndEexec() {
	s.eexecMode = 0
	s.peek = s.peek[:0]
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
