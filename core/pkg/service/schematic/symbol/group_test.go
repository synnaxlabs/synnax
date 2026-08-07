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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("DeleteGroup", func() {
	// Deletes run on the per-spec tx so every row rolls back, but the fixtures they
	// read must be committed, so each one is created outside the tx.
	createGroup := func(ctx SpecContext, parent ontology.ID) group.Group {
		GinkgoHelper()
		g := MustSucceed(groupSvc.NewWriter(nil).Create(ctx, "deletable", parent))
		DeferCleanup(func(ctx SpecContext) {
			// The spec may already have deleted it inside a committed tx.
			Expect(errors.Skip(
				groupSvc.NewWriter(nil).Delete(ctx, g.Key), query.ErrNotFound,
			)).To(Succeed())
		})
		return g
	}
	createSymbol := func(ctx SpecContext, g group.Group) symbol.Symbol {
		GinkgoHelper()
		sym := symbol.Symbol{
			Name: "Inlet",
			Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		}
		Expect(svc.NewWriter(nil).Create(ctx, &sym, g.OntologyID())).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(errors.Skip(
				svc.NewWriter(nil).Delete(ctx, sym.Key), query.ErrNotFound,
			)).To(Succeed())
		})
		return sym
	}

	It("Should delete the group and every symbol in it", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		sym := createSymbol(ctx, g)
		Expect(svc.DeleteGroup(ctx, tx, g.Key)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
	})
	It("Should delete an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		Expect(svc.DeleteGroup(ctx, tx, g.Key)).To(Succeed())
	})
	It("Should return not found for a missing group", func(ctx SpecContext) {
		Expect(svc.DeleteGroup(ctx, tx, uuid.New())).To(MatchError(query.ErrNotFound))
	})
	It("Should refuse a group holding a non-symbol child", func(ctx SpecContext) {
		g := createGroup(ctx, proj.OntologyID())
		createGroup(ctx, g.OntologyID())
		Expect(svc.DeleteGroup(ctx, tx, g.Key)).To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("RetrieveGroupSymbols", func() {
	It("Should return only the group's symbol children", func(ctx SpecContext) {
		g := MustSucceed(
			groupSvc.NewWriter(nil).Create(ctx, "members", proj.OntologyID()),
		)
		nested := MustSucceed(
			groupSvc.NewWriter(nil).Create(ctx, "nested", g.OntologyID()),
		)
		sym := symbol.Symbol{
			Name: "Inlet",
			Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		}
		Expect(svc.NewWriter(nil).Create(ctx, &sym, g.OntologyID())).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
			Expect(groupSvc.NewWriter(nil).Delete(ctx, nested.Key)).To(Succeed())
			Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
		})
		Expect(svc.RetrieveGroupSymbols(ctx, g.Key)).
			To(ConsistOf(symbol.OntologyID(sym.Key)))
	})
	It("Should return not found for a missing group", func(ctx SpecContext) {
		Expect(svc.RetrieveGroupSymbols(ctx, uuid.New())).Error().
			To(MatchError(query.ErrNotFound))
	})
})
