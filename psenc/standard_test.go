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

package psenc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStandardEncoding(t *testing.T) {
	enc := make([]string, 256)
	for i := 0; i < 256; i++ {
		enc[i] = ".notdef"
	}
	for name, c := range StandardEncodingRev {
		enc[c] = name
	}

	if d := cmp.Diff(enc, StandardEncoding[:]); d != "" {
		t.Errorf("mismatch: %s", d)
	}
}
