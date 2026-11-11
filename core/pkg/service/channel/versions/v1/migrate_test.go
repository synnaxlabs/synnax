// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("IndexGroupMigration", func() {
	var (
		db      *gorp.DB
		g, user = uuid.New(), uuid.New()
	)
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	migrate := func(ctx context.Context, namespace string) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  namespace,
			Migrations: []gorp.Migration{v1.IndexGroupMigration},
		})).To(Succeed())
	}
	groupOf := func(ctx context.Context, localKey v1.LocalKey) *uuid.UUID {
		GinkgoHelper()
		var ch v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Where(gorp.MatchKeys[v1.Key, v1.Channel](
				v1.Channel{Leaseholder: node.KeyFree, LocalKey: localKey}.Key(),
			)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch.Group
	}

	It("Should move each ungrouped index into its calculated channel's group",
		func(ctx SpecContext) {
			free := func(local, index v1.LocalKey, expr string, grp *uuid.UUID) v1.Channel {
				return v1.Channel{
					Name:        uuid.New().String(),
					Leaseholder: node.KeyFree,
					LocalKey:    local,
					LocalIndex:  index,
					IsIndex:     local == index,
					Virtual:     true,
					Expression:  expr,
					Group:       grp,
				}
			}
			Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(
				ctx,
				free(1, 1, "", nil),
				free(2, 1, "return a", &g),
				free(3, 3, "", &user),
				free(4, 3, "return b", &g),
				free(5, 5, "", nil),
				free(6, 5, "return c", nil),
			)).To(Succeed())
			migrate(ctx, "first")
			Expect(groupOf(ctx, 1)).To(HaveValue(Equal(g)))
			Expect(groupOf(ctx, 3)).To(HaveValue(Equal(user)))
			Expect(groupOf(ctx, 5)).To(BeNil())
			By("Changing nothing on a second run")
			migrate(ctx, "second")
			Expect(groupOf(ctx, 1)).To(HaveValue(Equal(g)))
			Expect(groupOf(ctx, 3)).To(HaveValue(Equal(user)))
			Expect(groupOf(ctx, 5)).To(BeNil())
		},
	)
})
