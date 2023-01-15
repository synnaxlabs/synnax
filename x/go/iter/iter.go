// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iter

// Endlessly returns a function that iterates over a collection of values
// indefinitely.
func Endlessly[T any](values []T) func() T {
	i := 0
	return func() T {
		val := values[i]
		if i < (len(values) - 1) {
			i++
		} else {
			i = 0
		}
		return val
	}
}

func ForEachUntilError[T any](values []T, fn func(T) error) error {
	for _, value := range values {
		if err := fn(value); err != nil {
			return err
		}
	}
	return nil
}

func MapForEachUntilError[K comparable, V any](values map[K]V, fn func(K, V) error) error {
	for key, value := range values {
		if err := fn(key, value); err != nil {
			return err
		}
	}
	return nil
}
