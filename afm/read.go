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
	"io"
	"strconv"
	"strings"

	"seehuhn.de/go/postscript/funit"
)

// Read reads an AFM file.
func Read(fd io.Reader) (*Metrics, error) {
	res := &Metrics{
		Glyphs: make(map[string]*GlyphInfo),
	}

	res.Encoding = make([]string, 256)
	for i := range res.Encoding {
		res.Encoding[i] = ".notdef"
	}

	noLigs := make(map[string]string) // shared map for all glyphs without ligatures

	charMetrics := false
	kernPairs := false
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "EndCharMetrics") {
			charMetrics = false
			continue
		}
		if charMetrics {
			var name string
			var width funit.Int16
			var code int
			var BBox funit.Rect16

			ligTmp := make(map[string]string)

			keyVals := strings.Split(line, ";")
			for _, keyVal := range keyVals {
				ff := strings.Fields(keyVal)
				if len(ff) < 2 {
					continue
				}
				switch ff[0] {
				case "C":
					code, _ = strconv.Atoi(ff[1])
				case "WX":
					tmp, _ := strconv.Atoi(ff[1])
					width = funit.Int16(tmp)
				case "N":
					name = ff[1]
				case "B":
					conv := func(in string) funit.Int16 {
						x, _ := strconv.Atoi(in)
						return funit.Int16(x)
					}
					BBox.LLx = conv(ff[1])
					BBox.LLy = conv(ff[2])
					BBox.URx = conv(ff[3])
					BBox.URy = conv(ff[4])
				case "L":
					ligTmp[ff[1]] = ff[2]
				default:
					panic("not implemented")
				}
			}

			if code >= 0 && code < 256 {
				res.Encoding[code] = name
			}

			if len(ligTmp) == 0 {
				ligTmp = noLigs
			}

			res.Glyphs[name] = &GlyphInfo{
				WidthX:    float64(width),
				BBox:      BBox,
				Ligatures: ligTmp,
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == "EndKernPairs" {
			kernPairs = false
			continue
		}
		if kernPairs {
			x, _ := strconv.Atoi(fields[3])
			res.Kern = append(res.Kern, &KernPair{
				Left:   fields[1],
				Right:  fields[2],
				Adjust: funit.Int16(x),
			})
			continue
		}

		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "FontName":
			res.FontName = fields[1]
		case "FullName":
			res.FullName = strings.Join(fields[1:], " ")
		case "CapHeight":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.CapHeight = x
		case "XHeight":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.XHeight = x
		case "Ascender":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.Ascent = x
		case "Descender":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.Descent = x
		case "UnderlinePosition":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.UnderlinePosition = x
		case "UnderlineThickness":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.UnderlineThickness = x
		case "ItalicAngle":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.ItalicAngle = x
		case "IsFixedPitch":
			res.IsFixedPitch = fields[1] == "true"
		case "StartCharMetrics":
			charMetrics = true
		case "StartKernPairs":
			kernPairs = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// TODO(voss): remove?
	if _, ok := res.Glyphs[".notdef"]; !ok {
		var width float64
		if gi, ok := res.Glyphs["space"]; ok {
			width = gi.WidthX
		}
		res.Glyphs[".notdef"] = &GlyphInfo{
			WidthX: width,
		}
	}

	return res, nil
}
