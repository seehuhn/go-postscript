// Copyright 2022 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

// Package funit contains types for representing font design units.
package funit

// Int16 is a 16-bit integer in font design units.
type Int16 int16

// AsFloat returns x*scale as a float64.
func (x Int16) AsFloat(scale float64) float64 {
	return float64(x) * scale
}

// Uint16 is an unsigned 16-bit integer in font design units.
// It corresponds to the UFWORD type in the OpenType specification and is
// used for non-negative quantities such as advance widths.
type Uint16 uint16

// AsFloat returns x*scale as a float64.
func (x Uint16) AsFloat(scale float64) float64 {
	return float64(x) * scale
}

// Int is an integer in font design units.
type Int int

// AsFloat returns x*scale as a float64.
func (x Int) AsFloat(scale float64) float64 {
	return float64(x) * scale
}

// Rect16 represents a rectangle in font design units.
type Rect16 struct {
	LLx, LLy, URx, URy Int16
}

func (rect Rect16) Dx() Int16 {
	return rect.URx - rect.LLx
}

func (rect Rect16) Dy() Int16 {
	return rect.URy - rect.LLy
}

// IsZero is true if the glyph leaves no marks on the page.
func (rect Rect16) IsZero() bool {
	return rect.LLx == 0 && rect.LLy == 0 && rect.URx == 0 && rect.URy == 0
}

// Extend enlarges the rectangle to also cover `other`.
func (rect *Rect16) Extend(other Rect16) {
	if other.IsZero() {
		return
	}
	if rect.IsZero() {
		*rect = other
		return
	}
	if other.LLx < rect.LLx {
		rect.LLx = other.LLx
	}
	if other.LLy < rect.LLy {
		rect.LLy = other.LLy
	}
	if other.URx > rect.URx {
		rect.URx = other.URx
	}
	if other.URy > rect.URy {
		rect.URy = other.URy
	}
}

// Rect represents a rectangle in font design units.
type Rect struct {
	LLx, LLy, URx, URy Int
}

// IsZero is true if the glyph leaves no marks on the page.
func (rect Rect) IsZero() bool {
	return rect.LLx == 0 && rect.LLy == 0 && rect.URx == 0 && rect.URy == 0
}

// Extend enlarges the rectangle to also cover `other`.
func (rect *Rect) Extend(other Rect) {
	if other.IsZero() {
		return
	}
	if rect.IsZero() {
		*rect = other
		return
	}
	if other.LLx < rect.LLx {
		rect.LLx = other.LLx
	}
	if other.LLy < rect.LLy {
		rect.LLy = other.LLy
	}
	if other.URx > rect.URx {
		rect.URx = other.URx
	}
	if other.URy > rect.URy {
		rect.URy = other.URy
	}
}

type Float64 float64
