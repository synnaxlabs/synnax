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
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Paths", Ordered, func() {
	var (
		db  *gorp.DB
		svc *group.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	})
	create := func(ctx SpecContext, g group.Group) group.Group {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	in := func(name string, parent *group.Group) group.Group {
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(parent.ResourceID())
		}
		return g
	}
	var names func(p group.Path) []string
	text := func(p group.Path) string { return strings.Join(names(p), ".") }
	names = func(p group.Path) []string {
		out := make([]string, len(p))
		for i, g := range p {
			out[i] = g.Name
		}
		return out
	}

	It("Should return the groups from the top down to each group", func(
		ctx SpecContext,
	) {
		stand := create(ctx, in("stand", nil))
		sensors := create(ctx, in("sensors", &stand))
		pressure := create(ctx, in("pressure", &sensors))
		other := create(ctx, in("other", &stand))
		paths := MustSucceed(group.Paths(ctx, db, pressure.Key, other.Key, stand.Key))
		Expect(paths).To(HaveLen(3))
		Expect(names(paths[pressure.Key])).To(Equal([]string{
			"stand", "sensors", "pressure",
		}))
		Expect(text(paths[pressure.Key])).To(Equal("stand.sensors.pressure"))
		Expect(text(paths[other.Key])).To(Equal("stand.other"))
		Expect(text(paths[stand.Key])).To(Equal("stand"))
	})

	It("Should stop a path at a group whose parent is not a group", func(
		ctx SpecContext,
	) {
		// No project table is open, so a raw write stores the groups.
		top := group.Group{
			Key:    uuid.New(),
			Name:   "views",
			Scope:  resource.TypeProject,
			Parent: &resource.ID{Type: resource.TypeProject, Key: "p1"},
		}
		below := group.Group{
			Key:    uuid.New(),
			Name:   "plots",
			Scope:  resource.TypeProject,
			Parent: new(top.ResourceID()),
		}
		Expect(gorp.WrapWriter[group.Key, group.Group](db).Set(ctx, top, below)).
			To(Succeed())
		paths := MustSucceed(group.Paths(ctx, db, below.Key))
		Expect(text(paths[below.Key])).To(Equal("views.plots"))
	})

	It("Should see the groups that a transaction staged", func(ctx SpecContext) {
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		top := in("staged", nil)
		Expect(svc.NewWriter(tx).Create(ctx, &top)).To(Succeed())
		below := in("below", &top)
		Expect(svc.NewWriter(tx).Create(ctx, &below)).To(Succeed())
		paths := MustSucceed(group.Paths(ctx, tx, below.Key))
		Expect(text(paths[below.Key])).To(Equal("staged.below"))
	})

	It("Should return no paths for no keys", func(ctx SpecContext) {
		Expect(group.Paths(ctx, db)).To(BeEmpty())
	})

	It("Should fail when a group does not exist", func(ctx SpecContext) {
		Expect(group.Paths(ctx, db, uuid.New())).Error().
			To(MatchError(query.ErrNotFound))
	})
})
