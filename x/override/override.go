// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package override

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/types"
	"reflect"
)

func If[T any](value T, override T, condition bool) T {
	return lo.Ternary(condition, override, value)
}

func Numeric[N types.Numeric](value N, override N) N {
	return lo.Ternary(override != 0, override, value)
}

func String[T ~string](value T, override T) T {
	return lo.Ternary(override != "", override, value)
}

func Zero[T ~struct{}](value T, override T) T {
	return lo.Ternary(reflect.DeepEqual(override, T{}), value, override)
}

func Nil[T any](value T, override T) T {
	overrideV := reflect.ValueOf(override)
	if overrideV.IsValid() && (isInterface[T]() || !overrideV.IsNil()) {
		return override
	}
	return value
}

func Slice[T any](value []T, override []T) []T {
	return lo.Ternary(len(override) > 0, override, value)
}

func BooleanTrue(value bool, override bool) bool {
	return lo.Ternary(override, override, value)
}

func isInterface[T any]() bool {
	return reflect.TypeOf((*T)(nil)).Elem().Kind() == reflect.Interface
}
