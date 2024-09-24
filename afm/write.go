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
	"fmt"
	"io"
	"strings"

	"seehuhn.de/go/postscript/funit"
)

// Write writes the metrics to the given writer in AFM format.
func (m *Metrics) Write(w io.Writer) error {
	write := func(format string, a ...interface{}) error {
		_, err := fmt.Fprintf(w, format+"\n", a...)
		return err
	}

	// Write header
	if err := write("StartFontMetrics 4.1"); err != nil {
		return err
	}

	// Write global font information
	if err := write("FontName %s", m.FontName); err != nil {
		return err
	}
	if err := write("FullName %s", m.FullName); err != nil {
		return err
	}
	if err := write("FamilyName %s", strings.Split(m.FullName, " ")[0]); err != nil {
		return err
	}
	if err := write("Weight %s", strings.Join(strings.Split(m.FullName, " ")[1:], " ")); err != nil {
		return err
	}

	bbox := &funit.Rect16{}
	for _, g := range m.Glyphs {
		bbox.Extend(g.BBox)
	}
	if err := write("FontBBox %d %d %d %d", bbox.LLx, bbox.LLy, bbox.URx, bbox.URy); err != nil {
		return err
	}

	if err := write("ItalicAngle %.1f", m.ItalicAngle); err != nil {
		return err
	}
	if err := write("IsFixedPitch %t", m.IsFixedPitch); err != nil {
		return err
	}
	if err := write("UnderlinePosition %.0f", m.UnderlinePosition); err != nil {
		return err
	}
	if err := write("UnderlineThickness %.0f", m.UnderlineThickness); err != nil {
		return err
	}
	if err := write("CapHeight %.0f", m.CapHeight); err != nil {
		return err
	}
	if err := write("XHeight %.0f", m.XHeight); err != nil {
		return err
	}
	if err := write("Ascender %.0f", m.Ascent); err != nil {
		return err
	}
	if err := write("Descender %.0f", m.Descent); err != nil {
		return err
	}

	// Write character metrics
	if err := write("StartCharMetrics %d", len(m.Glyphs)); err != nil {
		return err
	}
	glyphList := m.GlyphList()
	for _, name := range glyphList {
		g := m.Glyphs[name]
		if g == nil {
			continue
		}
		charCode := -1
		for i, n := range m.Encoding {
			if n == name {
				charCode = i
				break
			}
		}
		line := fmt.Sprintf("C %d ; WX %.0f ; N %s ; B %d %d %d %d ;",
			charCode, g.WidthX, name, g.BBox.LLx, g.BBox.LLy, g.BBox.URx, g.BBox.URy)
		for succ, lig := range g.Ligatures {
			line += fmt.Sprintf(" L %s %s ;", succ, lig)
		}
		if err := write(line); err != nil {
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
			if err := write("KPX %s %s %d", k.Left, k.Right, k.Adjust); err != nil {
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
