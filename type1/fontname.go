// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
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
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxFontNameLen is the greatest number of bytes a PostScript font name may
// occupy.
const MaxFontNameLen = 127

// allowedInName reports whether r may appear in a PostScript name.  The
// PostScript delimiters and white space are excluded, together with the
// characters which cannot be shown: the control characters, the format
// characters, and the spaces outside ASCII.
func allowedInName(r rune) bool {
	switch r {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%', ' ':
		return false
	}
	return unicode.IsPrint(r)
}

// RepairFontName makes a font name read from a font file usable, by removing
// the characters a PostScript font name may not hold, together with the byte
// sequences which are not valid UTF-8.  A name which is still too long
// afterwards is dropped rather than truncated, since a truncated name would
// stand for a different font, and the empty string is returned instead.
//
// This is for readers, which must accept whatever a file holds but may only
// keep a value they can write back out again.  Callers which reject an invalid
// name instead use [CheckFontName].
//
// The bytes above 127 are kept where they spell characters, so a name written
// in UTF-8 survives.  A font name is carried in two ways: a CFF Name INDEX
// states no encoding at all and its bytes are read as UTF-8, which is what a
// font naming itself outside ASCII uses in practice, whereas bytes which are
// not valid UTF-8 mean nothing a caller can act on.
func RepairFontName(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.Map(func(r rune) rune {
		if allowedInName(r) {
			return r
		}
		return -1
	}, s)
	if len(s) > MaxFontNameLen {
		return ""
	}
	return s
}

// CheckFontName returns an error if s cannot be written as a PostScript font
// name, because it is longer than [MaxFontNameLen], is not valid UTF-8, or
// holds a character such a name may not contain: white space, one of the
// delimiters "(", ")", "<", ">", "[", "]", "{", "}", "/" and "%", or a
// character which cannot be shown, such as a control character.
//
// The empty string is allowed; callers which require a name check for this
// separately.
func CheckFontName(s string) error {
	if len(s) > MaxFontNameLen {
		return fmt.Errorf("font name too long (%d bytes)", len(s))
	}
	if !utf8.ValidString(s) {
		return errors.New("font name is not valid UTF-8")
	}
	for _, r := range s {
		if !allowedInName(r) {
			return errors.New("invalid character in font name")
		}
	}
	return nil
}

// maxGlyphNameLen is the greatest number of bytes a glyph name may occupy.  A
// glyph name is a PostScript name like any other, so the limit is the one
// [MaxFontNameLen] states.
const maxGlyphNameLen = MaxFontNameLen

// asciiNameOK reports whether s is a name of usable length, built only from
// the printable ASCII characters a PostScript name may hold.  This is the
// common case by far, and settles it without decoding runes.
func asciiNameOK(s string) bool {
	if s == "" || len(s) > maxGlyphNameLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !nameByte[s[i]] {
			return false
		}
	}
	return true
}

// nameByte tells, for each byte, whether it may appear in a PostScript name
// spelled in ASCII: the printable characters, less white space and the
// delimiters.
var nameByte = func() (t [256]bool) {
	for c := byte('!'); c < 0x7F; c++ {
		t[c] = true
	}
	for _, c := range []byte("()<>[]{}/%") {
		t[c] = false
	}
	return t
}()

// RepairGlyphName makes a glyph name read from a font file usable, by removing
// the characters a PostScript name may not hold, together with the byte
// sequences which are not valid UTF-8.  A name which is still longer than 127
// bytes afterwards is dropped rather than truncated, since a truncated name
// would stand for a different glyph.  The result is the empty string if
// nothing is left, in which case the glyph cannot be named and the caller
// drops it.
//
// This is for readers, which must accept whatever a file holds but may only
// keep a value they can write back out again.  Callers which reject an invalid
// name instead use [CheckGlyphName].
func RepairGlyphName(s string) string {
	if asciiNameOK(s) {
		return s
	}
	s = strings.ToValidUTF8(s, "")
	s = strings.Map(func(r rune) rune {
		if allowedInName(r) {
			return r
		}
		return -1
	}, s)
	if len(s) > maxGlyphNameLen {
		return ""
	}
	return s
}

// CheckGlyphName returns an error if s cannot be written as a PostScript glyph
// name, because it is empty, is longer than 127 bytes, is not valid UTF-8, or
// holds a character such a name may not contain: white space, one of the
// delimiters "(", ")", "<", ">", "[", "]", "{", "}", "/" and "%", or a
// character which cannot be shown, such as a control character.
func CheckGlyphName(s string) error {
	if asciiNameOK(s) {
		return nil
	}
	if s == "" {
		return errors.New("empty glyph name")
	}
	if len(s) > maxGlyphNameLen {
		return fmt.Errorf("glyph name too long (%d bytes)", len(s))
	}
	if !utf8.ValidString(s) {
		return errors.New("glyph name is not valid UTF-8")
	}
	for _, r := range s {
		if !allowedInName(r) {
			return errors.New("invalid character in glyph name")
		}
	}
	return nil
}
