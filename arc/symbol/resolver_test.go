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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("MapResolver", func() {
	Describe("Resolve", func() {
		It("Should resolve existing symbol", func() {
			resolver := symbol.MapResolver{
				"pi":    symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
				"count": symbol.Symbol{Name: "count", Kind: symbol.KindVariable, Type: types.I32()},
			}
			sym, err := resolver.Resolve(bCtx, "pi")
			Expect(err).ToNot(HaveOccurred())
			Expect(sym.Name).To(Equal("pi"))
			Expect(sym.Kind).To(Equal(symbol.KindConfig))
			Expect(sym.Type).To(Equal(types.F64()))
		})

		It("Should return error for non-existent symbol", func() {
			resolver := symbol.MapResolver{
				"x": symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			}
			_, err := resolver.Resolve(bCtx, "y")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})

		It("Should work with empty resolver", func() {
			resolver := symbol.MapResolver{}
			_, err := resolver.Resolve(bCtx, "anything")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})
	})
})
