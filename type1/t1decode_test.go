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

package type1

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/geom/vec"
	"seehuhn.de/go/membudget"
)

// encInt encodes v as a five-byte Type 1 charstring integer operand.
func encInt(v int) []byte {
	return []byte{0xff, byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

// buildBlendCharstring assembles a charstring that invokes the MM blend
// othersubr `othersubr` with the given per-value base and delta operands,
// then retrieves the m blended results with m pop operators, emitting each
// as the dx of an rmoveto so the results surface as cumulative moveto
// coordinates.  deltas[i] holds the k-1 master deltas for value i.
func buildBlendCharstring(othersubr int, base []float64, deltas [][]float64) []byte {
	var cs []byte
	cs = append(cs, encInt(0)...)   // sidebearing
	cs = append(cs, encInt(100)...) // width
	cs = append(cs, 0x0d)           // hsbw
	n := 0
	for _, b := range base {
		cs = append(cs, encInt(int(b))...)
		n++
	}
	for _, dv := range deltas {
		for _, d := range dv {
			cs = append(cs, encInt(int(d))...)
			n++
		}
	}
	cs = append(cs, encInt(n)...)         // argument count
	cs = append(cs, encInt(othersubr)...) // othersubr number
	cs = append(cs, 0x0c, 0x10)           // callothersubr
	for range base {
		cs = append(cs, 0x0c, 0x11)   // pop
		cs = append(cs, encInt(0)...) // dy
		cs = append(cs, 0x15)         // rmoveto
	}
	cs = append(cs, 0x0e) // endchar
	return cs
}

// blendInputs returns distinguishable base and delta operands for a blend
// of m values across k masters: base values in the thousands, deltas small
// so the two are never confused.
func blendInputs(m, k int) (base []float64, deltas [][]float64) {
	base = make([]float64, m)
	deltas = make([][]float64, m)
	for i := range base {
		base[i] = float64(1000 * (i + 1))
		deltas[i] = make([]float64, k-1)
		for j := range deltas[i] {
			deltas[i][j] = float64((i+1)*10 + (j + 1))
		}
	}
	return base, deltas
}

// wantBlendCoords computes the cumulative moveto coordinates that
// buildBlendCharstring must yield for the given operands and weights.
func wantBlendCoords(base []float64, deltas [][]float64, wv []float64) []vec.Vec2 {
	want := make([]vec.Vec2, len(base))
	cum := 0.0
	for i := range base {
		r := base[i]
		for j := 1; j < len(wv); j++ {
			r += deltas[i][j-1] * wv[j]
		}
		cum += r
		want[i] = vec.Vec2{X: cum, Y: 0}
	}
	return want
}

// TestDecodeCharStringBlend checks the multiple master blend othersubrs
// (14-18) against analytically computed results.  The base and delta
// operands are chosen so a wrong operand ordering would produce visibly
// different coordinates.
func TestDecodeCharStringBlend(t *testing.T) {
	wv2 := []float64{0.25, 0.75}
	wv4 := []float64{0.125, 0.25, 0.5, 0.125}
	cases := []struct {
		name      string
		othersubr int
		m         int
		wv        []float64
	}{
		{"os14_2master", 14, 1, wv2},
		{"os15_2master", 15, 2, wv2},
		{"os16_2master", 16, 3, wv2},
		{"os17_2master", 17, 4, wv2},
		{"os18_2master", 18, 6, wv2},
		{"os14_4master", 14, 1, wv4},
		{"os15_4master", 15, 2, wv4},
		{"os16_4master", 16, 3, wv4},
		{"os17_4master", 17, 4, wv4},
		{"os18_4master", 18, 6, wv4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base, deltas := blendInputs(tc.m, len(tc.wv))
			cs := buildBlendCharstring(tc.othersubr, base, deltas)
			info := &decodeInfo{budget: newTestBudget(), weightVector: tc.wv}
			g := info.decodeCharString(cs, "blend")
			want := wantBlendCoords(base, deltas, tc.wv)
			if diff := cmp.Diff(want, g.Outline.Coords); diff != "" {
				t.Errorf("blend coords mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// newTestBudget returns a budget generous enough that no well-formed test
// charstring trips it.
func newTestBudget() *membudget.Budget { return membudget.New(64 << 20) }

// TestDecodeCharStringMalformedNoPanic checks that decodeCharString
// returns a blank stub rather than panicking on malformed input.
// The inputs exercise the bounds and arity checks added to
// t1callothersubr.
func TestDecodeCharStringMalformedNoPanic(t *testing.T) {
	cases := []struct {
		name string
		cs   []byte
	}{
		// 0, 0, callothersubr — flex-end with argN=0
		{"othersubr0_argN0", []byte{0x8b, 0x8b, 0x0c, 0x10}},
		// 0, 0, 2, 1, callothersubr — flex-start with argN=2
		{"othersubr1_argN2", []byte{0x8b, 0x8b, 0x8d, 0x8c, 0x0c, 0x10}},
		// 0, 1, 2, callothersubr — flex coord pair with argN=1
		{"othersubr2_argN1", []byte{0x8b, 0x8c, 0x8d, 0x0c, 0x10}},
		// 0, 3, callothersubr — hint replacement with argN=0
		{"othersubr3_argN0", []byte{0x8b, 0x8e, 0x0c, 0x10}},
		// -1 (5-byte int), 0, callothersubr — negative argN
		{"argN_negative", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0x8b, 0x0c, 0x10}},
		// 2_000_000 (5-byte int), 0, callothersubr — argN above the threshold
		{"argN_excessive", []byte{0xff, 0x00, 0x1e, 0x84, 0x80, 0x8b, 0x0c, 0x10}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := &decodeInfo{budget: newTestBudget()}
			g := info.decodeCharString(tc.cs, "test")
			if g == nil {
				t.Fatal("expected non-nil stub")
			}
			if len(g.Outline.Cmds) != 0 || len(g.HStem) != 0 || len(g.VStem) != 0 {
				t.Errorf("expected blank stub, got %+v", g)
			}
		})
	}
}

// TestDecodeCharStringBlendNilWeightVector checks that a blend othersubr in
// a non-MM font (weightVector == nil) is ignored: the raw operands stay on
// the postscript stack for the following pops, exactly as an unknown
// othersubr behaves, so the popped value is the raw base operand.
func TestDecodeCharStringBlendNilWeightVector(t *testing.T) {
	base := []float64{1000}
	deltas := [][]float64{{100}}
	cs := buildBlendCharstring(14, base, deltas)
	info := &decodeInfo{budget: newTestBudget()} // weightVector nil
	g := info.decodeCharString(cs, "blend")
	// pop returns the first operand pushed, i.e. the raw base value 1000
	want := []vec.Vec2{{X: 1000, Y: 0}}
	if diff := cmp.Diff(want, g.Outline.Coords); diff != "" {
		t.Errorf("nil-weightVector blend must be a no-op (-want +got):\n%s", diff)
	}
}

// TestDecodeCharStringBlendWrongArgN checks that a blend othersubr whose
// operand count does not equal m*k yields a blank stub via the bail path,
// without error or panic.
func TestDecodeCharStringBlendWrongArgN(t *testing.T) {
	// othersubr 14 wants m*k = 1*2 = 2 operands; supply 3.
	base := []float64{1000}
	deltas := [][]float64{{100, 200}}
	cs := buildBlendCharstring(14, base, deltas)
	info := &decodeInfo{budget: newTestBudget(), weightVector: []float64{0.25, 0.75}}
	g := info.decodeCharString(cs, "blend")
	if g == nil {
		t.Fatal("expected non-nil stub")
	}
	if len(g.Outline.Cmds) != 0 {
		t.Errorf("expected blank stub on wrong argN, got %d cmds", len(g.Outline.Cmds))
	}
	if g.WidthX != 100 {
		t.Errorf("stub must preserve width, got %v, want 100", g.WidthX)
	}
}

// TestDecodeCharStringBlendStackLimit checks the conditional operand-stack
// limit: a 16-master blend of 6 values pushes 96 operands, which decodes
// when weightVector is set but trips the ordinary limit when it is not.
func TestDecodeCharStringBlendStackLimit(t *testing.T) {
	wv := make([]float64, 16)
	wv[0] = 0.5
	wv[15] = 0.5
	base, deltas := blendInputs(6, 16)
	cs := buildBlendCharstring(18, base, deltas)

	// with a weight vector the raised limit admits all 96 operands
	info := &decodeInfo{budget: newTestBudget(), weightVector: wv}
	g := info.decodeCharString(cs, "blend")
	want := wantBlendCoords(base, deltas, wv)
	if diff := cmp.Diff(want, g.Outline.Coords); diff != "" {
		t.Errorf("16-master blend coords mismatch (-want +got):\n%s", diff)
	}

	// without one, the same 96 operands hit the ordinary 24-operand limit
	info2 := &decodeInfo{budget: newTestBudget()}
	g2 := info2.decodeCharString(cs, "blend")
	if len(g2.Outline.Cmds) != 0 {
		t.Errorf("expected blank stub from tripped limit, got %d cmds", len(g2.Outline.Cmds))
	}
}

// TestReadMalformedCharstringRoundTrip checks that a font containing
// a malformed glyph charstring reads successfully (with the bad glyph
// substituted by a blank glyph), and that the resulting font writes
// back and re-reads identically.
func TestReadMalformedCharstringRoundTrip(t *testing.T) {
	encoding := makeEmptyEncoding()
	encoding[65] = "A"
	F := &Font{
		FontInfo: &FontInfo{
			FontName:   "Test",
			FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		},
		Outlines: &Outlines{
			Private:  &PrivateDict{},
			Glyphs:   map[string]*Glyph{},
			Encoding: encoding,
		},
	}
	g := F.NewGlyph(".notdef", 100)
	g.MoveTo(10, 10)
	g.LineTo(20, 10)
	g.LineTo(20, 20)
	g.LineTo(10, 20)
	g.ClosePath()
	g = F.NewGlyph("A", 200)
	g.MoveTo(0, 10)
	g.LineTo(200, 10)
	g.LineTo(100, 110)
	g.ClosePath()

	var buf bytes.Buffer
	if err := F.Write(&buf, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}

	// replace /A's charstring with `0 200 hsbw 0 0 callothersubr` —
	// the leading hsbw sets WidthX before the trailing bytes panic, so
	// reading must preserve the width on the substituted glyph.
	malicious := obfuscateCharstring(
		[]byte{0x8b, 0xf7, 0x5c, 0x0d, 0x8b, 0x8b, 0x0c, 0x10},
		[]byte{0, 0, 0, 0})
	patched, ok := patchCharstring(buf.Bytes(), "A", malicious)
	if !ok {
		t.Fatal("could not locate /A charstring in PFA output")
	}

	// reading must succeed and substitute a blank glyph for /A
	F1, err := Read(bytes.NewReader(patched))
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	gA, ok := F1.Glyphs["A"]
	if !ok || gA == nil {
		t.Fatal("glyph A missing after substitution")
	}
	if gA.Outline == nil || len(gA.Outline.Cmds) != 0 {
		t.Errorf("glyph A: expected blank outline, got %d cmds", len(gA.Outline.Cmds))
	}
	if gA.WidthX != 200 {
		t.Errorf("glyph A: WidthX not preserved, got %v, want 200", gA.WidthX)
	}
	if len(gA.HStem) != 0 || len(gA.VStem) != 0 {
		t.Errorf("glyph A: hint state must not leak into the substituted glyph (HStem=%v, VStem=%v)",
			gA.HStem, gA.VStem)
	}
	gNotdef, ok := F1.Glyphs[".notdef"]
	if !ok || gNotdef == nil || len(gNotdef.Outline.Cmds) == 0 {
		t.Errorf(".notdef should be intact, got %v", gNotdef)
	}

	// round-trip: write F1 and read again, must match F1 exactly
	var buf2 bytes.Buffer
	if err := F1.Write(&buf2, &WriterOptions{Format: FormatNoEExec}); err != nil {
		t.Fatal(err)
	}
	F2, err := Read(bytes.NewReader(buf2.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(F1, F2); diff != "" {
		t.Errorf("round-trip differs (-F1 +F2):\n%s", diff)
	}
}

// patchCharstring replaces the obfuscated bytes of glyph `name` in a
// FormatNoEExec PFA, returning the modified buffer and true on success.
// The input format is `/<name> <len> RD <bytes...> ND`.
func patchCharstring(pfa []byte, name string, newBytes []byte) ([]byte, bool) {
	needle := []byte("/" + name + " ")
	i := bytes.Index(pfa, needle)
	if i < 0 {
		return nil, false
	}
	j := i + len(needle)
	k := j
	for k < len(pfa) && pfa[k] >= '0' && pfa[k] <= '9' {
		k++
	}
	oldLen, err := strconv.Atoi(string(pfa[j:k]))
	if err != nil {
		return nil, false
	}
	if !bytes.HasPrefix(pfa[k:], []byte(" RD ")) {
		return nil, false
	}
	bytesStart := k + len(" RD ")
	bytesEnd := bytesStart + oldLen
	if bytesEnd > len(pfa) {
		return nil, false
	}
	out := make([]byte, 0, len(pfa)+len(newBytes)-oldLen+8)
	out = append(out, pfa[:j]...)
	out = append(out, strconv.Itoa(len(newBytes))...)
	out = append(out, " RD "...)
	out = append(out, newBytes...)
	out = append(out, pfa[bytesEnd:]...)
	return out, true
}

// TestDecodeCharStringFanoutBomb checks that decodeCharString terminates on a
// malicious charstring whose subrs fan out exponentially.
//
// Each subr i calls subr i+1 `fan` times before returning. Because every call
// fully unwinds before the next sibling call begins, the live command stack
// never exceeds `depth` frames (well under the depth-10 cap), yet the total
// work is fan^(depth-1). Without an execution budget this hangs the reader on
// input far smaller than the work it triggers — a denial-of-service.
//
// The chain is placed at subr indices 4..4+depth-1 to avoid index 3, which the
// interpreter treats as a predefined no-op.
func TestDecodeCharStringFanoutBomb(t *testing.T) {
	const depth = 9 // keep < 10 so the legitimate depth cap is not what stops us
	const fan = 20  // 20^8 = 2.56e10 executions: impossible to finish honestly

	const base = 4
	subrs := make([][]byte, base+depth)
	subrs[base+depth-1] = []byte{0x0b} // innermost: return
	for i := depth - 2; i >= 0; i-- {
		child := base + i + 1
		var body []byte
		for range fan {
			body = append(body, byte(child+139), 0x0a) // push child index, callsubr
		}
		body = append(body, 0x0b) // return
		subrs[base+i] = body
	}
	// 0 0 hsbw, <base> callsubr, endchar
	cs := []byte{0x8b, 0x8b, 0x0d, byte(base + 139), 0x0a, 0x0e}

	// size the budget exactly as production does for this font
	subrBytes := 0
	for _, s := range subrs {
		subrBytes += len(s)
	}
	info := &decodeInfo{
		subrs:  subrs,
		budget: newCharstringBudget(subrBytes + len(cs)),
	}

	// run in a goroutine so a regression that removes the budget fails the
	// test (via timeout) instead of hanging it forever
	done := make(chan struct{})
	var g *Glyph
	go func() {
		g = info.decodeCharString(cs, "bomb")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("decodeCharString did not terminate: subr fan-out has no execution budget")
	}
	// the bomb cannot finish honestly, so termination means the budget
	// tripped and bail() returned a blank stub
	if len(g.Outline.Cmds) != 0 {
		t.Errorf("expected blank stub from tripped budget, got %d outline cmds", len(g.Outline.Cmds))
	}
}
