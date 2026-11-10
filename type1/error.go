// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

// InvalidFontError indicates a problem with font data.
type InvalidFontError struct {
	Reason string
}

func (err *InvalidFontError) Error() string {
	return "type1: " + err.Reason
}

func invalidSince(reason string) error {
	return &InvalidFontError{
		Reason: reason,
	}
}
