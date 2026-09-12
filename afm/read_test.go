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

package afm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"seehuhn.de/go/geom/rect"
)

func TestSplitFields(t *testing.T) {
	type testCase struct {
		in     string
		fields []string
		num    int
	}
	cases := []testCase{
		{"", nil, 0},
		{"   ", nil, 0},
		{"C 32", []string{"C", "32"}, 2},
		{"  B\t-10   0\r 100 200  ", []string{"B", "-10", "0", "100", "200"}, 5},
		{"B 1 2 3 4 5", []string{"B", "1", "2", "3", "4"}, 6},
	}
	for _, c := range cases {
		fields, num := splitFields(c.in)
		if num != c.num {
			t.Errorf("%q: got %d fields, want %d", c.in, num, c.num)
		}
		got := fields[:min(num, maxFields)]
		if diff := cmp.Diff(c.fields, got, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("%q: wrong fields (-want +got):\n%s", c.in, diff)
		}
	}
}

// TestKernPrealloc checks that a wrong kern pair count in the file does not
// change the data read.
func TestKernPrealloc(t *testing.T) {
	const body = `StartFontMetrics 4.1
FontName Test
StartCharMetrics 1
C 65 ; WX 100 ; N A ; B 0 0 100 100 ;
EndCharMetrics
StartKernPairs %s
KPX A A -20
EndKernPairs
EndFontMetrics
`
	for _, count := range []string{"1", "0", "-5", "999999999999", "x"} {
		m, err := Read(strings.NewReader(fmt.Sprintf(body, count)))
		if err != nil {
			t.Fatalf("count %s: %v", count, err)
		}
		want := []KernPair{{Left: "A", Right: "A", Adjust: -20}}
		if diff := cmp.Diff(want, m.Kern); diff != "" {
			t.Errorf("count %s: wrong kern pairs (-want +got):\n%s", count, diff)
		}
	}
}

// TestNoKernPairs checks that a font without kerning pairs has no kern data,
// even if the file announces a kern section.
func TestNoKernPairs(t *testing.T) {
	const body = `StartFontMetrics 4.1
FontName Test
StartKernPairs 7
EndKernPairs
EndFontMetrics
`
	m, err := Read(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if m.Kern != nil {
		t.Errorf("got %v, want nil", m.Kern)
	}
}

// TestRepeatedHeaderAllocation checks that a file of repeated section headers
// is read using memory in proportion to its size.  Each header names an entry
// count, and sizing a container for every one of them would let a small file
// claim a large amount of memory.
func TestRepeatedHeaderAllocation(t *testing.T) {
	var body strings.Builder
	for body.Len() < 1<<20 {
		body.WriteString("StartCharMetrics 999999\nEndCharMetrics\n")
		body.WriteString("StartKernPairs 999999\nEndKernPairs\n")
	}
	in := body.String()

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	if _, err := Read(strings.NewReader(in)); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)

	used := after.TotalAlloc - before.TotalAlloc
	// A faithful read stays near 1; the check only has to exclude the
	// amplification which sizing a container per header line causes.
	const maxRatio = 50
	if ratio := float64(used) / float64(len(in)); ratio > maxRatio {
		t.Errorf("allocated %d bytes for %d bytes of input (ratio %.0f, want <%d)",
			used, len(in), ratio, maxRatio)
	}
}

// benchMetrics builds a font of the size typical for a text font: a full
// 8-bit encoding, some ligatures, and a dense kerning table.
func benchMetrics() *Metrics {
	m := &Metrics{
		Glyphs:             make(map[string]*GlyphInfo),
		Encoding:           make([]string, 256),
		FontName:           "Benchmark-Regular",
		FullName:           "Benchmark Regular Text Font",
		Version:            "001.005",
		Notice:             "Copyright (c) 2026 nobody. All rights reserved.",
		CapHeight:          662,
		XHeight:            450,
		Ascent:             683,
		Descent:            -217,
		UnderlinePosition:  -100,
		UnderlineThickness: 50,
	}
	names := make([]string, 300)
	for i := range names {
		names[i] = fmt.Sprintf("glyph%03d", i)
		m.Glyphs[names[i]] = &GlyphInfo{
			WidthX: float64(200 + i%600),
			BBox:   rect.Rect{LLx: -10, LLy: -12, URx: float64(190 + i%600), URy: 700},
		}
	}
	for i := range m.Encoding {
		m.Encoding[i] = names[i%len(names)]
	}
	m.Glyphs[names[0]].Ligatures = map[string]string{names[1]: names[2]}
	for i := range 2000 {
		m.Kern = append(m.Kern, KernPair{
			Left:   names[i%len(names)],
			Right:  names[(i*7)%len(names)],
			Adjust: float64(-i % 60),
		})
	}
	return m
}

func BenchmarkRead(b *testing.B) {
	buf := &bytes.Buffer{}
	if err := benchMetrics().Write(buf); err != nil {
		b.Fatal(err)
	}
	data := buf.Bytes()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Read(bytes.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}

// TestUnusableValues checks that values which cannot be represented are
// replaced, rather than being passed on or silently wrapping around.
func TestUnusableValues(t *testing.T) {
	const header = "StartFontMetrics 4.1\nFontName Test\n"

	t.Run("headerMetrics", func(t *testing.T) {
		cases := []struct {
			key, value string
			get        func(*Metrics) float64
		}{
			{"Descender", "-2147483648", func(m *Metrics) float64 { return m.Descent }},
			{"Ascender", "1e300", func(m *Metrics) float64 { return m.Ascent }},
			{"CapHeight", "nan", func(m *Metrics) float64 { return m.CapHeight }},
			{"XHeight", "+Inf", func(m *Metrics) float64 { return m.XHeight }},
			{"UnderlinePosition", "-16777217", func(m *Metrics) float64 { return m.UnderlinePosition }},
		}
		for _, c := range cases {
			in := header + c.key + " " + c.value + "\nEndFontMetrics\n"
			m, err := Read(strings.NewReader(in))
			if err != nil {
				t.Fatalf("%s: %v", c.key, err)
			}
			if got := c.get(m); got != 0 {
				t.Errorf("%s %s: got %g, want 0", c.key, c.value, got)
			}
		}
	})

	t.Run("usableValuesKept", func(t *testing.T) {
		in := header + "CapHeight 693\nXHeight 485\nAscender 750\nDescender -250\nEndFontMetrics\n"
		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if m.CapHeight != 693 || m.XHeight != 485 || m.Ascent != 750 || m.Descent != -250 {
			t.Errorf("got %g %g %g %g", m.CapHeight, m.XHeight, m.Ascent, m.Descent)
		}
	})

	t.Run("glyphWidth", func(t *testing.T) {
		// a fractional width is legal, a placeholder is not
		in := header + "StartCharMetrics 2\nC 65 ; WX 500.5 ; N A ;\nC 66 ; WX -2147483648 ; N B ;\nEndCharMetrics\nEndFontMetrics\n"
		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if got := m.Glyphs["A"].WidthX; got != 500.5 {
			t.Errorf("got width %v, want 500.5", got)
		}
		if got := m.Glyphs["B"].WidthX; got != 0 {
			t.Errorf("got width %v, want 0", got)
		}
	})

	t.Run("glyphBBox", func(t *testing.T) {
		// One unusable coordinate costs the whole box: keeping the other three
		// would describe a glyph the file never did, and can leave a
		// rectangle whose corners are the wrong way round.
		in := header + "StartCharMetrics 2\n" +
			"C 65 ; WX 500 ; N A ; B 100 0 1e400 200 ;\n" +
			"C 66 ; WX 500 ; N B ; B 10 -20 30 40 ;\n" +
			"EndCharMetrics\nEndFontMetrics\n"
		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if got := m.Glyphs["A"].BBox; (got != rect.Rect{}) {
			t.Errorf("got bbox %v, want the zero rectangle", got)
		}
		want := rect.Rect{LLx: 10, LLy: -20, URx: 30, URy: 40}
		if got := m.Glyphs["B"].BBox; got != want {
			t.Errorf("got bbox %v, want %v", got, want)
		}
	})

	t.Run("limit", func(t *testing.T) {
		// the limit itself is still a usable value
		in := header +
			fmt.Sprintf("CapHeight %d\nXHeight %d\n", metricMax, metricMax+1) +
			"EndFontMetrics\n"
		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if m.CapHeight != metricMax {
			t.Errorf("got CapHeight %g, want %d", m.CapHeight, metricMax)
		}
		if m.XHeight != 0 {
			t.Errorf("got XHeight %g, want 0", m.XHeight)
		}
	})

	t.Run("kernAdjustment", func(t *testing.T) {
		// a fractional adjustment is legal, a placeholder is not
		in := header + "StartKernData\nStartKernPairs 2\nKPX A V -12.5\nKPX A W -2147483648\nEndKernPairs\nEndKernData\nEndFontMetrics\n"
		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Kern) != 2 {
			t.Fatalf("got %d kern pairs, want 2", len(m.Kern))
		}
		if m.Kern[0].Adjust != -12.5 || m.Kern[1].Adjust != 0 {
			t.Errorf("got adjustments %v and %v, want -12.5 and 0",
				m.Kern[0].Adjust, m.Kern[1].Adjust)
		}
	})
}

// TestTextWhichIsNotANumber checks that a field which holds no number at all
// costs the reader that field, rather than the rest of the file.
func TestTextWhichIsNotANumber(t *testing.T) {
	const in = "StartFontMetrics 4.1\nFontName Test\n" +
		"CapHeight seven hundred\n" +
		"StartCharMetrics 2\n" +
		"C xx ; WX abc ; N A ; B q w e r ;\n" +
		"C 66 ; WX 500 ; N B ;\n" +
		"EndCharMetrics\n" +
		"StartKernData\nStartKernPairs 1\nKPX A B xyz\nEndKernPairs\nEndKernData\n" +
		"EndFontMetrics\n"

	m, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if m.CapHeight != 0 {
		t.Errorf("got CapHeight %g, want 0", m.CapHeight)
	}
	a, ok := m.Glyphs["A"]
	if !ok {
		t.Fatal("glyph A missing")
	}
	if a.WidthX != 0 {
		t.Errorf("got width %g, want 0", a.WidthX)
	}
	if (a.BBox != rect.Rect{}) {
		t.Errorf("got bbox %v, want the zero rectangle", a.BBox)
	}
	// an unreadable character code leaves the glyph unencoded
	if m.Encoding[65] != ".notdef" {
		t.Errorf("glyph A is encoded as %q", m.Encoding[65])
	}
	if m.Encoding[66] != "B" {
		t.Errorf("got %q at code 66, want B", m.Encoding[66])
	}
	if len(m.Kern) != 1 || m.Kern[0].Adjust != 0 {
		t.Errorf("got kern pairs %v, want one pair with no adjustment", m.Kern)
	}
}

// TestRepeatedKey checks that a second, unusable value does not discard the
// value read from the first copy of a key.
func TestRepeatedKey(t *testing.T) {
	const in = "StartFontMetrics 4.1\nFontName Test\n" +
		"CapHeight 693\nCapHeight nan\nEndFontMetrics\n"

	m, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if m.CapHeight != 693 {
		t.Errorf("got CapHeight %g, want 693", m.CapHeight)
	}
}

// TestRepairedNames checks that a name the file format cannot carry is
// repaired on read, so that the metrics survive and can be written again.
func TestRepairedNames(t *testing.T) {
	const in = "StartFontMetrics 4.1\nFontName Test\n" +
		"StartCharMetrics 3\n" +
		"C 65 ; WX 500 ; N A(B ; L f(x fi ;\n" +
		"C 66 ; WX 600 ; N (/) ;\n" +
		"C 67 ; WX 700 ; N C ;\n" +
		"EndCharMetrics\n" +
		"StartKernData\nStartKernPairs 2\n" +
		"KPX A(B C -10\nKPX (/) C -20\n" +
		"EndKernPairs\nEndKernData\nEndFontMetrics\n"

	m, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	g, ok := m.Glyphs["AB"]
	if !ok {
		t.Fatalf("glyph AB missing, got %v", m.GlyphList())
	}
	if got := g.Ligatures["fx"]; got != "fi" {
		t.Errorf("got ligature %q, want fi", got)
	}
	if m.Encoding[65] != "AB" {
		t.Errorf("got %q at code 65, want AB", m.Encoding[65])
	}
	// a name which the repair empties leaves no glyph and no kern pair
	if len(m.Glyphs) != 2 {
		t.Errorf("got %d glyphs, want 2: %v", len(m.Glyphs), m.GlyphList())
	}
	if len(m.Kern) != 1 || m.Kern[0].Left != "AB" || m.Kern[0].Right != "C" {
		t.Errorf("got kern pairs %v, want one AB/C pair", m.Kern)
	}

	// what Read gives back, Write accepts
	if err := m.Write(io.Discard); err != nil {
		t.Errorf("write refused repaired metrics: %v", err)
	}
}

// TestLineLimit checks the two ways a line can be too long: a line which is
// already over the limit in the file, and one which would grow past it when
// written back.
func TestLineLimit(t *testing.T) {
	const header = "StartFontMetrics 4.1\nFontName Test\n"

	// charMetrics builds a character metrics line for glyph name with n
	// ligatures, packed as tightly as the file format allows.
	charMetrics := func(code int, name string, n int) string {
		var b strings.Builder
		fmt.Fprintf(&b, "C %d;WX 500;N %s;B 0 0 0 0", code, name)
		for i := range n {
			fmt.Fprintf(&b, ";L a%04d b%04d", i, i)
		}
		return b.String()
	}

	t.Run("lineDropped", func(t *testing.T) {
		// a line over the limit costs its glyph, but not the rest of the file
		long := charMetrics(65, "A", 4500)
		if len(long) <= maxLineLen {
			t.Fatalf("test line is only %d bytes", len(long))
		}
		in := header + "StartCharMetrics 2\n" + long + "\n" +
			charMetrics(66, "B", 0) + "\nEndCharMetrics\nEndFontMetrics\n"

		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := m.Glyphs["A"]; ok {
			t.Error("glyph A survived an over-long line")
		}
		if _, ok := m.Glyphs["B"]; !ok {
			t.Error("glyph B was lost with the line before it")
		}
	})

	t.Run("ligaturesTrimmed", func(t *testing.T) {
		// A line under the limit is kept, but only as many ligatures as still
		// fit once the line is written out in the wider form Write uses.
		const numLig = 1100
		long := charMetrics(65, "A", numLig)
		if len(long) > maxLineLen {
			t.Fatalf("test line is %d bytes, over the limit", len(long))
		}
		in := header + "StartCharMetrics 1\n" + long + "\nEndCharMetrics\nEndFontMetrics\n"

		m1, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		got := len(m1.Glyphs["A"].Ligatures)
		if got == 0 || got >= numLig {
			t.Fatalf("got %d ligatures, want some but fewer than %d", got, numLig)
		}

		// what survived is what Write can produce, and it reads back the same
		buf := &bytes.Buffer{}
		if err := m1.Write(buf); err != nil {
			t.Fatal(err)
		}
		m2, err := Read(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		if d := cmp.Diff(m1, m2); d != "" {
			t.Errorf("round trip failed (-want +got):\n%s", d)
		}
	})

	t.Run("longKernPair", func(t *testing.T) {
		// a kerning pair whose names leave no room for the adjustment is
		// dropped, and one with room is kept
		long := strings.Repeat("x", maxLineLen-kernPairFixed)
		in := header + "StartKernData\nStartKernPairs 2\n" +
			"KPX " + long + " y -10\n" +
			"KPX A V -20\n" +
			"EndKernPairs\nEndKernData\nEndFontMetrics\n"

		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Kern) != 1 || m.Kern[0].Left != "A" {
			t.Errorf("got kern pairs %v, want the A/V pair alone", m.Kern)
		}
	})
}

// TestHexCharacterCode checks that the hexadecimal form of the character code
// encodes the glyph just as the decimal form does.
func TestHexCharacterCode(t *testing.T) {
	const header = "StartFontMetrics 4.1\nFontName Test\n"

	in := header + "StartCharMetrics 4\n" +
		"CH <41> ; WX 500 ; N A ;\n" +
		"CH <4a> ; WX 500 ; N J ;\n" +
		"CH 5A ; WX 500 ; N Z ;\n" +
		"CH <zz> ; WX 500 ; N Q ;\n" +
		"EndCharMetrics\nEndFontMetrics\n"

	m1, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	for code, want := range map[int]string{0x41: "A", 0x4a: "J", 0x5a: "Z"} {
		if got := m1.Encoding[code]; got != want {
			t.Errorf("got %q at code %d, want %q", got, code, want)
		}
	}
	// a code we cannot read leaves the glyph unencoded, but keeps its metrics
	if _, ok := m1.Glyphs["Q"]; !ok {
		t.Error("glyph Q was lost with its code")
	}
	if slices.Contains(m1.Encoding, "Q") {
		t.Error("glyph Q was encoded from an unusable code")
	}

	// Write spells the codes out in the decimal form, which reads back the same
	buf := &bytes.Buffer{}
	if err := m1.Write(buf); err != nil {
		t.Fatal(err)
	}
	m2, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(m1, m2); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}

// TestSectionMarkers checks that a marker opens its section whether or not the
// count follows it.  The count only sizes the containers, so a file which
// leaves it out, or spells it in a way the reader cannot use, still yields the
// entries of the section.
func TestSectionMarkers(t *testing.T) {
	const body = "C 65 ; WX 500 ; N A ; B 0 0 400 700 ;\nC 66 ; WX 600 ; N B ;\n"
	const kern = "KPX A B -20\nKPX B A -10\n"

	cases := []struct{ name, count string }{
		{"count", " 2"},
		{"noCount", ""},
		{"unusableCount", " many"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := "StartFontMetrics 4.1\nFontName Test\n" +
				"StartCharMetrics" + c.count + "\n" + body + "EndCharMetrics\n" +
				"StartKernData\nStartKernPairs" + c.count + "\n" + kern +
				"EndKernPairs\nEndKernData\nEndFontMetrics\n"

			m, err := Read(strings.NewReader(in))
			if err != nil {
				t.Fatal(err)
			}
			if len(m.Glyphs) != 2 {
				t.Errorf("got %d glyphs, want 2", len(m.Glyphs))
			}
			if len(m.Kern) != 2 {
				t.Errorf("got %d kerning pairs, want 2", len(m.Kern))
			}
		})
	}
}

// TestLongValuesAreNotKept checks that Read holds back the values Write could
// not put on a line: a glyph name over the limit the type1 package sets, and a
// text value too long for the widest keyword Write builds from it.
func TestLongValuesAreNotKept(t *testing.T) {
	const header = "StartFontMetrics 4.1\nFontName Test\n"

	t.Run("glyphName", func(t *testing.T) {
		fits := strings.Repeat("x", maxNameLen(t))
		tooLong := strings.Repeat("y", maxNameLen(t)+1)
		in := header + "StartCharMetrics 2\n" +
			"C 65 ; WX 500 ; N " + fits + " ;\n" +
			"C 66 ; WX 500 ; N " + tooLong + " ;\n" +
			"EndCharMetrics\nEndFontMetrics\n"

		m, err := Read(strings.NewReader(in))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := m.Glyphs[fits]; !ok {
			t.Error("a name at the limit was dropped")
		}
		if _, ok := m.Glyphs[tooLong]; ok {
			t.Error("a name over the limit was kept")
		}
		// the glyph is gone, so its code names nothing
		if m.Encoding[66] != ".notdef" {
			t.Errorf("got %q at code 66, want .notdef", m.Encoding[66])
		}
		if err := m.Write(io.Discard); err != nil {
			t.Errorf("write refused what read returned: %v", err)
		}
	})

	t.Run("textValues", func(t *testing.T) {
		cases := []struct {
			key string
			get func(*Metrics) string
		}{
			{"FullName", func(m *Metrics) string { return m.FullName }},
			{"FamilyName", func(m *Metrics) string { return m.FamilyName }},
			{"Weight", func(m *Metrics) string { return m.Weight }},
			{"Version", func(m *Metrics) string { return m.Version }},
			{"Notice", func(m *Metrics) string { return m.Notice }},
		}
		for _, c := range cases {
			fits := strings.Repeat("x", maxTextValueLen)
			tooLong := strings.Repeat("y", maxTextValueLen+1)
			in := header + c.key + " " + fits + "\n" +
				c.key + " " + tooLong + "\nEndFontMetrics\n"

			m, err := Read(strings.NewReader(in))
			if err != nil {
				t.Fatalf("%s: %v", c.key, err)
			}
			if got := c.get(m); got != fits {
				t.Errorf("%s: got %d bytes, want the %d-byte value",
					c.key, len(got), len(fits))
			}
			if err := m.Write(io.Discard); err != nil {
				t.Errorf("%s: write refused what read returned: %v", c.key, err)
			}
		}
	})
}

// TestMaxNumberLen checks the assumption the line length bounds rest on: that
// no value Write accepts is spelled in more than maxNumberLen bytes.
func TestMaxNumberLen(t *testing.T) {
	check := func(x float64) {
		t.Helper()
		if checkMetric("test", x) != nil {
			return
		}
		for _, v := range []float64{x, -x} {
			if n := len(strconv.FormatFloat(v, 'f', -1, 64)); n > maxNumberLen {
				t.Fatalf("%g needs %d bytes, over the %d allowed", v, n, maxNumberLen)
			}
		}
	}

	check(0)
	check(metricMin)
	check(metricMax)
	check(math.Nextafter(metricMin, 0))
	check(math.Nextafter(metricMax, 0))
	check(1.0 / 3)
	check(metricMax - 1.0/3)

	// a spread of magnitudes over the whole allowed range
	rng := rand.New(rand.NewPCG(1, 2))
	lo, hi := math.Log(metricMin), math.Log(metricMax)
	for range 200000 {
		check(math.Exp(rng.Float64()*(hi-lo) + lo))
	}
}

// TestLineReader checks the line reader at the edges: the last line with and
// without a terminator, CRLF, and an over-long line in either position.
func TestLineReader(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"noFinalNewline", "a\nb", []string{"a", "b"}},
		{"finalNewline", "a\nb\n", []string{"a", "b"}},
		{"crlf", "a\r\nb\r\n", []string{"a", "b"}},
		{"blankLines", "\n\na\n", []string{"", "", "a"}},
		{"longFirst", strings.Repeat("x", maxLineLen+1) + "\na\n", []string{"a"}},
		{"longLast", "a\n" + strings.Repeat("x", maxLineLen+1), []string{"a"}},
		{"longLastNewline", "a\n" + strings.Repeat("x", maxLineLen+1) + "\n", []string{"a"}},
		{"atLimit", strings.Repeat("x", maxLineLen) + "\na\n",
			[]string{strings.Repeat("x", maxLineLen), "a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lr := &lineReader{r: bufio.NewReader(strings.NewReader(c.in))}
			var got []string
			for {
				line, ok, err := lr.next()
				if err == io.EOF {
					break
				} else if err != nil {
					t.Fatal(err)
				}
				if ok {
					got = append(got, line)
				}
			}
			if d := cmp.Diff(c.want, got); d != "" {
				t.Errorf("wrong lines (-want +got):\n%s", d)
			}
		})
	}
}
