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
	"regexp"
	"strconv"
	"strings"

	v0 "github.com/synnaxlabs/x/color/types/v0"
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
var FromHex = v0.FromHex

// MustFromHex parses a hex color string into a Color, panicking on error.
func MustFromHex(s string) Color {
	c, err := FromHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// rgbPattern matches an rgb()/rgba() string with three 0-255 channels and an
// optional alpha.
var rgbPattern = regexp.MustCompile(
	`^(rgba?)\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(?:,\s*([0-9.]+)\s*)?\)$`,
)

// FromCSS parses a CSS-style color string: a hex value with a leading '#' or an
// rgb()/rgba() function. Unlike FromHex, the leading '#' is required.
func FromCSS(s string) (Color, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		return FromHex(s)
	}
	if m := rgbPattern.FindStringSubmatch(s); m != nil {
		return fromRGBMatch(m, s)
	}
	return Color{}, errors.Wrapf(
		validate.ErrValidation,
		"color must be a hex value (e.g. \"#3bc454\") or rgb(r,g,b): %q", s,
	)
}

func fromRGBMatch(m []string, s string) (Color, error) {
	hasAlpha := m[5] != ""
	if m[1] == "rgb" && hasAlpha {
		return Color{}, errors.Wrapf(validate.ErrValidation, "rgb() takes 3 channels; use rgba() for alpha: %q", s)
	}
	if m[1] == "rgba" && !hasAlpha {
		return Color{}, errors.Wrapf(validate.ErrValidation, "rgba() requires a 4th alpha channel: %q", s)
	}
	r, err := strconv.ParseUint(m[2], 10, 8)
	if err != nil {
		return Color{}, errors.Wrapf(validate.ErrValidation, "rgb channels must be 0-255: %q", s)
	}
	g, err := strconv.ParseUint(m[3], 10, 8)
	if err != nil {
		return Color{}, errors.Wrapf(validate.ErrValidation, "rgb channels must be 0-255: %q", s)
	}
	b, err := strconv.ParseUint(m[4], 10, 8)
	if err != nil {
		return Color{}, errors.Wrapf(validate.ErrValidation, "rgb channels must be 0-255: %q", s)
	}
	a := 1.0
	if hasAlpha {
		if a, err = strconv.ParseFloat(m[5], 64); err != nil {
			return Color{}, errors.Wrapf(validate.ErrValidation, "invalid rgb alpha: %q", s)
		}
		if a < 0 || a > 1 {
			return Color{}, errors.Wrapf(validate.ErrValidation, "rgba alpha must be 0-1: %q", s)
		}
	}
	return Color{R: uint8(r), G: uint8(g), B: uint8(b), A: a}, nil
}
