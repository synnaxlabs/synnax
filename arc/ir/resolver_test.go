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
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("MapResolver", func() {
	Describe("Resolve", func() {
		It("Should resolve existing symbol", func() {
			resolver := ir.MapResolver{
				"pi":    ir.Symbol{Name: "pi", Kind: ir.KindConfigParam, Type: ir.F64{}},
				"count": ir.Symbol{Name: "count", Kind: ir.KindVariable, Type: ir.I32{}},
			}
			sym, err := resolver.Resolve(ctx, "pi")
			Expect(err).ToNot(HaveOccurred())
			Expect(sym.Name).To(Equal("pi"))
			Expect(sym.Kind).To(Equal(ir.KindConfigParam))
			Expect(sym.Type).To(Equal(ir.F64{}))
		})

		It("Should return error for non-existent symbol", func() {
			resolver := ir.MapResolver{
				"x": ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}},
			}
			_, err := resolver.Resolve(ctx, "y")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})

		It("Should work with empty resolver", func() {
			resolver := ir.MapResolver{}
			_, err := resolver.Resolve(ctx, "anything")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})
	})
})
