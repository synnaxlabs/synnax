// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Sibling breaker 2", func() {
	badParent := func() group.Group {
		return group.Group{
			Key:    uuid.New(),
			Name:   "bad parent " + uuid.New().String(),
			Scope:  resource.TypeChannel,
			Parent: &resource.ID{Type: resource.TypeGroup, Key: "not-a-uuid"},
		}
	}

	DescribeTable(
		"Should refuse a channel folder under a parent that is not a group "+
			"before it reaches the sibling index",
		func(ctx SpecContext, parent resource.ID, parentScope string) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			svc := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
			g := group.Group{Name: "pt", Scope: resource.TypeChannel, Parent: &parent}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &g)
			})).To(MatchError(ContainSubstring(
				"a channel group cannot sit under a " + parentScope + " parent",
			)))
		},
		Entry(
			"a project",
			resource.ID{Type: resource.TypeProject, Key: "p"},
			"project",
		),
		Entry("a rack", resource.ID{Type: resource.TypeRack, Key: "1"}, "device"),
	)

	It("Should not panic on a raw write of a channel folder with a bad parent key",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
			g := badParent()
			var err error
			Expect(func() {
				err = db.WithTx(ctx, func(tx gorp.Tx) error {
					return gorp.WrapWriter[group.Key, group.Group](tx).Set(ctx, g)
				})
			}).ToNot(Panic())
			Expect(err).To(Succeed())
		},
	)

	It("Should not panic at open over a stored channel folder with a bad parent key",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			// The first open runs the migrations, so the reopen keeps the folder.
			first := MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db}))
			Expect(first.Close()).To(Succeed())
			g := badParent()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.WrapWriter[group.Key, group.Group](tx).Set(ctx, g)
			})).To(Succeed())
			svc := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
			// The create waits for the index scan of the stored folder, which exempts
			// the folder whose parent is not a group.
			other := group.Group{Name: "other", Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
		},
	)

	It("Should keep a name of a scope that is not namespaced raw in CreateOrRetrieve",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			svc := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
			suffix := uuid.New().String()
			nfd := MustSucceed(svc.CreateOrRetrieve(
				ctx, "A\u030A "+suffix, resource.TypeProject,
			))
			Expect(nfd.Name).To(Equal("A\u030A " + suffix))
			again := MustSucceed(svc.CreateOrRetrieve(
				ctx, "A\u030A "+suffix, resource.TypeProject,
			))
			Expect(again.Key).To(Equal(nfd.Key))
		},
	)

	// openOverNFD opens a group service over a stored root channel folder whose name a
	// v0.62 Core kept as sent, in NFD.
	openOverNFD := func(ctx SpecContext) (*gorp.DB, *group.Service, group.Group) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		first := MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		Expect(first.Close()).To(Succeed())
		stored := group.Group{
			Key:   uuid.New(),
			Name:  "Stand A\u030A",
			Scope: resource.TypeChannel,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[group.Key, group.Group](tx).Set(ctx, stored)
		})).To(Succeed())
		return db, MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db})), stored
	}

	It("Should refuse a root folder beside a stored root folder of its name in NFD",
		func(ctx SpecContext) {
			db, svc, _ := openOverNFD(ctx)
			again := group.Group{Name: "Stand \u00C5", Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &again)
			})).To(MatchError(ContainSubstring("already exists at the top level")))
		},
	)
})
