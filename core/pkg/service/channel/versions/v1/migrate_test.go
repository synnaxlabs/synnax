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
	"github.com/samber/lo"
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
	groupOf := func(
		ctx context.Context,
		localKey v1.LocalKey,
		leaseholder ...node.Key,
	) *uuid.UUID {
		GinkgoHelper()
		var ch v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Where(gorp.MatchKeys[v1.Key, v1.Channel](v1.Channel{
				Leaseholder: lo.FirstOr(leaseholder, node.KeyFree),
				LocalKey:    localKey,
			}.Key())).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch.Group
	}

	It("Should move each ungrouped index into its calculated channel's group",
		func(ctx SpecContext) {
			free := func(
				name string,
				local, index v1.LocalKey,
				expr string,
				grp *uuid.UUID,
			) v1.Channel {
				return v1.Channel{
					Name:        name,
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
				free("a_time", 1, 1, "", nil),
				free("a", 2, 1, "return x", &g),
				free("b_time", 3, 3, "", &user),
				free("b", 4, 3, "return y", &g),
				free("c_time", 5, 5, "", nil),
				free("c", 6, 5, "return z", nil),
				free("renamed_time", 7, 7, "", nil),
				free("d", 8, 7, "return w", &g),
				free("e_time", 9, 9, "", nil),
				free("f_time", 10, 10, "", nil),
				free("f", 11, 10, "return v", &g),
				free("data", 12, 10, "", nil),
				free("h", 14, 13, "return u", &g),
				free("a2_time", 15, 15, "", nil),
				free("renamed_a2", 16, 15, "return t", &g),
				free("a2", 17, 15, "return s", &g),
				v1.Channel{
					Name:        "sensor_time",
					Leaseholder: 1,
					LocalKey:    13,
					IsIndex:     true,
				},
			),
			).To(Succeed())
			expectGroups := func() {
				GinkgoHelper()
				Expect(groupOf(ctx, 1)).To(HaveValue(Equal(g)))
				Expect(groupOf(ctx, 3)).To(HaveValue(Equal(user)))
				Expect(groupOf(ctx, 5)).To(BeNil())
				Expect(groupOf(ctx, 7)).To(HaveValue(Equal(g)))
				Expect(groupOf(ctx, 9)).To(BeNil())
				Expect(groupOf(ctx, 10)).To(BeNil())
				Expect(groupOf(ctx, 13, 1)).To(BeNil())
				Expect(groupOf(ctx, 15)).To(BeNil())
			}
			migrate(ctx, "first")
			expectGroups()
			By("Changing nothing on a second run")
			migrate(ctx, "second")
			expectGroups()
		},
	)
})
