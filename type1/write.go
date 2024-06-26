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

package type1

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"text/template"
	"time"

	"seehuhn.de/go/postscript"
	"seehuhn.de/go/postscript/funit"
	"seehuhn.de/go/postscript/psenc"
)

// FileFormat specifies the on-disk format of a font file.
type FileFormat int

// List of supported file formats.
const (
	FormatPFA     FileFormat = iota + 1 // hex eexec
	FormatPFB                           // hex eexec, pfb wrapper
	FormatBinary                        // binary eexec
	FormatNoEExec                       // no eexec
)

// WriterOptions contains options for writing a font.
type WriterOptions struct {
	Format FileFormat // which file format to write (default: FormatPFA)
}

var defaultWriterOptions = &WriterOptions{
	Format: FormatPFA,
}

// Write writes the font to the given writer.
func (f *Font) Write(w io.Writer, opt *WriterOptions) error {
	if opt == nil {
		opt = defaultWriterOptions
	}
	format := opt.Format
	if format == 0 {
		format = FormatPFA
	}

	info := f.makeTemplateData(opt)

	switch format {
	case FormatPFA:
		err := tmpl.ExecuteTemplate(w, "SectionA", info)
		if err != nil {
			return err
		}

		wh := &hexWriter{w: w}
		we, err := newEExecWriter(wh)
		if err != nil {
			return err
		}
		err = tmpl.ExecuteTemplate(we, "SectionB", info)
		if err != nil {
			return err
		}
		err = we.Close()
		if err != nil {
			return err
		}
		err = wh.Close()
		if err != nil {
			return err
		}

		return tmpl.ExecuteTemplate(w, "SectionC", info)

	case FormatPFB:
		buf := &bytes.Buffer{}

		err := tmpl.ExecuteTemplate(buf, "SectionA", info)
		if err != nil {
			return err
		}
		n := uint32(buf.Len())
		_, err = w.Write([]byte{128, 1, byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)})
		if err != nil {
			return err
		}
		_, err = w.Write(buf.Bytes())
		if err != nil {
			return err
		}

		buf.Reset()
		we, err := newEExecWriter(buf)
		if err != nil {
			return err
		}
		err = tmpl.ExecuteTemplate(we, "SectionB", info)
		if err != nil {
			return err
		}
		err = we.Close()
		if err != nil {
			return err
		}
		n = uint32(buf.Len())
		_, err = w.Write([]byte{128, 2, byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)})
		if err != nil {
			return err
		}
		_, err = w.Write(buf.Bytes())
		if err != nil {
			return err
		}

		buf.Reset()
		err = tmpl.ExecuteTemplate(buf, "SectionC", info)
		if err != nil {
			return err
		}
		n = uint32(buf.Len())
		_, err = w.Write([]byte{128, 1, byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)})
		if err != nil {
			return err
		}
		_, err = w.Write(buf.Bytes())
		if err != nil {
			return err
		}

		_, err = w.Write([]byte{128, 3})
		if err != nil {
			return err
		}

	case FormatBinary:
		err := tmpl.ExecuteTemplate(w, "SectionA", info)
		if err != nil {
			return err
		}

		we, err := newEExecWriter(w)
		if err != nil {
			return err
		}
		err = tmpl.ExecuteTemplate(we, "SectionB", info)
		if err != nil {
			return err
		}
		err = we.Close()
		if err != nil {
			return err
		}

		_, err = w.Write([]byte{'\n'})
		if err != nil {
			return err
		}
		return tmpl.ExecuteTemplate(w, "SectionC", info)

	case FormatNoEExec:
		return tmpl.Execute(w, info)

	default:
		panic("invalid font file format")
	}

	return nil
}

// WritePDF writes the font in the format required for embedding in a PDF file.
func (f *Font) WritePDF(w io.Writer) (int, int, error) {
	opt := &WriterOptions{Format: FormatBinary}
	info := f.makeTemplateData(opt)

	wc := &countingWriter{w: w}

	err := tmpl.ExecuteTemplate(wc, "SectionA", info)
	if err != nil {
		return 0, 0, err
	}
	length1 := wc.n

	we, err := newEExecWriter(wc)
	if err != nil {
		return 0, 0, err
	}
	err = tmpl.ExecuteTemplate(we, "SectionB", info)
	if err != nil {
		return 0, 0, err
	}
	err = we.Close()
	if err != nil {
		return 0, 0, err
	}
	length2 := wc.n - length1

	return length1, length2, nil
}

type countingWriter struct {
	w io.Writer
	n int
}

func (w *countingWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.n += n
	return n, err
}

func (f *Font) makeTemplateData(opt *WriterOptions) *fontInfo {
	fontMatrix := f.FontInfo.FontMatrix
	if len(fontMatrix) != 6 {
		fontMatrix = [6]float64{0.001, 0, 0, 0.001, 0, 0}
	}

	info := &fontInfo{
		BlueFuzz:           f.Private.BlueFuzz,
		BlueScale:          f.Private.BlueScale,
		BlueShift:          f.Private.BlueShift,
		BlueValues:         f.Private.BlueValues,
		CharStrings:        f.encodeCharstrings(),
		Copyright:          f.FontInfo.Copyright,
		CreationDate:       f.CreationDate,
		Encoding:           f.Encoding,
		FamilyName:         f.FontInfo.FamilyName,
		FontMatrix:         fontMatrix,
		FontName:           f.FontInfo.FontName,
		ForceBold:          f.Private.ForceBold,
		FullName:           f.FontInfo.FullName,
		IsFixedPitch:       f.FontInfo.IsFixedPitch,
		ItalicAngle:        f.FontInfo.ItalicAngle,
		Notice:             f.FontInfo.Notice,
		OtherBlues:         f.Private.OtherBlues,
		UnderlinePosition:  float64(f.FontInfo.UnderlinePosition),
		UnderlineThickness: float64(f.FontInfo.UnderlineThickness),
		Version:            f.FontInfo.Version,
		Weight:             f.FontInfo.Weight,
		EExec:              opt.Format != FormatNoEExec,
	}
	if f.Private.StdHW != 0 {
		info.StdHW = []float64{f.Private.StdHW}
	}
	if f.Private.StdVW != 0 {
		info.StdVW = []float64{f.Private.StdVW}
	}
	return info
}

func (f *Font) encodeCharstrings() map[string]string {
	charStrings := make(map[string]string)
	for name, g := range f.Glyphs {
		cs := g.encodeCharString(int32(math.Round(g.WidthX)), int32(math.Round(g.WidthY)))

		var obf []byte
		iv := []byte{0, 0, 0, 0}
		for {
			obf = obfuscateCharstring(cs, iv)
			if obf[0] > 32 {
				couldBeHex := true
				for _, b := range obf[:4] {
					if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F') {
						couldBeHex = false
						break
					}
				}
				if !couldBeHex {
					break
				}
			}

			pos := 0
			for pos < 4 {
				iv[pos]++
				if iv[pos] != 0 {
					break
				}
				pos++
			}
		}
		charStrings[name] = string(obf)
	}
	return charStrings
}

func writeEncoding(encoding []string) string {
	if len(encoding) != 256 {
		return ""
	}
	if isStandardEncoding(encoding) {
		return "/Encoding StandardEncoding def\n"
	}

	b := &strings.Builder{}
	b.WriteString("/Encoding 256 array\n")
	b.WriteString("0 1 255 {1 index exch /.notdef put} for\n")
	for i, name := range encoding {
		if name == ".notdef" {
			continue
		}
		fmt.Fprintf(b, "dup %d %s put\n", i, postscript.Name(name).PS())
	}
	b.WriteString("readonly def\n")
	return b.String()
}

func isStandardEncoding(encoding []string) bool {
	if len(encoding) != 256 {
		return false
	}
	for i, s := range encoding {
		if s != psenc.StandardEncoding[i] && s != ".notdef" {
			return false
		}
	}
	return true
}

var tmpl = template.Must(template.New("type1").Funcs(template.FuncMap{
	"PS": func(s string) string {
		x := postscript.String(s)
		return x.PS()
	},
	"PN": func(s string) string {
		x := postscript.Name(s)
		return x.PS()
	},
	"E": writeEncoding,
}).Parse(`{{define "SectionA" -}}
%!FontType1-1.1: {{.FontName}} {{.Version}}
{{if not .CreationDate.IsZero}}%%CreationDate: {{.CreationDate.Format "2006-01-02 15:04:05 -0700 MST"}}
{{end -}}
10 dict begin
/FontInfo 11 dict dup begin
/version {{.Version|PS}} def
{{if .Notice}}/Notice {{.Notice|PS}} def
{{end -}}
{{if .Copyright}}/Copyright {{.Copyright|PS}} def
{{end -}}
/FullName {{.FullName|PS}} def
/FamilyName {{.FamilyName|PS}} def
/Weight {{.Weight|PS}} def
/ItalicAngle {{.ItalicAngle}} def
/isFixedPitch {{.IsFixedPitch}} def
/UnderlinePosition {{.UnderlinePosition}} def
/UnderlineThickness {{.UnderlineThickness}} def
end def
/FontName {{.FontName|PN}} def
{{ .Encoding|E -}}
/PaintType 0 def
/FontType 1 def
/FontMatrix {{ .FontMatrix }} def
/FontBBox [0 0 0 0] def
currentdict end
{{if .EExec}}currentfile eexec
{{end -}}
{{end -}}

{{define "SectionB" -}}
dup /Private 15 dict dup begin
/RD {string currentfile exch readstring pop} executeonly def
/ND {def} executeonly def
/NP {put} executeonly def
/Subrs {{ len .Subrs }} array
{{ range $index, $subr := .Subrs -}}
dup {{ $index }} {{ len $subr }} RD {{ $subr }} NP
{{ end -}}
{{ if .BlueValues}}/BlueValues {{ .BlueValues }} def
{{end -}}
{{ if .OtherBlues}}/OtherBlues {{ .OtherBlues }} def
{{end -}}
{{ if (or (lt .BlueScale .039624) (gt .BlueScale .039626)) -}}
/BlueScale {{.BlueScale}} def
{{end -}}
{{ if ne .BlueShift 7 }}/BlueShift {{.BlueShift}} def
{{end -}}
{{ if ne .BlueFuzz 1 }}/BlueFuzz {{.BlueFuzz}} def
{{end -}}
{{ if .StdHW }}/StdHW {{ .StdHW }} def
{{end -}}
{{ if .StdVW }}/StdVW {{ .StdVW }} def
{{end -}}
/ForceBold {{ .ForceBold }} def
/password 5839 def
/MinFeature {16 16} def
ND
2 index /CharStrings {{ len .CharStrings }} dict dup begin
{{ range $name, $cs := .CharStrings -}}
{{ $name|PN }} {{ len $cs }} RD {{ $cs }} ND
{{ end -}}
end
end
readonly put
put
dup /FontName get exch definefont pop
{{if .EExec}}mark currentfile closefile
{{end -}}
{{end -}}

{{define "SectionC" -}}
{{if .EExec -}}
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
cleartomark
{{end -}}
{{end -}}

{{template "SectionA" . -}}
{{template "SectionB" . -}}
{{template "SectionC" . -}}
`))

type fontInfo struct {
	BlueFuzz           int32
	BlueScale          float64
	BlueShift          int32
	BlueValues         []funit.Int16
	CharStrings        map[string]string
	Copyright          string
	CreationDate       time.Time
	Encoding           []string
	FamilyName         string
	FontMatrix         [6]float64
	FontName           string
	ForceBold          bool
	FullName           string
	IsFixedPitch       bool
	ItalicAngle        float64
	Notice             string
	OtherBlues         []funit.Int16
	StdHW              []float64
	StdVW              []float64
	Subrs              []string
	UnderlinePosition  float64
	UnderlineThickness float64
	Version            string
	Weight             string

	EExec bool
}
