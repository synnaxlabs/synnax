// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Functions", func() {
	var (
		fn1, fn2, fn3 ir.Function
		fns           ir.Functions
	)

	BeforeEach(func() {
		fn1 = ir.Function{Key: "add"}
		fn2 = ir.Function{Key: "multiply"}
		fn3 = ir.Function{Key: "divide"}
		fns = ir.Functions{fn1, fn2, fn3}
	})

	Describe("Find", func() {
		It("Should find existing function by key", func() {
			fn := MustBeOk(fns.Find("multiply"))
			Expect(fn.Key).To(Equal("multiply"))
		})

		It("Should return false for non-existent key", func() {
			_, found := fns.Find("nonexistent")
			Expect(found).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("Should get existing function by key", func() {
			fn := fns.Get("add")
			Expect(fn.Key).To(Equal("add"))
		})

		It("Should panic for non-existent key", func() {
			Expect(func() {
				_ = fns.Get("nonexistent")
			}).To(Panic())
		})
	})

	Describe("Empty Collection", func() {
		It("Should handle Find on empty collection", func() {
			empty := ir.Functions{}
			_, found := empty.Find("anything")
			Expect(found).To(BeFalse())
		})

		It("Should panic on Get with empty collection", func() {
			empty := ir.Functions{}
			Expect(func() {
				_ = empty.Get("anything")
			}).To(Panic())
		})
	})

	Describe("Function.Type", func() {
		It("Should return function type with all properties", func() {
			inputs := types.Params{
				{Name: "x", Type: types.I64()},
				{Name: "y", Type: types.I64()},
			}
			outputs := types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}

			fn := ir.Function{
				Key:     "test",
				Inputs:  inputs,
				Outputs: outputs,
			}

			t := fn.Type()
			Expect(t.Kind).To(Equal(types.KindFunction))
			Expect(t.Inputs).To(HaveLen(2))
			Expect(t.Outputs).To(HaveLen(1))
		})
	})
})
