// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"encoding/hex"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var functionPattern = regexp.MustCompile(`^(rgba?|hsla?)\((.*)\)$`)

var argSeparator = regexp.MustCompile(`[\s,/]+`)

var numberPattern = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)$`)

// Parse parses a color string. It accepts a CSS color name or the transparent keyword,
// a hex value of 3, 4, 6, or 8 digits with or without a leading '#', an rgb() or rgba()
// function, and an hsl() or hsla() function, in comma or space syntax with an optional
// alpha. The alpha of a color without one is 255. Parse returns an error wrapping
// validate.ErrValidation for any other string or for a value out of range.
func Parse(s string) (Color, error) {
	t := strings.ToLower(strings.TrimSpace(s))
	if c, ok := names[t]; ok {
		return c, nil
	}
	if m := functionPattern.FindStringSubmatch(t); m != nil {
		c, err := parseFunction(m[1][:3], m[2])
		if err != nil {
			return Color{}, errors.Wrapf(validate.ErrValidation, "%v: %q", err, s)
		}
		return c, nil
	}
	c, err := parseHex(strings.TrimPrefix(t, "#"))
	if err != nil {
		return Color{}, errors.Wrapf(validate.ErrValidation, "%v: %q", err, s)
	}
	return c, nil
}

func parseHex(digits string) (Color, error) {
	if len(digits) == 3 || len(digits) == 4 {
		var b strings.Builder
		for _, d := range digits {
			b.WriteRune(d)
			b.WriteRune(d)
		}
		digits = b.String()
	}
	if len(digits) != 6 && len(digits) != 8 {
		return Color{}, errors.New("invalid color")
	}
	b, err := hex.DecodeString(digits)
	if err != nil {
		return Color{}, errors.New("invalid hex color")
	}
	c := Color{R: b[0], G: b[1], B: b[2], A: 255}
	if len(b) == 4 {
		c.A = b[3]
	}
	return c, nil
}

func parseFunction(name, body string) (Color, error) {
	args := strings.Fields(argSeparator.ReplaceAllString(body, " "))
	if len(args) != 3 && len(args) != 4 {
		return Color{}, errors.Newf("%s() takes 3 or 4 values", name)
	}
	a := uint8(255)
	if len(args) == 4 {
		v, err := parseNumber(args[3], 1)
		if err != nil || v < 0 || v > 1 {
			return Color{}, errors.Newf("%s() alpha must be 0-1", name)
		}
		a = uint8(math.Round(v * 255))
	}
	if name == "rgb" {
		var ch [3]uint8
		for i, arg := range args[:3] {
			v, err := parseNumber(arg, 255)
			if err != nil || v < 0 || v > 255 {
				return Color{}, errors.New("rgb() channels must be 0-255")
			}
			ch[i] = uint8(math.Round(v))
		}
		return Color{R: ch[0], G: ch[1], B: ch[2], A: a}, nil
	}
	h, err := parseDecimal(strings.TrimSuffix(args[0], "deg"))
	if err != nil {
		return Color{}, errors.New("invalid hsl() hue")
	}
	sat, err := parseNumber(strings.TrimSuffix(args[1], "%"), 100)
	if err != nil || sat < 0 || sat > 100 {
		return Color{}, errors.New("hsl() saturation must be 0-100")
	}
	l, err := parseNumber(strings.TrimSuffix(args[2], "%"), 100)
	if err != nil || l < 0 || l > 100 {
		return Color{}, errors.New("hsl() lightness must be 0-100")
	}
	r, g, b := hslToRGB(h, sat/100, l/100)
	return Color{R: r, G: g, B: b, A: a}, nil
}

// parseNumber parses a number, reading a percentage as a fraction of scale.
func parseNumber(arg string, scale float64) (float64, error) {
	if p, ok := strings.CutSuffix(arg, "%"); ok {
		v, err := parseDecimal(p)
		return v * scale / 100, err
	}
	return parseDecimal(arg)
}

// parseDecimal parses a plain decimal number, rejecting forms such as NaN, Inf, and
// exponents that strconv.ParseFloat accepts.
func parseDecimal(s string) (float64, error) {
	if !numberPattern.MatchString(s) {
		return 0, errors.New("invalid number")
	}
	return strconv.ParseFloat(s, 64)
}

// hslToRGB converts a hue in degrees and 0-1 saturation and lightness to 0-255
// channels. The hue wraps, as it does in CSS.
func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	toChannel := func(v float64) uint8 { return uint8(math.Round((v + m) * 255)) }
	return toChannel(r), toChannel(g), toChannel(b)
}
