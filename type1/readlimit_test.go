// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
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
	"bytes"
	"testing"

	"seehuhn.de/go/geom/matrix"
)

// TestReadMemoryLimit checks that Read bounds the interpreter's memory budget,
// so a hostile embedded font program cannot retain unbounded memory.  A valid
// font is built, confirmed to parse, then a memory-bomb loop is injected after
// the header.  Without a budget the loop parses fine (the allocations are
// discarded); with the budget in place Read must reject it.
func TestReadMemoryLimit(t *testing.T) {
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		FontInfo: &FontInfo{
			FontName:   "Bomb",
			FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{},
			Glyphs:   map[string]*Glyph{},
			Encoding: encoding,
		},
	}
	g := F.NewGlyph(".notdef", 100)
	g.MoveTo(0, 0)
	g.LineTo(10, 0)
	g.LineTo(10, 10)
	g.ClosePath()
	g = F.NewGlyph("A", 200)
	g.MoveTo(0, 0)
	g.LineTo(100, 0)
	g.LineTo(50, 100)
	g.ClosePath()

	buf := &bytes.Buffer{}
	if err := F.Write(buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	pfa := buf.Bytes()

	// the unmodified font must parse: the budget is generous enough for a real
	// font
	if _, err := Read(bytes.NewReader(pfa)); err != nil {
		t.Fatalf("baseline font does not parse: %v", err)
	}

	// inject a loop allocating far more than the memory budget, right after the
	// "%!" header line
	nl := bytes.IndexByte(pfa, '\n')
	if nl < 0 {
		t.Fatal("no newline in generated PFA")
	}
	bomb := []byte("\n0 1 2000 { pop 65536 array pop } for")
	hostile := append(append(append([]byte{}, pfa[:nl]...), bomb...), pfa[nl:]...)

	if _, err := Read(bytes.NewReader(hostile)); err == nil {
		t.Error("expected error for over-budget font program, got nil")
	}
}
