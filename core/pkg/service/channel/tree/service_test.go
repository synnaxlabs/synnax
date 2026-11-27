// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tree_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel/tree"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		db     *gorp.DB
		groups *group.Service
		svc    *tree.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc = MustSucceed(tree.OpenService(tree.ServiceConfig{Group: groups}))
	})

	create := func(
		ctx context.Context,
		scope resource.Type,
		parent *group.Group,
	) group.Group {
		GinkgoHelper()
		g := group.Group{Name: uuid.New().String(), Scope: scope}
		if parent != nil {
			g.Parent = new(parent.ResourceID())
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groups.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	deleteFolders := func(ctx context.Context, keys ...group.Key) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.DeleteFolders(ctx, tx, keys...)
		})
	}
	exists := func(ctx context.Context, key group.Key) bool {
		GinkgoHelper()
		return MustSucceed(groups.NewRetrieve().
			Where(group.MatchKeys(key)).
			Exists(ctx, db))
	}

	Describe("DeleteFolders", func() {
		It("Should delete channel folders and the folders inside them", func(
			ctx SpecContext,
		) {
			outer := create(ctx, resource.TypeChannel, nil)
			inner := create(ctx, resource.TypeChannel, &outer)
			Expect(deleteFolders(ctx, outer.Key, inner.Key)).To(Succeed())
			Expect(exists(ctx, outer.Key)).To(BeFalse())
			Expect(exists(ctx, inner.Key)).To(BeFalse())
		})

		It(
			"Should refuse a folder that does not hold channels and delete nothing",
			func(
				ctx SpecContext,
			) {
				channels := create(ctx, resource.TypeChannel, nil)
				schematics := create(ctx, resource.TypeSchematic, nil)
				Expect(deleteFolders(ctx, channels.Key, schematics.Key)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"folder "+schematics.Name+" does not hold channels",
					)),
				))
				Expect(exists(ctx, channels.Key)).To(BeTrue())
				Expect(exists(ctx, schematics.Key)).To(BeTrue())
			},
		)

		It("Should delete nothing for a key that names no folder", func(
			ctx SpecContext,
		) {
			kept := create(ctx, resource.TypeChannel, nil)
			Expect(deleteFolders(ctx, uuid.New())).To(Succeed())
			Expect(exists(ctx, kept.Key)).To(BeTrue())
		})
	})
})
