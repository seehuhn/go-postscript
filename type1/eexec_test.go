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

package type1

import (
	"math"
	"testing"
)

func TestObfuscation(t *testing.T) {
	iv := []byte{1, 2, 3, 4}
	msg := "Hello World!"
	plain := []byte(msg)
	cipher := obfuscateCharstring(plain, iv)
	if len(cipher) != len(plain)+len(iv) {
		t.Errorf("cipher has wrong length")
	}
	if len(cipher) != cap(cipher) {
		t.Errorf("cipher has wrong capacity")
	}

	plain2 := deobfuscateCharstring(cipher, len(iv))
	if string(plain2) != msg {
		t.Errorf("deobfuscation failed")
	}
	if len(plain2) != cap(plain2) {
		t.Errorf("plain2 has wrong capacity")
	}
}

// TestDeobfuscateNegativeLenIV checks that a negative lenIV (which can come
// from a malicious /Private dict in a Type 1 font) is rejected instead of
// triggering a makeslice panic via len(cipher)-n overflow.
func TestDeobfuscateNegativeLenIV(t *testing.T) {
	cipher := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	if plain := deobfuscateCharstring(cipher, math.MinInt); plain != nil {
		t.Errorf("expected nil for negative n, got %d bytes", len(plain))
	}
	if plain := deobfuscateCharstring(cipher, -1); plain != nil {
		t.Errorf("expected nil for negative n, got %d bytes", len(plain))
	}
}
