// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package set

type Set[T comparable, V any] map[T]V

func (s Set[T, V]) Add(values ...T) {
	var v V
	for _, k := range values {
		s[k] = v
	}
}

func (s Set[T, V]) Remove(keys ...T) {
	for _, v := range keys {
		delete(s, v)
	}
}

func (s Set[T, V]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

func (s Set[T, V]) Keys() []T {
	values := make([]T, 0, len(s))
	for k := range s {
		values = append(values, k)
	}
	return values
}

func (s Set[T, V]) Values() []V {
	values := make([]V, 0, len(s))
	for _, v := range s {
		values = append(values, v)
	}
	return values
}
