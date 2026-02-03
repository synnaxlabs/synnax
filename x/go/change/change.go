// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package change

type Variant uint8

//go:generate stringer -type=Variant
const (
	VariantSet Variant = iota + 1
	VariantDelete
)

// Change is a mutation to a generic key-value pair. This change can either be a 'Label'
// or a 'DeleteChannel'. In the case of a 'Label', the Value field will be populated with the
// new value. In the case of a 'DeleteChannel', the Value field will be the zero value of the
// type of the Value field.
type Change[K, V any] struct {
	// Key is the key of the key-value pair.
	Key K
	// Value is the value of the key-value pair. On a 'DeleteChannel' change, this will be the
	// zero value of the Value field.
	Value V
	// Variant is the type of change.
	Variant Variant
}

func Map[K comparable, V comparable](
	prev,
	next map[K]V,
	equal func(prev, next V) bool,
) []Change[K, V] {
	if equal == nil {
		equal = func(prev, next V) bool { return prev == next }
	}
	changes := make([]Change[K, V], 0, len(prev)+len(next))
	for k, v := range prev {
		next, ok := next[k]
		if !ok || !equal(v, next) {
			changes = append(changes, Change[K, V]{Key: k, Value: v, Variant: VariantDelete})
		}
	}
	for k := range next {
		if _, ok := prev[k]; !ok {
			changes = append(changes, Change[K, V]{Key: k, Value: next[k], Variant: VariantSet})
		}
	}
	return changes
}
