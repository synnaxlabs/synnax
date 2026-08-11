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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func createGroup(ctx SpecContext, parent ontology.ID) group.Group {
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
	Expect(svc.NewWriter(nil).Create(ctx, &sym, g.OntologyID())).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
	})
	return sym
}

var _ = Describe("DeleteGroup", func() {
	// The api reads the members through the same tx it deletes under, which every spec
	// below mirrors.
	deleteGroup := func(ctx SpecContext, key group.Key) error {
		GinkgoHelper()
		return svc.DeleteGroup(ctx, tx, key, MustSucceed(
			svc.RetrieveGroupSymbols(ctx, tx, key),
		))
	}
	It("Should delete the group and every symbol in it", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		sym := createSymbol(ctx, g)
		Expect(deleteGroup(ctx, g.Key)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
	})
	It("Should delete an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		Expect(deleteGroup(ctx, g.Key)).To(Succeed())
	})
	It("Should reject an id no symbol key can come from", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		Expect(svc.DeleteGroup(ctx, tx, g.Key, []ontology.ID{
			{Type: ontology.ResourceTypeSchematicSymbol, Key: "not-a-uuid"},
		})).To(MatchError(ContainSubstring("invalid UUID")))
	})
})

var _ = Describe("RetrieveGroupSymbols", func() {
	It("Should return the group's symbols", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		sym := createSymbol(ctx, g)
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).
			To(ConsistOf(symbol.OntologyID(sym.Key)))
	})
	It("Should return nothing for an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).To(BeEmpty())
	})
	It("Should reject a group holding a non-symbol child", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		createGroup(ctx, g.OntologyID())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("not a schematic symbol")),
		))
	})
	It("Should return not found for a missing group", func(ctx SpecContext) {
		Expect(svc.RetrieveGroupSymbols(ctx, tx, uuid.New())).Error().
			To(MatchError(query.ErrNotFound))
	})
})
