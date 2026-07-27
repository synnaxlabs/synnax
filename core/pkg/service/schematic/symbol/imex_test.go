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
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// wireRoundTrip marshals env to JSON and back, binding the codec Decode needs. Exported
// envelopes carry a body but no codec, so a decode must first pass through the wire.
func wireRoundTrip(env imex.Envelope) imex.Envelope {
	b := MustSucceed(json.Marshal(env))
	var out imex.Envelope
	Expect(json.Unmarshal(b, &out)).To(Succeed())
	return out
}

var _ = Describe("ImEx", Ordered, func() {
	// A self-contained service keeps the committed export symbol out of the shared
	// suite DB, whose retrieve specs count every symbol.
	var svc *symbol.Service
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		exportDB := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: exportDB}))
		searchIdx := MustOpen(search.OpenIndex())
		svc = MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
			DB:       exportDB,
			Ontology: otg,
			Search:   searchIdx,
			ImEx:     imex.NewService(),
		}))
	})

	Describe("Export", func() {
		It("Should export a symbol as a versioned envelope", func(ctx SpecContext) {
			sym := symbol.Symbol{Name: "exported", Data: map[string]any{"svg": "<svg/>"}}
			Expect(svc.NewWriter(nil).Create(ctx, &sym, ontology.RootID)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, symbol.OntologyID(sym.Key)))
			Expect(env.Version).To(Equal(symbol.Version))
			Expect(env.Type).To(Equal("schematic_symbol"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[symbol.Symbol](ctx, wireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.Data).To(HaveKeyWithValue("svg", "<svg/>"))
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
