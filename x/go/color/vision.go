// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import "math"

// Deficiency is a full (dichromatic) color vision deficiency.
type Deficiency uint8

const (
	// protanopia is the absence of red cones.
	protanopia Deficiency = iota
	// deuteranopia is the absence of green cones.
	deuteranopia
	// tritanopia is the absence of blue cones.
	tritanopia
)

// Deficiencies holds each Deficiency that Simulate can simulate: protanopia,
// deuteranopia, and tritanopia, in that order.
var Deficiencies = [...]Deficiency{protanopia, deuteranopia, tritanopia}

// Machado, Oliveira, and Fernandes (2009), Table 1, severity 1.0, for linear RGB.
var deficiencyMatrices = [...][3][3]float64{
	protanopia: {
		{0.152286, 1.052583, -0.204868},
		{0.114503, 0.786281, 0.099216},
		{-0.003882, -0.048116, 1.051998},
	},
	deuteranopia: {
		{0.367322, 0.860646, -0.227968},
		{0.280085, 0.672501, 0.047413},
		{-0.01182, 0.04294, 0.968881},
	},
	tritanopia: {
		{1.255528, -0.076749, -0.178779},
		{-0.078411, 0.930809, 0.147602},
		{0.004733, 0.691367, 0.3039},
	},
}

func toLinear(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// fromLinear converts a linear-light value to an sRGB byte, clamping it into gamut.
func fromLinear(v float64) uint8 {
	c := 12.92 * v
	if v > 0.0031308 {
		c = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(math.Round(min(1, max(0, c)) * 255))
}

// Simulate returns c as a person with deficiency d sees it. The alpha does not change.
func Simulate(c Color, d Deficiency) Color {
	rgb := [3]float64{toLinear(c.R), toLinear(c.G), toLinear(c.B)}
	var out [3]uint8
	for i, row := range deficiencyMatrices[d] {
		out[i] = fromLinear(row[0]*rgb[0] + row[1]*rgb[1] + row[2]*rgb[2])
	}
	return Color{R: out[0], G: out[1], B: out[2], A: c.A}
}

func oklab(c Color) [3]float64 {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return [3]float64{
		0.2104542553*l + 0.793617785*m - 0.0040720468*s,
		1.9779984951*l - 2.428592205*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.808675766*s,
	}
}

// Distance returns the perceptual distance between a and b: the Euclidean distance
// between them in OKLab. Alpha is ignored. Black and white are 1 apart.
func Distance(a, b Color) float64 {
	x, y := oklab(a), oklab(b)
	dl, da, db := x[0]-y[0], x[1]-y[1], x[2]-y[2]
	return math.Sqrt(dl*dl + da*da + db*db)
}
