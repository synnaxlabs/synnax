// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import "github.com/synnaxlabs/x/color/versions"

// Parse parses a color string: a CSS color name or the transparent keyword, a hex
// value of 3, 4, 6, or 8 digits with or without a leading '#', an rgb() or rgba()
// function, or an hsl() or hsla() function. The alpha of a color without one is 255.
// Parse returns an error wrapping validate.ErrValidation for any other string or for a
// value out of range.
func Parse(s string) (Color, error) { return versions.Parse(s) }

// FromUint32 unpacks a color from its 0xRRGGBBAA form.
func FromUint32(v uint32) Color { return versions.FromUint32(v) }
