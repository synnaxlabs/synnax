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

// Order returns a function that orders keys of type K as Gorp stores them: strings
// byte by byte, and fixed-size keys by their big-endian encoding.
func Order[K primitive]() func(a, b K) int {
	var zero K
	if reflect.TypeOf(zero).Kind() == reflect.String {
		return func(a, b K) int {
			return strings.Compare(
				*(*string)(unsafe.Pointer(&a)),
				*(*string)(unsafe.Pointer(&b)),
			)
		}
	}
	switch size := int(unsafe.Sizeof(zero)); size {
	case 1:
		return orderAs[K, uint8]
	case 2:
		return orderAs[K, uint16]
	case 4:
		return orderAs[K, uint32]
	case 8:
		return orderAs[K, uint64]
	default:
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
	}
}

func orderAs[K primitive, T cmp.Ordered](a, b K) int {
	return cmp.Compare(*(*T)(unsafe.Pointer(&a)), *(*T)(unsafe.Pointer(&b)))
}
