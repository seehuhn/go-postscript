// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2025  Jochen Voss <voss@seehuhn.de>
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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/rect"
)

func TestOutlinesGlyphBBox(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *Outlines
		matrix    matrix.Matrix
		glyphName string
		expected  rect.Rect
	}{
		{
			name: "missing glyph without notdef",
			setup: func() *Outlines {
				return &Outlines{
					Glyphs: map[string]*Glyph{},
				}
			},
			matrix:    matrix.Identity,
			glyphName: "missing",
			expected:  rect.Rect{},
		},
		{
			name: "missing glyph with notdef",
			setup: func() *Outlines {
				g := &Glyph{WidthX: 100}
				g.MoveTo(10, 5)
				g.LineTo(30, 5)
				g.LineTo(30, 20)
				g.LineTo(10, 20)
				g.ClosePath()
				return &Outlines{
					Glyphs: map[string]*Glyph{
						".notdef": g,
					},
				}
			},
			matrix:    matrix.Identity,
			glyphName: "missing",
			expected:  rect.Rect{LLx: 10, LLy: 5, URx: 30, URy: 20},
		},
		{
			name: "blank glyph",
			setup: func() *Outlines {
				g := &Glyph{WidthX: 100}
				// No drawing commands, so blank
				return &Outlines{
					Glyphs: map[string]*Glyph{
						"space": g,
					},
				}
			},
			matrix:    matrix.Identity,
			glyphName: "space",
			expected:  rect.Rect{},
		},
		{
			name: "simple rectangle",
			setup: func() *Outlines {
				g := &Glyph{WidthX: 100}
				g.MoveTo(10, 10)
				g.LineTo(20, 10)
				g.LineTo(20, 20)
				g.LineTo(10, 20)
				g.ClosePath()
				return &Outlines{
					Glyphs: map[string]*Glyph{
						"A": g,
					},
				}
			},
			matrix:    matrix.Matrix{1, 0, 0, 1, 0, 0}, // identity
			glyphName: "A",
			expected:  rect.Rect{LLx: 10, LLy: 10, URx: 20, URy: 20},
		},
		{
			name: "scaled rectangle",
			setup: func() *Outlines {
				g := &Glyph{WidthX: 100}
				g.MoveTo(10, 10)
				g.LineTo(20, 10)
				g.LineTo(20, 20)
				g.LineTo(10, 20)
				g.ClosePath()
				return &Outlines{
					Glyphs: map[string]*Glyph{
						"B": g,
					},
				}
			},
			matrix:    matrix.Scale(2, 3), // scale by 2x, 3y
			glyphName: "B",
			expected:  rect.Rect{LLx: 20, LLy: 30, URx: 40, URy: 60},
		},
		{
			name: "shifted rectangle",
			setup: func() *Outlines {
				g := &Glyph{WidthX: 100}
				g.MoveTo(0, 0)
				g.LineTo(10, 0)
				g.LineTo(10, 10)
				g.LineTo(0, 10)
				g.ClosePath()
				return &Outlines{
					Glyphs: map[string]*Glyph{
						"C": g,
					},
				}
			},
			matrix:    matrix.Translate(5, 7), // translate by (5, 7)
			glyphName: "C",
			expected:  rect.Rect{LLx: 5, LLy: 7, URx: 15, URy: 17},
		},
		{
			name: "glyph with cubic curve",
			setup: func() *Outlines {
				g := &Glyph{WidthX: 100}
				g.MoveTo(0, 0)
				g.CurveTo(10, 20, 30, 40, 50, 10)
				return &Outlines{
					Glyphs: map[string]*Glyph{
						"D": g,
					},
				}
			},
			matrix:    matrix.Identity,
			glyphName: "D",
			// BBox includes control points as conservative approximation
			expected: rect.Rect{LLx: 0, LLy: 0, URx: 50, URy: 40},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outlines := tt.setup()
			bbox := outlines.GlyphBBox(tt.matrix, tt.glyphName)

			if diff := cmp.Diff(bbox, tt.expected); diff != "" {
				t.Errorf("bounding box mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
