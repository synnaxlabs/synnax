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
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// createGroup creates a symbol group, under parent when one is given.
func createGroup(ctx SpecContext, parent *group.Group) group.Group {
	GinkgoHelper()
	g := group.Group{Name: "deletable", Scope: resource.TypeSchematicSymbol}
	if parent != nil {
		g.Parent = new(parent.ResourceID())
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return groupSvc.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
		})).To(Succeed())
	})
	return g
}

func createSymbol(ctx SpecContext, g group.Group) symbol.Symbol {
	GinkgoHelper()
	sym := defaults.Fill(symbol.Symbol{
		Name:  "Inlet",
		Data:  symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		Group: &g.Key,
	})
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.NewWriter(tx).Create(ctx, &sym)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, sym.Key)
		})).To(Succeed())
	})
	return sym
}

var _ = Describe("DeleteGroup", func() {
	It("Should delete the group and every symbol in it", func(ctx SpecContext) {
		g := createGroup(ctx, nil)
		sym := createSymbol(ctx, g)
		tx := DeferClose(db.OpenTx())
		Expect(svc.DeleteGroup(ctx, tx, g.Key)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
	})
	It("Should delete an empty group", func(ctx SpecContext) {
		g := createGroup(ctx, nil)
		tx := DeferClose(db.OpenTx())
		Expect(svc.DeleteGroup(ctx, tx, g.Key)).To(Succeed())
	})
	It(
		"Should refuse to delete a group that holds another group",
		func(ctx SpecContext) {
			g := createGroup(ctx, nil)
			createGroup(ctx, &g)
			tx := DeferClose(db.OpenTx())
			Expect(svc.DeleteGroup(ctx, tx, g.Key)).To(MatchError(
				ContainSubstring(
					"cannot delete group deletable because it is not empty",
				),
			))
		},
	)
	It("Should return not found for a missing group", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		Expect(svc.DeleteGroup(ctx, tx, uuid.New())).To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Writer in concurrent transactions", func() {
	It(
		"Should keep the group that a move writes during concurrent applies",
		func(ctx SpecContext) {
			groups := make([]group.Group, 50)
			for i := range groups {
				groups[i] = group.Group{
					Name:  fmt.Sprintf("Concurrent %d", i),
					Scope: resource.TypeSchematicSymbol,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return groupSvc.NewWriter(tx).Create(ctx, &groups[i])
				})).To(Succeed())
			}
			DeferCleanup(func(ctx SpecContext) {
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
					})).To(Succeed())
				}
			})
			created := defaults.Fill(symbol.Symbol{
				Name: "concurrent",
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &created)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, created.Key)
				})).To(Succeed())
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
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Rename(ctx, created.Key, "renamed")
						})).
							To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).SetGroup(ctx, &g.Key, created.Key)
					})).
						To(Succeed())
					Expect(retrieveGroup()).To(Equal(&g.Key))
				}
			})
			wg.Wait()
			Expect(retrieveGroup()).To(Equal(&groups[len(groups)-1].Key))
		},
	)
})
