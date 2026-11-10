// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

// Package postscript implements a rudimentary PostScript interpreter.
//
// The package implements enough of the PostScript language to be able to read
// most Type 1 fonts and CMap files.  It does not contain a rasterizer and does
// not implement any drawing operations.
package postscript
