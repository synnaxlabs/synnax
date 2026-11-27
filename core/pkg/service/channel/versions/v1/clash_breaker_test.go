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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Clash breaker", func() {
	const host node.Key = 1
	var (
		db     *gorp.DB
		first  = uuid.MustParse("00000000-0000-4000-8000-000000000001")
		second = uuid.MustParse("00000000-0000-4000-8000-000000000002")
	)
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	channel := func(local v1.LocalKey, name string, folder *uuid.UUID) v1.Channel {
		return v1.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    local,
			LocalIndex:  local,
			IsIndex:     true,
			Virtual:     true,
			Group:       folder,
		}
	}
	folder := func(key uuid.UUID, name string) groupv1.Group {
		return groupv1.Group{Key: key, Name: name, Scope: resource.TypeChannel}
	}
	run := func(ctx SpecContext, channels []v1.Channel, folders ...groupv1.Group) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(ctx, channels...)).
			To(Succeed())
		Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](db).Set(ctx, folders...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "channel",
			Migrations: []gorp.Migration{
				v1.NameRuleMigration,
				v1.NewClashMigration(host),
				v1.NewCalculationTextMigration(parser.Config{}),
			},
		})).To(Succeed())
	}
	expressionOf := func(ctx SpecContext, local v1.LocalKey) string {
		GinkgoHelper()
		var c v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Where(gorp.MatchKeys[v1.Key, v1.Channel](channel(local, "", nil).Key())).
			Entry(&c).
			Exec(ctx, db)).To(Succeed())
		return c.Expression
	}

	It("Should stage a second top-level metrics folder as a repeat, not a clash",
		func(ctx SpecContext) {
			run(
				ctx,
				nil,
				folder(first, groupv1.MetricsFolder),
				folder(second, groupv1.MetricsFolder),
			)
			Expect(MustSucceed(resource.StagedRenames(db))).To(ConsistOf(
				resource.Rename{
					ID:     folder(second, "").ResourceID(),
					From:   groupv1.MetricsFolder,
					Reason: resource.ReasonRepeat,
				},
			))
		})

	It("Should write the new name of the channel that v0.62 bound when a channel in "+
		"another folder keeps the old name", func(ctx SpecContext) {
		calc := channel(3, "doubled", nil)
		calc.Expression = "return Metrics * 2"
		run(
			ctx,
			// v0.62 bound "Metrics" to the higher key, the top-level channel.
			[]v1.Channel{
				channel(1, groupv1.MetricsFolder, &first),
				channel(2, groupv1.MetricsFolder, nil),
				calc,
			},
			folder(first, "Plant"),
			folder(second, groupv1.MetricsFolder),
		)
		Expect(expressionOf(ctx, 3)).To(Equal("return Metrics_2 * 2"))
	})
})
