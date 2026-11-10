// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestPeekReader(t *testing.T) {
	msg := "Hello World!"
	for i := range 2 {
		var r io.Reader
		switch i {
		case 0:
			r = strings.NewReader(msg)
		case 1:
			buf := &bytes.Buffer{}
			buf.WriteString(msg)
			r = buf
		}
		head, r2, err := peek(r, 1)
		if err != nil {
			t.Fatal(err)
		}
		if string(head) != msg[:1] {
			t.Errorf("%d: got %q, want %q", i, head, msg[:1])
		}
		all, err := io.ReadAll(r2)
		if err != nil {
			t.Fatal(err)
		}
		if string(all) != msg {
			t.Errorf("%d: got %q, want %q", i, all, msg)
		}
	}
}
