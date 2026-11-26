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
	"encoding/json/jsontext"
	"fmt"
	"math"

	xjson "github.com/synnaxlabs/x/json"
	xmath "github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/types"
)

// CheckDefault returns an error wrapping validate.ErrValidation when f has a default
// that it cannot hold: an infinite float, text that is not base64 or not valid JSON, or
// a number that decodes outside the min and max of f. The error does not name the
// default.
func CheckDefault(f JSONField) error {
	return problemError(defaultProblem(f))
}

func defaultProblem(f JSONField) (string, bool) {
	switch v := f.Variant.(type) {
	case FloatJSONField:
		if v.Default != nil && math.IsInf(float64(*v.Default), 0) {
			return "cannot be infinite", true
		}
		return defaultRangeProblem(v.BaseField, v.Default)
	case IntegerJSONField:
		return defaultRangeProblem(v.BaseField, v.Default)
	case UnsignedJSONField:
		return defaultRangeProblem(v.BaseField, v.Default)
	case BytesJSONField:
		if v.Default != nil {
			if _, err := xjson.DecodeBase64(*v.Default); err != nil {
				return "must be base64", true
			}
		}
	case JSONJSONField:
		if v.Default != nil && !jsontext.Value(*v.Default).IsValid() {
			return "must be valid JSON", true
		}
	}
	return "", false
}

// CheckBinaryDefault returns an error wrapping validate.ErrValidation when f, a field
// of a valid bit length, has a default that it cannot hold: an infinite float, a value
// that its bits cannot hold, or a number that decodes outside the min and max of f. The
// error does not name the default.
func CheckBinaryDefault(f BinaryField) error {
	return problemError(binaryDefaultProblem(f))
}

// CheckTextDefault returns an error wrapping validate.ErrValidation when the default of
// f is infinite or decodes outside the min and max of f. The error does not name the
// default.
func CheckTextDefault(f TextField) error {
	return problemError(textDefaultProblem(f))
}

func binaryDefaultProblem(f BinaryField) (string, bool) {
	n := f.Base().BitLength
	switch v := f.Variant.(type) {
	case FloatBinaryField:
		if v.Default == nil {
			return "", false
		}
		d := float64(*v.Default)
		switch {
		case math.IsInf(d, 0):
			return "cannot be infinite", true
		case n == 32 && math.Abs(d) > math.MaxFloat32:
			return fmt.Sprintf("%g does not fit a 32-bit float", d), true
		}
		return defaultRangeProblem(v.BaseField, v.Default)
	case IntegerBinaryField:
		if v.Default != nil && !fits(int64(*v.Default), f) {
			return fmt.Sprintf("%d does not fit the field", *v.Default), true
		}
		return defaultRangeProblem(v.BaseField, v.Default)
	case UnsignedBinaryField:
		if v.Default != nil && n < 64 && uint64(*v.Default) >= 1<<n {
			return fmt.Sprintf("%d does not fit the field", *v.Default), true
		}
		return defaultRangeProblem(v.BaseField, v.Default)
	}
	return "", false
}

func textDefaultProblem(f TextField) (string, bool) {
	b := f.Base()
	if b.Default != nil && math.IsInf(float64(*b.Default), 0) {
		return "cannot be infinite", true
	}
	return defaultRangeProblem(b.BaseField, b.Default)
}

// defaultRangeProblem returns how the finite default d of a number field with base b
// decodes outside its min and max, and false when it does not.
func defaultRangeProblem[N types.Numeric](b BaseField, d *N) (string, bool) {
	if d == nil || !xmath.Finite(float64(*d)) {
		return "", false
	}
	return outOfRawRange(b, float64(*d))
}
