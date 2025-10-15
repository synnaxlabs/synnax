// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("MapResolver", func() {
	Describe("Resolve", func() {
		It("Should resolve existing symbol", func() {
			resolver := symbol.MapResolver{
				"pi":    Symbol{Name: "pi", Kind: SymbolKindConfigParam, Type: types.F64()},
				"count": Symbol{Name: "count", Kind: SymbolKindVariable, Type: types.I32{}},
			}
			sym, err := resolver.Resolve(ir.ctx, "pi")
			Expect(err).ToNot(HaveOccurred())
			Expect(sym.Name).To(Equal("pi"))
			Expect(sym.Kind).To(Equal(SymbolKindConfigParam))
			Expect(sym.Type).To(Equal(types.F64()))
		})

		It("Should return error for non-existent symbol", func() {
			resolver := symbol.MapResolver{
				"x": Symbol{Name: "x", Kind: SymbolKindVariable, Type: types.I32{}},
			}
			_, err := resolver.Resolve(ir.ctx, "y")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})

		It("Should work with empty resolver", func() {
			resolver := symbol.MapResolver{}
			_, err := resolver.Resolve(ir.ctx, "anything")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})
	})
})
