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
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/postscript/funit"
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
		Notice:             "Copyright (c) 2026 nobody.  All rights reserved.",
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
			Adjust: funit.Int16(-i % 60),
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
