// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key

import (
	"cmp"
	"reflect"
	"strings"
	"unsafe"
)

// Order returns a function that orders keys of type K. Numbers order by value, false
// orders before true, strings order byte by byte, and byte arrays order from the last
// byte to the first, as Gorp stores them.
func Order[K primitive]() func(a, b K) int {
	var zero K
	switch t := reflect.TypeOf(zero); t.Kind() {
	case reflect.String:
		return func(a, b K) int {
			return strings.Compare(
				*(*string)(unsafe.Pointer(&a)),
				*(*string)(unsafe.Pointer(&b)),
			)
		}
	case reflect.Array:
		size := int(t.Size())
		return func(a, b K) int {
			x := unsafe.Slice((*byte)(unsafe.Pointer(&a)), size)
			y := unsafe.Slice((*byte)(unsafe.Pointer(&b)), size)
			for i := size - 1; i >= 0; i-- {
				if c := cmp.Compare(x[i], y[i]); c != 0 {
					return c
				}
			}
			return 0
		}
	case reflect.Bool:
		return func(a, b K) int {
			x, y := *(*bool)(unsafe.Pointer(&a)), *(*bool)(unsafe.Pointer(&b))
			switch {
			case x == y:
				return 0
			case y:
				return -1
			default:
				return 1
			}
		}
	case reflect.Int8:
		return orderAs[K, int8]
	case reflect.Int16:
		return orderAs[K, int16]
	case reflect.Int32:
		return orderAs[K, int32]
	case reflect.Int64:
		return orderAs[K, int64]
	case reflect.Uint8:
		return orderAs[K, uint8]
	case reflect.Uint16:
		return orderAs[K, uint16]
	case reflect.Uint32:
		return orderAs[K, uint32]
	case reflect.Uint64:
		return orderAs[K, uint64]
	case reflect.Float32:
		return orderAs[K, float32]
	case reflect.Float64:
		return orderAs[K, float64]
	default:
		panic("gorp: no order for key kind " + t.Kind().String())
	}
}

func orderAs[K primitive, T cmp.Ordered](a, b K) int {
	return cmp.Compare(*(*T)(unsafe.Pointer(&a)), *(*T)(unsafe.Pointer(&b)))
}
