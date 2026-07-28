// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	Describe("Export", func() {
		It("Should export a symbol as a versioned envelope", func(ctx SpecContext) {
			sym := symbol.Symbol{
				Name: "exported",
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			}
			// Export reads committed data, so the symbol is created outside the
			// per-spec tx and must be deleted to keep the shared DB's counts intact.
			Expect(svc.NewWriter(nil).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
			})
			env := MustSucceed(svc.Export(ctx, symbol.OntologyID(sym.Key)))
			Expect(env.Version).To(Equal(symbol.Version))
			Expect(env.Type).To(Equal("schematic_symbol"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[symbol.Symbol](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.Data.SVG).To(Equal("<svg/>"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{
				Type: ontology.ResourceTypeSchematicSymbol,
				Key:  uuid.NewString(),
			}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeSchematicSymbol, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})
})
