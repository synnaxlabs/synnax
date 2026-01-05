// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import (
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/telem"
)

// applyOperator applies the comparison operator to the actual and expected values.
func applyOperator(op Operator, actual any, expected any, subject ontology.ID) bool {
	switch op {
	case OpEqual:
		return actual == expected
	case OpNotEqual:
		return actual != expected
	case OpEqualSubject:
		if id, ok := actual.(ontology.ID); ok {
			return id == subject
		}
		return false
	case OpIn:
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		return lo.Contains(expectedList, actualStr)
	case OpNotIn:
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		return !lo.Contains(expectedList, actualStr)
	case OpContains:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedStr, ok := expected.(string)
		if !ok {
			return false
		}
		return lo.Contains(actualList, expectedStr)
	case OpContainsAny:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		for _, exp := range expectedList {
			if lo.Contains(actualList, exp) {
				return true
			}
		}
		return false
	case OpContainsAll:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		for _, exp := range expectedList {
			if !lo.Contains(actualList, exp) {
				return false
			}
		}
		return true
	case OpContainsNone:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		for _, exp := range expectedList {
			if lo.Contains(actualList, exp) {
				return false
			}
		}
		return true
	case OpWithin:
		actualTR, ok := actual.(telem.TimeRange)
		if !ok {
			return false
		}
		expectedTR, ok := expected.(telem.TimeRange)
		if !ok {
			return false
		}
		return expectedTR.ContainsRange(actualTR)
	case OpSubsetOf:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		for _, item := range actualList {
			if !lo.Contains(expectedList, item) {
				return false
			}
		}
		return true
	case OpLessThan:
		return compareNumeric(actual, expected) < 0
	case OpLessThanOrEqual:
		return compareNumeric(actual, expected) <= 0
	case OpGreaterThan:
		return compareNumeric(actual, expected) > 0
	case OpGreaterThanOrEqual:
		return compareNumeric(actual, expected) >= 0
	default:
		return false
	}
}

func toStringSlice(v any) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			str, ok := item.(string)
			if !ok {
				return nil, false
			}
			result[i] = str
		}
		return result, true
	default:
		return nil, false
	}
}

func compareNumeric(a, b any) int {
	if aDur, ok := a.(time.Duration); ok {
		if bDur, ok := b.(time.Duration); ok {
			if aDur < bDur {
				return -1
			} else if aDur > bDur {
				return 1
			}
			return 0
		}
	}
	if aSpan, ok := a.(telem.TimeSpan); ok {
		if bSpan, ok := b.(telem.TimeSpan); ok {
			if aSpan < bSpan {
				return -1
			} else if aSpan > bSpan {
				return 1
			}
			return 0
		}
	}
	if aInt, ok := toInt64(a); ok {
		if bInt, ok := toInt64(b); ok {
			if aInt < bInt {
				return -1
			} else if aInt > bInt {
				return 1
			}
			return 0
		}
	}
	if aFloat, ok := toFloat64(a); ok {
		if bFloat, ok := toFloat64(b); ok {
			if aFloat < bFloat {
				return -1
			} else if aFloat > bFloat {
				return 1
			}
			return 0
		}
	}
	return 0
}

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	default:
		return 0, false
	}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}
