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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each panel by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := panel.Panel{Project: &proj.Key, Name: "resource-panel"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{p.ResourceID()})).
			To(Equal([]resource.Resource{{ID: p.ResourceID(), Name: p.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypePanel, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid panel key "pump"`)),
		))
	})

	It("Should leave out an ID that names no panel", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := panel.Panel{Project: &proj.Key, Name: "listed-panel"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		missing := resource.ID{Type: resource.TypePanel, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, p.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: p.ResourceID(), Name: p.Name}}))
	})

	It("Should list each panel by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := panel.Panel{Project: &proj.Key, Name: "listed-panel"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: p.ResourceID(), Name: p.Name},
		))
	})
})

var _ = Describe("Resource", func() {
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
			}})).Error().To(MatchError(ContainSubstring("invalid uuid")))
		})
	})

	Describe("Search", func() {
		It("Should search panels by name", func(ctx SpecContext) {
			p := panel.Panel{Name: "turbopump overview", Project: &proj.Key}
			Expect(writer.Create(ctx, &p)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(writer.Delete(ctx, p.Key)).To(Succeed())
			})
			q := search.Query[resource.ID]{Term: "turbopump", Limit: 10}
			Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
				To(Equal([]resource.ID{p.ResourceID()}))
		})
	})
})
