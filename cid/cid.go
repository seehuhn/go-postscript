// Copyright 2024 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package cid

import "strconv"

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

func (ROS *SystemInfo) String() string {
	return ROS.Registry + "-" + ROS.Ordering + "-" + strconv.Itoa(int(ROS.Supplement))
}

// CID represents a character identifier.  This identifies a character within
// a character collection.
//
// The special value 0 is used to indicate a missing glyph.
type CID uint32
