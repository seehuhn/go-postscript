// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2024  Jochen Voss <voss@seehuhn.de>
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
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"seehuhn.de/go/geom/rect"
)

var (
	testMetrics = &Metrics{
		Glyphs: map[string]*GlyphInfo{
			".notdef": {
				WidthX: 500,
				BBox: rect.Rect{
					URx: 500,
					URy: 800,
				},
			},
			"f": {
				WidthX: 400,
				BBox: rect.Rect{
					LLx: 20,
					LLy: -100,
					URx: 500,
					URy: 800,
				},
				Ligatures: map[string]string{"f": "ff"},
			},
			"ff": {
				WidthX: 700,
				BBox: rect.Rect{
					LLx: 20,
					LLy: 100,
					URx: 750,
					URy: 810,
				},
			},
			"qr": {
				WidthX: 1000,
				BBox: rect.Rect{
					URx: 1000,
					URy: 1000,
				},
			},
		},
		Encoding: []string{
			".notdef",
			"f",
			".notdef",
			"ff",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
			".notdef",
		},
		FontName:           "Test",
		FullName:           "Test Font",
		FamilyName:         "Test",
		Weight:             "Regular",
		Version:            "001.002",
		Notice:             "Copyright (c) 2026 nobody. All rights reserved.",
		CapHeight:          750,
		XHeight:            451,
		Ascent:             812,
		Descent:            -203,
		UnderlinePosition:  -400,
		UnderlineThickness: 5,
		ItalicAngle:        -6,
		IsFixedPitch:       false,
		Kern: []KernPair{
			{"f", "f", -20},
		},
	}
)

func TestWriteReadCycle(t *testing.T) {
	buf := &bytes.Buffer{}

	err := testMetrics.Write(buf)
	if err != nil {
		t.Fatal(err)
	}

	m2, err := Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(testMetrics, m2); d != "" {
		t.Fatalf("mismatch (-want +got):\n%s", d)
	}
}

func FuzzReadAFM(f *testing.F) {
	buf := &bytes.Buffer{}
	err := testMetrics.Write(buf)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(buf.Bytes())

	// seeds around the line limit, so that the corpus reaches the point where
	// Read must hold back to keep what it returns writable
	for _, numLig := range []int{1, 1100, 4500} {
		var b strings.Builder
		b.WriteString("StartFontMetrics 4.1\nFontName Test\nStartCharMetrics 1\n")
		fmt.Fprintf(&b, "C 65;WX 500;N A;B 0 0 0 0")
		for i := range numLig {
			fmt.Fprintf(&b, ";L a%04d b%04d", i, i)
		}
		b.WriteString("\nEndCharMetrics\nEndFontMetrics\n")
		f.Add([]byte(b.String()))
	}

	f.Fuzz(func(t *testing.T, data1 []byte) {
		info1, err := Read(bytes.NewReader(data1))
		if err != nil {
			return
		}

		buf := &bytes.Buffer{}
		err = info1.Write(buf)
		if err != nil {
			t.Fatal(err)
		}

		data2 := buf.Bytes()
		info2, err := Read(bytes.NewReader(data2))
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(info1, info2) {
			os.WriteFile("test1.afm", data1, 0644)
			os.WriteFile("test2.afm", data2, 0644)
			t.Fatalf("mismatch: %s", cmp.Diff(info1, info2))
		}
	})
}

// TestWriteRejectsUnusableValues checks that Write refuses numbers which
// [Read] would not give back.
func TestWriteRejectsUnusableValues(t *testing.T) {
	base := func() *Metrics {
		return &Metrics{
			FontName: "Test",
			FullName: "Test Font",
			Glyphs: map[string]*GlyphInfo{
				".notdef": {WidthX: 500},
				"A":       {WidthX: 600, BBox: rect.Rect{URx: 600, URy: 700}},
			},
			Encoding: make([]string, 256),
			Kern:     []KernPair{{"A", "A", -20}},
		}
	}

	cases := []struct {
		name    string
		breakIt func(*Metrics)
	}{
		{"CapHeight", func(m *Metrics) { m.CapHeight = math.NaN() }},
		{"Descender", func(m *Metrics) { m.Descent = math.Inf(-1) }},
		{"ItalicAngle", func(m *Metrics) { m.ItalicAngle = 2 * metricMax }},
		{"width", func(m *Metrics) { m.Glyphs["A"].WidthX = math.Inf(1) }},
		{"bbox", func(m *Metrics) { m.Glyphs["A"].BBox.URy = -2 * metricMax }},
		{"kern", func(m *Metrics) { m.Kern[0].Adjust = math.NaN() }},
		{"tiny", func(m *Metrics) { m.XHeight = metricMin / 2 }},
		{"tinyNegative", func(m *Metrics) { m.Glyphs["A"].WidthX = -1e-300 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			c.breakIt(m)
			if err := m.Write(io.Discard); err == nil {
				t.Error("no error for an unusable value")
			}

			// the same font writes and reads back once the value is usable
			m = base()
			buf := &bytes.Buffer{}
			if err := m.Write(buf); err != nil {
				t.Fatal(err)
			}
			if _, err := Read(buf); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestWriteRejectsUnusableNames checks that Write refuses names and text
// values which [Read] would not give back.
func TestWriteRejectsUnusableNames(t *testing.T) {
	base := func() *Metrics {
		return &Metrics{
			FontName: "Test",
			FullName: "Test Font",
			Glyphs: map[string]*GlyphInfo{
				".notdef": {WidthX: 500},
				"A":       {WidthX: 600, BBox: rect.Rect{URx: 600, URy: 700}},
			},
			Encoding: make([]string, 256),
			Kern:     []KernPair{{Left: "A", Right: ".notdef", Adjust: -20}},
		}
	}

	// the unbroken font writes and reads back
	buf := &bytes.Buffer{}
	if err := base().Write(buf); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(buf); err != nil {
		t.Fatal(err)
	}

	rename := func(m *Metrics, name string) {
		m.Glyphs[name] = m.Glyphs["A"]
		delete(m.Glyphs, "A")
	}
	cases := []struct {
		name    string
		breakIt func(*Metrics)
	}{
		{"glyphSpace", func(m *Metrics) { rename(m, "A B") }},
		{"glyphSemicolon", func(m *Metrics) { rename(m, "A;B") }},
		{"glyphNewline", func(m *Metrics) { rename(m, "A\nB") }},
		{"glyphDelimiter", func(m *Metrics) { rename(m, "A/B") }},
		{"glyphEmpty", func(m *Metrics) { rename(m, "") }},
		{"ligature", func(m *Metrics) { m.Glyphs["A"].Ligatures = map[string]string{"f": "f i"} }},
		{"ligatureKey", func(m *Metrics) { m.Glyphs["A"].Ligatures = map[string]string{"f;": "fi"} }},
		{"kernLeft", func(m *Metrics) { m.Kern[0].Left = "A B" }},
		{"kernRight", func(m *Metrics) { m.Kern[0].Right = "" }},
		{"fullName", func(m *Metrics) { m.FullName = "Test  Font" }},
		{"familyName", func(m *Metrics) { m.FamilyName = "Test\tFamily" }},
		{"weight", func(m *Metrics) { m.Weight = "Regular " }},
		{"version", func(m *Metrics) { m.Version = "1.0\n2.0" }},
		{"notice", func(m *Metrics) { m.Notice = " leading" }},
		{"fullNameTooLong", func(m *Metrics) { m.FullName = strings.Repeat("x", maxTextValueLen+1) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			c.breakIt(m)
			if err := m.Write(io.Discard); err == nil {
				t.Error("no error for an unusable name")
			}
		})
	}
}

// TestWriteIsDeterministic checks that the same metrics always give the same
// file, including the ligatures, which are held in a map.
func TestWriteIsDeterministic(t *testing.T) {
	m := &Metrics{
		FontName: "Test",
		FullName: "Test Font",
		Glyphs: map[string]*GlyphInfo{
			"f": {WidthX: 300, Ligatures: map[string]string{
				"i": "fi", "l": "fl", "f": "ff", "j": "fj", "k": "fk", "b": "fb",
			}},
		},
	}

	buf := &bytes.Buffer{}
	if err := m.Write(buf); err != nil {
		t.Fatal(err)
	}
	want := buf.String()
	for range 20 {
		buf.Reset()
		if err := m.Write(buf); err != nil {
			t.Fatal(err)
		}
		if buf.String() != want {
			t.Fatalf("two writes differ:\n%s\nvs\n%s", want, buf.String())
		}
	}
}

// TestWriteRejectsUnusableEncoding checks that Write refuses an encoding the
// character metrics cannot record.
func TestWriteRejectsUnusableEncoding(t *testing.T) {
	base := func() *Metrics {
		enc := make([]string, 256)
		for i := range enc {
			enc[i] = ".notdef"
		}
		enc[65] = "A"
		return &Metrics{
			FontName: "Test",
			FullName: "Test Font",
			Glyphs: map[string]*GlyphInfo{
				".notdef": {WidthX: 500},
				"A":       {WidthX: 600, BBox: rect.Rect{URx: 600, URy: 700}},
			},
			Encoding: enc,
		}
	}

	// the unbroken font writes and reads back
	buf := &bytes.Buffer{}
	if err := base().Write(buf); err != nil {
		t.Fatal(err)
	}
	m, err := Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if m.Encoding[65] != "A" {
		t.Errorf("got %q at code 65, want A", m.Encoding[65])
	}

	// a glyph named at two codes keeps the lower one, rather than being refused
	twice := base()
	twice.Encoding[66] = "A"
	buf.Reset()
	if err := twice.Write(buf); err != nil {
		t.Fatal(err)
	}
	m, err = Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if m.Encoding[65] != "A" || m.Encoding[66] != ".notdef" {
		t.Errorf("got %q at 65 and %q at 66, want A and .notdef",
			m.Encoding[65], m.Encoding[66])
	}

	cases := []struct {
		name    string
		breakIt func(*Metrics)
	}{
		{"noMetrics", func(m *Metrics) { m.Encoding[66] = "B" }},
		{"tooLong", func(m *Metrics) { m.Encoding = append(m.Encoding, ".notdef") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			c.breakIt(m)
			if err := m.Write(io.Discard); err == nil {
				t.Error("no error for an unusable encoding")
			}
		})
	}
}

// TestWriteNeverExceedsLineLimit checks the bound [Read] relies on: that a
// line Write produces never runs past maxLineLen, even for metrics built at
// the edge of what Write accepts.
func TestWriteNeverExceedsLineLimit(t *testing.T) {
	// the extremes of the range checkMetric allows, which also spell the
	// longest numbers
	extremes := []float64{
		0, metricMin, -metricMin, metricMax, -metricMax,
		math.Nextafter(metricMax, 0), 1.0 / 3, -1.0 / 3,
	}
	long := strings.Repeat("x", maxNameLen(t))

	m := &Metrics{
		FontName:           "Test",
		FullName:           strings.Repeat("a ", 600) + "b",
		Version:            strings.Repeat("v ", 600) + "w",
		Notice:             strings.Repeat("n ", 600) + "o",
		CapHeight:          metricMax,
		XHeight:            -metricMax,
		Ascent:             math.Nextafter(metricMax, 0),
		Descent:            1.0 / 3,
		ItalicAngle:        -1.0 / 3,
		UnderlinePosition:  metricMin,
		UnderlineThickness: math.Nextafter(1, 2),
		Glyphs:             make(map[string]*GlyphInfo),
		Encoding:           make([]string, 256),
	}
	for i := range m.Encoding {
		m.Encoding[i] = ".notdef"
	}

	// a glyph with the longest name and the longest numbers, carrying as many
	// ligatures as the reader's budget would allow it
	g := &GlyphInfo{
		WidthX:    1.0 / 3,
		BBox:      rect.Rect{LLx: -metricMax, LLy: 1.0 / 3, URx: metricMax, URy: metricMin},
		Ligatures: make(map[string]string),
	}
	lineLen := charMetricsFixed + len(long)
	for i := 0; ; i++ {
		succ := fmt.Sprintf("s%06d", i)
		repl := fmt.Sprintf("r%06d", i)
		need := ligatureFixed + len(succ) + len(repl)
		if lineLen+need > maxLineLen {
			break
		}
		lineLen += need
		g.Ligatures[succ] = repl
	}
	if len(g.Ligatures) == 0 {
		t.Fatal("no ligatures fitted")
	}
	m.Glyphs[long] = g
	m.Encoding[255] = long

	// more glyphs, each pairing an extreme value with an extreme name
	for i, x := range extremes {
		name := strings.Repeat("y", 1+i*(maxNameLen(t)-1)/len(extremes))
		m.Glyphs[name] = &GlyphInfo{
			WidthX: x,
			BBox:   rect.Rect{LLx: x, LLy: -x, URx: x, URy: -x},
		}
		m.Kern = append(m.Kern, KernPair{Left: name, Right: long, Adjust: x})
	}

	buf := &bytes.Buffer{}
	if err := m.Write(buf); err != nil {
		t.Fatal(err)
	}
	for i, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		if len(line) > maxLineLen {
			t.Errorf("line %d is %d bytes, over the %d allowed", i+1, len(line), maxLineLen)
		}
	}

	// and the file reads back as what was written
	m2, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(m, m2); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}

// maxNameLen returns the length of the longest glyph name Write accepts.  The
// limit belongs to the type1 package, which does not export it, so the tests
// find it by probing rather than restating it.
func maxNameLen(t *testing.T) int {
	t.Helper()
	n := 1
	for n < maxLineLen && checkName(strings.Repeat("x", n+1)) == nil {
		n++
	}
	if n == 1 || n >= maxLineLen {
		t.Fatalf("no usable name length limit, got %d", n)
	}
	return n
}

// TestWriteRejectsLongLine checks that metrics which no AFM line could hold
// are refused rather than written out for a reader to drop.
func TestWriteRejectsLongLine(t *testing.T) {
	base := func() *Metrics {
		return &Metrics{
			FontName: "Test",
			FullName: "Test Font",
			Glyphs:   map[string]*GlyphInfo{"A": {WidthX: 600}},
		}
	}

	cases := []struct {
		name    string
		breakIt func(*Metrics)
	}{
		{"notice", func(m *Metrics) { m.Notice = strings.Repeat("x", maxLineLen) }},
		{"glyphName", func(m *Metrics) {
			m.Glyphs[strings.Repeat("x", maxLineLen)] = m.Glyphs["A"]
			delete(m.Glyphs, "A")
		}},
		{"ligatures", func(m *Metrics) {
			lig := make(map[string]string)
			for i := range 2000 {
				lig[fmt.Sprintf("s%06d", i)] = fmt.Sprintf("r%06d", i)
			}
			m.Glyphs["A"].Ligatures = lig
		}},
		{"kernNames", func(m *Metrics) {
			long := strings.Repeat("x", maxLineLen)
			m.Glyphs[long] = &GlyphInfo{WidthX: 1}
			m.Kern = []KernPair{{Left: long, Right: "A", Adjust: -10}}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			c.breakIt(m)
			if err := m.Write(io.Discard); err == nil {
				t.Error("no error for a line over the limit")
			}
		})
	}
}
