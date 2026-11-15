// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestReadCMapValid verifies that a minimal well-formed CMap is parsed into
// both the dictionary and the *CMapInfo return values.
func TestReadCMapValid(t *testing.T) {
	src := `/CIDInit /ProcSet findresource begin
12 dict begin
/CMapType 1 def
/CMapName /test def
begincmap
1 begincodespacerange
<00> <ff>
endcodespacerange
2 begincidchar
<41> 65
<42> 66
endcidchar
endcmap
/test currentdict /CMap defineresource pop
end
`
	dict, codeMap, err := ReadCMap(strings.NewReader(src), "")
	if err != nil {
		t.Fatalf("ReadCMap failed: %v", err)
	}
	if name, _ := dict["CMapName"].(Name); name != "test" {
		t.Errorf("CMapName: got %q, want %q", name, "test")
	}
	if codeMap == nil {
		t.Fatal("CMapInfo is nil")
	}
	if got, want := len(codeMap.CodeSpaceRanges), 1; got != want {
		t.Errorf("CodeSpaceRanges: got %d entries, want %d", got, want)
	}
	if got, want := len(codeMap.CidChars), 2; got != want {
		t.Errorf("CidChars: got %d entries, want %d", got, want)
	}
}

// TestReadCMapKeepsOrder verifies that mappings are returned in file
// order, since a later mapping takes precedence over an earlier one.
func TestReadCMapKeepsOrder(t *testing.T) {
	src := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /test def
1 begincodespacerange
<00> <ff>
endcodespacerange
2 begincidchar
<42> 2
<41> 1
endcidchar
2 begincidrange
<10> <20> 100
<00> <ff> 1
endcidrange
endcmap
/test currentdict /CMap defineresource pop
end
`
	_, codeMap, err := ReadCMap(strings.NewReader(src), "")
	if err != nil {
		t.Fatal(err)
	}
	wantChars := []CharMap{
		{Src: []byte{0x42}, Dst: Integer(2)},
		{Src: []byte{0x41}, Dst: Integer(1)},
	}
	if diff := cmp.Diff(wantChars, codeMap.CidChars); diff != "" {
		t.Errorf("cid chars differ (-want +got):\n%s", diff)
	}
	wantRanges := []RangeMap{
		{Low: []byte{0x10}, High: []byte{0x20}, Dst: Integer(100)},
		{Low: []byte{0x00}, High: []byte{0xff}, Dst: Integer(1)},
	}
	if diff := cmp.Diff(wantRanges, codeMap.CidRanges); diff != "" {
		t.Errorf("cid ranges differ (-want +got):\n%s", diff)
	}
}

// TestReadCMapEmpty verifies that input which does not define any CMap
// is rejected.
func TestReadCMapEmpty(t *testing.T) {
	src := `% no CMap defined
`
	if _, _, err := ReadCMap(strings.NewReader(src), ""); err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

// TestReadCMapMultiple checks the choice between several CMaps in one file:
// the last one with the requested name, or else the last one defined.
func TestReadCMapMultiple(t *testing.T) {
	src := `/CIDInit /ProcSet findresource begin
12 dict begin
/CMapName /first def
begincmap
1 begincidchar
<01> 1
endcidchar
endcmap
CMapName currentdict /CMap defineresource pop
end
12 dict begin
/CMapName /second def
begincmap
/first usecmap
1 begincidchar
<01> 2
endcidchar
endcmap
/other currentdict /CMap defineresource pop
end
end
`
	tests := []struct {
		name     Name
		wantName Name
		wantCID  Integer
	}{
		{"", "second", 2},
		{"first", "first", 1},
		{"second", "second", 2},
		{"other", "second", 2},
		{"missing", "second", 2},
	}
	for _, tc := range tests {
		dict, codeMap, err := ReadCMap(strings.NewReader(src), tc.name)
		if err != nil {
			t.Fatalf("%q: %v", tc.name, err)
		}
		if name, _ := dict["CMapName"].(Name); name != tc.wantName {
			t.Errorf("%q: got CMap %q, want %q", tc.name, name, tc.wantName)
		}
		if len(codeMap.CidChars) != 1 || codeMap.CidChars[0].Dst != tc.wantCID {
			t.Errorf("%q: wrong mappings %v", tc.name, codeMap.CidChars)
		}
	}
}

// TestReadCMapMutatedCodeMap exercises a malicious CMap that overwrites the
// "CodeMap" slot after defineresource.  ReadCMap must return an error rather
// than handing the caller a Dict whose CodeMap is no longer a *CMapInfo.
func TestReadCMapMutatedCodeMap(t *testing.T) {
	src := `/CIDInit /ProcSet findresource begin
12 dict begin
/CMapType 1 def
/CMapName /attack def
begincmap
endcmap
/attack currentdict /CMap defineresource pop
currentdict /CodeMap (junk) put
end
`
	if _, _, err := ReadCMap(strings.NewReader(src), ""); err == nil {
		t.Fatal("expected error for mutated CodeMap, got nil")
	}
}

// TestReadCMapPermissive checks that common defects of CMap files lose
// only the affected entries, not the whole CMap.
func TestReadCMapPermissive(t *testing.T) {
	const head = "/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n" +
		"/CMapName /test def\n1 begincodespacerange\n<00> <ff>\nendcodespacerange\n"
	const tail = "endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n"

	var many strings.Builder
	for i := range 300 {
		fmt.Fprintf(&many, "<%02x> %d\n", i%256, i)
	}

	type test struct {
		name   string
		src    string
		want   []CharMap
		ranges []RangeMap
		bf     []CharMap
	}
	tests := []test{
		{
			name: "count too small",
			src:  head + "1 begincidchar\n<01> 1\n<02> 2\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}, {[]byte{2}, Integer(2)}},
		},
		{
			name: "count too large",
			src:  head + "5 begincidchar\n<01> 1\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
		{
			name: "count missing",
			src:  head + "begincidchar\n<01> 1\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
		{
			name: "real CID",
			src:  head + "1 begincidchar\n<01> 1.0\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Real(1)}},
		},
		{
			name: "malformed entry",
			src:  head + "3 begincidchar\n<01> 1\n<02> (x)\n<03> 3\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}, {[]byte{3}, Integer(3)}},
		},
		{
			name: "excess operand",
			src:  head + "2 begincidchar\n7 <01> 1\n<02> 2\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}, {[]byte{2}, Integer(2)}},
		},
		{
			name: "missing end operator",
			src: head + "1 begincidchar\n<01> 1\n" +
				"1 begincidrange\n<10> <1f> 16\nendcidrange\n" + tail,
			want:   []CharMap{{[]byte{1}, Integer(1)}},
			ranges: []RangeMap{{[]byte{0x10}, []byte{0x1f}, Integer(16)}},
		},
		{
			name: "invalid range",
			src: head + "2 begincidrange\n<1f> <10> 16\n<20> <2f> 32\nendcidrange\n" +
				"1 begincidrange\n<30> <003f> 48\nendcidrange\n" + tail,
			ranges: []RangeMap{{[]byte{0x20}, []byte{0x2f}, Integer(32)}},
		},
		{
			name: "missing CIDInit",
			src: "12 dict begin\nbegincmap\n/CMapName /test def\n" +
				"1 begincidchar\n<01> 1\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
		{
			name: "missing begincmap",
			src: "/CIDInit /ProcSet findresource begin\n12 dict begin\n" +
				"/CMapName /test def\n1 begincidchar\n<01> 1\nendcidchar\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
		{
			name: "missing endcmap and defineresource",
			src:  head + "1 begincidchar\n<01> 1\nendcidchar\nend\nend\n",
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
		{
			name: "truncated",
			src:  head + "2 begincidchar\n<01> 1\n<02> 2\n<0",
			want: []CharMap{{[]byte{1}, Integer(1)}, {[]byte{2}, Integer(2)}},
		},
		{
			name: "truncated mid-entry",
			src:  head + "3 begincidchar\n<01> 1\n<02> 2\n<03>",
			want: []CharMap{{[]byte{1}, Integer(1)}, {[]byte{2}, Integer(2)}},
		},
		{
			name:   "truncated mid-range",
			src:    head + "2 begincidrange\n<10> <1f> 16\n<20> <2f>",
			ranges: []RangeMap{{[]byte{0x10}, []byte{0x1f}, Integer(16)}},
		},
		{
			name: "truncated mid-bfchar",
			src:  head + "3 beginbfchar\n<01> <0041>\n<02> <0042>\n<03>",
			bf: []CharMap{
				{[]byte{1}, String{0x00, 0x41}},
				{[]byte{2}, String{0x00, 0x42}},
			},
		},
		{
			name: "missing endcmap",
			src: head + "1 begincidchar\n<01> 1\nendcidchar\n" +
				"CMapName currentdict /CMap defineresource pop\nend\nend\n",
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
		{
			name: "count missing after open block",
			src: head + "1 begincidchar\n<01> 1\n" +
				"begincidrange\n<10> <1f> 16\nendcidrange\n" + tail,
			want:   []CharMap{{[]byte{1}, Integer(1)}},
			ranges: []RangeMap{{[]byte{0x10}, []byte{0x1f}, Integer(16)}},
		},
		{
			name: "mismatched end operator",
			src:  head + "1 begincidchar\n<01> 1\nendcidrange\n" + tail,
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
		{
			name: "entries after endcmap",
			src: head + "1 begincidchar\n<01> 1\nendcidchar\nendcmap\n" +
				"1 begincidchar\n<02> 2\nendcidchar\n" +
				"CMapName currentdict /CMap defineresource pop\nend\nend\n",
			want: []CharMap{{[]byte{1}, Integer(1)}, {[]byte{2}, Integer(2)}},
		},
		{
			name: "trailing junk",
			src:  head + "1 begincidchar\n<01> 1\nendcidchar\n" + tail + "junk }\n",
			want: []CharMap{{[]byte{1}, Integer(1)}},
		},
	}

	var wantMany []CharMap
	for i := range 300 {
		wantMany = append(wantMany, CharMap{[]byte{byte(i % 256)}, Integer(i)})
	}
	tests = append(tests, test{
		name: "large block",
		src:  head + "300 begincidchar\n" + many.String() + "endcidchar\n" + tail,
		want: wantMany,
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dict, codeMap, err := ReadCMap(strings.NewReader(tc.src), "")
			if err != nil {
				t.Fatal(err)
			}
			if name, _ := dict["CMapName"].(Name); name != "test" {
				t.Errorf("wrong CMapName %q", name)
			}
			if diff := cmp.Diff(tc.want, codeMap.CidChars); diff != "" {
				t.Errorf("wrong cidchar entries (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.ranges, codeMap.CidRanges); diff != "" {
				t.Errorf("wrong cidrange entries (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.bf, codeMap.BfChars); diff != "" {
				t.Errorf("wrong bfchar entries (-want +got):\n%s", diff)
			}
		})
	}
}

// TestReadCMapMissingEndcmap checks that a CMap without endcmap does not
// pass its entries on to the next CMap in the file, whether or not the next
// CMap starts with begincmap.
func TestReadCMapMissingEndcmap(t *testing.T) {
	const first = `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /a def
1 begincidchar
<01> 1
endcidchar
end
12 dict begin
`
	const second = `/CMapName /b def
1 begincidchar
<02> 2
endcidchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`
	for _, src := range []string{first + "begincmap\n" + second, first + second} {
		dict, codeMap, err := ReadCMap(strings.NewReader(src), "")
		if err != nil {
			t.Fatal(err)
		}
		if name, _ := dict["CMapName"].(Name); name != "b" {
			t.Errorf("got CMap %q, want %q", name, "b")
		}
		want := []CharMap{{[]byte{2}, Integer(2)}}
		if diff := cmp.Diff(want, codeMap.CidChars); diff != "" {
			t.Errorf("wrong cidchar entries (-want +got):\n%s", diff)
		}
	}
}

// TestReadCMapLastUsable checks that a defined CMap which lost its CodeMap
// is passed over in favour of an earlier usable one.
func TestReadCMapLastUsable(t *testing.T) {
	src := `/CIDInit /ProcSet findresource begin
12 dict begin
/CMapName /first def
begincmap
1 begincidchar
<01> 1
endcidchar
endcmap
CMapName currentdict /CMap defineresource pop
end
12 dict begin
/CMapName /second def
begincmap
endcmap
CMapName currentdict /CMap defineresource pop
currentdict /CodeMap (junk) put
end
end
`
	for _, name := range []Name{"", "first", "second"} {
		dict, codeMap, err := ReadCMap(strings.NewReader(src), name)
		if err != nil {
			t.Fatalf("%q: %v", name, err)
		}
		if got, _ := dict["CMapName"].(Name); got != "first" {
			t.Errorf("%q: got CMap %q, want %q", name, got, "first")
		}
		if len(codeMap.CidChars) != 1 {
			t.Errorf("%q: wrong mappings %v", name, codeMap.CidChars)
		}
	}
}

// TestReadCMapUnequalCodeSpace checks that a code space range whose bounds
// differ in length is dropped.
func TestReadCMapUnequalCodeSpace(t *testing.T) {
	src := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /test def
2 begincodespacerange
<00> <00ff>
<00> <7f>
endcodespacerange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`
	_, codeMap, err := ReadCMap(strings.NewReader(src), "")
	if err != nil {
		t.Fatal(err)
	}
	want := []CodeSpaceRange{{Low: []byte{0x00}, High: []byte{0x7f}}}
	if diff := cmp.Diff(want, codeMap.CodeSpaceRanges); diff != "" {
		t.Errorf("wrong code space ranges (-want +got):\n%s", diff)
	}
}

// TestReadCMapUseFont checks that usefont directs the mappings to the font
// selected, and that reading continues after it.
func TestReadCMapUseFont(t *testing.T) {
	const head = "/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n" +
		"/CMapName /test def\n1 begincodespacerange\n<00> <ff>\nendcodespacerange\n"
	const tail = "endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n"

	type test struct {
		name string
		src  string
		want CMapInfo
	}
	space := []CodeSpaceRange{{[]byte{0x00}, []byte{0xff}}}
	c1 := []CharMap{{[]byte{1}, Integer(1)}}
	c2 := []CharMap{{[]byte{2}, Integer(2)}}
	r3 := []RangeMap{{[]byte{0x10}, []byte{0x1f}, Integer(3)}}
	tests := []test{
		{
			name: "font 0",
			src:  head + "0 usefont\n1 begincidchar\n<01> 1\nendcidchar\n" + tail,
			want: CMapInfo{CodeSpaceRanges: space, CMapEntries: CMapEntries{CidChars: c1}},
		},
		{
			name: "several fonts",
			src: head + "0 usefont\n1 begincidchar\n<01> 1\nendcidchar\n" +
				"1 usefont\n1 begincidchar\n<02> 2\nendcidchar\n" +
				"2 usefont\n1 begincidrange\n<10> <1f> 3\nendcidrange\n" + tail,
			want: CMapInfo{
				CodeSpaceRanges: space,
				CMapEntries:     CMapEntries{CidChars: c1},
				Fonts: map[int]*CMapEntries{
					1: {CidChars: c2},
					2: {CidRanges: r3},
				},
			},
		},
		{
			name: "back to font 0",
			src: head + "1 usefont\n1 begincidchar\n<02> 2\nendcidchar\n" +
				"0 usefont\n1 begincidchar\n<01> 1\nendcidchar\n" +
				"1 usefont\n1 begincidrange\n<10> <1f> 3\nendcidrange\n" + tail,
			want: CMapInfo{
				CodeSpaceRanges: space,
				CMapEntries:     CMapEntries{CidChars: c1},
				Fonts:           map[int]*CMapEntries{1: {CidChars: c2, CidRanges: r3}},
			},
		},
		{
			name: "missing end operator",
			src: head + "1 begincidchar\n<01> 1\n" +
				"1 usefont\n1 begincidchar\n<02> 2\nendcidchar\n" + tail,
			want: CMapInfo{
				CodeSpaceRanges: space,
				CMapEntries:     CMapEntries{CidChars: c1},
				Fonts:           map[int]*CMapEntries{1: {CidChars: c2}},
			},
		},
		{
			name: "invalid font number",
			src: head + "-1 usefont\n1 begincidchar\n<02> 2\nendcidchar\n" +
				"(x) usefont\n1 begincidchar\n<02> 2\nendcidchar\n" +
				"usefont\n1 begincidchar\n<02> 2\nendcidchar\n" +
				"99999999 usefont\n1 begincidchar\n<02> 2\nendcidchar\n" +
				"0 usefont\n1 begincidchar\n<01> 1\nendcidchar\n" + tail,
			want: CMapInfo{CodeSpaceRanges: space, CMapEntries: CMapEntries{CidChars: c1}},
		},
		{
			name: "matrix and rearranged font",
			src: head + "/F [/A /B] beginrearrangedfont\n" +
				"1 beginusematrix [1 0 0 1 0 0] endusematrix\n" +
				"1 begincidchar\n<01> 1\nendcidchar\nendrearrangedfont\n" +
				"endusematrix\nbeginusematrix\nbeginrearrangedfont\n" + tail,
			want: CMapInfo{CodeSpaceRanges: space, CMapEntries: CMapEntries{CidChars: c1}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, codeMap, err := ReadCMap(strings.NewReader(tc.src), "")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(&tc.want, codeMap); diff != "" {
				t.Errorf("wrong CMap (-want +got):\n%s", diff)
			}
		})
	}
}

// TestUseFontNewCMap checks that a new CMap starts with font 0.
func TestUseFontNewCMap(t *testing.T) {
	const src = "/CIDInit /ProcSet findresource begin\n" +
		"12 dict begin\nbegincmap\n/CMapName /a def\n" +
		"1 usefont\n1 begincidchar\n<02> 2\nendcidchar\n" +
		"endcmap\nCMapName currentdict /CMap defineresource pop\nend\n" +
		"12 dict begin\nbegincmap\n/CMapName /b def\n" +
		"1 begincidchar\n<01> 1\nendcidchar\n" +
		"endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n"
	_, codeMap, err := ReadCMap(strings.NewReader(src), "b")
	if err != nil {
		t.Fatal(err)
	}
	want := &CMapInfo{CMapEntries: CMapEntries{CidChars: []CharMap{{[]byte{1}, Integer(1)}}}}
	if diff := cmp.Diff(want, codeMap); diff != "" {
		t.Errorf("wrong CMap (-want +got):\n%s", diff)
	}
}
