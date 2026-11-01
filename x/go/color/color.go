// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/color/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// Schema is a zyn schema for parsing a color.
var Schema = zyn.Object(map[string]zyn.Schema{
	"r": zyn.Number().Uint8().Coerce(),
	"g": zyn.Number().Uint8().Coerce(),
	"b": zyn.Number().Uint8().Coerce(),
	"a": zyn.Number().Float64().Coerce(),
})

// FromHex parses a hex color string into a Color. Supports 6 or 8 character hex
// strings with or without a leading '#'.
func FromHex(s string) (Color, error) { return versions.FromHex(s) }

// rgbPattern matches an rgb()/rgba() string with three 0-255 channels and an
// optional alpha.
var rgbPattern = regexp.MustCompile(
	`^(rgba?)\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(?:,\s*([0-9.]+)\s*)?\)$`,
)

// hslPattern matches hsl(h, s%, l%) or hsla(h, s%, l%, a).
var hslPattern = regexp.MustCompile(
	`^(hsla?)\(\s*(-?[0-9.]+)\s*,\s*([0-9.]+)%\s*,\s*([0-9.]+)%` +
		`\s*(?:,\s*([0-9.]+)\s*)?\)$`,
)

// FromCSS parses a CSS-style color string: a hex value with a leading '#', an
// rgb()/rgba() function, or an hsl()/hsla() function. Unlike FromHex, the leading
// '#' is required.
func FromCSS(s string) (Color, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		return FromHex(s)
	}
	if m := rgbPattern.FindStringSubmatch(s); m != nil {
		return fromRGBMatch(m, s)
	}
	if m := hslPattern.FindStringSubmatch(s); m != nil {
		return fromHSLMatch(m, s)
	}
	return Color{}, errors.Wrapf(
		validate.ErrValidation,
		"color must be a hex value (e.g. \"#3bc454\"), rgb(r,g,b), or hsl(h,s%%,l%%): %q",
		s,
	)
}

// parseAlpha validates the fn/fna pairing against the presence of a 4th channel
// and parses it. fn is the base function name ("rgb" or "hsl").
func parseAlpha(fn, name, raw, s string) (float64, error) {
	hasAlpha := raw != ""
	if name == fn && hasAlpha {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"%s() takes 3 channels; use %sa() for alpha: %q",
			fn, fn, s,
		)
	}
	if name != fn && !hasAlpha {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"%sa() requires a 4th alpha channel: %q",
			fn, s,
		)
	}
	if !hasAlpha {
		return 1, nil
	}
	a, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, errors.Wrapf(validate.ErrValidation, "invalid %s alpha: %q", fn, s)
	}
	if a < 0 || a > 1 {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"%sa alpha must be 0-1: %q",
			fn, s,
		)
	}
	return a, nil
}

func fromRGBMatch(m []string, s string) (Color, error) {
	a, err := parseAlpha("rgb", m[1], m[5], s)
	if err != nil {
		return Color{}, err
	}
	r, err := strconv.ParseUint(m[2], 10, 8)
	if err != nil {
		return Color{}, errors.Wrapf(
			validate.ErrValidation,
			"rgb channels must be 0-255: %q",
			s,
		)
	}
	g, err := strconv.ParseUint(m[3], 10, 8)
	if err != nil {
		return Color{}, errors.Wrapf(
			validate.ErrValidation,
			"rgb channels must be 0-255: %q",
			s,
		)
	}
	b, err := strconv.ParseUint(m[4], 10, 8)
	if err != nil {
		return Color{}, errors.Wrapf(
			validate.ErrValidation,
			"rgb channels must be 0-255: %q",
			s,
		)
	}
	return Color{R: uint8(r), G: uint8(g), B: uint8(b), A: a}, nil
}

func fromHSLMatch(m []string, s string) (Color, error) {
	a, err := parseAlpha("hsl", m[1], m[5], s)
	if err != nil {
		return Color{}, err
	}
	h, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return Color{}, errors.Wrapf(validate.ErrValidation, "invalid hsl hue: %q", s)
	}
	sat, err := parsePercent(m[3])
	if err != nil {
		return Color{}, errors.Wrapf(
			validate.ErrValidation,
			"hsl saturation must be 0-100%%: %q",
			s,
		)
	}
	l, err := parsePercent(m[4])
	if err != nil {
		return Color{}, errors.Wrapf(
			validate.ErrValidation,
			"hsl lightness must be 0-100%%: %q",
			s,
		)
	}
	r, g, b := hslToRGB(h, sat, l)
	return Color{R: r, G: g, B: b, A: a}, nil
}

// parsePercent parses a 0-100 percentage into a 0-1 fraction.
func parsePercent(raw string) (float64, error) {
	p, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	if p < 0 || p > 100 {
		return 0, errors.New("out of range")
	}
	return p / 100, nil
}

// hslToRGB converts a hue in degrees and 0-1 saturation and lightness fractions to
// 0-255 channels. The hue wraps, as it does in CSS.
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
