// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2023  Jochen Voss <voss@seehuhn.de>
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
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/postscript/funit"
	"seehuhn.de/go/postscript/internal/debug"
)

func makeEmptyEncoding() []string {
	encoding := make([]string, 256)
	for i := range encoding {
		encoding[i] = ".notdef"
	}
	return encoding
}

func TestReadMissingFontInfo(t *testing.T) {
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		FontInfo: &FontInfo{
			FontName:   "NoInfo",
			FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{},
			Glyphs:   map[string]*Glyph{},
			Encoding: encoding,
		},
	}
	g := F.NewGlyph(".notdef", 100)
	g.MoveTo(10, 10)
	g.LineTo(20, 10)
	g.LineTo(20, 20)
	g.LineTo(10, 20)
	g.ClosePath()
	g = F.NewGlyph("A", 200)
	g.MoveTo(0, 10)
	g.LineTo(200, 10)
	g.LineTo(100, 110)
	g.ClosePath()

	buf := &bytes.Buffer{}
	err := F.Write(buf, &WriterOptions{Format: FormatNoEExec})
	if err != nil {
		t.Fatal(err)
	}

	// strip the FontInfo block from the PFA
	re := regexp.MustCompile(`(?s)/FontInfo \d+ dict dup begin\n.*?end def\n`)
	pfa := re.ReplaceAll(buf.Bytes(), nil)

	G, err := Read(bytes.NewReader(pfa))
	if err != nil {
		t.Fatalf("reading font without FontInfo: %v", err)
	}
	if G.FontName != "NoInfo" {
		t.Errorf("FontName: got %q, want %q", G.FontName, "NoInfo")
	}
	if G.FontMatrix != F.FontMatrix {
		t.Errorf("FontMatrix: got %v, want %v", G.FontMatrix, F.FontMatrix)
	}
	if len(G.Glyphs) != 2 {
		t.Errorf("glyph count: got %d, want 2", len(G.Glyphs))
	}
}

func FuzzFont(f *testing.F) {
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		CreationDate: time.Now(),
		FontInfo: &FontInfo{
			FontName:           "Test",
			Version:            "1.000",
			Notice:             "Notice",
			Copyright:          "Copyright",
			FullName:           "Test Font",
			FamilyName:         "Test Family",
			Weight:             "Bold",
			ItalicAngle:        -11.5,
			IsFixedPitch:       false,
			UnderlinePosition:  12,
			UnderlineThickness: 14,
			FontMatrix:         [6]float64{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private: &PrivateDict{
				BlueValues: []funit.Int16{0, 10, 40, 50, 100, 120},
				OtherBlues: []funit.Int16{-20, -10},
				BlueScale:  0.1,
				BlueShift:  8,
				BlueFuzz:   2,
				StdHW:      10,
				StdVW:      20,
				ForceBold:  true,
			},
			Glyphs:   map[string]*Glyph{},
			Encoding: encoding,
		},
	}
	g := F.NewGlyph(".notdef", 100)
	g.MoveTo(10, 10)
	g.LineTo(20, 10)
	g.LineTo(20, 20)
	g.LineTo(10, 20)
	g = F.NewGlyph("A", 200)
	g.MoveTo(0, 10)
	g.LineTo(200, 10)
	g.LineTo(100, 110)

	buf := &bytes.Buffer{}
	ff := []FileFormat{FormatPFA, FormatPFB, FormatBinary, FormatNoEExec}
	for _, format := range ff {
		buf.Reset()
		err := F.Write(buf, &WriterOptions{Format: format})
		if err != nil {
			f.Fatal(err)
		}
		f.Add(buf.Bytes(), uint8(format))
	}

	// a synthetic multiple master font, to exercise the MM reader and blend
	// charstring decoder
	mm := debug.MakeMMFont()
	f.Add(mm, uint8(FormatPFA))
	f.Add(mm, uint8(FormatPFB))

	f.Fuzz(func(t *testing.T, data []byte, format uint8) {
		i1, err := Read(bytes.NewReader(data))
		if err != nil {
			return
		}

		buf := &bytes.Buffer{}
		err = i1.Write(buf, &WriterOptions{Format: FileFormat(format % uint8(len(ff)))})
		if err != nil {
			t.Fatal(err)
		}

		i2, err := Read(bytes.NewReader(buf.Bytes()))
		if err != nil {
			os.WriteFile("debug.pfa", buf.Bytes(), 0644)
			t.Fatal(err)
		}

		// The writer does not re-emit multiple master data: writing a MM font
		// emits a single-master snapshot of its default instance, so i1 and i2
		// need not match.  The settled contract is idempotence from the first
		// write onward: i2 must be an ordinary font, and writing it again must
		// reproduce it exactly.
		if i1.MM != nil {
			if i2.MM != nil {
				t.Fatal("MM data survived a write/read round trip")
			}

			buf2 := &bytes.Buffer{}
			err = i2.Write(buf2, &WriterOptions{Format: FileFormat(format % uint8(len(ff)))})
			if err != nil {
				t.Fatal(err)
			}
			i3, err := Read(bytes.NewReader(buf2.Bytes()))
			if err != nil {
				os.WriteFile("debug.pfa", buf2.Bytes(), 0644)
				t.Fatal(err)
			}
			if !reflect.DeepEqual(i2, i3) {
				t.Fatal(cmp.Diff(i2, i3))
			}
			return
		}

		if !reflect.DeepEqual(i1, i2) {
			t.Fatal(cmp.Diff(i1, i2))
		}
	})
}

// TestGlyphHeightPDF checks the measured cap height and x-height, including
// the cases where the glyph is absent, blank, or below the baseline.
func TestGlyphHeightPDF(t *testing.T) {
	newFont := func(fontMatrix matrix.Matrix) *Font {
		return &Font{
			FontInfo: &FontInfo{
				FontName:   "Test",
				FontMatrix: fontMatrix,
			},
			Outlines: &Outlines{
				Private:  &PrivateDict{},
				Glyphs:   map[string]*Glyph{},
				Encoding: makeEmptyEncoding(),
			},
		}
	}

	// a box glyph of the given height, in font design units
	addBox := func(F *Font, name string, height float64) {
		g := F.NewGlyph(name, 500)
		g.MoveTo(0, 0)
		g.LineTo(400, 0)
		g.LineTo(400, height)
		g.LineTo(0, height)
		g.ClosePath()
	}

	t.Run("measured", func(t *testing.T) {
		F := newFont(matrix.Matrix{0.001, 0, 0, 0.001, 0, 0})
		addBox(F, "H", 700)
		addBox(F, "x", 450)
		if got := F.CapHeightPDF(); got != 700 {
			t.Errorf("CapHeightPDF = %g, want 700", got)
		}
		if got := F.XHeightPDF(); got != 450 {
			t.Errorf("XHeightPDF = %g, want 450", got)
		}
	})

	// the font matrix scales the measurement into PDF glyph space
	t.Run("font matrix", func(t *testing.T) {
		F := newFont(matrix.Matrix{0.0005, 0, 0, 0.0005, 0, 0})
		addBox(F, "H", 1400)
		if got := F.CapHeightPDF(); got != 700 {
			t.Errorf("CapHeightPDF = %g, want 700", got)
		}
	})

	// where the preferred glyph is absent, the next one is measured
	t.Run("fallback", func(t *testing.T) {
		F := newFont(matrix.Matrix{0.001, 0, 0, 0.001, 0, 0})
		addBox(F, "L", 680)
		addBox(F, "v", 430)
		if got := F.CapHeightPDF(); got != 680 {
			t.Errorf("CapHeightPDF = %g, want 680", got)
		}
		if got := F.XHeightPDF(); got != 430 {
			t.Errorf("XHeightPDF = %g, want 430", got)
		}
	})

	// a blank glyph is skipped in favour of the next candidate
	t.Run("blank then fallback", func(t *testing.T) {
		F := newFont(matrix.Matrix{0.001, 0, 0, 0.001, 0, 0})
		F.NewGlyph("H", 500)
		addBox(F, "I", 690)
		if got := F.CapHeightPDF(); got != 690 {
			t.Errorf("CapHeightPDF = %g, want 690", got)
		}
	})

	// a missing glyph must not be measured via the .notdef substitute
	t.Run("missing", func(t *testing.T) {
		F := newFont(matrix.Matrix{0.001, 0, 0, 0.001, 0, 0})
		addBox(F, ".notdef", 700)
		if got := F.CapHeightPDF(); got != 0 {
			t.Errorf("CapHeightPDF = %g, want 0", got)
		}
		if got := F.XHeightPDF(); got != 0 {
			t.Errorf("XHeightPDF = %g, want 0", got)
		}
	})

	t.Run("blank", func(t *testing.T) {
		F := newFont(matrix.Matrix{0.001, 0, 0, 0.001, 0, 0})
		F.NewGlyph("H", 500)
		if got := F.CapHeightPDF(); got != 0 {
			t.Errorf("CapHeightPDF = %g, want 0", got)
		}
	})

	// a glyph which sits entirely below the baseline has no usable height
	t.Run("below baseline", func(t *testing.T) {
		F := newFont(matrix.Matrix{0.001, 0, 0, 0.001, 0, 0})
		g := F.NewGlyph("H", 500)
		g.MoveTo(0, -300)
		g.LineTo(400, -300)
		g.LineTo(400, -100)
		g.ClosePath()
		if got := F.CapHeightPDF(); got != 0 {
			t.Errorf("CapHeightPDF = %g, want 0", got)
		}
	})
}
