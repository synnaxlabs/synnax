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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Ontology", func() {
	Describe("ID helpers", func() {
		It("Should build a resource ID of the panel type", func() {
			key := uuid.New()
			id := panel.ResourceID(key)
			Expect(id.Type).To(Equal(resource.TypePanel))
			Expect(id.Key).To(Equal(key.String()))
		})

		It("Should map keys and panels to resource IDs", func() {
			a, b := uuid.New(), uuid.New()
			Expect(panel.ResourceIDs([]panel.Key{a, b})).To(Equal([]resource.ID{
				panel.ResourceID(a), panel.ResourceID(b),
			}))
			Expect(
				panel.ResourceIDsFromPanels([]panel.Panel{{Key: a}, {Key: b}}),
			).To(Equal(
				[]resource.ID{panel.ResourceID(a), panel.ResourceID(b)},
			))
		})

		It("Should round-trip resource IDs back to keys", func() {
			a, b := uuid.New(), uuid.New()
			Expect(panel.KeysFromResourceIDs([]resource.ID{
				panel.ResourceID(a), panel.ResourceID(b),
			})).To(Equal([]panel.Key{a, b}))
		})

		It("Should error when a resource ID key is not a UUID", func() {
			Expect(panel.KeysFromResourceIDs([]resource.ID{{
				Type: resource.TypePanel, Key: "not-a-uuid",
			}})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`invalid panel key "not-a-uuid"`)),
			))
		})
	})

	Describe("Service interface", func() {
		It("Should report the panel resource type", func() {
			Expect(svc.Type()).To(Equal(resource.TypePanel))
		})

		It("Should retrieve a panel as an ontology resource", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			p := panel.Panel{Name: "resource", Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, p.Key.String(), tx))
			Expect(res.ID).To(Equal(p.ResourceID()))
			Expect(res.Name).To(Equal("resource"))
		})

		It("Should carry the icon and the color", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			c := MustSucceed(color.FromHex("#dc136c"))
			p := panel.Panel{
				Name: "styled", Project: &proj.Key, Icon: new("valve"), Color: &c,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, p.Key.String(), tx))
			Expect(res.Data).To(HaveKeyWithValue("icon", "valve"))
			var got color.Color
			Expect(color.Schema.Parse(res.Data.(map[string]any)["color"], &got)).
				To(Succeed())
			Expect(got).To(Equal(c))
		})

		It("Should omit an unset icon and color", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			p := panel.Panel{Name: "plain", Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, p.Key.String(), tx))
			Expect(res.Data).ToNot(HaveKey("icon"))
			Expect(res.Data).ToNot(HaveKey("color"))
		})

		It(
			"Should error when retrieving a resource with a non-UUID key",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				Expect(svc.RetrieveResource(ctx, "not-a-uuid", tx)).Error().
					To(MatchError(ContainSubstring("invalid uuid")))
			},
		)

		It("Should emit a Set change when a panel is created", func(ctx SpecContext) {
			var (
				mu      sync.Mutex
				changes []ontology.Change
			)
			DeferCleanup(
				svc.OnChange(func(_ context.Context, seq iter.Seq[ontology.Change]) {
					mu.Lock()
					defer mu.Unlock()
					changes = append(changes, slices.Collect(seq)...)
				}),
			)
			p := panel.Panel{Name: "observed", Project: &proj.Key}
			Expect(writer.Create(ctx, &p)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(writer.Delete(ctx, p.Key)).To(Succeed())
			})
			Eventually(func(g Gomega) {
				mu.Lock()
				defer mu.Unlock()
				g.Expect(changes).To(ContainElement(SatisfyAll(
					HaveField("Variant", Equal(change.VariantSet)),
					HaveField("Key", Equal(p.ResourceID().String())),
					HaveField("Value.Name", Equal("observed")),
				)))
			}).Should(Succeed())
		})

		It("Should search panels by name", func(ctx SpecContext) {
			p := panel.Panel{Name: "turbopump overview", Project: &proj.Key}
			Expect(writer.Create(ctx, &p)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(writer.Delete(ctx, p.Key)).To(Succeed())
			})
			q := search.Query[resource.ID]{Term: "turbopump", Limit: 10}
			Expect(MustSucceed(svc.Search(ctx, db, q)).Keys).
				To(Equal([]resource.ID{p.ResourceID()}))
		})
	})
})
