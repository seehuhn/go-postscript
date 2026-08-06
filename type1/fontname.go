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

// allowedInFontName reports whether r may appear in a PostScript font name.
// The PostScript delimiters and white space are excluded, together with the
// characters which cannot be shown: the control characters, the format
// characters, and the spaces outside ASCII.
func allowedInFontName(r rune) bool {
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
		if allowedInFontName(r) {
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
		if !allowedInFontName(r) {
			return errors.New("invalid character in font name")
		}
	}
	return nil
}
