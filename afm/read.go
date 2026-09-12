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
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"

	"seehuhn.de/go/geom/rect"
	"seehuhn.de/go/postscript/type1"
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

// metricMax is the greatest magnitude a metric may have.  The file format sets
// no range, but generators can write nonsense in place of a value they failed
// to measure: FontForge prints -2147483648 as the descender of a font which
// has none of the glyphs it measures.  Genuine metrics reach a few thousand
// units, so this limit lies far above real data and far below such artefacts.
const metricMax = 1 << 24

// metricMin is the smallest magnitude a metric may have.  AFM lengths are in
// units of 1/1000 of the em square, so this stands for 10^-7 em: far below any
// measurement, and small enough that reading such a value as zero loses
// nothing.  The bound is what keeps the written form of a number short, since
// the shortest decimal spelling of a tiny value is long — 10^-300 needs three
// hundred digits — and the length of a line is bounded in terms of it.
const metricMin = 1e-4

// maxNumberLen is the greatest number of bytes [Metrics.Write] can spend on a
// single number.  Every value it writes is zero or lies between metricMin and
// metricMax in magnitude, and TestMaxNumberLen checks that no such value is
// spelled longer than this.
const maxNumberLen = 32

// maxCodeLen is the greatest number of bytes [Metrics.Write] can spend on a
// character code.  An encoding holds at most 256 entries, and a glyph the
// encoding does not name is written as -1.
const maxCodeLen = 3

// maxLineLen bounds the length of a line, in bytes.  [Read] discards a longer
// line, which bounds the memory a single line can claim however long the line
// in the file is, and [Metrics.Write] refuses to produce one.  The two limits
// are the same, so that neither function can hand the other a line it will not
// take.  The longest line found in real AFM files is about 3.5 kB.
const maxLineLen = 16384

// maxTextValueLen bounds the length of the text values in the header.
// [Metrics.Write] spells such a value out after its keyword, and "FamilyName"
// is the longest of those keywords.
const maxTextValueLen = maxLineLen - len("FamilyName ")

// The character metrics line [Metrics.Write] produces has the form
//
//	"C <code> ; WX <wx> ; N <name> ; B <llx> <lly> <urx> <ury> ;"
//
// followed by " L <succ> <repl> ;" for each ligature, and a kerning pair is
// written as "KPX <left> <right> <adjust>".  The constants below over-estimate
// the length of those lines: the pieces spell out the literal parts of the two
// formats, and every number is charged the longest form maxNumberLen allows.
//
// The estimate for a character metrics line is a fixed part plus a sum over
// the ligatures, so [Read] can decide as it goes whether the line it is
// building can be written back, without regard to the order the entries arrive
// in or the order [Metrics.Write] puts them in.
const (
	charMetricsFixed = len("C ") + maxCodeLen +
		len(" ; WX ") + maxNumberLen +
		len(" ; N ") + // the glyph name is charged by the caller
		len(" ; B ") + 4*maxNumberLen + 3*len(" ") +
		len(" ;")
	ligatureFixed = len(" L ") + len(" ") + len(" ;")
	kernPairFixed = len("KPX ") + len(" ") + len(" ") + maxNumberLen
)

// ligature is one entry of a character metrics line, held in the order the
// file gives before the line is known to fit.
type ligature struct{ succ, repl string }

// readNumber parses a number from an AFM file.  The second return value
// reports whether the text is a number which can serve as a metric; text which
// is not a number at all, and values which are not finite or lie beyond
// metricMax, are all rejected.  A magnitude below metricMin is read as zero
// rather than rejected.
//
// A rejected value is never an error: the caller substitutes a default, so that
// one unusable field does not cost the reader the rest of the file.
func readNumber(s string) (float64, bool) {
	x, err := strconv.ParseFloat(s, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, false
	}
	// the comparison also rejects NaN and the infinities
	if !(x >= -metricMax && x <= metricMax) {
		return 0, false
	}
	if x > -metricMin && x < metricMin {
		return 0, true
	}
	return x, true
}

// repairName makes a name read from an AFM file usable.  Beyond the PostScript
// rules, a semicolon is removed: a character metrics line uses semicolons to
// separate its entries, so a name holding one could not be written back.  The
// same repair is applied everywhere a name is read, so that a kerning pair
// still matches the glyph it kerns.
//
// The result is the empty string if nothing is left, and the caller then drops
// the entry.
func repairName(s string) string {
	return type1.RepairGlyphName(strings.ReplaceAll(s, ";", ""))
}

// lineReader reads an AFM file one line at a time, keeping no more than
// maxLineLen bytes of any one line.
type lineReader struct {
	r   *bufio.Reader
	buf []byte
}

// next returns the following line, without its line terminator.  A line longer
// than maxLineLen is consumed but not kept, and ok is false; the caller skips
// it.  io.EOF is returned once the file is exhausted.
func (lr *lineReader) next() (string, bool, error) {
	chunk, err := lr.r.ReadSlice('\n')
	if err == nil || (err == io.EOF && len(chunk) > 0) {
		// the common case: the whole line sat in the reader's buffer
		line := trimEOL(chunk)
		if len(line) > maxLineLen {
			return "", false, nil
		}
		return string(line), true, nil
	}
	if err == io.EOF {
		return "", false, io.EOF
	}
	if err != bufio.ErrBufferFull {
		return "", false, err
	}

	// The line spans several buffers.  Once it is known to be too long the
	// bytes are dropped instead of kept, so that the memory used does not
	// follow the length of the line in the file.  The two bytes of slack
	// cover a "\r\n" terminator, which does not count towards the limit.
	const maxRaw = maxLineLen + 2
	tooLong := len(chunk) > maxRaw
	lr.buf = lr.buf[:0]
	if !tooLong {
		lr.buf = append(lr.buf, chunk...)
	}
	for {
		chunk, err = lr.r.ReadSlice('\n')
		if !tooLong {
			if len(lr.buf)+len(chunk) > maxRaw {
				tooLong = true
				lr.buf = lr.buf[:0]
			} else {
				lr.buf = append(lr.buf, chunk...)
			}
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil && err != io.EOF {
			return "", false, err
		}
		line := trimEOL(lr.buf)
		if tooLong || len(line) > maxLineLen {
			return "", false, nil
		}
		return string(line), true, nil
	}
}

// trimEOL removes the line terminator, which is either "\n" or "\r\n".
func trimEOL(b []byte) []byte {
	b = bytes.TrimSuffix(b, []byte("\n"))
	return bytes.TrimSuffix(b, []byte("\r"))
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
	lr := &lineReader{r: bufio.NewReader(fd)}
	for {
		// next returns a freshly allocated string for each line, so the glyph
		// and kern pair names below can be retained as substrings of it
		// without copying.
		line, ok, err := lr.next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if !ok {
			// a line too long to keep leaves out the glyph or the value it
			// carried, which is the same repair as a line the reader cannot
			// make sense of
			continue
		}
		if strings.HasPrefix(line, "EndCharMetrics") {
			charMetrics = false
			continue
		}
		if charMetrics {
			var name string
			var width float64
			code := -1
			var BBox rect.Rect

			var ligs []ligature

			keyVals := strings.SplitSeq(line, ";")
			for keyVal := range keyVals {
				ff, numFields := splitFields(keyVal)
				if numFields < 2 {
					continue
				}
				switch ff[0] {
				case "C":
					// a code we cannot read leaves the glyph unencoded
					if c, err := strconv.Atoi(ff[1]); err == nil {
						code = c
					}
				case "CH":
					// the hexadecimal form of C, written as "CH <41>".  The
					// brackets are optional here, so that a file which leaves
					// them out is still understood.
					h := strings.Trim(ff[1], "<>")
					if c, err := strconv.ParseInt(h, 16, 32); err == nil {
						code = int(c)
					}
				case "WX":
					if x, ok := readNumber(ff[1]); ok {
						width = x
					}
				case "N":
					// The name is repaired rather than refused, so that a
					// file naming a glyph in a way PostScript cannot write
					// still yields metrics.  A name left empty by the repair
					// drops the glyph below.
					name = repairName(ff[1])
				case "B":
					if numFields != 5 {
						continue
					}
					// a box is usable only whole: one coordinate we cannot
					// read would otherwise leave a box the file never
					// described
					llx, ok1 := readNumber(ff[1])
					lly, ok2 := readNumber(ff[2])
					urx, ok3 := readNumber(ff[3])
					ury, ok4 := readNumber(ff[4])
					if ok1 && ok2 && ok3 && ok4 {
						BBox = rect.Rect{LLx: llx, LLy: lly, URx: urx, URy: ury}
					}
				case "L":
					if numFields >= 3 {
						succ := repairName(ff[1])
						repl := repairName(ff[2])
						if succ == "" || repl == "" {
							continue
						}
						ligs = append(ligs, ligature{succ, repl})
					}
				}
			}
			_, seen := res.Glyphs[name]
			if name == "" || seen {
				continue
			}
			// A glyph is kept only if its line can be written back.  The
			// ligatures are then added one by one for as long as the line has
			// room, so that what Read returns is always something Write can
			// produce.
			lineLen := charMetricsFixed + len(name)
			if lineLen > maxLineLen {
				continue
			}
			if code >= 0 && code < 256 {
				res.Encoding[code] = name
			}
			var ligTmp map[string]string
			for _, l := range ligs {
				need := ligatureFixed + len(l.succ) + len(l.repl)
				if lineLen+need > maxLineLen {
					break
				}
				lineLen += need
				if ligTmp == nil {
					ligTmp = make(map[string]string, len(ligs))
				}
				ligTmp[l.succ] = l.repl
			}

			res.Glyphs[name] = &GlyphInfo{
				WidthX:    width,
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
		// The section markers are matched before the keys below, which all
		// need a value: a marker opens its section whether or not the count
		// follows, as the closing markers above need no argument either.
		if fields[0] == "StartCharMetrics" {
			charMetrics = true
			// The header states how many entries follow.  Use this to size the
			// map, but limit how much memory a bogus count can claim.  Only
			// the first section counts, so that a later one neither discards
			// the entries already read nor makes a file of repeated headers
			// allocate a map per line.
			if numFields >= 2 && !glyphsSized {
				if n, err := strconv.Atoi(fields[1]); err == nil {
					glyphsSized = true
					res.Glyphs = make(map[string]*GlyphInfo, min(max(n, 0), maxPrealloc))
				}
			}
			continue
		}
		if fields[0] == "StartKernPairs" {
			kernPairs = true
			// as above, for the kerning pairs
			if numFields >= 2 && res.Kern == nil {
				if n, err := strconv.Atoi(fields[1]); err == nil {
					res.Kern = make([]KernPair, 0, min(max(n, 0), maxPrealloc))
				}
			}
			continue
		}
		if kernPairs && numFields == 4 && fields[0] == "KPX" {
			// the names are repaired as in the character metrics, so that a
			// pair still matches the glyphs it kerns
			left := repairName(fields[1])
			right := repairName(fields[2])
			if left == "" || right == "" {
				continue
			}
			// as above: a pair which would not fit on a line of its own is
			// not kept
			if kernPairFixed+len(left)+len(right) > maxLineLen {
				continue
			}
			// an adjustment we cannot read means no kerning for the pair
			x, _ := readNumber(fields[3])
			res.Kern = append(res.Kern, KernPair{
				Left:   left,
				Right:  right,
				Adjust: x,
			})
			continue
		}
		if numFields < 2 {
			continue
		}
		switch fields[0] {
		case "FontName":
			res.FontName = type1.RepairFontName(fields[1])

		// A text value too long to write back leaves the field alone, as does
		// an unusable number below.
		case "FullName":
			if v := lineValue(line); len(v) <= maxTextValueLen {
				res.FullName = v
			}
		case "FamilyName":
			if v := lineValue(line); len(v) <= maxTextValueLen {
				res.FamilyName = v
			}
		case "Weight":
			if v := lineValue(line); len(v) <= maxTextValueLen {
				res.Weight = v
			}
		case "Version":
			if v := lineValue(line); len(v) <= maxTextValueLen {
				res.Version = v
			}
		case "Notice":
			if v := lineValue(line); len(v) <= maxTextValueLen {
				res.Notice = v
			}

		// For the metrics below, a value which cannot be used leaves the field
		// alone too, so that a repeated key with an unusable second value
		// keeps the first.
		case "CapHeight":
			if x, ok := readNumber(fields[1]); ok {
				res.CapHeight = x
			}
		case "XHeight":
			if x, ok := readNumber(fields[1]); ok {
				res.XHeight = x
			}
		case "Ascender":
			if x, ok := readNumber(fields[1]); ok {
				res.Ascent = x
			}
		case "Descender":
			if x, ok := readNumber(fields[1]); ok {
				res.Descent = x
			}
		case "UnderlinePosition":
			if x, ok := readNumber(fields[1]); ok {
				res.UnderlinePosition = x
			}
		case "UnderlineThickness":
			if x, ok := readNumber(fields[1]); ok {
				res.UnderlineThickness = x
			}
		case "ItalicAngle":
			if x, ok := readNumber(fields[1]); ok {
				res.ItalicAngle = x
			}
		case "IsFixedPitch":
			res.IsFixedPitch = fields[1] == "true"
		}
	}
	// a font without kerning pairs has no kern slice, even if the file
	// announced a kern section
	if len(res.Kern) == 0 {
		res.Kern = nil
	}

	return res, nil
}
