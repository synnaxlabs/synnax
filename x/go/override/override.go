// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package override

import (
	"reflect"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/uuid"
)

// If returns the override value if the condition is true, otherwise it returns the
// base value.
func If[T any](base T, override T, condition bool) T {
	return lo.Ternary(condition, override, base)
}

// Numeric returns the override value if it is not zero, otherwise it returns the
// base value.
func Numeric[N types.Numeric](base N, override N) N {
	return If(base, override, override != 0)
}

// String returns the override value if it is not empty, otherwise it returns the
// base value.
func String[T ~string](base T, override T) T {
	return If(base, override, override != "")
}

// Zero returns the override if its IsZero method returns false, otherwise it returns the
// base value.
func Zero[T Zeroable](base T, override T) T {
	return If(base, override, !override.IsZero())
}

// Nil returns the override value if it is not nil, otherwise it returns the base value.
// Both values must be interfaces or pointers, and Nil will panic if they are not.
func Nil[T any](base T, override T) T {
	overrideV := reflect.ValueOf(override)
	return If(
		base,
		override,
		overrideV.IsValid() && (isInterface[T]() || !overrideV.IsNil()),
	)
}

// Slice returns the override value if it is not empty, otherwise it returns the
// base value.
func Slice[T any](base []T, override []T) []T {
	return If(base, override, len(override) > 0)
}

// UUID returns the override value if it is not the zero value, otherwise it returns the
// base value.
func UUID(base uuid.UUID, override uuid.UUID) uuid.UUID {
	return If(base, override, override != uuid.Nil)
}

func isInterface[T any]() bool {
	return reflect.TypeOf((*T)(nil)).Elem().Kind() == reflect.Interface
}

// Zeroable is a type that can be checked for its zero value using the IsZero method.
type Zeroable interface {
	// IsZero returns true if the value is zero.
	IsZero() bool
}
