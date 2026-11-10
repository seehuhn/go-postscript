// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

// Package afm provides a parser for Adobe Font Metrics (AFM) files.
//
// The AFM file format is documented in
// Adobe technical note #5004, "Adobe Font Metrics File Format Specification".
//
// All lengths in an AFM file are in units of 1/1000 of the em square.  This is
// independent of the font matrix of the font program the metrics describe, so
// a consumer which works in font design units must convert.
package afm
