// Copyright 2025 Synnax Labs, Inc.
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
)

var _ = Describe("FreshType", func() {
	It("Should rename type variables with prefix", func() {
		tv := types.TypeVariable("T", nil)
		fresh := ir.FreshType(tv, "node1")
		Expect(fresh.Kind).To(Equal(types.KindTypeVariable))
		Expect(fresh.Name).To(Equal("node1_T"))
	})

	It("Should recursively freshen constrained type variables", func() {
		constraint := types.I64()
		tv := types.TypeVariable("T", &constraint)
		fresh := ir.FreshType(tv, "node2")
		Expect(fresh.Name).To(Equal("node2_T"))
		Expect(fresh.Constraint).ToNot(BeNil())
	})

	It("Should freshen channel types recursively", func() {
		tv := types.TypeVariable("T", nil)
		chanType := types.Chan(tv)
		fresh := ir.FreshType(chanType, "node3")
		Expect(fresh.Kind).To(Equal(types.KindChan))
		Expect(fresh.ValueType).ToNot(BeNil())
		Expect(fresh.ValueType.Kind).To(Equal(types.KindTypeVariable))
		Expect(fresh.ValueType.Name).To(Equal("node3_T"))
	})

	It("Should freshen series types recursively", func() {
		tv := types.TypeVariable("T", nil)
		seriesType := types.Series(tv)
		fresh := ir.FreshType(seriesType, "node4")
		Expect(fresh.Kind).To(Equal(types.KindSeries))
		Expect(fresh.ValueType).ToNot(BeNil())
		Expect(fresh.ValueType.Kind).To(Equal(types.KindTypeVariable))
		Expect(fresh.ValueType.Name).To(Equal("node4_T"))
	})

	It("Should copy function types", func() {
		inputs := types.Params{}
		inputs.Put("x", types.I64())
		fnType := types.Function(types.FunctionProperties{Inputs: &inputs})
		fresh := ir.FreshType(fnType, "node5")
		Expect(fresh.Kind).To(Equal(types.KindFunction))
		Expect(fresh.Inputs).ToNot(BeNil())
		Expect(fresh.Inputs.Count()).To(Equal(1))
	})

	It("Should return primitive types unchanged", func() {
		primitives := []types.Type{
			types.I64(),
			types.F64(),
			types.U32(),
			types.String(),
			types.TimeStamp(),
			types.TimeSpan(),
		}
		for _, prim := range primitives {
			fresh := ir.FreshType(prim, "prefix")
			Expect(fresh.Kind).To(Equal(prim.Kind))
		}
	})

	It("Should handle nested type variables in channels", func() {
		constraint := types.I64()
		tv := types.TypeVariable("T", &constraint)
		chanType := types.Chan(tv)
		fresh := ir.FreshType(chanType, "test")
		Expect(fresh.ValueType.Name).To(Equal("test_T"))
		Expect(fresh.ValueType.Constraint).ToNot(BeNil())
		Expect(fresh.ValueType.Constraint.Kind).To(Equal(types.KindI64))
	})
})
