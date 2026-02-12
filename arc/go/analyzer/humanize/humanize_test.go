// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package humanize_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/humanize"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Type", func() {
	Describe("Concrete Types", func() {
		It("Should return string representation for i64", func() {
			Expect(humanize.Type(types.I64())).To(Equal("i64"))
		})

		It("Should return string representation for f32", func() {
			Expect(humanize.Type(types.F32())).To(Equal("f32"))
		})

		It("Should return string representation for string", func() {
			Expect(humanize.Type(types.String())).To(Equal("str"))
		})
	})

	Describe("Type Variables", func() {
		It("Should return 'integer' for integer constraint", func() {
			intConstraint := types.IntegerConstraint()
			tv := types.Variable("$T1", &intConstraint)
			Expect(humanize.Type(tv)).To(Equal("integer"))
		})

		It("Should return 'float' for float constraint", func() {
			floatConstraint := types.FloatConstraint()
			tv := types.Variable("$T2", &floatConstraint)
			Expect(humanize.Type(tv)).To(Equal("float"))
		})

		It("Should return 'numeric' for numeric constraint", func() {
			numConstraint := types.NumericConstraint()
			tv := types.Variable("$T3", &numConstraint)
			Expect(humanize.Type(tv)).To(Equal("numeric"))
		})

		It("Should return 'unknown type' for unconstrained variable", func() {
			tv := types.Variable("$T4", nil)
			Expect(humanize.Type(tv)).To(Equal("unknown type"))
		})
	})

	Describe("Compound Types", func() {
		It("Should humanize chan element type", func() {
			intConstraint := types.IntegerConstraint()
			tv := types.Variable("$T", &intConstraint)
			ch := types.Chan(tv)
			Expect(humanize.Type(ch)).To(Equal("chan integer"))
		})

		It("Should humanize series element type", func() {
			s := types.Series(types.F64())
			Expect(humanize.Type(s)).To(Equal("series f64"))
		})
	})
})
