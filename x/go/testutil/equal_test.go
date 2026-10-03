// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

type rate float64

// unit carries an Equal method that follows IEEE semantics, so NaN never equals itself
// through it.
type unit struct{ scale float64 }

func (u unit) Equal(other unit) bool { return u.scale == other.scale }

type record struct {
	Name   string
	Rate   rate
	Tags   []string
	Attrs  map[string]int
	Next   *record
	hidden int
}

var _ = Describe("DeepEqual", func() {
	DescribeTable("Should compare values structurally",
		func(a, b any, expected bool) { Expect(DeepEqual(a, b)).To(Equal(expected)) },
		Entry("equal ints", 1, 1, true),
		Entry("different ints", 1, 2, false),
		Entry("different types", int32(1), int64(1), false),
		Entry("both nil", nil, nil, true),
		Entry("nil and value", nil, 1, false),
		Entry("NaN float64", math.NaN(), math.NaN(), true),
		Entry("NaN float32", float32(math.NaN()), float32(math.NaN()), true),
		Entry("NaN named float", rate(math.NaN()), rate(math.NaN()), true),
		Entry("NaN and number", math.NaN(), 1.0, false),
		Entry("NaN complex", complex(math.NaN(), 0), complex(math.NaN(), 0), true),
		Entry("Equal method ignored", unit{math.NaN()}, unit{math.NaN()}, true),
		Entry("Equal method differing scale", unit{1}, unit{2}, false),
		Entry("nil and empty slice", []int(nil), []int{}, false),
		Entry("equal slices", []int{1, 2}, []int{1, 2}, true),
		Entry("different slice lengths", []int{1}, []int{1, 2}, false),
		Entry("NaN in slice", []float64{math.NaN()}, []float64{math.NaN()}, true),
		Entry("equal arrays", [2]int{1, 2}, [2]int{1, 2}, true),
		Entry("equal maps", map[string]int{"a": 1}, map[string]int{"a": 1}, true),
		Entry("map missing key", map[string]int{"a": 1}, map[string]int{"b": 1}, false),
		Entry("nil and empty map", map[string]int(nil), map[string]int{}, false),
		Entry("nil pointers", (*record)(nil), (*record)(nil), true),
		Entry("nil and set pointer", (*record)(nil), &record{}, false),
		Entry("pointed values", &record{Name: "a"}, &record{Name: "a"}, true),
		Entry("interface elements of different types", []any{1}, []any{"1"}, false),
		Entry("nil interface elements", []any{nil}, []any{nil}, true),
		Entry(
			"nested records",
			record{Name: "a", Rate: rate(math.NaN()), Tags: []string{"x"}},
			record{Name: "a", Rate: rate(math.NaN()), Tags: []string{"x"}},
			true,
		),
		Entry("unexported field", record{hidden: 1}, record{hidden: 2}, false),
		Entry("nil funcs", (func())(nil), (func())(nil), true),
		Entry("set funcs", func() {}, func() {}, false),
	)

	It("Should terminate on a cyclic value", func() {
		a := &record{Name: "loop"}
		a.Next = a
		b := &record{Name: "loop"}
		b.Next = b
		Expect(DeepEqual(a, b)).To(BeTrue())
	})

	It("Should compare a channel by identity", func() {
		ch := make(chan int)
		Expect(DeepEqual(ch, ch)).To(BeTrue())
		Expect(DeepEqual(ch, make(chan int))).To(BeFalse())
	})
})
