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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

func createGroup(ctx SpecContext, parent resource.ID) group.Group {
	GinkgoHelper()
	g := MustSucceed(groupSvc.NewWriter(nil).Create(ctx, "deletable", parent))
	DeferCleanup(func(ctx SpecContext) {
		Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
	})
	return g
}

func createSymbol(ctx SpecContext, g group.Group) symbol.Symbol {
	GinkgoHelper()
	sym := symbol.Symbol{
		Name: "Inlet",
		Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
	}
	Expect(svc.NewWriter(nil).Create(ctx, &sym, g.ResourceID())).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
	})
	return sym
}

var _ = Describe("DeleteGroup", func() {
	// The api reads the members through the same tx it deletes under, which every spec
	// below mirrors.
	deleteGroup := func(ctx SpecContext, tx gorp.Tx, key group.Key) error {
		GinkgoHelper()
		return svc.DeleteGroup(ctx, tx, key, MustSucceed(
			svc.RetrieveGroupSymbols(ctx, tx, key),
		))
	}
	It("Should delete the group and every symbol in it", func(ctx SpecContext) {
		g := createGroup(ctx, proj.ResourceID())
		sym := createSymbol(ctx, g)
		tx := DeferClose(db.OpenTx())
		Expect(deleteGroup(ctx, tx, g.Key)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
	})
	It("Should delete an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, proj.ResourceID())
		tx := DeferClose(db.OpenTx())
		Expect(deleteGroup(ctx, tx, g.Key)).To(Succeed())
	})
	It("Should move a child that is not a schematic symbol to the permanent group",
		func(ctx SpecContext) {
			g := createGroup(ctx, proj.ResourceID())
			stray := createGroup(ctx, g.ResourceID())
			tx := DeferClose(db.OpenTx())
			Expect(deleteGroup(ctx, tx, g.Key)).To(Succeed())
			var parents []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(stray.ResourceID()).
				TraverseTo(ontology.ParentsTraverser).
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).To(HaveLen(1))
			Expect(parents[0].ID).To(Equal(svc.Group().ResourceID()))
		},
	)
	It("Should reject an id no symbol key can come from", func(ctx SpecContext) {
		g := createGroup(ctx, proj.ResourceID())
		tx := DeferClose(db.OpenTx())
		Expect(svc.DeleteGroup(ctx, tx, g.Key, []resource.ID{
			{Type: resource.TypeSchematicSymbol, Key: "not-a-uuid"},
		})).To(MatchError(ContainSubstring("invalid uuid")))
	})
})

var _ = Describe("RetrieveGroupSymbols", func() {
	It("Should return the group's symbols", func(ctx SpecContext) {
		g := createGroup(ctx, proj.ResourceID())
		sym := createSymbol(ctx, g)
		tx := DeferClose(db.OpenTx())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).
			To(ConsistOf(symbol.ResourceID(sym.Key)))
	})
	It("Should return nothing for an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, proj.ResourceID())
		tx := DeferClose(db.OpenTx())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).To(BeEmpty())
	})
	It("Should skip a child that is not a schematic symbol", func(ctx SpecContext) {
		g := createGroup(ctx, proj.ResourceID())
		createGroup(ctx, g.ResourceID())
		sym := createSymbol(ctx, g)
		tx := DeferClose(db.OpenTx())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).
			To(ConsistOf(symbol.ResourceID(sym.Key)))
	})
	It("Should return not found for a missing group", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, uuid.New())).Error().
			To(MatchError(query.ErrNotFound))
	})
})
