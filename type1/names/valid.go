// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2025  Jochen Voss <voss@seehuhn.de>
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

package names

const maxNameLength = 31

// IsValid checks if s is a valid glyph name.
//
// See https://github.com/adobe-type-tools/agl-specification for details.
func IsValid(s string) bool {
	if s == ".notdef" {
		return true
	}

	if len(s) < 1 || len(s) > maxNameLength {
		return false
	}

	firstChar := s[0]
	if (firstChar >= '0' && firstChar <= '9') || firstChar == '.' {
		return false
	}

	for _, char := range s {
		if !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' || char == '.' || char == '_') {
			return false
		}
	}

	return true
}
