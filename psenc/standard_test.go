// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package psenc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStandardEncoding(t *testing.T) {
	enc := make([]string, 256)
	for i := range 256 {
		enc[i] = ".notdef"
	}
	for name, c := range StandardEncodingRev {
		enc[c] = name
	}

	if d := cmp.Diff(enc, StandardEncoding[:]); d != "" {
		t.Errorf("mismatch: %s", d)
	}
}
