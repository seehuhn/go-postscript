// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2024  Jochen Voss <voss@seehuhn.de>
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

// TODO(voss): remove this package
package cid

// CID represents a character identifier.  This is the index of a character in
// a character collection.
//
// TODO(voss): Move this somewhere else.
// TODO(voss): Split into different types, depending on font type?
type CID uint32
