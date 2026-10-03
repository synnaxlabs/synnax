// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"math"
	"reflect"

	"github.com/synnaxlabs/x/set"
)

// DeepEqual reports whether a and b are structurally equal. It walks fields, elements
// and pointers directly, never calls an Equal method, and treats any two NaN floats as
// equal. A nil slice or map differs from an empty one, as in reflect.DeepEqual. Use it
// to check that a decoded value survives an encode/decode cycle when the type compares
// floats with == or may hold NaN.
func DeepEqual(a, b any) bool {
	return deepEqual(reflect.ValueOf(a), reflect.ValueOf(b), set.New[visit]())
}

// visit records a pair of references already under comparison, so a cyclic value
// terminates.
type visit struct {
	a, b uintptr
	t    reflect.Type
}

func floatEqual(x, y float64) bool {
	return x == y || (math.IsNaN(x) && math.IsNaN(y))
}

func deepEqual(a, b reflect.Value, seen set.Set[visit]) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}
	if a.Type() != b.Type() {
		return false
	}
	switch a.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		key := visit{a: a.Pointer(), b: b.Pointer(), t: a.Type()}
		if seen.Contains(key) {
			return true
		}
		seen.Add(key)
	}
	switch a.Kind() {
	case reflect.Bool:
		return a.Bool() == b.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		return a.Uint() == b.Uint()
	case reflect.Float32, reflect.Float64:
		return floatEqual(a.Float(), b.Float())
	case reflect.Complex64, reflect.Complex128:
		x, y := a.Complex(), b.Complex()
		return floatEqual(real(x), real(y)) && floatEqual(imag(x), imag(y))
	case reflect.String:
		return a.String() == b.String()
	case reflect.Array, reflect.Slice:
		if a.Len() != b.Len() {
			return false
		}
		for i := range a.Len() {
			if !deepEqual(a.Index(i), b.Index(i), seen) {
				return false
			}
		}
		return true
	case reflect.Map:
		if a.Len() != b.Len() {
			return false
		}
		for iter := a.MapRange(); iter.Next(); {
			bv := b.MapIndex(iter.Key())
			if !bv.IsValid() || !deepEqual(iter.Value(), bv, seen) {
				return false
			}
		}
		return true
	case reflect.Pointer:
		return deepEqual(a.Elem(), b.Elem(), seen)
	case reflect.Interface:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return deepEqual(a.Elem(), b.Elem(), seen)
	case reflect.Struct:
		for i := range a.NumField() {
			if !deepEqual(a.Field(i), b.Field(i), seen) {
				return false
			}
		}
		return true
	case reflect.Func:
		return a.IsNil() && b.IsNil()
	case reflect.Chan, reflect.UnsafePointer:
		return a.Pointer() == b.Pointer()
	}
	return false
}
