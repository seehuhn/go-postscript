// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

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
