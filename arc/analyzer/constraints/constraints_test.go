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

	BeforeEach(func() {
		system = constraints.New()
	})

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
			result := system.ApplySubstitutions(tv1)
			Expect(result).To(Equal(types.I64()))
		})

		It("should apply substitutions in compound types", func() {
			tv := types.NewTypeVariable("T", nil)
			chanType := types.Chan(tv)
			system.SetSubstitution("T", types.F64())
			result := system.ApplySubstitutions(chanType)
			Expect(result).To(Equal(types.Chan(types.F64())))
		})

		It("should apply substitutions in series types", func() {
			constraint := types.NumericConstraint()
			tv := types.NewTypeVariable("T", &constraint)
			seriesType := types.Series(tv)
			system.SetSubstitution("T", types.I32())
			result := system.ApplySubstitutions(seriesType)
			Expect(result).To(Equal(types.Series(types.I32())))
		})
	})
})
