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

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"
)

func TestGlyphPath(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Glyph)
		expected []pathSegment
	}{
		{
			name:     "empty glyph",
			setup:    func(g *Glyph) {},
			expected: nil,
		},
		{
			name: "simple rectangle",
			setup: func(g *Glyph) {
				g.MoveTo(10, 10)
				g.LineTo(20, 10)
				g.LineTo(20, 20)
				g.LineTo(10, 20)
				g.ClosePath()
			},
			expected: []pathSegment{
				{cmd: path.CmdMoveTo, points: []vec.Vec2{{X: 10, Y: 10}}},
				{cmd: path.CmdLineTo, points: []vec.Vec2{{X: 20, Y: 10}}},
				{cmd: path.CmdLineTo, points: []vec.Vec2{{X: 20, Y: 20}}},
				{cmd: path.CmdLineTo, points: []vec.Vec2{{X: 10, Y: 20}}},
				{cmd: path.CmdClose, points: nil},
			},
		},
		{
			name: "path with cubic curve",
			setup: func(g *Glyph) {
				g.MoveTo(0, 0)
				g.CurveTo(10, 5, 20, 15, 30, 10)
				g.LineTo(40, 20)
			},
			expected: []pathSegment{
				{cmd: path.CmdMoveTo, points: []vec.Vec2{{X: 0, Y: 0}}},
				{cmd: path.CmdCubeTo, points: []vec.Vec2{{X: 10, Y: 5}, {X: 20, Y: 15}, {X: 30, Y: 10}}},
				{cmd: path.CmdLineTo, points: []vec.Vec2{{X: 40, Y: 20}}},
			},
		},
		{
			name: "multiple subpaths",
			setup: func(g *Glyph) {
				g.MoveTo(0, 0)
				g.LineTo(10, 0)
				g.ClosePath()
				g.MoveTo(20, 20)
				g.LineTo(30, 30)
			},
			expected: []pathSegment{
				{cmd: path.CmdMoveTo, points: []vec.Vec2{{X: 0, Y: 0}}},
				{cmd: path.CmdLineTo, points: []vec.Vec2{{X: 10, Y: 0}}},
				{cmd: path.CmdClose, points: nil},
				{cmd: path.CmdMoveTo, points: []vec.Vec2{{X: 20, Y: 20}}},
				{cmd: path.CmdLineTo, points: []vec.Vec2{{X: 30, Y: 30}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Glyph{WidthX: 100}
			tt.setup(g)

			var segments []pathSegment
			for cmd, points := range g.Path() {
				segments = append(segments, pathSegment{
					cmd:    cmd,
					points: append([]vec.Vec2(nil), points...),
				})
			}

			if len(segments) != len(tt.expected) {
				t.Errorf("got %d segments, want %d", len(segments), len(tt.expected))
				return
			}

			for i, got := range segments {
				expected := tt.expected[i]
				if got.cmd != expected.cmd {
					t.Errorf("segment %d: got command %v, want %v", i, got.cmd, expected.cmd)
				}
				if diff := cmp.Diff(got.points, expected.points); diff != "" {
					t.Errorf("segment %d points mismatch (-got +want):\n%s", i, diff)
				}
			}
		})
	}
}

// pathSegment represents a single path command with its points for testing
type pathSegment struct {
	cmd    path.Command
	points []vec.Vec2
}
