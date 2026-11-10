// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import "io"

func peek(r io.Reader, n int) ([]byte, io.Reader, error) {
	buf := make([]byte, n)

	if r, ok := r.(io.ReadSeeker); ok {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, nil, err
		}
		k, err := io.ReadFull(r, buf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return nil, nil, err
		}
		_, err = r.Seek(pos, io.SeekStart)
		if err != nil {
			return nil, nil, err
		}
		return buf[:k], r, nil
	}

	k, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, nil, err
	}

	return buf[:k], &peekReader{r: r, buf: buf[:k]}, nil
}

type peekReader struct {
	r   io.Reader
	buf []byte
}

func (r *peekReader) Read(b []byte) (n int, err error) {
	if len(r.buf) == 0 {
		return r.r.Read(b)
	}
	k := min(len(b), len(r.buf))
	copy(b, r.buf[:k])
	r.buf = r.buf[k:]
	return k, nil
}
