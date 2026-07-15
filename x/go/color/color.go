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
	latest "github.com/synnaxlabs/x/color/types/v0"
)

// Schema is a zyn schema for parsing a color.
var Schema = latest.Schema

// FromHex parses a hex color string into a Color. Supports 6 or 8 character hex
// strings with or without a leading '#'.
func FromHex(s string) (Color, error) { return latest.FromHex(s) }

// MustFromHex parses a hex color string into a Color, panicking on error.
func MustFromHex(s string) Color { return latest.MustFromHex(s) }

// FromCSS parses a CSS-style color string: a hex value with a leading '#' or an
// rgb()/rgba() function. Unlike FromHex, the leading '#' is required.
func FromCSS(s string) (Color, error) { return latest.FromCSS(s) }
