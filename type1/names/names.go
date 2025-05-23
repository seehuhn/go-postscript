// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2021  Jochen Voss <voss@seehuhn.de>
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

package names

import (
	"bufio"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ToUnicode maps a Type1 Glyph name to a sequence of unicode characters.
// This implements the algorithm documented at
// https://github.com/adobe-type-tools/agl-specification .
//
// For the ".notdef" glyph, an empty string is returned.
func ToUnicode(name string, fontName string) string {
	var res []rune

	idx := strings.IndexByte(name, '.')
	if idx >= 0 {
		name = name[:idx]
	}

	parts := strings.Split(name, "_")
	for _, part := range parts {
		if fontName == "ZapfDingbats" {
			c, ok := glyph.lookup("zapfdingbats", part)
			if ok {
				res = append(res, c)
				continue
			}
		}

		c, ok := glyph.lookup("glyphlist", part)
		if ok {
			res = append(res, c)
			continue
		}

		if strings.HasPrefix(part, "uni") && len(part)%4 == 3 {
			good := true
			var candidates []rune
			var val rune
		hexLoop:
			for i, c := range part[3:] {
				switch {
				case c >= '0' && c <= '9':
					val = val*16 + c - '0'
				case c >= 'A' && c <= 'F':
					val = val*16 + c - 'A' + 10
				default:
					good = false
					break hexLoop
				}
				if i%4 == 3 {
					if val >= 0xD800 && val < 0xE000 {
						good = false
						break hexLoop
					}
					candidates = append(candidates, val)
					val = 0
				}
			}
			if good {
				res = append(res, candidates...)
				continue
			}
		}

		if len(part) >= 5 && len(part) <= 7 && part[0] == 'u' {
			good := true
			var val rune
		hexLoop2:
			for _, c := range part[1:] {
				switch {
				case c >= '0' && c <= '9':
					val = val*16 + c - '0'
				case c >= 'A' && c <= 'F':
					val = val*16 + c - 'A' + 10
				default:
					good = false
					break hexLoop2
				}
			}
			if good && (val < 0xD800 || val >= 0xE000 && val < 0x110000) {
				res = append(res, val)
				continue
			}
		}
	}

	return string(res)
}

// FromUnicode find the PostScript glyph name for a unicode character.
//
// If text is empty, the empty string is returned.
//
// TODO(voss): add special handling for ZapfDingbats?
func FromUnicode(text string) string {
	var parts []string
	for _, r := range text {
		parts = append(parts, glyph.encode(r))
	}
	return strings.Join(parts, "_")
}

type glyphMap struct {
	sync.Mutex
	nameToRune map[string]map[string]rune
	runeToName map[rune]string
}

func (gm *glyphMap) encode(r rune) string {
	rMap := gm.getEncode()

	rr := expand(r)
	var parts []string
	for _, r := range rr {
		name, ok := rMap[r]
		if !ok {
			name = fmt.Sprintf("u%04X", r)
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, "_")
}

func (gm *glyphMap) getEncode() map[rune]string {
	gm.Lock()
	defer gm.Unlock()

	if gm.runeToName != nil {
		return gm.runeToName
	}

	r2n := make(map[rune]string)
	fd, _ := glyphData.Open("agl-aglfn/aglfn.txt")

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		ww := strings.SplitN(line, ";", 3)
		code, _ := strconv.ParseInt(ww[0], 16, 32)
		name := ww[1]
		r2n[rune(code)] = name
	}
	if err := scanner.Err(); err != nil {
		panic("corrupted glyph map aglfn.txt")
	}

	gm.runeToName = r2n
	return r2n
}

func (gm *glyphMap) lookup(file, name string) (rune, bool) {
	gm.Lock()
	defer gm.Unlock()

	fMap := gm.getFile(file)
	c, ok := fMap[name]
	return c, ok
}

func (gm *glyphMap) getFile(file string) map[string]rune {
	fMap := gm.nameToRune[file]
	if fMap != nil {
		return fMap
	}
	fMap = make(map[string]rune)

	fd, err := glyphData.Open("agl-aglfn/" + file + ".txt")
	if err != nil {
		panic("invalid glyph map " + file)
	}

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		ww := strings.SplitN(line, ";", 2)
		name := ww[0]
		code, _ := strconv.ParseInt(ww[1], 16, 32)

		// fix up some swapped character codes
		switch {
		case name == "Tcommaaccent" && code == 0x0162:
			code = 0x021A
		case name == "tcommaaccent" && code == 0x0163:
			code = 0x021B
		}

		fMap[name] = rune(code)
	}
	if err := scanner.Err(); err != nil {
		panic("corrupted glyph map " + file)
	}

	gm.nameToRune[file] = fMap
	return fMap
}

var glyph = &glyphMap{
	nameToRune: make(map[string]map[string]rune),
}

//go:embed agl-aglfn/*.txt
var glyphData embed.FS
