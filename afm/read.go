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

package afm

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/postscript/funit"
)

// maxPrealloc bounds the number of entries allocated in advance from a count
// given in the file, so that a wrong count cannot force a large allocation.
const maxPrealloc = 2048

// maxFields is the number of fields retained from a line.  The widest entry
// read below is the character bounding box, "B llx lly urx ury"; a case which
// needs more fields must raise this.
const maxFields = 5

// splitFields splits s into whitespace-separated fields.  It returns the
// first maxFields fields, together with the total number of fields in s.
// Returning an array avoids an allocation for every line of the file, and
// makes an index beyond maxFields a compile-time error.
func splitFields(s string) (fields [maxFields]string, n int) {
	for f := range strings.FieldsSeq(s) {
		if n < maxFields {
			fields[n] = f
		}
		n++
	}
	return fields, n
}

// lineValue returns the part of a key/value line which follows the key, with
// runs of white space collapsed into single spaces.  The line must contain at
// least one field.
func lineValue(line string) string {
	fields := strings.Fields(line)
	return strings.Join(fields[1:], " ")
}

// Read reads an AFM file.
func Read(fd io.Reader) (*Metrics, error) {
	res := &Metrics{
		Glyphs: make(map[string]*GlyphInfo),
	}

	res.Encoding = make([]string, 256)
	for i := range res.Encoding {
		res.Encoding[i] = ".notdef"
	}

	charMetrics := false
	kernPairs := false
	glyphsSized := false
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		// Text returns a freshly allocated string for each line, so the glyph
		// and kern pair names below can be retained as substrings of it
		// without copying.
		line := scanner.Text()
		if strings.HasPrefix(line, "EndCharMetrics") {
			charMetrics = false
			continue
		}
		if charMetrics {
			var name string
			var width funit.Int16
			code := -1
			var BBox rect.Rect

			var ligTmp map[string]string

			keyVals := strings.SplitSeq(line, ";")
			for keyVal := range keyVals {
				ff, numFields := splitFields(keyVal)
				if numFields < 2 {
					continue
				}
				switch ff[0] {
				case "C":
					var err error
					code, err = strconv.Atoi(ff[1])
					if err != nil {
						return nil, fmt.Errorf("invalid character code %q: %v", ff[1], err)
					}
				case "WX":
					tmp, err := strconv.Atoi(ff[1])
					if err != nil {
						return nil, fmt.Errorf("invalid character width %q: %v", ff[1], err)
					}
					width = funit.Int16(tmp)
				case "N":
					name = ff[1]
				case "B":
					if numFields != 5 {
						continue
					}
					conv := func(in string) (float64, error) {
						return strconv.ParseFloat(in, 64)
					}
					var err error
					if BBox.LLx, err = conv(ff[1]); err != nil {
						return nil, fmt.Errorf("invalid bounding box LLx: %v", err)
					}
					if BBox.LLy, err = conv(ff[2]); err != nil {
						return nil, fmt.Errorf("invalid bounding box LLy: %v", err)
					}
					if BBox.URx, err = conv(ff[3]); err != nil {
						return nil, fmt.Errorf("invalid bounding box URx: %v", err)
					}
					if BBox.URy, err = conv(ff[4]); err != nil {
						return nil, fmt.Errorf("invalid bounding box URy: %v", err)
					}
				case "L":
					if numFields >= 3 {
						if ligTmp == nil {
							ligTmp = make(map[string]string)
						}
						ligTmp[ff[1]] = ff[2]
					}
				}
			}
			_, seen := res.Glyphs[name]
			if name == "" || seen {
				continue
			}
			if code >= 0 && code < 256 {
				res.Encoding[code] = name
			}
			res.Glyphs[name] = &GlyphInfo{
				WidthX:    float64(width),
				BBox:      BBox,
				Ligatures: ligTmp,
			}
			continue
		}
		fields, numFields := splitFields(line)
		if numFields == 0 {
			continue
		}
		if fields[0] == "EndKernPairs" {
			kernPairs = false
			continue
		}
		if kernPairs && numFields == 4 && fields[0] == "KPX" {
			x, err := strconv.Atoi(fields[3])
			if err != nil {
				return nil, fmt.Errorf("invalid kerning pair adjustment: %v", err)
			}
			res.Kern = append(res.Kern, KernPair{
				Left:   fields[1],
				Right:  fields[2],
				Adjust: funit.Int16(x),
			})
			continue
		}
		if numFields < 2 {
			continue
		}
		switch fields[0] {
		case "FontName":
			res.FontName = fields[1]
		case "FullName":
			res.FullName = lineValue(line)
		case "Version":
			res.Version = lineValue(line)
		case "Notice":
			res.Notice = lineValue(line)
		case "CapHeight":
			x, _ := strconv.ParseFloat(fields[1], 64)
			if x >= math.MinInt32 && x <= math.MaxInt32 {
				// Note that the above test also excludes NaN values and infinities.
				res.CapHeight = x
			}
		case "XHeight":
			x, _ := strconv.ParseFloat(fields[1], 64)
			if x >= math.MinInt32 && x <= math.MaxInt32 {
				// Note that the above test also excludes NaN values and infinities.
				res.XHeight = x
			}
		case "Ascender":
			x, _ := strconv.ParseFloat(fields[1], 64)
			if x >= math.MinInt32 && x <= math.MaxInt32 {
				// Note that the above test also excludes NaN values and infinities.
				res.Ascent = x
			}
		case "Descender":
			x, _ := strconv.ParseFloat(fields[1], 64)
			if x >= math.MinInt32 && x <= math.MaxInt32 {
				// Note that the above test also excludes NaN values and infinities.
				res.Descent = x
			}
		case "UnderlinePosition":
			x, _ := strconv.ParseFloat(fields[1], 64)
			if x >= math.MinInt32 && x <= math.MaxInt32 {
				// Note that the above test also excludes NaN values and infinities.
				res.UnderlinePosition = x
			}
		case "UnderlineThickness":
			x, _ := strconv.ParseFloat(fields[1], 64)
			if x >= math.MinInt32 && x <= math.MaxInt32 {
				// Note that the above test also excludes NaN values and infinities.
				res.UnderlineThickness = x
			}
		case "ItalicAngle":
			x, _ := strconv.ParseFloat(fields[1], 64)
			if x >= math.MinInt32 && x <= math.MaxInt32 {
				// Note that the above test also excludes NaN values and infinities.
				res.ItalicAngle = x
			}
		case "IsFixedPitch":
			res.IsFixedPitch = fields[1] == "true"
		case "StartCharMetrics":
			charMetrics = true
			// The header states how many entries follow.  Use this to size the
			// map, but limit how much memory a bogus count can claim.  Only
			// the first section counts, so that a later one neither discards
			// the entries already read nor makes a file of repeated headers
			// allocate a map per line.
			if n, err := strconv.Atoi(fields[1]); err == nil && !glyphsSized {
				glyphsSized = true
				res.Glyphs = make(map[string]*GlyphInfo, min(max(n, 0), maxPrealloc))
			}
		case "StartKernPairs":
			kernPairs = true
			// as above, for the kerning pairs
			if n, err := strconv.Atoi(fields[1]); err == nil && res.Kern == nil {
				res.Kern = make([]KernPair, 0, min(max(n, 0), maxPrealloc))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// a font without kerning pairs has no kern slice, even if the file
	// announced a kern section
	if len(res.Kern) == 0 {
		res.Kern = nil
	}

	return res, nil
}
