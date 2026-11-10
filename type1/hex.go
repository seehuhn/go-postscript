// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"io"
)

type hexWriter struct {
	w   io.Writer
	buf []byte
}

func (w *hexWriter) Write(p []byte) (n int, err error) {
	for _, c := range p {
		w.buf = append(w.buf, hex[c>>4], hex[c&0x0f])
		n++
		if len(w.buf) >= 78 {
			if err = w.flush(); err != nil {
				return n, err
			}
		}
	}
	return len(p), nil
}

func (w *hexWriter) Close() error {
	return w.flush()
}

func (w *hexWriter) flush() error {
	if len(w.buf) == 0 {
		return nil
	}
	w.buf = append(w.buf, '\n')
	_, err := w.w.Write(w.buf)
	w.buf = w.buf[:0]
	return err
}

const hex = "0123456789abcdef"
