// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/path"
	"seehuhn.de/go/geom/vec"

	"seehuhn.de/go/postscript/internal/debug"
)

func readSeacFont(t *testing.T) *Font {
	t.Helper()
	F, err := Read(bytes.NewReader(debug.MakeSeacFont()))
	if err != nil {
		t.Fatal(err)
	}
	return F
}

// TestSeacAccentPlacement checks that seac components are found through
// StandardEncoding and that the accent is shifted by adx + sbx - asb.
func TestSeacAccentPlacement(t *testing.T) {
	F := readSeacFont(t)
	g := F.Glyphs["Aacute"]
	if g == nil {
		t.Fatal("glyph Aacute missing")
	}

	// the accent is shifted by adx + sbx - asb = 120 + 30 - 60 = 90, and
	// by ady = 200
	want := (&path.Data{}).
		MoveTo(vec.Vec2{X: 30, Y: 0}).
		LineTo(vec.Vec2{X: 330, Y: 0}).
		LineTo(vec.Vec2{X: 330, Y: 700}).
		Close().
		MoveTo(vec.Vec2{X: 150, Y: 800}).
		LineTo(vec.Vec2{X: 250, Y: 800}).
		LineTo(vec.Vec2{X: 250, Y: 900}).
		Close()
	if d := cmp.Diff(want, g.Outline); d != "" {
		t.Errorf("outline mismatch (-want +got):\n%s", d)
	}
	if g.WidthX != 500 {
		t.Errorf("wrong width: got %g, want 500", g.WidthX)
	}

	// the components are unchanged
	if n := len(F.Glyphs["A"].Outline.Cmds); n != 4 {
		t.Errorf("component A modified: %d commands", n)
	}
}

// TestSeacWidth checks that a seac glyph keeps the width from its own hsbw,
// even where this differs from the base glyph's width.
func TestSeacWidth(t *testing.T) {
	F := readSeacFont(t)
	g := F.Glyphs["Awide"]
	if g == nil {
		t.Fatal("glyph Awide missing")
	}
	if g.WidthX != 600 || g.WidthY != 0 {
		t.Errorf("wrong width: got (%g, %g), want (600, 0)", g.WidthX, g.WidthY)
	}
	if d := cmp.Diff(F.Glyphs["Aacute"].Outline, g.Outline); d != "" {
		t.Errorf("outline differs from Aacute (-want +got):\n%s", d)
	}
}

// TestSeacMissingComponent checks that a seac glyph is left as decoded if a
// component is missing from the font, or if its code is unused in
// StandardEncoding.
func TestSeacMissingComponent(t *testing.T) {
	F := readSeacFont(t)
	for _, name := range []string{"Bacute", "Xacute"} {
		g := F.Glyphs[name]
		if g == nil {
			t.Fatalf("glyph %s missing", name)
		}
		if g.Outline != nil && len(g.Outline.Cmds) != 0 {
			t.Errorf("%s: unexpected outline for incomplete composite: %v", name, g.Outline)
		}
	}
}

// TestReadErrorAfterFont checks that a font is returned if the PostScript
// code fails only after the font has been defined, and that the error is
// reported if no font has been defined.
func TestReadErrorAfterFont(t *testing.T) {
	want := readSeacFont(t)

	for _, junk := range []string{"\n{ 1", "\n}", "\nundefinedname"} {
		data := append(debug.MakeSeacFont(), junk...)
		got, err := Read(bytes.NewReader(data))
		if err != nil {
			t.Errorf("%q: %v", junk, err)
			continue
		}
		if d := cmp.Diff(want, got); d != "" {
			t.Errorf("%q: font mismatch (-want +got):\n%s", junk, d)
		}
	}

	if _, err := Read(strings.NewReader("%!\n{ 1")); err == nil {
		t.Error("no font: got no error")
	}
}
