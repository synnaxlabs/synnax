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

	xjson "github.com/synnaxlabs/x/json"
	xmath "github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/types"
)

// CheckDefault returns an error wrapping validate.ErrValidation when f has a default
// that it cannot hold: a float that is not finite, text that is not base64 or not valid
// JSON, or a number that decodes outside the min and max of f. The error does not name
// the default.
func CheckDefault(f JSONField) error {
	return problemError(defaultProblem(f))
}

func defaultProblem(f JSONField) (string, bool) {
	switch v := f.Variant.(type) {
	case FloatJSONField:
		if v.Default != nil && !xmath.Finite(*v.Default) {
			return "must be finite", true
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

// defaultRangeProblem returns how the finite default d of a number field with base b
// decodes outside its min and max, and false when it does not.
func defaultRangeProblem[N types.Numeric](b BaseField, d *N) (string, bool) {
	if d == nil || !xmath.Finite(float64(*d)) {
		return "", false
	}
	return outOfRawRange(b, float64(*d))
}
