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

// The tests here live outside the type1 package, since they read AFM metrics
// and the afm package imports type1.
package type1_test

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"seehuhn.de/go/postscript/afm"
	"seehuhn.de/go/postscript/type1"
)

// TestMMFontReal exercises the reader on real Adobe Multiple Master fonts, if
// the user has supplied any.  It looks for *.pfb files in the "mm" subdirectory
// of QUIRE_TESTFONTS and skips cleanly when none are present.
func TestMMFontReal(t *testing.T) {
	base := os.Getenv("QUIRE_TESTFONTS")
	if base == "" {
		t.Skip("external test fonts not available (set QUIRE_TESTFONTS)")
	}
	dir := filepath.Join(base, "mm")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no mm test fonts: %v", err)
	}

	found := 0
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".pfb") {
			continue
		}
		found++
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			F, err := type1.Read(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if F.MM == nil {
				t.Fatal("font parsed without MM data")
			}
			if len(F.MM.Axes) == 0 {
				t.Error("MM font has no axes")
			}
			if len(F.Glyphs) == 0 {
				t.Error("MM font has no glyphs")
			}

			// spot-check default-instance advance widths against the AFM
			afmPath := strings.TrimSuffix(filepath.Join(dir, name), filepath.Ext(name)) + ".afm"
			checkAdvanceWidths(t, F, afmPath)
		})
	}
	if found == 0 {
		t.Skip("no *.pfb MM test fonts present")
	}
}

func checkAdvanceWidths(t *testing.T, F *type1.Font, afmPath string) {
	t.Helper()
	fd, err := os.Open(afmPath)
	if err != nil {
		return // no metrics alongside the font
	}
	defer fd.Close()
	metrics, err := afm.Read(fd)
	if err != nil {
		t.Logf("skipping width check, AFM unreadable: %v", err)
		return
	}
	checked := 0
	for name, gi := range metrics.Glyphs {
		g := F.Glyphs[name]
		if g == nil {
			continue
		}
		if math.Abs(g.WidthX-gi.WidthX) > 1 {
			t.Errorf("glyph %q width: font %g, AFM %g", name, g.WidthX, gi.WidthX)
		}
		checked++
		if checked >= 5 {
			break
		}
	}
}
