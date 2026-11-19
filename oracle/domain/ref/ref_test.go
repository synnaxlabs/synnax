// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ref_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/ref"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/x/testutil"
)

func expr(name string, idents ...string) resolution.Expression {
	values := make([]resolution.ExpressionValue, len(idents))
	for i, v := range idents {
		values[i] = resolution.ExpressionValue{
			IdentValue: v,
			Kind:       resolution.ValueKindIdent,
		}
	}
	return resolution.Expression{Name: name, Values: values}
}

func field(exprs ...resolution.Expression) resolution.Field {
	f := resolution.Field{Name: "channel", Domains: map[string]resolution.Domain{}}
	if len(exprs) > 0 {
		f.Domains["ref"] = resolution.Domain{Name: "ref", Expressions: exprs}
	}
	return f
}

var _ = Describe("Ref", func() {
	Describe("Declares", func() {
		It("Should declare a reference for a @ref with an on_delete", func() {
			Expect(ref.Declares(field(
				expr("on_delete", "weak"),
				expr("direction", "read"),
			))).To(BeTrue())
		})

		It("Should not declare a reference for a @ref with only a direction", func() {
			Expect(ref.Declares(field(expr("direction", "read")))).To(BeFalse())
		})

		It("Should declare a reference for a @ref with no expressions", func() {
			f := field()
			f.Domains["ref"] = resolution.Domain{Name: "ref"}
			Expect(ref.Declares(f)).To(BeTrue())
		})

		It("Should not declare a reference for a field with no @ref", func() {
			Expect(ref.Declares(field())).To(BeFalse())
		})
	})

	Describe("DirectionOf", func() {
		DescribeTable(
			"Should read the direction",
			func(f resolution.Field, dir ref.Direction) {
				Expect(MustSucceed(ref.DirectionOf(f))).To(Equal(dir))
			},
			Entry("read", field(expr("direction", "read")), ref.Read),
			Entry("write", field(expr("direction", "write")), ref.Write),
			Entry("none on a reference", field(expr("on_delete", "weak")), ref.None),
			Entry("none with no @ref", field(), ref.None),
		)

		DescribeTable(
			"Should refuse a direction other than read or write",
			func(f resolution.Field) {
				Expect(ref.DirectionOf(f)).Error().To(MatchError(
					"@ref field channel needs direction read or write",
				))
			},
			Entry("an unknown value", field(expr("direction", "both"))),
			Entry("no value", field(expr("direction"))),
			Entry("two values", field(expr("direction", "read", "write"))),
		)
	})

	Describe("Direction", func() {
		It("Should keep the values that generated tags hold", func() {
			Expect(uint8(ref.None)).To(Equal(uint8(0)))
			Expect(uint8(ref.Read)).To(Equal(uint8(1)))
			Expect(uint8(ref.Write)).To(Equal(uint8(2)))
		})

		It("Should name each direction", func() {
			Expect(ref.Read.String()).To(Equal("read"))
			Expect(ref.Write.String()).To(Equal("write"))
			Expect(ref.None.String()).To(Equal("none"))
		})
	})
})
