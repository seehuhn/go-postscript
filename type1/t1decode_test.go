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
	"seehuhn.de/go/geom/path"
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
// The specification does not permit a stack this deep, so the first case
// tests deliberate leniency rather than conforming input.
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

// TestDecodeCharStringStackDepth checks both edges of the operand-stack
// limit: a charstring may fill the stack to exactly the limit and still have
// a command consume it, but one operand more is rejected.
func TestDecodeCharStringStackDepth(t *testing.T) {
	wv := make([]float64, 16)
	wv[0] = 0.5
	wv[15] = 0.5

	// Draws a line and then leaves n operands on the stack.  The drawing
	// comes first so that the outline is non-empty whenever the charstring
	// runs to completion; tripping the limit discards it via bail.
	build := func(n int) []byte {
		var cs []byte
		cs = append(cs, encInt(0)...)   // sidebearing
		cs = append(cs, encInt(500)...) // width
		cs = append(cs, 0x0d)           // hsbw
		cs = append(cs, encInt(10)...)
		cs = append(cs, encInt(10)...)
		cs = append(cs, 0x15) // rmoveto, clears the stack
		cs = append(cs, encInt(50)...)
		cs = append(cs, encInt(0)...)
		cs = append(cs, 0x05) // rlineto, clears the stack
		for range n {
			cs = append(cs, encInt(1)...)
		}
		cs = append(cs, 0x0e) // endchar
		return cs
	}

	cases := []struct {
		name    string
		wv      []float64
		nPushed int
		ok      bool
	}{
		{"plain at limit", nil, stackLimit, true},
		{"plain past limit", nil, stackLimit + 1, false},
		{"blend at limit", wv, stackLimitBlend, true},
		{"blend past limit", wv, stackLimitBlend + 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			info := &decodeInfo{budget: newTestBudget(), weightVector: c.wv}
			g := info.decodeCharString(build(c.nPushed), "test")
			if got := len(g.Outline.Cmds) > 0; got != c.ok {
				t.Errorf("%d operands: decoded=%v, want %v", c.nPushed, got, c.ok)
			}
		})
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

// appendPair encodes two charstring integer operands.
func appendPair(cs []byte, a, b int) []byte {
	cs = append(cs, encInt(a)...)
	return append(cs, encInt(b)...)
}

// TestDecodeCharStringCloseThenDraw checks the handling of a charstring which
// continues drawing after a closepath.  The Type 1 closepath command leaves
// the current point unchanged, so the decoded outline must start a new
// sub-path there rather than at the start of the closed sub-path.
func TestDecodeCharStringCloseThenDraw(t *testing.T) {
	var cs []byte
	cs = append(cs, encInt(0)...)   // sidebearing
	cs = append(cs, encInt(600)...) // width
	cs = append(cs, 0x0d)           // hsbw
	cs = appendPair(cs, 0, 0)
	cs = append(cs, 0x15) // rmoveto
	cs = appendPair(cs, 600, 0)
	cs = append(cs, 0x05) // rlineto
	cs = appendPair(cs, -300, 700)
	cs = append(cs, 0x05) // rlineto
	cs = append(cs, 0x09) // closepath
	cs = appendPair(cs, -200, -300)
	cs = append(cs, 0x05) // rlineto
	cs = append(cs, 0x0e) // endchar

	info := &decodeInfo{budget: newTestBudget()}
	g := info.decodeCharString(cs, "test")

	// The line after the closepath starts at the current point, which the
	// closepath left at (300, 700).
	want := &path.Data{}
	want.MoveTo(vec.Vec2{X: 0, Y: 0})
	want.LineTo(vec.Vec2{X: 600, Y: 0})
	want.LineTo(vec.Vec2{X: 300, Y: 700})
	want.Close()
	want.MoveTo(vec.Vec2{X: 300, Y: 700})
	want.LineTo(vec.Vec2{X: 100, Y: 400})
	want.Close()

	if d := cmp.Diff(want, g.Outline); d != "" {
		t.Errorf("outline differs (-want +got):\n%s", d)
	}
}

// TestDecodeCharStringCloseThenCurve checks that a curve which follows a
// closepath starts a new sub-path at the current point, which the closepath
// left unchanged.
func TestDecodeCharStringCloseThenCurve(t *testing.T) {
	var cs []byte
	cs = append(cs, encInt(0)...)   // sidebearing
	cs = append(cs, encInt(600)...) // width
	cs = append(cs, 0x0d)           // hsbw
	cs = appendPair(cs, 0, 0)
	cs = append(cs, 0x15) // rmoveto
	cs = appendPair(cs, 600, 0)
	cs = append(cs, 0x05) // rlineto
	cs = appendPair(cs, -300, 700)
	cs = append(cs, 0x05) // rlineto
	cs = append(cs, 0x09) // closepath
	cs = appendPair(cs, -100, -100)
	cs = appendPair(cs, -100, -200)
	cs = appendPair(cs, -100, -400)
	cs = append(cs, 0x08) // rrcurveto
	cs = append(cs, 0x0e) // endchar

	info := &decodeInfo{budget: newTestBudget()}
	g := info.decodeCharString(cs, "test")

	// The curve after the closepath starts at (300, 700); its control points
	// and endpoint follow from the three relative pairs.
	want := &path.Data{}
	want.MoveTo(vec.Vec2{X: 0, Y: 0})
	want.LineTo(vec.Vec2{X: 600, Y: 0})
	want.LineTo(vec.Vec2{X: 300, Y: 700})
	want.Close()
	want.MoveTo(vec.Vec2{X: 300, Y: 700})
	want.CubeTo(
		vec.Vec2{X: 200, Y: 600},
		vec.Vec2{X: 100, Y: 400},
		vec.Vec2{X: 0, Y: 0},
	)
	want.Close()

	if d := cmp.Diff(want, g.Outline); d != "" {
		t.Errorf("outline differs (-want +got):\n%s", d)
	}
}

// appendFlex appends a flex sequence to cs.  The sequence consists of othersubr
// 1, seven rmoveto commands each followed by othersubr 2, and othersubr 0.
// The first pair gives the reference point relative to the current point; the
// remaining six give the control points and endpoints of the two curves, each
// relative to its predecessor.  End is the absolute endpoint of the second
// curve, which the trailing setcurrentpoint restores as the current point.
func appendFlex(cs []byte, pairs [7][2]int, end [2]int) []byte {
	cs = appendPair(cs, 0, 1)
	cs = append(cs, 0x0c, 0x10) // callothersubr (flex start)
	for _, p := range pairs {
		cs = appendPair(cs, p[0], p[1])
		cs = append(cs, 0x15) // rmoveto
		cs = appendPair(cs, 0, 2)
		cs = append(cs, 0x0c, 0x10) // callothersubr (flex coordinate pair)
	}
	cs = append(cs, encInt(50)...) // flex height
	cs = appendPair(cs, end[0], end[1])
	cs = appendPair(cs, 3, 0)
	cs = append(cs, 0x0c, 0x10) // callothersubr (flex end)
	cs = append(cs, 0x0c, 0x11) // pop
	cs = append(cs, 0x0c, 0x11) // pop
	cs = append(cs, 0x0c, 0x21) // setcurrentpoint
	return cs
}

// TestDecodeCharStringFlex checks that a flex sequence decodes to the two
// cubic curves it describes, dropping the reference point.  The curves start
// at the current point from before the flex, so after a closepath the flex
// must start a new sub-path there.
func TestDecodeCharStringFlex(t *testing.T) {
	// starting at (100, 0), with reference point (100, 50)
	pairs := [7][2]int{
		{0, 50},   // reference point
		{10, -50}, // (110, 0)
		{10, 10},  // (120, 10)
		{10, 0},   // (130, 10)
		{10, 0},   // (140, 10)
		{10, -10}, // (150, 0)
		{10, 0},   // (160, 0)
	}
	curves := func(d *path.Data) {
		d.CubeTo(
			vec.Vec2{X: 110, Y: 0},
			vec.Vec2{X: 120, Y: 10},
			vec.Vec2{X: 130, Y: 10},
		)
		d.CubeTo(
			vec.Vec2{X: 140, Y: 10},
			vec.Vec2{X: 150, Y: 0},
			vec.Vec2{X: 160, Y: 0},
		)
	}

	prefix := func() []byte {
		var cs []byte
		cs = append(cs, encInt(0)...)   // sidebearing
		cs = append(cs, encInt(600)...) // width
		cs = append(cs, 0x0d)           // hsbw
		cs = appendPair(cs, 0, 0)
		cs = append(cs, 0x15) // rmoveto
		cs = appendPair(cs, 100, 0)
		cs = append(cs, 0x05) // rlineto, current point is now (100, 0)
		return cs
	}

	t.Run("openSubPath", func(t *testing.T) {
		cs := appendFlex(prefix(), pairs, [2]int{160, 0})
		cs = append(cs, 0x0e) // endchar

		info := &decodeInfo{budget: newTestBudget()}
		g := info.decodeCharString(cs, "test")

		// the flex continues the sub-path opened by the rmoveto
		want := &path.Data{}
		want.MoveTo(vec.Vec2{X: 0, Y: 0})
		want.LineTo(vec.Vec2{X: 100, Y: 0})
		curves(want)
		want.Close()

		if d := cmp.Diff(want, g.Outline); d != "" {
			t.Errorf("outline differs (-want +got):\n%s", d)
		}
	})

	t.Run("afterClosePath", func(t *testing.T) {
		cs := prefix()
		cs = append(cs, 0x09) // closepath
		cs = appendFlex(cs, pairs, [2]int{160, 0})
		cs = append(cs, 0x0e) // endchar

		info := &decodeInfo{budget: newTestBudget()}
		g := info.decodeCharString(cs, "test")

		// the flex starts a new sub-path at (100, 0)
		want := &path.Data{}
		want.MoveTo(vec.Vec2{X: 0, Y: 0})
		want.LineTo(vec.Vec2{X: 100, Y: 0})
		want.Close()
		want.MoveTo(vec.Vec2{X: 100, Y: 0})
		curves(want)
		want.Close()

		if d := cmp.Diff(want, g.Outline); d != "" {
			t.Errorf("outline differs (-want +got):\n%s", d)
		}
	})
}

// TestDecodeCharStringRedundantClose checks that closepath commands which do
// not close an open sub-path are ignored, instead of adding stray Close
// commands to the outline.
func TestDecodeCharStringRedundantClose(t *testing.T) {
	prefix := func() []byte {
		var cs []byte
		cs = append(cs, encInt(0)...)   // sidebearing
		cs = append(cs, encInt(600)...) // width
		cs = append(cs, 0x0d)           // hsbw
		return cs
	}

	t.Run("leading", func(t *testing.T) {
		cs := prefix()
		cs = append(cs, 0x09) // closepath
		cs = append(cs, 0x0e) // endchar

		info := &decodeInfo{budget: newTestBudget()}
		g := info.decodeCharString(cs, "test")

		want := &path.Data{}
		if d := cmp.Diff(want, g.Outline); d != "" {
			t.Errorf("outline differs (-want +got):\n%s", d)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		cs := prefix()
		cs = appendPair(cs, 0, 0)
		cs = append(cs, 0x15) // rmoveto
		cs = appendPair(cs, 100, 0)
		cs = append(cs, 0x05) // rlineto
		cs = append(cs, 0x09) // closepath
		cs = append(cs, 0x09) // closepath (redundant)
		cs = append(cs, 0x0e) // endchar

		info := &decodeInfo{budget: newTestBudget()}
		g := info.decodeCharString(cs, "test")

		want := &path.Data{}
		want.MoveTo(vec.Vec2{X: 0, Y: 0})
		want.LineTo(vec.Vec2{X: 100, Y: 0})
		want.Close()
		if d := cmp.Diff(want, g.Outline); d != "" {
			t.Errorf("outline differs (-want +got):\n%s", d)
		}
	})
}

// lineGlyphCharstring builds a charstring drawing a closed path of n line
// segments, so that the size of the resulting outline is known.
func lineGlyphCharstring(n int) []byte {
	var cs []byte
	cs = append(cs, encInt(0)...)   // sidebearing
	cs = append(cs, encInt(500)...) // width
	cs = append(cs, 0x0d)           // hsbw
	cs = append(cs, encInt(10)...)
	cs = append(cs, encInt(20)...)
	cs = append(cs, 0x15) // rmoveto
	for i := range n {
		cs = append(cs, encInt(30+i)...)
		cs = append(cs, encInt(i-40)...)
		cs = append(cs, 0x05) // rlineto
	}
	cs = append(cs, 0x09) // closepath
	cs = append(cs, 0x0e) // endchar
	return cs
}

// TestDecodeCharStringScratchReuse checks that decoding a sequence of glyphs
// through one decodeInfo gives the same outlines as decoding each with a
// fresh one.  The outline scratch buffer is shared between glyphs, so it
// must neither carry commands over from the preceding glyph nor stay aliased
// by an outline which has already been returned.
func TestDecodeCharStringScratchReuse(t *testing.T) {
	long := lineGlyphCharstring(12)
	short := lineGlyphCharstring(1)

	wantLong := (&decodeInfo{budget: newTestBudget()}).decodeCharString(long, "long")
	wantShort := (&decodeInfo{budget: newTestBudget()}).decodeCharString(short, "short")
	if len(wantLong.Outline.Cmds) <= len(wantShort.Outline.Cmds) {
		t.Fatal("test glyphs are not of different sizes")
	}

	// A shorter glyph follows a longer one, so leftover commands would show
	// up, and the longer glyph is then decoded again.
	shared := &decodeInfo{budget: newTestBudget()}
	gotLong := shared.decodeCharString(long, "long")
	gotShort := shared.decodeCharString(short, "short")
	gotLongAgain := shared.decodeCharString(long, "long")

	// The comparisons run only after the last decode, so an outline still
	// pointing into the scratch buffer shows up as a mismatch.
	if diff := cmp.Diff(wantLong, gotLong); diff != "" {
		t.Errorf("first glyph (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantShort, gotShort); diff != "" {
		t.Errorf("shorter following glyph (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantLong, gotLongAgain); diff != "" {
		t.Errorf("repeated glyph (-want +got):\n%s", diff)
	}

	// The returned outlines must not share storage with each other.
	if &gotLong.Outline.Coords[0] == &gotShort.Outline.Coords[0] {
		t.Error("outlines of successive glyphs share their coordinates")
	}
	if &gotLong.Outline.Cmds[0] == &gotLongAgain.Outline.Cmds[0] {
		t.Error("outlines of successive glyphs share their commands")
	}
}

// flexEndCharstring builds a charstring which ends a flex sequence it never
// started, drawing a single line.  The flex end is honoured only if seven
// coordinate pairs have been collected, which this charstring never does.
func flexEndCharstring() []byte {
	var cs []byte
	cs = append(cs, encInt(0)...)   // sidebearing
	cs = append(cs, encInt(500)...) // width
	cs = append(cs, 0x0d)           // hsbw
	cs = appendPair(cs, 10, 20)
	cs = append(cs, 0x15) // rmoveto
	cs = appendPair(cs, 60, 0)
	cs = append(cs, 0x05)          // rlineto
	cs = append(cs, encInt(50)...) // flex height
	cs = appendPair(cs, 70, 20)    // endpoint
	cs = appendPair(cs, 3, 0)      // argument count, othersubr 0
	cs = append(cs, 0x0c, 0x10)    // callothersubr (flex end)
	cs = append(cs, 0x0c, 0x11)    // pop
	cs = append(cs, 0x0c, 0x11)    // pop
	cs = append(cs, 0x0c, 0x21)    // setcurrentpoint
	cs = append(cs, 0x0e)          // endchar
	return cs
}

// TestDecodeCharStringFlexReuse checks that the flex coordinates collected
// for one glyph do not reach the next.  The buffer is shared between the
// glyphs of a font, and a flex end does not require a matching flex start,
// so a stale buffer would draw the preceding glyph's curves here.
func TestDecodeCharStringFlexReuse(t *testing.T) {
	want := (&decodeInfo{budget: newTestBudget()}).decodeCharString(flexEndCharstring(), "flexEnd")

	shared := &decodeInfo{budget: newTestBudget()}
	shared.decodeCharString(flexGlyphCharstring(3), "flex")
	got := shared.decodeCharString(flexEndCharstring(), "flexEnd")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flex data carried over (-want +got):\n%s", diff)
	}
}

// flexGlyphCharstring builds a charstring of the shape a text glyph in a real
// font has: stem hints, a hint replacement, n line segments and a flex.  Hint
// replacement and flex drive the PostScript operand stack, which the plain
// line glyphs leave untouched.
func flexGlyphCharstring(n int) []byte {
	var cs []byte
	cs = append(cs, encInt(0)...)   // sidebearing
	cs = append(cs, encInt(500)...) // width
	cs = append(cs, 0x0d)           // hsbw
	cs = append(cs, encInt(-20)...)
	cs = append(cs, encInt(40)...)
	cs = append(cs, 0x01) // hstem
	cs = append(cs, encInt(0)...)
	cs = append(cs, encInt(60)...)
	cs = append(cs, 0x03) // vstem

	// hint replacement, "subr# 1 3 callothersubr pop callsubr"
	cs = append(cs, encInt(3)...) // subr number
	cs = append(cs, encInt(1)...) // argument count
	cs = append(cs, encInt(3)...) // othersubr number
	cs = append(cs, 0x0c, 0x10)   // callothersubr
	cs = append(cs, 0x0c, 0x11)   // pop
	cs = append(cs, 0x0a)         // callsubr

	x, y := 10, 20
	cs = appendPair(cs, x, y)
	cs = append(cs, 0x15) // rmoveto
	for i := range n {
		dx, dy := 30+i, i-40
		x, y = x+dx, y+dy
		cs = appendPair(cs, dx, dy)
		cs = append(cs, 0x05) // rlineto
	}

	pairs := [7][2]int{{20, 0}, {10, 5}, {10, 10}, {10, 5}, {10, -5}, {10, -10}, {10, -5}}
	for _, p := range pairs {
		x, y = x+p[0], y+p[1]
	}
	cs = appendFlex(cs, pairs, [2]int{x, y})

	cs = append(cs, 0x09) // closepath
	cs = append(cs, 0x0e) // endchar
	return cs
}

// BenchmarkDecodeCharString measures decoding a font-sized set of glyphs
// through a single decodeInfo, the way decodeGlyphs does.  The glyph shapes
// differ in which of the decoder's buffers they use.
func BenchmarkDecodeCharString(b *testing.B) {
	cases := []struct {
		name  string
		build func(n int) []byte
		wv    []float64
	}{
		{name: "lines", build: lineGlyphCharstring},
		{name: "flex", build: flexGlyphCharstring},
		{name: "blend", build: blendGlyphCharstring, wv: benchWeightVector()},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			charstrings := make([][]byte, 200)
			total := 0
			for i := range charstrings {
				charstrings[i] = c.build(10 + i%50)
				total += len(charstrings[i])
			}
			b.SetBytes(int64(total))
			b.ReportAllocs()
			for b.Loop() {
				info := &decodeInfo{
					weightVector: c.wv,
					budget:       membudget.New(64 << 20),
				}
				for _, cs := range charstrings {
					info.decodeCharString(cs, "glyph")
				}
			}
		})
	}
}

// TestBenchmarkGlyphs checks that the charstrings used by the benchmarks
// decode in full.  A malformed one would still be timed, but would measure
// the bail-out path instead of the glyph it describes.
func TestBenchmarkGlyphs(t *testing.T) {
	cases := []struct {
		name     string
		cs       []byte
		wv       []float64
		wantCmds int
	}{
		// moveto, 10 linetos, closepath
		{name: "lines", cs: lineGlyphCharstring(10), wantCmds: 12},
		// as above, plus the two curves of the flex
		{name: "flex", cs: flexGlyphCharstring(10), wantCmds: 14},
		// one moveto per blended value
		{name: "blend", cs: blendGlyphCharstring(0), wv: benchWeightVector(), wantCmds: 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			info := &decodeInfo{weightVector: c.wv, budget: newTestBudget()}
			g := info.decodeCharString(c.cs, c.name)
			if got := len(g.Outline.Cmds); got != c.wantCmds {
				t.Errorf("got %d path commands, want %d", got, c.wantCmds)
			}
		})
	}
}

// benchWeightVector returns the blend weights of a four-master font.
func benchWeightVector() []float64 {
	return []float64{0.4, 0.3, 0.2, 0.1}
}

// blendGlyphCharstring builds a charstring which blends six values across the
// four masters of [benchWeightVector], the widest blend othersubr 18 allows.
// n is ignored; the operand count is fixed by the othersubr.
func blendGlyphCharstring(n int) []byte {
	base, deltas := blendInputs(6, len(benchWeightVector()))
	return buildBlendCharstring(18, base, deltas)
}
