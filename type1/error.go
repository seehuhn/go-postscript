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

var (
	errStackOverflow = invalidSince("type 1 buildchar stack overflow")
	errIncomplete    = invalidSince("incomplete type 1 charstring")
)
