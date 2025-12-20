// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Freshen", func() {
	It("Should rename type variables with prefix", func() {
		tv := types.Variable("T", nil)
		fresh := types.Freshen(tv, "node1")
		Expect(fresh.Kind).To(Equal(types.KindVariable))
		Expect(fresh.Name).To(Equal("node1_T"))
	})

	It("Should recursively freshen constrained type variables", func() {
		constraint := types.I64()
		tv := types.Variable("T", &constraint)
		fresh := types.Freshen(tv, "node2")
		Expect(fresh.Name).To(Equal("node2_T"))
		Expect(fresh.Constraint).ToNot(BeNil())
	})

	It("Should freshen channel types recursively", func() {
		tv := types.Variable("T", nil)
		chanType := types.Chan(tv)
		fresh := types.Freshen(chanType, "node3")
		Expect(fresh.Kind).To(Equal(types.KindChan))
		Expect(fresh.Elem).ToNot(BeNil())
		Expect(fresh.Elem.Kind).To(Equal(types.KindVariable))
		Expect(fresh.Elem.Name).To(Equal("node3_T"))
	})

	It("Should freshen series types recursively", func() {
		tv := types.Variable("T", nil)
		seriesType := types.Series(tv)
		fresh := types.Freshen(seriesType, "node4")
		Expect(fresh.Kind).To(Equal(types.KindSeries))
		Expect(fresh.Elem).ToNot(BeNil())
		Expect(fresh.Elem.Kind).To(Equal(types.KindVariable))
		Expect(fresh.Elem.Name).To(Equal("node4_T"))
	})

	It("Should copy function types", func() {
		inputs := types.Params{}
		inputs = append(inputs, types.Param{Name: "x", Type: types.I64()})
		fnType := types.Function(types.FunctionProperties{Inputs: inputs})
		fresh := types.Freshen(fnType, "node5")
		Expect(fresh.Kind).To(Equal(types.KindFunction))
		Expect(fresh.Inputs).ToNot(BeNil())
		Expect(len(fresh.Inputs)).To(Equal(1))
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
			fresh := types.Freshen(prim, "prefix")
			Expect(fresh.Kind).To(Equal(prim.Kind))
		}
	})

	It("Should handle nested type variables in channels", func() {
		constraint := types.I64()
		tv := types.Variable("T", &constraint)
		chanType := types.Chan(tv)
		fresh := types.Freshen(chanType, "test")
		Expect(fresh.Elem.Name).To(Equal("test_T"))
		Expect(fresh.Elem.Constraint).ToNot(BeNil())
		Expect(fresh.Elem.Constraint.Kind).To(Equal(types.KindI64))
	})

	It("Should maintain consistent mapping for repeated type variables", func() {
		// Test that T maps to same fresh variable in multiple locations
		tv := types.Variable("T", nil)
		inputs := types.Params{{Name: "a", Type: tv}, {Name: "b", Type: tv}}
		fnType := types.Function(types.FunctionProperties{Inputs: inputs})
		fresh := types.Freshen(fnType, "test")

		// Both inputs should have same fresh type variable name
		aType := fresh.Inputs[0]
		bType := fresh.Inputs[1]
		Expect(aType.Type.Name).To(Equal("test_T"))
		Expect(bType.Type.Name).To(Equal("test_T"))
		Expect(aType.Type.Name).To(Equal(bType.Type.Name))
	})

	It("Should handle deeply nested constrained type variables", func() {
		// Create a chain: T3 <: T2 <: T1 <: i64
		var (
			i64Constraint = types.I64()
			t1            = types.Variable("T1", &i64Constraint)
			t2            = types.Variable("T2", &t1)
			t3            = types.Variable("T3", &t2)
		)
		fresh := types.Freshen(t3, "node")
		Expect(fresh.Name).To(Equal("node_T3"))
		Expect(fresh.Constraint).ToNot(BeNil())
		Expect(fresh.Constraint.Name).To(Equal("node_T2"))
		Expect(fresh.Constraint.Constraint).ToNot(BeNil())
		Expect(fresh.Constraint.Constraint.Name).To(Equal("node_T1"))
		Expect(fresh.Constraint.Constraint.Constraint).ToNot(BeNil())
		Expect(fresh.Constraint.Constraint.Constraint.Kind).To(Equal(types.KindI64))
	})

	It("Should handle series of channels of type variables", func() {
		var (
			tv         = types.Variable("T", nil)
			chanType   = types.Chan(tv)
			seriesType = types.Series(chanType)
		)
		fresh := types.Freshen(seriesType, "prefix")
		Expect(fresh.Kind).To(Equal(types.KindSeries))
		Expect(fresh.Elem.Kind).To(Equal(types.KindChan))
		Expect(fresh.Elem.Elem.Kind).To(Equal(types.KindVariable))
		Expect(fresh.Elem.Elem.Name).To(Equal("prefix_T"))
	})

	It("Should handle function with mixed generic and concrete parameters", func() {
		tv := types.Variable("T", nil)
		inputs := types.Params{}
		inputs = append(inputs, types.Param{Name: "generic", Type: tv})
		inputs = append(inputs, types.Param{Name: "concrete", Type: types.I64()})

		outputs := types.Params{}
		outputs = append(outputs, types.Param{Name: "result", Type: tv})

		fnType := types.Function(types.FunctionProperties{
			Inputs:  inputs,
			Outputs: outputs,
		})

		fresh := types.Freshen(fnType, "node")

		// Generic params should be freshened
		genericInput := fresh.Inputs[0]
		Expect(genericInput.Type.Kind).To(Equal(types.KindVariable))
		Expect(genericInput.Type.Name).To(Equal("node_T"))

		// Concrete params should remain unchanged
		concreteInput := fresh.Inputs[1]
		Expect(concreteInput.Type.Kind).To(Equal(types.KindI64))

		// Output should use same freshened variable
		output := fresh.Outputs[0]
		Expect(output.Type.Name).To(Equal("node_T"))
		Expect(output.Type.Name).To(Equal(genericInput.Type.Name))
	})

	It("Should handle empty function parameters", func() {
		fnType := types.Function(types.FunctionProperties{})
		fresh := types.Freshen(fnType, "test")

		Expect(fresh.Kind).To(Equal(types.KindFunction))
		Expect(fresh.Inputs).To(BeNil())
		Expect(fresh.Outputs).To(BeNil())
		Expect(fresh.Config).To(BeNil())
	})

	It("Should handle multiple distinct type variables", func() {
		tvA := types.Variable("A", nil)
		tvB := types.Variable("B", nil)

		inputs := types.Params{}
		inputs = append(inputs, types.Param{Name: "a", Type: tvA})
		inputs = append(inputs, types.Param{Name: "b", Type: tvB})

		fnType := types.Function(types.FunctionProperties{Inputs: inputs})
		fresh := types.Freshen(fnType, "node")

		aType := MustBeOk(fresh.Inputs.Get("a"))
		bType := MustBeOk(fresh.Inputs.Get("b"))

		Expect(aType.Type.Name).To(Equal("node_A"))
		Expect(bType.Type.Name).To(Equal("node_B"))
		Expect(aType.Type.Name).ToNot(Equal(bType.Type.Name))
	})
})
