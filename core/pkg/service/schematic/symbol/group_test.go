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
	"fmt"
	"sync"
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// createGroup creates a symbol group, under parent when one is given.
func createGroup(ctx SpecContext, parent *group.Group) group.Group {
	GinkgoHelper()
	g := group.Group{Name: "deletable", Scope: resource.TypeSchematicSymbol}
	if parent != nil {
		g.Parent = new(parent.ResourceID())
	}
	Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
	})
	return g
}

func createSymbol(ctx SpecContext, g group.Group) symbol.Symbol {
	GinkgoHelper()
	sym := symbol.Symbol{
		Name:  "Inlet",
		Data:  symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		Group: &g.Key,
	}
	Expect(svc.NewWriter(nil).Create(ctx, &sym)).To(Succeed())
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
		g := createGroup(ctx, nil)
		sym := createSymbol(ctx, g)
		tx := DeferClose(db.OpenTx())
		Expect(deleteGroup(ctx, tx, g.Key)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
	})
	It("Should delete an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, nil)
		tx := DeferClose(db.OpenTx())
		Expect(deleteGroup(ctx, tx, g.Key)).To(Succeed())
	})
	It(
		"Should refuse to delete a group that holds another group",
		func(ctx SpecContext) {
			g := createGroup(ctx, nil)
			createGroup(ctx, &g)
			tx := DeferClose(db.OpenTx())
			Expect(deleteGroup(ctx, tx, g.Key)).To(MatchError(
				ContainSubstring(
					"cannot delete group deletable because it is not empty",
				),
			))
		},
	)
	It("Should reject an id no symbol key can come from", func(ctx SpecContext) {
		g := createGroup(ctx, nil)
		tx := DeferClose(db.OpenTx())
		Expect(svc.DeleteGroup(ctx, tx, g.Key, []resource.ID{
			{Type: resource.TypeSchematicSymbol, Key: "not-a-uuid"},
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`key "not-a-uuid"`)),
		))
	})
})

var _ = Describe("RetrieveGroupSymbols", func() {
	It("Should return the group's symbols", func(ctx SpecContext) {
		g := createGroup(ctx, nil)
		sym := createSymbol(ctx, g)
		tx := DeferClose(db.OpenTx())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).
			To(ConsistOf(symbol.ResourceID(sym.Key)))
	})
	It("Should return nothing for an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, nil)
		tx := DeferClose(db.OpenTx())
		Expect(svc.RetrieveGroupSymbols(ctx, tx, g.Key)).To(BeEmpty())
	})
	It("Should skip a child that is not a schematic symbol", func(ctx SpecContext) {
		g := createGroup(ctx, nil)
		createGroup(ctx, &g)
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

var _ = Describe("Writer with no transaction", func() {
	It(
		"Should keep the group that a move writes during concurrent creates",
		func(ctx SpecContext) {
			groups := make([]group.Group, 50)
			for i := range groups {
				groups[i] = group.Group{
					Name:  fmt.Sprintf("Concurrent %d", i),
					Scope: resource.TypeSchematicSymbol,
				}
				Expect(groupSvc.NewWriter(nil).Create(ctx, &groups[i])).To(Succeed())
			}
			DeferCleanup(func(ctx SpecContext) {
				for _, g := range groups {
					Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
				}
			})
			created := symbol.Symbol{
				Name: "concurrent",
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			}
			Expect(svc.NewWriter(nil).Create(ctx, &created)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, created.Key)).To(Succeed())
			})
			retrieveGroup := func() *group.Key {
				GinkgoHelper()
				var res symbol.Symbol
				Expect(svc.NewRetrieve().
					Where(symbol.MatchKeys(created.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				return res.Group
			}
			var (
				wg    sync.WaitGroup
				moved atomic.Bool
			)
			for range 4 {
				wg.Go(func() {
					defer GinkgoRecover()
					for !moved.Load() {
						again := created
						Expect(svc.NewWriter(nil).Create(ctx, &again)).
							To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(svc.NewWriter(nil).SetGroup(ctx, &g.Key, created.Key)).
						To(Succeed())
					Expect(retrieveGroup()).To(Equal(&g.Key))
				}
			})
			wg.Wait()
			Expect(retrieveGroup()).To(Equal(&groups[len(groups)-1].Key))
		},
	)
})
