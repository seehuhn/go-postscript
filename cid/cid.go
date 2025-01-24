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

package cid

// SystemInfo describes a character collection.
// The characters within the collection are identified by a [CID].
// The meaning of CID values is specific to the character collection.
//
// See section 5.11.2 of the PLRM.
type SystemInfo struct {
	// Registry identifies the issuer of the character collection.
	Registry string

	// Ordering uniquely identifies a character collection issued by specific
	// registry
	Ordering string

	// The Supplement of an original character collection is 0. Whenever
	// additional CIDs are assigned in a character collection, the supplement
	// number is increased.
	Supplement int32
}

// CID represents a character identifier.  This identifies a character within
// a character collection.
//
// The special value 0 is used to indicate a missing glyph.
type CID uint32
