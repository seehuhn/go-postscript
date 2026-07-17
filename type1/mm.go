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
	"math"

	"seehuhn.de/go/postscript"
)

// MMInfo holds the multiple master data of a Type 1 font.
//
// A multiple master font describes a design space with one to four axes.
// Every point in that space is a weighted average ("blend") of a small set
// of master designs.  MMInfo records the axis descriptions, the master
// positions, the weight vector of the font's default instance and the
// per-master values needed to derive an interpolated instance.
type MMInfo struct {
	// Axes describes the design axes.  Its length n is between 1 and 4.
	Axes []MMAxis

	// Masters holds the normalized design-space position of each master.
	// Its length k equals the number of masters; each entry has length n.
	Masters [][]float64

	// WeightVector holds the blend weight of each master for the font's
	// default instance.  Its length k is between 2 and 16.
	WeightVector []float64

	// Blend holds the per-master values of the blended font entries.
	// It may be nil.
	Blend *MMBlend

	// deobfuscated charstrings and subrs, retained for instantiation.
	charstrings map[string][]byte
	subrs       [][]byte
}

// MMAxis describes a single design axis of a multiple master font.
type MMAxis struct {
	// Name is the axis type, for example "Weight" or "Width".
	Name string

	// Map converts design coordinates to normalized coordinates.
	// The points are in ascending order of their Design value and there
	// are at least two of them.
	Map []MMMapPoint
}

// MMMapPoint is one point of an axis' design-to-normalized map.
type MMMapPoint struct {
	Design, Normalized float64
}

// MMBlend holds the per-master values of the blended font entries.
// For the two-dimensional fields the outer index selects the element and
// the inner index selects the master, so each inner slice has length k.
type MMBlend struct {
	// FontBBox holds the four bounding-box coordinates, or nil.
	FontBBox [][]float64

	// BlueValues, OtherBlues, StemSnapH and StemSnapV hold the per-master
	// values of the corresponding Private dictionary entries, or nil.
	BlueValues [][]float64
	OtherBlues [][]float64
	StemSnapH  [][]float64
	StemSnapV  [][]float64

	// StdHW and StdVW hold k per-master values, or nil.
	StdHW []float64
	StdVW []float64

	// UnderlinePosition, UnderlineThickness and ItalicAngle hold the
	// per-master values of the corresponding FontInfo entries, or nil.
	UnderlinePosition  []float64
	UnderlineThickness []float64
	ItalicAngle        []float64
}

// MM data caps, guarding against hostile fonts.
const (
	mmMinMasters   = 2
	mmMaxMasters   = 16
	mmMinAxes      = 1
	mmMaxAxes      = 4
	mmMaxMapPoints = 12
)

// readMMInfo extracts the multiple master data of a font.  It returns nil
// for an ordinary (single master) font.  Reading is permissive: malformed
// blend data degrades to a font that parses but is not instantiable rather
// than failing the whole font.
func readMMInfo(fd, fontInfo postscript.Dict) *MMInfo {
	wv := readNumberArray(fd["WeightVector"])
	k := len(wv)
	if k < mmMinMasters || k > mmMaxMasters {
		return nil
	}
	for _, v := range wv {
		if !isFinite(v) {
			return nil
		}
	}

	mm := &MMInfo{WeightVector: wv}

	axes, masters, ok := readAxesAndMasters(fontInfo, k)
	if !ok {
		// keep WeightVector for the charstring decoder, but the font is
		// not instantiable
		return mm
	}
	mm.Axes = axes
	mm.Masters = masters
	mm.Blend = readMMBlend(fd["Blend"], k)
	return mm
}

// readAxesAndMasters parses /BlendAxisTypes, /BlendDesignPositions and
// /BlendDesignMap.  It returns ok == false on any inconsistency.
func readAxesAndMasters(fontInfo postscript.Dict, k int) ([]MMAxis, [][]float64, bool) {
	axisTypes, _ := fontInfo["BlendAxisTypes"].(postscript.Array)
	n := len(axisTypes)
	if n < mmMinAxes || n > mmMaxAxes {
		return nil, nil, false
	}
	names := make([]string, n)
	for i, v := range axisTypes {
		name, ok := v.(postscript.Name)
		if !ok {
			return nil, nil, false
		}
		names[i] = string(name)
	}

	positions, _ := fontInfo["BlendDesignPositions"].(postscript.Array)
	if len(positions) != k {
		return nil, nil, false
	}
	masters := make([][]float64, k)
	for i, rowObj := range positions {
		row := readNumberArray(rowObj)
		if len(row) != n {
			return nil, nil, false
		}
		for _, v := range row {
			if !isFinite(v) {
				return nil, nil, false
			}
		}
		masters[i] = row
	}

	designMap, _ := fontInfo["BlendDesignMap"].(postscript.Array)
	if len(designMap) != n {
		return nil, nil, false
	}
	axes := make([]MMAxis, n)
	for i, axisObj := range designMap {
		seg, ok := axisObj.(postscript.Array)
		if !ok {
			return nil, nil, false
		}
		if len(seg) > mmMaxMapPoints {
			seg = seg[:mmMaxMapPoints]
		}
		if len(seg) < 2 {
			return nil, nil, false
		}
		points := make([]MMMapPoint, len(seg))
		for j, pointObj := range seg {
			pair := readNumberArray(pointObj)
			if len(pair) != 2 || !isFinite(pair[0]) || !isFinite(pair[1]) {
				return nil, nil, false
			}
			if j > 0 && pair[0] <= points[j-1].Design {
				return nil, nil, false
			}
			points[j] = MMMapPoint{Design: pair[0], Normalized: pair[1]}
		}
		axes[i] = MMAxis{Name: names[i], Map: points}
	}

	return axes, masters, true
}

// readMMBlend parses the /Blend dictionary.  Each entry is optional and a
// malformed entry is dropped silently.  It returns nil if no /Blend
// dictionary is present.
func readMMBlend(obj postscript.Object, k int) *MMBlend {
	blend, ok := obj.(postscript.Dict)
	if !ok {
		return nil
	}
	res := &MMBlend{}

	if bbox := readBlend2D(blend["FontBBox"], k); len(bbox) == 4 {
		res.FontBBox = bbox
	}

	if priv, ok := blend["Private"].(postscript.Dict); ok {
		res.BlueValues = readBlend2D(priv["BlueValues"], k)
		res.OtherBlues = readBlend2D(priv["OtherBlues"], k)
		res.StemSnapH = readBlend2D(priv["StemSnapH"], k)
		res.StemSnapV = readBlend2D(priv["StemSnapV"], k)
		res.StdHW = readBlendScalar(priv["StdHW"], k)
		res.StdVW = readBlendScalar(priv["StdVW"], k)
	}

	if fi, ok := blend["FontInfo"].(postscript.Dict); ok {
		res.UnderlinePosition = readBlend1D(fi["UnderlinePosition"], k)
		res.UnderlineThickness = readBlend1D(fi["UnderlineThickness"], k)
		res.ItalicAngle = readBlend1D(fi["ItalicAngle"], k)
	}

	return res
}

// readNumberArray returns the elements of a PostScript number array as
// float64 values, or nil if obj is not an array of numbers.
func readNumberArray(obj postscript.Object) []float64 {
	arr, ok := obj.(postscript.Array)
	if !ok {
		return nil
	}
	res := make([]float64, len(arr))
	for i, v := range arr {
		f, ok := getReal(v)
		if !ok {
			return nil
		}
		res[i] = f
	}
	return res
}

// readBlend2D parses an element-major array of per-master values, where
// each element is an array of k finite numbers.  It returns nil on any
// deviation from that shape.
func readBlend2D(obj postscript.Object, k int) [][]float64 {
	arr, ok := obj.(postscript.Array)
	if !ok || len(arr) == 0 {
		return nil
	}
	res := make([][]float64, len(arr))
	for i, rowObj := range arr {
		row := readBlend1D(rowObj, k)
		if row == nil {
			return nil
		}
		res[i] = row
	}
	return res
}

// readBlend1D parses an array of exactly k finite numbers, returning nil
// on any deviation.
func readBlend1D(obj postscript.Object, k int) []float64 {
	row := readNumberArray(obj)
	if len(row) != k {
		return nil
	}
	for _, v := range row {
		if !isFinite(v) {
			return nil
		}
	}
	return row
}

// readBlendScalar parses the one-element-array form of a blended scalar,
// [[v1 ... vk]], into k per-master values.  It returns nil on any deviation.
func readBlendScalar(obj postscript.Object, k int) []float64 {
	arr, ok := obj.(postscript.Array)
	if !ok || len(arr) != 1 {
		return nil
	}
	return readBlend1D(arr[0], k)
}

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}
