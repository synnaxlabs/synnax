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
	"github.com/synnaxlabs/x/maps"
)

var _ = Describe("Fresh Type Variables", func() {
	Describe("FreshType", func() {
		It("Should create fresh type variables with unique names", func() {
			tv := ir.TypeVariable{Name: "T", Constraint: ir.NumericConstraint{}}
			fresh1 := ir.FreshType(tv, "node1")
			fresh2 := ir.FreshType(tv, "node2")

			Expect(fresh1).To(BeAssignableToTypeOf(ir.TypeVariable{}))
			Expect(fresh2).To(BeAssignableToTypeOf(ir.TypeVariable{}))

			tv1 := fresh1.(ir.TypeVariable)
			tv2 := fresh2.(ir.TypeVariable)

			Expect(tv1.Name).To(Equal("node1_T"))
			Expect(tv2.Name).To(Equal("node2_T"))
			Expect(tv1.Constraint).To(Equal(ir.NumericConstraint{}))
			Expect(tv2.Constraint).To(Equal(ir.NumericConstraint{}))
		})

		It("Should preserve concrete types unchanged", func() {
			concreteTypes := []ir.Type{
				ir.F32{},
				ir.F64{},
				ir.I32{},
				ir.I64{},
				ir.U8{},
				ir.U32{},
				ir.U64{},
			}

			for _, t := range concreteTypes {
				fresh := ir.FreshType(t, "test")
				Expect(fresh).To(Equal(t))
			}
		})

		It("Should recursively freshen Chan types", func() {
			tv := ir.TypeVariable{Name: "T", Constraint: nil}
			channelType := ir.Chan{ValueType: tv}

			fresh := ir.FreshType(channelType, "node1")
			Expect(fresh).To(BeAssignableToTypeOf(ir.Chan{}))

			ch := fresh.(ir.Chan)
			Expect(ch.ValueType).To(BeAssignableToTypeOf(ir.TypeVariable{}))
			freshTV := ch.ValueType.(ir.TypeVariable)
			Expect(freshTV.Name).To(Equal("node1_T"))
		})

		It("Should recursively freshen Series types", func() {
			tv := ir.TypeVariable{Name: "T", Constraint: ir.NumericConstraint{}}
			seriesType := ir.Series{ValueType: tv}

			fresh := ir.FreshType(seriesType, "node1")
			Expect(fresh).To(BeAssignableToTypeOf(ir.Series{}))

			s := fresh.(ir.Series)
			Expect(s.ValueType).To(BeAssignableToTypeOf(ir.TypeVariable{}))
			freshTV := s.ValueType.(ir.TypeVariable)
			Expect(freshTV.Name).To(Equal("node1_T"))
			Expect(freshTV.Constraint).To(Equal(ir.NumericConstraint{}))
		})
	})

	Describe("FreshStage", func() {
		It("Should create fresh instances of polymorphic stages", func() {
			// Create a polymorphic "add" stage with type variable T
			tv := ir.TypeVariable{Name: "T", Constraint: ir.NumericConstraint{}}
			params := &maps.Ordered[string, ir.Type]{}
			params.Put("a", tv)
			params.Put("b", tv)

			outputs := &maps.Ordered[string, ir.Type]{}
			outputs.Put("output", tv)

			stage := ir.Stage{
				Key:     "add",
				Params:  *params,
				Outputs: *outputs,
			}

			// Create two fresh instances for different nodes
			fresh1 := ir.FreshStage(stage, "add_node1")
			fresh2 := ir.FreshStage(stage, "add_node2")

			// Both should have the same structure but different type variable names
			Expect(fresh1.Key).To(Equal("add"))
			Expect(fresh2.Key).To(Equal("add"))

			// Check that type variables are unique per instance
			aType1, _ := fresh1.Params.Get("a")
			aType2, _ := fresh2.Params.Get("a")

			tv1 := aType1.(ir.TypeVariable)
			tv2 := aType2.(ir.TypeVariable)

			Expect(tv1.Name).To(Equal("add_node1_T"))
			Expect(tv2.Name).To(Equal("add_node2_T"))

			// Verify outputs also have fresh variables
			outType1, _ := fresh1.Outputs.Get("output")
			outType2, _ := fresh2.Outputs.Get("output")

			tvOut1 := outType1.(ir.TypeVariable)
			tvOut2 := outType2.(ir.TypeVariable)

			Expect(tvOut1.Name).To(Equal("add_node1_T"))
			Expect(tvOut2.Name).To(Equal("add_node2_T"))
		})

		It("Should preserve concrete types in non-polymorphic stages", func() {
			params := &maps.Ordered[string, ir.Type]{}
			params.Put("input", ir.U8{})

			outputs := &maps.Ordered[string, ir.Type]{}
			outputs.Put("output", ir.U8{})

			stage := ir.Stage{
				Key:     "identity",
				Params:  *params,
				Outputs: *outputs,
			}

			fresh := ir.FreshStage(stage, "identity_node1")

			inType, _ := fresh.Params.Get("input")
			outType, _ := fresh.Outputs.Get("output")

			Expect(inType).To(Equal(ir.U8{}))
			Expect(outType).To(Equal(ir.U8{}))
		})

		It("Should handle stages with mixed polymorphic and concrete parameters", func() {
			tv := ir.TypeVariable{Name: "T", Constraint: ir.NumericConstraint{}}
			params := &maps.Ordered[string, ir.Type]{}
			params.Put("data", tv)
			params.Put("trigger", ir.U8{})

			outputs := &maps.Ordered[string, ir.Type]{}
			outputs.Put("output", tv)

			stage := ir.Stage{
				Key:     "gate",
				Params:  *params,
				Outputs: *outputs,
			}

			fresh := ir.FreshStage(stage, "gate_node1")

			dataType, _ := fresh.Params.Get("data")
			triggerType, _ := fresh.Params.Get("trigger")
			outType, _ := fresh.Outputs.Get("output")

			Expect(dataType.(ir.TypeVariable).Name).To(Equal("gate_node1_T"))
			Expect(triggerType).To(Equal(ir.U8{}))
			Expect(outType.(ir.TypeVariable).Name).To(Equal("gate_node1_T"))
		})

		It("Should prevent type variable collision between multiple instances", func() {
			// Regression test for the core bug: Multiple nodes using the same
			// polymorphic stage must get independent type variables
			tv := ir.TypeVariable{Name: "T", Constraint: ir.NumericConstraint{}}
			outputs := &maps.Ordered[string, ir.Type]{}
			outputs.Put("output", tv)

			onStage := ir.Stage{
				Key:     "on",
				Outputs: *outputs,
			}

			// Create two separate instances
			fresh1 := ir.FreshStage(onStage, "on_1")
			fresh2 := ir.FreshStage(onStage, "on_2")

			out1, _ := fresh1.Outputs.Get("output")
			out2, _ := fresh2.Outputs.Get("output")

			tv1 := out1.(ir.TypeVariable)
			tv2 := out2.(ir.TypeVariable)

			// Critical: these MUST have different names
			Expect(tv1.Name).To(Equal("on_1_T"))
			Expect(tv2.Name).To(Equal("on_2_T"))
			Expect(tv1.Name).NotTo(Equal(tv2.Name))
		})
	})

	Describe("FreshFunction", func() {
		It("Should create fresh instances of polymorphic functions", func() {
			tv := ir.TypeVariable{Name: "T", Constraint: nil}
			params := &maps.Ordered[string, ir.Type]{}
			params.Put("input", tv)

			outputs := &maps.Ordered[string, ir.Type]{}
			outputs.Put("output", tv)

			fn := ir.Function{
				Key:     "identity",
				Params:  *params,
				Outputs: *outputs,
			}

			fresh1 := ir.FreshFunction(fn, "call1")
			fresh2 := ir.FreshFunction(fn, "call2")

			in1, _ := fresh1.Params.Get("input")
			in2, _ := fresh2.Params.Get("input")

			Expect(in1.(ir.TypeVariable).Name).To(Equal("call1_T"))
			Expect(in2.(ir.TypeVariable).Name).To(Equal("call2_T"))
		})
	})
})
