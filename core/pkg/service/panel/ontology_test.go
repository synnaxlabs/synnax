// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"context"
	"iter"
	"slices"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/change"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", func() {
	Describe("ID helpers", func() {
		It("Should build an ontology ID of the panel type", func() {
			key := uuid.New()
			id := panel.OntologyID(key)
			Expect(id.Type).To(Equal(ontology.ResourceTypePanel))
			Expect(id.Key).To(Equal(key.String()))
		})

		It("Should map keys and panels to ontology IDs", func() {
			a, b := uuid.New(), uuid.New()
			Expect(panel.OntologyIDs([]panel.Key{a, b})).To(Equal([]ontology.ID{
				panel.OntologyID(a), panel.OntologyID(b),
			}))
			Expect(panel.OntologyIDsFromPanels([]panel.Panel{{Key: a}, {Key: b}})).To(Equal(
				[]ontology.ID{panel.OntologyID(a), panel.OntologyID(b)},
			))
		})

		It("Should round-trip ontology IDs back to keys", func() {
			a, b := uuid.New(), uuid.New()
			Expect(panel.KeysFromOntologyIDs([]ontology.ID{
				panel.OntologyID(a), panel.OntologyID(b),
			})).To(Equal([]panel.Key{a, b}))
		})

		It("Should error when an ontology ID key is not a UUID", func() {
			Expect(panel.KeysFromOntologyIDs([]ontology.ID{{
				Type: ontology.ResourceTypePanel, Key: "not-a-uuid",
			}})).Error().To(MatchError(ContainSubstring("invalid UUID")))
		})
	})

	Describe("Service interface", func() {
		It("Should report the panel resource type", func() {
			Expect(svc.Type()).To(Equal(ontology.ResourceTypePanel))
		})

		It("Should retrieve a panel as an ontology resource", func(ctx SpecContext) {
			p := panel.Panel{Name: "resource"}
			Expect(svc.NewWriter(tx).Create(ctx, &p, parentID)).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, p.Key.String(), tx))
			Expect(res.ID).To(Equal(panel.OntologyID(p.Key)))
			Expect(res.Name).To(Equal("resource"))
		})

		It("Should error when retrieving a resource with a non-UUID key", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, "not-a-uuid", tx)).Error().
				To(MatchError(ContainSubstring("invalid UUID")))
		})

		It("Should emit a Set change when a panel is created", func(ctx SpecContext) {
			var (
				mu      sync.Mutex
				changes []ontology.Change
			)
			DeferCleanup(svc.OnChange(func(_ context.Context, seq iter.Seq[ontology.Change]) {
				mu.Lock()
				defer mu.Unlock()
				changes = append(changes, slices.Collect(seq)...)
			}))
			p := panel.Panel{Name: "observed"}
			Expect(svc.NewWriter(nil).Create(ctx, &p, parentID)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) { Expect(svc.NewWriter(nil).Delete(ctx, p.Key)).To(Succeed()) })
			Eventually(func(g Gomega) {
				mu.Lock()
				defer mu.Unlock()
				g.Expect(changes).To(ContainElement(SatisfyAll(
					HaveField("Variant", Equal(change.VariantSet)),
					HaveField("Key", Equal(panel.OntologyID(p.Key).String())),
					HaveField("Value.Name", Equal("observed")),
				)))
			}).Should(Succeed())
		})

		It("Should iterate existing panels via OpenNexter", func(ctx SpecContext) {
			p := panel.Panel{Name: "nexted"}
			Expect(svc.NewWriter(nil).Create(ctx, &p, parentID)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) { Expect(svc.NewWriter(nil).Delete(ctx, p.Key)).To(Succeed()) })
			next, closer := MustSucceed2(svc.OpenNexter(ctx))
			defer func() { Expect(closer.Close()).To(Succeed()) }()
			var ids []ontology.ID
			for res := range next {
				ids = append(ids, res.ID)
			}
			Expect(ids).To(ContainElement(panel.OntologyID(p.Key)))
		})
	})
})
