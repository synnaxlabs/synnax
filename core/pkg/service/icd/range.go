// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"math"

	"github.com/synnaxlabs/x/errors"
	xstrconv "github.com/synnaxlabs/x/strconv"
	"github.com/synnaxlabs/x/validate"
)

// decodeRounding bounds the float64 rounding of raw × scale + offset, relative to
// |raw × scale| + |offset|.
const decodeRounding = 4 * 0x1p-52

// CheckRange returns an error wrapping validate.ErrValidation when v, a value after the
// scale and offset of b, is outside the min and max of b, or is NaN and b has a min or
// a max.
func CheckRange(b BaseField, v float64) error {
	return problemError(outOfRange(b, v, v, 0))
}

// CheckRawRange returns an error wrapping validate.ErrValidation when raw, a value
// before the scale and offset of b, decodes outside the min and max of b, or decodes to
// NaN and b has a min or a max. It allows the float64 rounding of the decode, so a
// value that decodes to a limit passes.
func CheckRawRange(b BaseField, raw float64) error {
	return problemError(outOfRawRange(b, raw))
}

// outOfRawRange returns how raw, a value before the scale and offset of b, decodes
// outside the min and max of b, and false when it does not. It allows the float64
// rounding of the decode, so a value at a limit passes.
func outOfRawRange(b BaseField, raw float64) (string, bool) {
	scaled := raw * b.Scale
	decoded := scaled + b.Offset
	slack := decodeRounding * (math.Abs(scaled) + math.Abs(b.Offset))
	// The slack of an infinite decode is infinite, which would pass every limit.
	if math.IsInf(decoded, 0) {
		slack = 0
	}
	return outOfRange(b, raw, decoded, slack)
}

// outOfRange returns how decoded, the decoded value of raw, is more than slack outside
// the min and max of b or is NaN when b has a limit, and false when it is neither.
func outOfRange(b BaseField, raw, decoded, slack float64) (string, bool) {
	if math.IsNaN(decoded) && (b.Min != nil || b.Max != nil) {
		return "NaN is outside the limits", true
	}
	var (
		side  string
		limit float64
	)
	switch {
	case b.Min != nil && decoded < *b.Min-slack:
		side, limit = "below the min", *b.Min
	case b.Max != nil && decoded > *b.Max+slack:
		side, limit = "above the max", *b.Max
	default:
		return "", false
	}
	text := func(v float64) string { return xstrconv.FormatFloat(v, 64) }
	if raw == decoded {
		return text(decoded) + " is " + side + " " + text(limit), true
	}
	return text(raw) + " decodes to " + text(decoded) + ", which is " + side + " " +
		text(limit), true
}

func problemError(message string, found bool) error {
	if !found {
		return nil
	}
	return errors.Wrap(validate.ErrValidation, message)
}
