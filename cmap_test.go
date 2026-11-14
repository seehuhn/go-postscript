// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
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
	dict, codeMap, err := ReadCMap(strings.NewReader(src))
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
	_, codeMap, err := ReadCMap(strings.NewReader(src))
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
	if _, _, err := ReadCMap(strings.NewReader(src)); err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

// TestReadCMapMultiple verifies that input which defines more than one CMap
// is rejected.  Real PDF CMap streams contain exactly one CMap, so this is
// almost certainly a malicious or corrupt file.
func TestReadCMapMultiple(t *testing.T) {
	src := `/CIDInit /ProcSet findresource begin
12 dict begin
/CMapType 1 def
/CMapName /first def
begincmap
endcmap
/first currentdict /CMap defineresource pop
end
12 dict begin
/CMapType 1 def
/CMapName /second def
begincmap
endcmap
/second currentdict /CMap defineresource pop
end
`
	if _, _, err := ReadCMap(strings.NewReader(src)); err == nil {
		t.Fatal("expected error for multiple CMaps, got nil")
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
	if _, _, err := ReadCMap(strings.NewReader(src)); err == nil {
		t.Fatal("expected error for mutated CodeMap, got nil")
	}
}
