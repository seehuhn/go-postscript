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
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"seehuhn.de/go/postscript/type1"
)

// checkMetric reports an error if x cannot be written to an AFM file in a form
// which reads back as x.  Values which are not finite and values beyond
// metricMax are refused by [Read], and a magnitude below metricMin is read
// back as zero, so all of them would be lost.
func checkMetric(field string, x float64) error {
	// the comparisons also reject NaN and the infinities
	if x == 0 || (math.Abs(x) >= metricMin && math.Abs(x) <= metricMax) {
		return nil
	}
	return fmt.Errorf("%s value %g out of range", field, x)
}

// checkMetrics reports an error if any number in m would be lost when the
// written file is read again.
func (m *Metrics) checkMetrics() error {
	for _, f := range []struct {
		name string
		x    float64
	}{
		{"ItalicAngle", m.ItalicAngle},
		{"UnderlinePosition", m.UnderlinePosition},
		{"UnderlineThickness", m.UnderlineThickness},
		{"CapHeight", m.CapHeight},
		{"XHeight", m.XHeight},
		{"Ascender", m.Ascent},
		{"Descender", m.Descent},
	} {
		if err := checkMetric(f.name, f.x); err != nil {
			return err
		}
	}
	for name, g := range m.Glyphs {
		if err := checkMetric("glyph "+shortName(name)+" width", g.WidthX); err != nil {
			return err
		}
		b := g.BBox
		for _, x := range []float64{b.LLx, b.LLy, b.URx, b.URy} {
			if err := checkMetric("glyph "+shortName(name)+" bounding box", x); err != nil {
				return err
			}
		}
	}
	for _, k := range m.Kern {
		if err := checkMetric("kern pair "+shortName(k.Left)+" "+shortName(k.Right), k.Adjust); err != nil {
			return err
		}
	}
	return nil
}

// shortName abbreviates a name for an error message, so that a long name does
// not run away with the line.
func shortName(s string) string {
	const max = 24
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// checkName reports an error if s cannot be written as a glyph name in an AFM
// file.  Beyond the PostScript rules a semicolon is excluded, since the file
// format uses semicolons to separate the entries of a character metrics line.
func checkName(s string) error {
	if s == "" {
		return errors.New("empty glyph name")
	}
	if err := type1.CheckGlyphName(s); err != nil {
		return fmt.Errorf("glyph name %q: %v", shortName(s), err)
	}
	if strings.ContainsRune(s, ';') {
		return fmt.Errorf("glyph name %q holds a semicolon", shortName(s))
	}
	return nil
}

// checkNames reports an error if any name in m would be lost or would corrupt
// the file.
func (m *Metrics) checkNames() error {
	for name, g := range m.Glyphs {
		if err := checkName(name); err != nil {
			return err
		}
		for succ, repl := range g.Ligatures {
			if err := checkName(succ); err != nil {
				return err
			}
			if err := checkName(repl); err != nil {
				return err
			}
		}
	}
	for _, k := range m.Kern {
		if err := checkName(k.Left); err != nil {
			return err
		}
		if err := checkName(k.Right); err != nil {
			return err
		}
	}
	return nil
}

// checkEncoding reports an error if the encoding of m cannot be written.  An
// AFM file addresses 256 codes, and names the glyph of a code on the character
// metrics line of that glyph, so a code beyond 256 and a code whose glyph has
// no metrics both have nowhere to go.
//
// A glyph named at several codes is not an error, even though only the lowest
// of its codes is written: fonts encode a glyph twice often enough that the
// limit belongs to the file format rather than to the caller.
//
// The name ".notdef" is exempt: [Read] uses it for the codes a file leaves
// unencoded, so reading gives it back whether it was written or not.
func (m *Metrics) checkEncoding() error {
	if len(m.Encoding) > 256 {
		return fmt.Errorf("encoding has %d entries, more than 256", len(m.Encoding))
	}
	for code, name := range m.Encoding {
		if name == "" || name == ".notdef" {
			continue
		}
		if _, ok := m.Glyphs[name]; !ok {
			return fmt.Errorf("code %d names glyph %s, which has no metrics", code, name)
		}
	}
	return nil
}

// checkStrings reports an error if a text value in m would read back changed.
// Such a value runs to the end of the line, and [Read] collapses runs of white
// space, so only a value already in that form survives.  A value too long for
// the line it is written to is refused here rather than below, so that the
// error names the field the caller set.
func (m *Metrics) checkStrings() error {
	for _, f := range []struct {
		key, s string
	}{
		{"FullName", m.FullName},
		{"FamilyName", m.FamilyName},
		{"Weight", m.Weight},
		{"Version", m.Version},
		{"Notice", m.Notice},
	} {
		if f.s != strings.Join(strings.Fields(f.s), " ") {
			return fmt.Errorf("%s holds unusable white space", f.key)
		}
		if len(f.s) > maxTextValueLen {
			return fmt.Errorf("%s too long (%d bytes)", f.key, len(f.s))
		}
	}
	return nil
}

// Write writes the metrics to the given writer in AFM format.
//
// The metrics are refused if a value could not be read back unchanged: a
// number beyond the range [Read] accepts, a glyph name the file format cannot
// carry, a text value holding white space which [Read] would collapse, an
// encoding entry the character metrics cannot record, or a line longer than
// [Read] accepts.  A glyph named at several codes keeps only the lowest, which
// is all an AFM file can hold.
func (m *Metrics) Write(w io.Writer) error {
	if err := type1.CheckFontName(m.FontName); err != nil {
		return err
	}
	if err := m.checkMetrics(); err != nil {
		return err
	}
	if err := m.checkNames(); err != nil {
		return err
	}
	if err := m.checkStrings(); err != nil {
		return err
	}
	if err := m.checkEncoding(); err != nil {
		return err
	}

	// Every line is assembled before it is written, so that one longer than
	// maxLineLen is refused rather than handed to a reader which would drop
	// it.  [Read] keeps nothing which needs a longer line, so this can only
	// fire for metrics which were not read from a file.
	var buf, out []byte
	writeLine := func(line []byte) error {
		if len(line) > maxLineLen {
			return fmt.Errorf("line too long (%d bytes): %s...", len(line), line[:24])
		}
		out = append(append(out[:0], line...), '\n')
		_, err := w.Write(out)
		return err
	}
	write := func(format string, a ...any) error {
		buf = fmt.Appendf(buf[:0], format, a...)
		return writeLine(buf)
	}

	// Write header
	if err := write("StartFontMetrics 4.1"); err != nil {
		return err
	}

	// a text value is left out when empty, since the keyword alone would carry
	// no value
	writeText := func(key, s string) error {
		if s == "" {
			return nil
		}
		return write("%s %s", key, s)
	}

	// Write global font information
	if err := write("FontName %s", m.FontName); err != nil {
		return err
	}
	if err := writeText("FullName", m.FullName); err != nil {
		return err
	}
	if err := writeText("FamilyName", m.FamilyName); err != nil {
		return err
	}
	if err := writeText("Weight", m.Weight); err != nil {
		return err
	}

	// The font bounding box needs no check of its own: rect.Extend only ever
	// carries a coordinate over from one of the glyph boxes checked above, or
	// leaves the box zero.
	bbox := m.FontBBoxPDF()
	llx := strconv.FormatFloat(bbox.LLx, 'f', -1, 64)
	lly := strconv.FormatFloat(bbox.LLy, 'f', -1, 64)
	urx := strconv.FormatFloat(bbox.URx, 'f', -1, 64)
	ury := strconv.FormatFloat(bbox.URy, 'f', -1, 64)
	if err := write("FontBBox %s %s %s %s", llx, lly, urx, ury); err != nil {
		return err
	}

	if err := write("ItalicAngle %s", strconv.FormatFloat(m.ItalicAngle, 'f', -1, 64)); err != nil {
		return err
	}
	if err := write("IsFixedPitch %t", m.IsFixedPitch); err != nil {
		return err
	}
	if err := write("UnderlinePosition %s", strconv.FormatFloat(m.UnderlinePosition, 'f', -1, 64)); err != nil {
		return err
	}
	if err := write("UnderlineThickness %s", strconv.FormatFloat(m.UnderlineThickness, 'f', -1, 64)); err != nil {
		return err
	}
	if err := writeText("Version", m.Version); err != nil {
		return err
	}
	if err := writeText("Notice", m.Notice); err != nil {
		return err
	}

	if err := write("CapHeight %s", strconv.FormatFloat(m.CapHeight, 'f', -1, 64)); err != nil {
		return err
	}
	if err := write("XHeight %s", strconv.FormatFloat(m.XHeight, 'f', -1, 64)); err != nil {
		return err
	}
	if err := write("Ascender %s", strconv.FormatFloat(m.Ascent, 'f', -1, 64)); err != nil {
		return err
	}
	if err := write("Descender %s", strconv.FormatFloat(m.Descent, 'f', -1, 64)); err != nil {
		return err
	}

	// Write character metrics
	if err := write("StartCharMetrics %d", len(m.Glyphs)); err != nil {
		return err
	}
	code := make(map[string]int, len(m.Encoding))
	for i, name := range m.Encoding {
		if _, ok := code[name]; !ok {
			code[name] = i
		}
	}
	glyphList := m.GlyphList()
	var line []byte
	for _, name := range glyphList {
		g := m.Glyphs[name]
		charCode, ok := code[name]
		if !ok {
			charCode = -1
		}
		llx := strconv.FormatFloat(g.BBox.LLx, 'f', -1, 64)
		lly := strconv.FormatFloat(g.BBox.LLy, 'f', -1, 64)
		urx := strconv.FormatFloat(g.BBox.URx, 'f', -1, 64)
		ury := strconv.FormatFloat(g.BBox.URy, 'f', -1, 64)
		wx := strconv.FormatFloat(g.WidthX, 'f', -1, 64)
		line = fmt.Appendf(line[:0], "C %d ; WX %s ; N %s ; B %s %s %s %s ;",
			charCode, wx, name, llx, lly, urx, ury)
		// the successors are sorted, so that the same metrics always give the
		// same file
		for _, succ := range slices.Sorted(maps.Keys(g.Ligatures)) {
			line = fmt.Appendf(line, " L %s %s ;", succ, g.Ligatures[succ])
		}
		if err := writeLine(line); err != nil {
			return err
		}
	}
	if err := write("EndCharMetrics"); err != nil {
		return err
	}

	// Write kerning data
	if len(m.Kern) > 0 {
		if err := write("StartKernData"); err != nil {
			return err
		}
		if err := write("StartKernPairs %d", len(m.Kern)); err != nil {
			return err
		}
		for _, k := range m.Kern {
			adjust := strconv.FormatFloat(k.Adjust, 'f', -1, 64)
			if err := write("KPX %s %s %s", k.Left, k.Right, adjust); err != nil {
				return err
			}
		}
		if err := write("EndKernPairs"); err != nil {
			return err
		}
		if err := write("EndKernData"); err != nil {
			return err
		}
	}

	// Write footer
	return write("EndFontMetrics")
}
