// Copyright 2026 Synnax Labs, Inc.
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
)

var _ = Describe("UnitsAssignable", func() {
	ns := &types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "ns"}
	psi := &types.Unit{Name: "psi", Scale: 1}
	bar := &types.Unit{Name: "bar", Scale: 1}

	DescribeTable("unit compatibility",
		func(a, b *types.Unit, want bool) {
			Expect(types.UnitsAssignable(a, b)).To(Equal(want))
		},
		Entry("nil + nil", (*types.Unit)(nil), (*types.Unit)(nil), true),
		Entry("nil + ns (wildcard)", (*types.Unit)(nil), ns, true),
		Entry("ns + nil (wildcard)", ns, (*types.Unit)(nil), true),
		Entry("ns + ns (match)", ns, ns, true),
		Entry("psi + bar (mismatch)", psi, bar, false),
	)
})

var _ = Describe("WithUnit", func() {
	s := &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}

	DescribeTable("Should set the unit of the value type",
		func(t types.Type) {
			Expect(types.WithUnit(t, s).Unwrap().Unit).To(BeIdenticalTo(s))
			Expect(t.Unwrap().Unit).To(BeNil())
		},
		Entry("a value", types.I64()),
		Entry("a series", types.Series(types.I64())),
		Entry("a channel", types.Chan(types.I64())),
	)
})

var _ = Describe("Concrete", func() {
	s := &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}
	variable := func(constraint *types.Type) types.Type {
		t := types.Variable("T", constraint)
		t.Unit = s
		return t
	}
	integer, float := types.IntegerConstraint(), types.FloatConstraint()

	DescribeTable("Should resolve the type of an untyped literal",
		func(t, want types.Type) {
			Expect(types.Concrete(t)).To(Equal(want))
		},
		Entry("an integer literal", variable(&integer), types.WithUnit(types.I64(), s)),
		Entry("a float literal", variable(&float), types.WithUnit(types.F64(), s)),
		Entry("a type variable with no constraint",
			variable(nil), types.WithUnit(types.F64(), s)),
		Entry("a type that is not a type variable", types.I32(), types.I32()),
	)
})

var _ = Describe("UnitsConvertible", func() {
	ns := &types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "ns"}
	s := &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}
	psi := &types.Unit{Dimensions: types.DimPressure, Scale: 6894.76, Name: "psi"}

	DescribeTable("unit conversion",
		func(a, b *types.Unit, want bool) {
			Expect(types.UnitsConvertible(a, b)).To(Equal(want))
		},
		Entry("s to ns", s, ns, true),
		Entry("ns to s", ns, s, true),
		Entry("the same unit", ns, ns, false),
		Entry("another dimension", psi, ns, false),
		Entry("no unit on the source", (*types.Unit)(nil), ns, false),
		Entry("no unit on the target", s, (*types.Unit)(nil), false),
	)
})

var _ = Describe("UnitsIncompatible", func() {
	ns := &types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "ns"}
	s := &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}
	psi := &types.Unit{Dimensions: types.DimPressure, Scale: 6894.76, Name: "psi"}

	DescribeTable("unit dimensions",
		func(a, b *types.Unit, want bool) {
			Expect(types.UnitsIncompatible(a, b)).To(Equal(want))
		},
		Entry("another dimension", psi, ns, true),
		Entry("the same dimension", s, ns, false),
		Entry("the same unit", ns, ns, false),
		Entry("no unit on the source", (*types.Unit)(nil), ns, false),
		Entry("no unit on the target", psi, (*types.Unit)(nil), false),
	)
})

var _ = Describe("CommonUnit", func() {
	ms := &types.Unit{Dimensions: types.DimTime, Scale: 1e6, Name: "ms"}
	s := &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}

	DescribeTable("unit of a sum",
		func(a, b, want *types.Unit) {
			Expect(types.CommonUnit(a, b)).To(BeIdenticalTo(want))
		},
		Entry("the smaller unit first", ms, s, ms),
		Entry("the smaller unit second", s, ms, ms),
		Entry("the same unit", s, s, s),
		Entry("no unit on the left", (*types.Unit)(nil), s, s),
		Entry("no unit on the right", s, (*types.Unit)(nil), s),
		Entry("no unit", (*types.Unit)(nil), (*types.Unit)(nil), (*types.Unit)(nil)),
	)
})

var _ = Describe("ProductUnit", func() {
	ms := &types.Unit{Dimensions: types.DimTime, Scale: 1e6, Name: "ms"}
	s := &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}

	DescribeTable("unit of a product",
		func(a, b, want *types.Unit) {
			Expect(types.ProductUnit(a, b)).To(BeIdenticalTo(want))
		},
		Entry("two units", s, ms, (*types.Unit)(nil)),
		Entry("the same unit twice", s, s, (*types.Unit)(nil)),
		Entry("no unit on the left", (*types.Unit)(nil), s, s),
		Entry("no unit on the right", s, (*types.Unit)(nil), s),
		Entry("no unit", (*types.Unit)(nil), (*types.Unit)(nil), (*types.Unit)(nil)),
	)

	DescribeTable("unit of a quotient",
		func(a, b, want *types.Unit) {
			Expect(types.QuotientUnit(a, b)).To(BeIdenticalTo(want))
		},
		Entry("two units", s, ms, (*types.Unit)(nil)),
		Entry("no unit on the left", (*types.Unit)(nil), s, (*types.Unit)(nil)),
		Entry("no unit on the right", s, (*types.Unit)(nil), s),
		Entry("no unit", (*types.Unit)(nil), (*types.Unit)(nil), (*types.Unit)(nil)),
	)
})

var _ = Describe("UnitScale", func() {
	in := func(t types.Type, scale float64) types.Type {
		t.Unit = &types.Unit{Dimensions: types.DimLength, Scale: scale}
		return t
	}
	to := func(scale float64) *types.Unit {
		return &types.Unit{Dimensions: types.DimLength, Scale: scale}
	}

	DescribeTable("factor that converts a value",
		func(t types.Type, to *types.Unit, factor float64, divide, exact bool) {
			gotFactor, gotDivide, gotExact := types.UnitScale(t, to)
			Expect(gotFactor).To(Equal(factor))
			Expect(gotDivide).To(Equal(divide))
			Expect(gotExact).To(Equal(exact))
		},
		Entry("an integer by a whole factor",
			in(types.I64(), 1e3), to(1), 1e3, false, true),
		Entry("an integer by a whole divisor",
			in(types.I64(), 1), to(1e3), 1e3, true, true),
		Entry("an integer by a factor that is not whole",
			in(types.I64(), 0.0254), to(1), 0.0254, false, false),
		Entry("a float by a factor that is not whole",
			in(types.F64(), 0.0254), to(1), 0.0254, false, true),
		Entry("a 32-bit integer by a factor that does not fit",
			in(types.I32(), 6e10), to(1), 6e10, false, false),
		Entry("a 64-bit integer by the same factor",
			in(types.I64(), 6e10), to(1), 6e10, false, true),
	)
})

var _ = Describe("UnitFactor", func() {
	unit := func(scale float64) *types.Unit {
		return &types.Unit{Dimensions: types.DimLength, Scale: scale}
	}

	DescribeTable("factor between two units",
		func(from, to, want float64) {
			Expect(types.UnitFactor(unit(from), unit(to))).To(Equal(want))
		},
		Entry("a whole factor", 1e3, 1.0, 1e3),
		Entry("a whole factor that is not exact in a float", 1e-6, 1e-9, 1e3),
		Entry("a factor that is not whole", 0.0254, 1.0, 0.0254),
		Entry("a factor below one", 1.0, 1e3, 1e-3),
	)
})

var _ = Describe("InputScale", func() {
	withUnit := func(t types.Type, name string, scale float64) types.Type {
		t.Unit = &types.Unit{Dimensions: types.DimTime, Scale: scale, Name: name}
		return t
	}
	seconds := withUnit(types.I64(), "s", 1e9)

	DescribeTable("scale of an input",
		func(t types.Type, want float64) {
			up, _ := types.InputScale(t)
			Expect(up).To(Equal(want))
		},
		Entry("s variable into an ns input",
			withUnit(types.VarRef(seconds, "v"), "ns", 1), 1e9),
		Entry(
			"ns variable into an s input",
			withUnit(
				types.VarRef(withUnit(types.I64(), "ns", 1), "v"),
				"s",
				1e9,
			),
			1e-9,
		),
		Entry(
			"a factor that is not exact in a float",
			withUnit(
				types.VarRef(withUnit(types.I64(), "um", 1e-6), "v"),
				"nm",
				1e-9,
			),
			1e3,
		),
		Entry("var ref with no unit of its own", types.VarRef(seconds, "v"), 1.0),
		Entry("variable with no unit",
			withUnit(types.VarRef(types.I64(), "v"), "ns", 1), 1.0),
		Entry("not a var ref", seconds, 1.0),
	)

	DescribeTable("inverse scale of an input",
		func(t types.Type, want float64) {
			_, down := types.InputScale(t)
			Expect(down).To(Equal(want))
		},
		Entry("ns variable into an s input",
			withUnit(types.VarRef(withUnit(types.I64(), "ns", 1), "v"), "s", 1e9), 1e9),
		Entry(
			"a factor that is not exact in a float",
			withUnit(
				types.VarRef(withUnit(types.I64(), "nm", 1e-9), "v"),
				"um",
				1e-6,
			),
			1e3,
		),
		Entry("not a var ref", seconds, 1.0),
	)
})
