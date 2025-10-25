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
	. "github.com/synnaxlabs/x/testutil"
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

	It("Should maintain consistent mapping for repeated type variables", func() {
		// Test that T maps to same fresh variable in multiple locations
		tv := types.TypeVariable("T", nil)
		inputs := types.Params{}
		inputs.Put("a", tv)
		inputs.Put("b", tv) // Same T
		fnType := types.Function(types.FunctionProperties{Inputs: &inputs})
		fresh := ir.FreshType(fnType, "test")

		// Both inputs should have same fresh type variable name
		aType := MustBeOk(fresh.Inputs.Get("a"))
		bType := MustBeOk(fresh.Inputs.Get("b"))
		Expect(aType.Name).To(Equal("test_T"))
		Expect(bType.Name).To(Equal("test_T"))
		Expect(aType.Name).To(Equal(bType.Name))
	})

	It("Should handle deeply nested constrained type variables", func() {
		// Create a chain: T3 <: T2 <: T1 <: i64
		i64Constraint := types.I64()
		t1 := types.TypeVariable("T1", &i64Constraint)
		t2 := types.TypeVariable("T2", &t1)
		t3 := types.TypeVariable("T3", &t2)

		fresh := ir.FreshType(t3, "node")
		Expect(fresh.Name).To(Equal("node_T3"))
		Expect(fresh.Constraint).ToNot(BeNil())
		Expect(fresh.Constraint.Name).To(Equal("node_T2"))
		Expect(fresh.Constraint.Constraint).ToNot(BeNil())
		Expect(fresh.Constraint.Constraint.Name).To(Equal("node_T1"))
		Expect(fresh.Constraint.Constraint.Constraint).ToNot(BeNil())
		Expect(fresh.Constraint.Constraint.Constraint.Kind).To(Equal(types.KindI64))
	})

	It("Should handle series of channels of type variables", func() {
		tv := types.TypeVariable("T", nil)
		chanType := types.Chan(tv)
		seriesType := types.Series(chanType)

		fresh := ir.FreshType(seriesType, "prefix")
		Expect(fresh.Kind).To(Equal(types.KindSeries))
		Expect(fresh.ValueType.Kind).To(Equal(types.KindChan))
		Expect(fresh.ValueType.ValueType.Kind).To(Equal(types.KindTypeVariable))
		Expect(fresh.ValueType.ValueType.Name).To(Equal("prefix_T"))
	})

	It("Should handle function with mixed generic and concrete parameters", func() {
		tv := types.TypeVariable("T", nil)
		inputs := types.Params{}
		inputs.Put("generic", tv)
		inputs.Put("concrete", types.I64())

		outputs := types.Params{}
		outputs.Put("result", tv)

		fnType := types.Function(types.FunctionProperties{
			Inputs:  &inputs,
			Outputs: &outputs,
		})

		fresh := ir.FreshType(fnType, "node")

		// Generic params should be freshened
		genericInput := MustBeOk(fresh.Inputs.Get("generic"))
		Expect(genericInput.Kind).To(Equal(types.KindTypeVariable))
		Expect(genericInput.Name).To(Equal("node_T"))

		// Concrete params should remain unchanged
		concreteInput := MustBeOk(fresh.Inputs.Get("concrete"))
		Expect(concreteInput.Kind).To(Equal(types.KindI64))

		// Output should use same freshened variable
		output := MustBeOk(fresh.Outputs.Get("result"))
		Expect(output.Name).To(Equal("node_T"))
		Expect(output.Name).To(Equal(genericInput.Name))
	})

	It("Should handle empty function parameters", func() {
		fnType := types.Function(types.FunctionProperties{})
		fresh := ir.FreshType(fnType, "test")

		Expect(fresh.Kind).To(Equal(types.KindFunction))
		Expect(fresh.Inputs).ToNot(BeNil())
		Expect(fresh.Outputs).ToNot(BeNil())
		Expect(fresh.Config).ToNot(BeNil())
		Expect(fresh.Inputs.Count()).To(Equal(0))
		Expect(fresh.Outputs.Count()).To(Equal(0))
		Expect(fresh.Config.Count()).To(Equal(0))
	})

	It("Should handle multiple distinct type variables", func() {
		tvA := types.TypeVariable("A", nil)
		tvB := types.TypeVariable("B", nil)

		inputs := types.Params{}
		inputs.Put("a", tvA)
		inputs.Put("b", tvB)

		fnType := types.Function(types.FunctionProperties{Inputs: &inputs})
		fresh := ir.FreshType(fnType, "node")

		aType := MustBeOk(fresh.Inputs.Get("a"))
		bType := MustBeOk(fresh.Inputs.Get("b"))

		Expect(aType.Name).To(Equal("node_A"))
		Expect(bType.Name).To(Equal("node_B"))
		Expect(aType.Name).ToNot(Equal(bType.Name))
	})
})
