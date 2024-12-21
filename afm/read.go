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
	"strconv"
	"strings"

	"seehuhn.de/go/geom/rect"
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
			code := -1
			var BBox rect.Rect

			ligTmp := make(map[string]string)

			keyVals := strings.Split(line, ";")
			for _, keyVal := range keyVals {
				ff := strings.Fields(keyVal)
				if len(ff) < 2 {
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
					if len(ff) != 5 {
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
					if len(ff) >= 3 {
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
			if len(ligTmp) == 0 {
				ligTmp = nil
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
		if kernPairs && len(fields) == 4 && fields[0] == "KPX" {
			x, err := strconv.Atoi(fields[3])
			if err != nil {
				return nil, fmt.Errorf("invalid kerning pair adjustment: %v", err)
			}
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
		case "Version":
			res.Version = strings.Join(fields[1:], " ")
		case "Notice":
			res.Notice = strings.Join(fields[1:], " ")
		case "CapHeight":
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid CapHeight: %v", err)
			}
			res.CapHeight = x
		case "XHeight":
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid XHeight: %v", err)
			}
			res.XHeight = x
		case "Ascender":
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Ascender: %v", err)
			}
			res.Ascent = x
		case "Descender":
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Descender: %v", err)
			}
			res.Descent = x
		case "UnderlinePosition":
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid UnderlinePosition: %v", err)
			}
			res.UnderlinePosition = x
		case "UnderlineThickness":
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid UnderlineThickness: %v", err)
			}
			res.UnderlineThickness = x
		case "ItalicAngle":
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid ItalicAngle: %v", err)
			}
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

	return res, nil
}
