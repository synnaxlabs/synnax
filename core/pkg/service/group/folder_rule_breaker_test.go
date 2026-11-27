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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/group/versions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Folder rule breaker", func() {
	var db *gorp.DB
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
	})
	// openOver stores a top-level channel folder named name, as the upgrade keeps a
	// v0.62 name that breaks the rule but not a path, and then opens the service.
	openOver := func(ctx SpecContext, name string) (*group.Service, group.Group) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  types.Name[group.Group](),
			Migrations: versions.Migrations,
		})).To(Succeed())
		stored := group.Group{Key: uuid.New(), Name: name, Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[group.Key, group.Group](tx).Set(ctx, stored)
		})).To(Succeed())
		svc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:        db,
			NameRules: nameRules,
		}))
		return svc, stored
	}
	write := func(
		ctx context.Context,
		svc *group.Service,
		f func(w group.Writer) error,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error { return f(svc.NewWriter(tx)) })
	}

	DescribeTable("Should move a stored channel folder whose kept name breaks the rule",
		func(ctx SpecContext, name string) {
			svc, stored := openOver(ctx, name)
			rigs := group.Group{Name: "Rigs", Scope: resource.TypeChannel}
			Expect(write(ctx, svc, func(w group.Writer) error {
				return w.Create(ctx, &rigs)
			})).To(Succeed())
			parent := group.ResourceID(rigs.Key)
			Expect(write(ctx, svc, func(w group.Writer) error {
				return w.SetParent(ctx, &parent, stored.Key)
			})).To(Succeed())
		},
		Entry("a zero-width space", "Stand\u200BA"),
		Entry("a no-break space alone", "\u00A0"),
		Entry("a tab", "Stand\tA"),
		Entry("spaces alone", "   "),
	)

	It("Should rename a stored channel folder to the kept name that it holds",
		func(ctx SpecContext) {
			svc, stored := openOver(ctx, "Stand\u200BA")
			Expect(write(ctx, svc, func(w group.Writer) error {
				return w.Rename(ctx, stored.Key, stored.Name)
			})).To(Succeed())
		})

	It("Should refuse a config with no name rule for the channel scope",
		func(ctx SpecContext) {
			svc, err := group.OpenService(ctx, group.ServiceConfig{DB: db})
			if err == nil {
				DeferClose(svc)
			}
			Expect(err).To(MatchError(ContainSubstring("channel")))
		})
})
