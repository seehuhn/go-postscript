// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package psenc

import "testing"

func TestISOLatin1Encoding(t *testing.T) {
	// codes where the encoding differs from StandardEncoding or has gaps
	want := map[int]string{
		0:   ".notdef",
		32:  "space",
		39:  "quoteright",
		45:  "minus",
		96:  "quoteleft",
		127: ".notdef",
		143: ".notdef",
		144: "dotlessi",
		153: ".notdef",
		156: ".notdef",
		159: "caron",
		160: "space",
		173: "hyphen",
		215: "multiply",
		247: "divide",
		255: "ydieresis",
	}
	for code, name := range want {
		if got := ISOLatin1Encoding[code]; got != name {
			t.Errorf("code %d: got %q, want %q", code, got, name)
		}
	}

	// the printable ASCII letters and digits agree with StandardEncoding
	for code := '0'; code <= 'z'; code++ {
		if code == '`' {
			continue
		}
		if ISOLatin1Encoding[code] != StandardEncoding[code] {
			t.Errorf("code %d: got %q, want %q",
				code, ISOLatin1Encoding[code], StandardEncoding[code])
		}
	}
}
