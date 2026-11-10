// Copyright 2025 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

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
