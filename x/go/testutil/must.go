// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import "github.com/onsi/gomega"

// MustSucceed is a test helper that asserts the error is nil and returns the value.
// This is useful for unwrapping Go's value-error return pattern in test assertions.
//
// Common use cases include:
//   - Function calls: result := MustSucceed(someFunc())
//   - Method calls: data := MustSucceed(file.Read())
//   - Channel operations: frame := MustSucceed(reader.Next())
//
// If err is non-nil, the test will fail at the call site with proper line number reporting.
//
// Example:
//
//	ch := client.CreateChannel("temperature")
//	data := MustSucceed(ch.Read()) // Returns data if err is nil
//	result := MustSucceed(failingFunc()) // Fails the test
func MustSucceed[T any](value T, err error) T {
	return MustSucceedWithOffset[T](1)(value, err)
}

// MustSucceedWithOffset returns a function that asserts the error is nil and returns the value,
// with an additional stack offset for proper error reporting. This is useful when wrapping
// MustSucceed in helper functions.
//
// The offset parameter adjusts the call stack depth for error reporting. Use offset=1 when
// wrapping this function directly, offset=2 when wrapping it in a function that wraps it, etc.
//
// Example:
//
//	func ReadHelper(ch Channel) Data {
//	    // Use offset=1 so errors point to the caller of ReadHelper
//	    return MustSucceedWithOffset[Data](1)(ch.Read())
//	}
//
//	func DoubleWrappedHelper(ch Channel) Data {
//	    // Use offset=2 since we're two levels deep
//	    return MustSucceedWithOffset[Data](2)(ch.Read())
//	}
func MustSucceedWithOffset[T any](offset int) func(value T, err error) T {
	return func(value T, err error) T {
		gomega.ExpectWithOffset(offset+1, err).ToNot(gomega.HaveOccurred())
		return value
	}
}

// MustSucceed2 is a test helper that asserts the error is nil and returns two values.
// This is useful for unwrapping functions that return two values and an error in test assertions.
//
// Example:
//
//	key, name := MustSucceed2(parseKeyAndName(input))
func MustSucceed2[A, B any](a A, b B, err error) (A, B) {
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return a, b
}

// MustSucceed3 is a test helper that asserts the error is nil and returns three values.
// This is useful for unwrapping functions that return three values and an error in test assertions.
//
// Example:
//
//	x, y, z := MustSucceed3(parseCoordinates(input))
func MustSucceed3[A, B, C any](a A, b B, c C, err error) (A, B, C) {
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return a, b, c
}

// MustSucceed4 is a test helper that asserts the error is nil and returns four values.
// This is useful for unwrapping functions that return four values and an error in test assertions.
//
// Example:
//
//	r, g, b, a := MustSucceed4(parseRGBA(input))
func MustSucceed4[A, B, C, D any](a A, b B, c C, d D, err error) (A, B, C, D) {
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return a, b, c, d
}

// MustBeOk is a test helper that asserts the ok value is true and returns the value.
// This is useful for unwrapping Go's comma-ok idiom in test assertions.
//
// Common use cases include:
//   - Map lookups: value := MustBeOk(myMap["key"])
//   - Type assertions: str := MustBeOk(myInterface.(string))
//   - Channel receives: msg := MustBeOk(<-myChan)
//
// If ok is false, the test will fail at the call site with proper line number reporting.
//
// Example:
//
//	m := map[string]int{"foo": 42}
//	val := MustBeOk(m["foo"]) // Returns 42
//	val := MustBeOk(m["bar"]) // Fails the test
func MustBeOk[T any](value T, ok bool) T {
	return MustBeOkWithOffset[T](1)(value, ok)
}

// MustBeOkWithOffset returns a function that asserts the ok value is true and returns the value,
// with an additional stack offset for proper error reporting. This is useful when wrapping
// MustBeOk in helper functions.
//
// The offset parameter adjusts the call stack depth for error reporting. Use offset=1 when
// wrapping this function directly, offset=2 when wrapping it in a function that wraps it, etc.
//
// Example:
//
//	func MyHelper(m map[string]int, key string) int {
//	    // Use offset=1 so errors point to the caller of MyHelper
//	    return MustBeOkWithOffset[int](1)(m[key])
//	}
//
//	func MyDoubleHelper(m map[string]int, key string) int {
//	    // Use offset=2 since we're two levels deep
//	    return MustBeOkWithOffset[int](2)(m[key])
//	}
func MustBeOkWithOffset[T any](offset int) func(value T, ok bool) T {
	return func(value T, ok bool) T {
		gomega.ExpectWithOffset(offset+1, ok).To(gomega.BeTrue())
		return value
	}
}
