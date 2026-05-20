// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2026  Jochen Voss <voss@seehuhn.de>
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

package postscript

import (
	"strings"
	"testing"
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
