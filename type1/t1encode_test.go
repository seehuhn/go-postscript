// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package type1

import (
	"bytes"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"

	"seehuhn.de/go/geom/path"
)

func TestMoveTo(t *testing.T) {
	baseX := 2.0
	baseY := 3.0
	type testCase struct {
		dx, dy float64
	}
	cases := []testCase{
		{0, 0},
		{0, 11},
		{13, 0},
		{17, 19},
	}
	for _, c := range cases {
		g1 := &Glyph{}
		g1.MoveTo(baseX, baseY)
		g1.MoveTo(baseX+c.dx, baseY+c.dy)
		buf := g1.encodeCharString()

		ctx := &decodeInfo{budget: newTestBudget()}
		g2 := ctx.decodeCharString(buf, "test")

		if d := cmp.Diff(g1, g2); d != "" {
			t.Errorf("unexpected diff: %s", d)
		}
	}
}

func TestLineTo(t *testing.T) {
	baseX := 2.0
	baseY := 3.0
	type testCase struct {
		dx, dy float64
	}
	cases := []testCase{
		{0, 0},
		{0, 11},
		{13, 0},
		{17, -19},
	}
	for _, c := range cases {
		g1 := &Glyph{}
		g1.MoveTo(baseX, baseY)
		g1.LineTo(baseX+c.dx, baseY+c.dy)
		g1.ClosePath()
		buf := g1.encodeCharString()

		ctx := &decodeInfo{budget: newTestBudget()}
		g2 := ctx.decodeCharString(buf, "test")

		if d := cmp.Diff(g1, g2); d != "" {
			t.Errorf("unexpected diff: %s", d)
		}
	}
}

func TestCurveTo(t *testing.T) {
	baseX := 2.0
	baseY := 3.0
	type testCase struct {
		a, b, c, d, e, f float64
	}
	cases := []testCase{
		{0, 0, 0, 0, 0, 0},
		{11, 0, 13, 17, 13, 19},
		{0, 20, 30, 60, 50, 60},
		{10, 20, 30, 40, 50, 60},
	}
	for _, c := range cases {
		g1 := &Glyph{}
		g1.MoveTo(baseX, baseY)
		g1.CurveTo(baseX+c.a, baseY+c.b, baseX+c.c, baseY+c.d, baseX+c.e, baseY+c.f)
		g1.ClosePath()
		buf := g1.encodeCharString()

		ctx := &decodeInfo{budget: newTestBudget()}
		g2 := ctx.decodeCharString(buf, "test")

		if d := cmp.Diff(g1, g2); d != "" {
			t.Errorf("unexpected diff: %s", d)
		}
	}
}

func TestAppendInt(t *testing.T) {
	for _, x := range []int32{0, 1, -1, 2, -2, 107, -107, 108, -108, 1131, -1131, 1132, -1132, 70000, -70000, 500_000_000, -500_000_000} {
		var buf []byte
		buf = appendInt(buf, 0)
		buf = appendInt(buf, 0)
		buf = appendOp(buf, t1hsbw)
		// divide, so that large values give a usable coordinate
		buf = appendInt(buf, x)
		buf = appendInt(buf, gridScale)
		buf = appendOp(buf, t1div)
		buf = appendOp(buf, t1hmoveto)
		buf = appendOp(buf, t1endchar)
		ctx := &decodeInfo{budget: newTestBudget()}
		g := ctx.decodeCharString(buf, "test")
		if g.Outline == nil || len(g.Outline.Cmds) != 1 || g.Outline.Cmds[0] != path.CmdMoveTo || len(g.Outline.Coords) != 1 {
			t.Fatalf("test is broken")
		}
		if got := g.Outline.Coords[0].X * gridScale; got != float64(x) {
			t.Errorf("x=%g, want %d", got, x)
		}
	}
}

func TestAppendNumber(t *testing.T) {
	values := []float64{0, 1, -1, 2, -2, 0.1, 0.5, -0.5, 2.5, 1.0 / 3, math.Pi,
		1.0 / gridScale, -1.0 / gridScale, 31999, -31999,
		maxOperand - 1.0/gridScale, -maxOperand + 1.0/gridScale}
	for _, v := range values {
		x := snap(v)
		var buf []byte
		buf = appendInt(buf, 0)
		buf = appendInt(buf, 0)
		buf = appendOp(buf, t1hsbw)
		buf = appendNumber(buf, x)
		buf = appendOp(buf, t1hmoveto)
		buf = appendOp(buf, t1endchar)

		ctx := &decodeInfo{budget: newTestBudget()}
		g := ctx.decodeCharString(buf, "test")
		if g.Outline == nil || len(g.Outline.Cmds) != 1 || g.Outline.Cmds[0] != path.CmdMoveTo || len(g.Outline.Coords) != 1 {
			t.Fatalf("x=%g: no outline", x)
		}
		if got := g.Outline.Coords[0].X; got != x {
			t.Errorf("x=%g: got %g", x, got)
		}
	}
}

// TestAppendNumberShortest checks that a fraction is written with the
// smallest power-of-two denominator.
func TestAppendNumberShortest(t *testing.T) {
	got := appendNumber(nil, 10.5)
	want := appendInt(appendInt(nil, 21), 2)
	want = appendOp(want, t1div)
	if !bytes.Equal(got, want) {
		t.Errorf("got % x, want % x", got, want)
	}
}

// TestAppendNumberDivisor checks that the divisor of every div written by
// appendNumber is a small number, i.e. at most 32000 in absolute value.
func TestAppendNumberDivisor(t *testing.T) {
	div := t1div.Bytes()
	for _, k := range []int64{1, -1, 3, -3, 12345, 16383, 16385,
		maxOperand*gridScale - 1, -maxOperand*gridScale + 1} {
		x := float64(k) / gridScale
		buf := appendNumber(nil, x)
		if !bytes.HasSuffix(buf, div) {
			t.Fatalf("x=%g: no div operator", x)
		}
		ops := decodeInts(t, buf[:len(buf)-len(div)])
		if len(ops) != 2 {
			t.Fatalf("x=%g: got %d operands, want 2", x, len(ops))
		}
		if q := ops[1]; q < -maxOperand || q > maxOperand {
			t.Errorf("x=%g: divisor %d is too large", x, q)
		}
	}
}

// decodeInts decodes a sequence of charstring integers.
func decodeInts(t *testing.T, cs []byte) []int32 {
	t.Helper()
	var res []int32
	for len(cs) > 0 {
		v := cs[0]
		switch {
		case v >= 32 && v <= 246:
			res = append(res, int32(v)-139)
			cs = cs[1:]
		case v >= 247 && v <= 250 && len(cs) >= 2:
			res = append(res, (int32(v)-247)*256+int32(cs[1])+108)
			cs = cs[2:]
		case v >= 251 && v <= 254 && len(cs) >= 2:
			res = append(res, -(int32(v)-251)*256-int32(cs[1])-108)
			cs = cs[2:]
		case v == 255 && len(cs) >= 5:
			res = append(res, int32(uint32(cs[1])<<24|uint32(cs[2])<<16|uint32(cs[3])<<8|uint32(cs[4])))
			cs = cs[5:]
		default:
			t.Fatalf("unexpected charstring byte %d", v)
		}
	}
	return res
}

// TestEncodeSbw checks that a glyph with a vertical width component, which
// is written using sbw, survives a write/read cycle.
func TestEncodeSbw(t *testing.T) {
	g1 := &Glyph{WidthX: 500.5, WidthY: -100.25}
	g1.MoveTo(10, 20)
	g1.LineTo(400.5, 20)
	g1.ClosePath()

	buf := g1.encodeCharString()
	g2 := (&decodeInfo{budget: newTestBudget()}).decodeCharString(buf, "test")
	if d := cmp.Diff(g1, g2); d != "" {
		t.Errorf("round trip failed (-want +got):\n%s", d)
	}
}
