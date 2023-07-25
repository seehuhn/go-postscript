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
	"seehuhn.de/go/postscript/type1"
)

func Read(fd io.Reader) (*type1.Font, error) {
	res := &type1.Font{
		Info:      &type1.FontInfo{},
		GlyphInfo: make(map[string]*type1.GlyphInfo),
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

			res.GlyphInfo[name] = &type1.GlyphInfo{
				WidthX:    width,
				Extent:    BBox,
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
			res.Kern = append(res.Kern, &type1.KernPair{
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
			res.Info.FontName = fields[1]
		case "FullName":
			res.Info.FullName = strings.Join(fields[1:], " ")
		case "CapHeight":
			x, _ := strconv.Atoi(fields[1])
			res.CapHeight = funit.Int16(x)
		case "XHeight":
			x, _ := strconv.Atoi(fields[1])
			res.XHeight = funit.Int16(x)
		case "Ascender":
			x, _ := strconv.Atoi(fields[1])
			res.Ascent = funit.Int16(x)
		case "Descender":
			x, _ := strconv.Atoi(fields[1])
			res.Descent = funit.Int16(x)
		case "UnderlinePosition":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.Info.UnderlinePosition = funit.Float64(x)
		case "UnderlineThickness":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.Info.UnderlineThickness = funit.Float64(x)
		case "ItalicAngle":
			x, _ := strconv.ParseFloat(fields[1], 64)
			res.Info.ItalicAngle = x
		case "IsFixedPitch":
			res.Info.IsFixedPitch = fields[1] == "true"
		case "StartCharMetrics":
			charMetrics = true
		case "StartKernPairs":
			kernPairs = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if _, ok := res.GlyphInfo[".notdef"]; !ok {
		var width funit.Int16
		if gi, ok := res.GlyphInfo["space"]; ok {
			width = gi.WidthX
		}
		res.GlyphInfo[".notdef"] = &type1.GlyphInfo{
			WidthX: width,
		}
	}

	res.UnitsPerEm = 1000 // TODO(voss): is there a better way?
	return res, nil
}
