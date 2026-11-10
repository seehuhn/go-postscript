// Copyright 2024 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

func isStringOrName(o Object) bool {
	switch o.(type) {
	case String, Name:
		return true
	default:
		return false
	}
}

func isStringOrArray(o Object) bool {
	switch o.(type) {
	case String, Array:
		return true
	default:
		return false
	}
}
