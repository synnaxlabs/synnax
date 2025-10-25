// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraints_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Constraint System", func() {
	var system *constraints.System
	BeforeEach(func() { system = constraints.New() })

	Describe("Basic Constraint Collection", func() {
		It("should track type variables", func() {
			tv1 := types.NewTypeVariable("T", nil)
			constraint := types.NumericConstraint()
			tv2 := types.NewTypeVariable("U", &constraint)
			system.AddEquality(tv1, types.F32(), nil, "test")
			system.AddEquality(tv2, types.I32(), nil, "test")
			Expect(system.TypeVariables()).To(HaveLen(2))
			Expect(system.Constraints()).To(HaveLen(2))
		})

		It("should track constraints between type variables", func() {
			tv1 := types.NewTypeVariable("T", nil)
			tv2 := types.NewTypeVariable("U", nil)
			system.AddEquality(tv1, tv2, nil, "T = U")
			Expect(system.TypeVariables()).To(HaveLen(2))
			Expect(system.Constraints()).To(HaveLen(1))
		})

		It("should handle nested type variables in channels", func() {
			constraint := types.NumericConstraint()
			tv := types.NewTypeVariable("T", &constraint)
			chanType := types.Chan(tv)
			system.AddEquality(chanType, types.Chan(types.F32()), nil, "channel types")
			Expect(system.TypeVariables()).To(HaveLen(1))
			Expect(system.TypeVariables()["T"]).NotTo(BeNil())
		})
	})

	Describe("HasTypeVariables", func() {
		It("should return false when no type variables exist", func() {
			Expect(system.HasTypeVariables()).To(BeFalse())
		})

		It("should return true when type variables exist", func() {
			tv := types.NewTypeVariable("T", nil)
			system.AddEquality(tv, types.F32(), nil, "test")
			Expect(system.HasTypeVariables()).To(BeTrue())
		})
	})

	Describe("ApplySubstitutions", func() {
		It("should apply simple substitutions", func() {
			tv := types.NewTypeVariable("T", nil)
			system.SetSubstitution("T", types.F32())
			result := system.ApplySubstitutions(tv)
			Expect(result).To(Equal(types.F32()))
		})

		It("should apply substitutions recursively", func() {
			tv1 := types.NewTypeVariable("T", nil)
			tv2 := types.NewTypeVariable("U", nil)
			system.SetSubstitution("T", tv2)
			system.SetSubstitution("U", types.I64())
			Expect(system.ApplySubstitutions(tv1)).To(Equal(types.I64()))
		})

		It("should apply substitutions in compound types", func() {
			tv := types.NewTypeVariable("T", nil)
			chanType := types.Chan(tv)
			system.SetSubstitution("T", types.F64())
			Expect(system.ApplySubstitutions(chanType)).To(Equal(types.Chan(types.F64())))
		})

		It("should apply substitutions in series types", func() {
			constraint := types.NumericConstraint()
			tv := types.NewTypeVariable("T", &constraint)
			seriesType := types.Series(tv)
			system.SetSubstitution("T", types.I32())
			Expect(system.ApplySubstitutions(seriesType)).To(Equal(types.Series(types.I32())))
		})

		It("should apply substitutions to function input types", func() {
			tv := types.NewTypeVariable("T", nil)
			props := types.NewFunctionProperties()
			props.Inputs.Put("x", tv)
			fnType := types.Function(props)
			system.SetSubstitution("T", types.F32())
			result := system.ApplySubstitutions(fnType)
			inputType, ok := result.Inputs.Get("x")
			Expect(ok).To(BeTrue())
			Expect(inputType).To(Equal(types.F32()))
		})

		It("should apply substitutions to function output types", func() {
			tv := types.NewTypeVariable("T", nil)
			props := types.NewFunctionProperties()
			props.Outputs.Put("result", tv)
			fnType := types.Function(props)
			system.SetSubstitution("T", types.I64())
			result := system.ApplySubstitutions(fnType)
			outputType, ok := result.Outputs.Get("result")
			Expect(ok).To(BeTrue())
			Expect(outputType).To(Equal(types.I64()))
		})

		It("should apply substitutions to function config types", func() {
			tv := types.NewTypeVariable("T", nil)
			props := types.NewFunctionProperties()
			props.Config.Put("threshold", tv)
			fnType := types.Function(props)
			system.SetSubstitution("T", types.F64())
			result := system.ApplySubstitutions(fnType)
			configType, ok := result.Config.Get("threshold")
			Expect(ok).To(BeTrue())
			Expect(configType).To(Equal(types.F64()))
		})

		It("should apply substitutions to multiple function parameters", func() {
			tv1 := types.NewTypeVariable("T1", nil)
			tv2 := types.NewTypeVariable("T2", nil)
			tv3 := types.NewTypeVariable("T3", nil)
			props := types.NewFunctionProperties()
			props.Inputs.Put("x", tv1)
			props.Outputs.Put("y", tv2)
			props.Config.Put("z", tv3)
			fnType := types.Function(props)
			system.SetSubstitution("T1", types.F32())
			system.SetSubstitution("T2", types.I32())
			system.SetSubstitution("T3", types.String())
			result := system.ApplySubstitutions(fnType)
			inputType, _ := result.Inputs.Get("x")
			outputType, _ := result.Outputs.Get("y")
			configType, _ := result.Config.Get("z")
			Expect(inputType).To(Equal(types.F32()))
			Expect(outputType).To(Equal(types.I32()))
			Expect(configType).To(Equal(types.String()))
		})

		It("should handle circular substitution chains correctly", func() {
			tv1 := types.NewTypeVariable("A", nil)
			tv2 := types.NewTypeVariable("B", nil)
			system.SetSubstitution("A", tv2)
			system.SetSubstitution("B", tv1)
			result := system.ApplySubstitutions(tv1)
			Expect(result.Kind).To(Equal(types.KindTypeVariable))
		})
	})

	Describe("String", func() {
		It("should format constraint system as string", func() {
			constraint := types.NumericConstraint()
			tv1 := types.NewTypeVariable("T1", &constraint)
			tv2 := types.NewTypeVariable("T2", nil)
			system.AddEquality(tv1, types.F32(), nil, "T1 = f32")
			system.AddCompatible(tv2, types.I32(), nil, "T2 ~ i32")
			system.SetSubstitution("T1", types.F32())
			str := system.String()
			Expect(str).To(ContainSubstring("Type Variables"))
			Expect(str).To(ContainSubstring("Constraints"))
			Expect(str).To(ContainSubstring("Substitutions"))
			Expect(str).To(ContainSubstring("T1"))
			Expect(str).To(ContainSubstring("T2"))
		})

		It("should show resolved and unresolved type variables", func() {
			constraint := types.NumericConstraint()
			tv1 := types.NewTypeVariable("Resolved", &constraint)
			tv2 := types.NewTypeVariable("Unresolved", nil)
			system.AddEquality(tv1, types.F32(), nil, "test")
			system.AddEquality(tv2, tv2, nil, "test")
			system.SetSubstitution("Resolved", types.F32())
			str := system.String()
			Expect(str).To(ContainSubstring("Resolved"))
			Expect(str).To(ContainSubstring("=>"))
			Expect(str).To(ContainSubstring("Unresolved"))
			Expect(str).To(ContainSubstring("(unresolved)"))
		})

		It("should distinguish equality vs compatible constraints in string output", func() {
			tv1 := types.NewTypeVariable("T1", nil)
			tv2 := types.NewTypeVariable("T2", nil)
			system.AddEquality(tv1, types.F32(), nil, "equality constraint")
			system.AddCompatible(tv2, types.I32(), nil, "compatible constraint")
			str := system.String()
			Expect(str).To(ContainSubstring("≡"))
			Expect(str).To(ContainSubstring("~"))
			Expect(str).To(ContainSubstring("equality constraint"))
			Expect(str).To(ContainSubstring("compatible constraint"))
		})

		It("should show constraint reasons", func() {
			tv := types.NewTypeVariable("T", nil)
			system.AddEquality(tv, types.F32(), nil, "because we need float precision")
			str := system.String()
			Expect(str).To(ContainSubstring("because we need float precision"))
		})
	})
})
