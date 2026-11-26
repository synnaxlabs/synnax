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
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NameRuleMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	channel := func(local v1.LocalKey, name string, group *uuid.UUID) v1.Channel {
		return v1.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    local,
			LocalIndex:  local,
			IsIndex:     true,
			Virtual:     true,
			Group:       group,
		}
	}
	// migrate stores channels, runs the migration, and returns the name of each channel
	// by local key.
	migrate := func(ctx context.Context, channels ...v1.Channel) map[v1.LocalKey]string {
		GinkgoHelper()
		Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(ctx, channels...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "channel",
			Migrations: []gorp.Migration{v1.NameRuleMigration},
		})).To(Succeed())
		var stored []v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		names := make(map[v1.LocalKey]string, len(stored))
		for _, c := range stored {
			names[c.LocalKey] = c.Name
		}
		return names
	}
	staged := func() map[string]string {
		GinkgoHelper()
		renames := MustSucceed(resource.StagedRenames(db))
		from := make(map[string]string, len(renames))
		for _, r := range renames {
			Expect(r.ID.Type).To(Equal(resource.TypeChannel))
			from[r.ID.Key] = r.From
		}
		return from
	}

	DescribeTable("Should rewrite a name that a path cannot hold",
		func(ctx SpecContext, name, expected string) {
			names := migrate(ctx, channel(1, name, nil))
			Expect(names[1]).To(Equal(expected))
			key := channel(1, name, nil).Key().String()
			Expect(staged()).To(Equal(map[string]string{key: name}))
		},
		Entry("a separator", "stand.pt", "stand_pt"),
		Entry("each invalid byte", "pt\xff\xfe", "pt\uFFFD\uFFFD"),
	)

	DescribeTable("Should keep a name that breaks the rule but not a path",
		func(ctx SpecContext, name string) {
			Expect(migrate(ctx, channel(1, name, nil))[1]).To(Equal(name))
			Expect(staged()).To(BeEmpty())
		},
		Entry("a dash and a space", "Stand A-01"),
		Entry("a control character", "pt\x00a"),
		Entry("a zero-width space", "pt\u200B01"),
		Entry("a no-break space", "pt\u00A001"),
		Entry("a name not in NFC", "cafe\u0301"),
	)

	It("Should name a channel channel_<key> when its name is empty",
		func(ctx SpecContext) {
			ch := channel(1, "", nil)
			Expect(migrate(ctx, ch)[1]).To(Equal("channel_" + ch.Key().String()))
		})

	It("Should keep each legal name, built-ins and repeats in other groups included",
		func(ctx SpecContext) {
			a, b := uuid.New(), uuid.New()
			builtIn := channel(3, "sy_node_1_metrics", nil)
			builtIn.Internal = true
			names := migrate(
				ctx,
				channel(1, "Stand A-01", nil),
				channel(2, "pressure", &a),
				channel(4, "pressure", &b),
				builtIn,
			)
			Expect(names).To(Equal(map[v1.LocalKey]string{
				1: "Stand A-01",
				2: "pressure",
				3: "sy_node_1_metrics",
				4: "pressure",
			}))
			Expect(staged()).To(BeEmpty())
		})

	It("Should keep the name of the higher key in a group and move the lower keys",
		func(ctx SpecContext) {
			g := uuid.New()
			names := migrate(
				ctx,
				channel(5, "pressure", &g),
				channel(2, "pressure", &g),
				channel(8, "pressure", &g),
				channel(7, "pressure", nil),
			)
			Expect(names).To(Equal(map[v1.LocalKey]string{
				2: "pressure_3",
				5: "pressure_2",
				7: "pressure",
				8: "pressure",
			}))
		})

	It("Should keep a legal name when a higher key matches it only after NFC",
		func(ctx SpecContext) {
			names := migrate(
				ctx,
				channel(2, "cafe\u0301", nil),
				channel(1, "caf\u00E9", nil),
			)
			Expect(names[1]).To(Equal("caf\u00E9"))
			Expect(names[2]).To(Equal("caf\u00E9_2"))
		})

	It("Should move a channel that holds the name of a built-in in its group, even "+
		"with a lower key", func(ctx SpecContext) {
		builtIn := channel(9, "sy_node_1_metrics", nil)
		builtIn.Internal = true
		names := migrate(ctx, channel(1, "sy_node_1_metrics", nil), builtIn)
		Expect(names[9]).To(Equal("sy_node_1_metrics"))
		Expect(names[1]).To(Equal("sy_node_1_metrics_2"))
	})

	It("Should keep the name of a built-in, even one that breaks the rule",
		func(ctx SpecContext) {
			builtIn := channel(9, "sy.metrics", nil)
			builtIn.Internal = true
			names := migrate(ctx, channel(1, "sy.metrics", nil), builtIn)
			Expect(names[9]).To(Equal("sy.metrics"))
			Expect(names[1]).To(Equal("sy_metrics"))
		})

	It("Should give a new name the first suffix that no channel in any group holds",
		func(ctx SpecContext) {
			g := uuid.New()
			names := migrate(
				ctx,
				channel(1, "stand_pt", &g),
				channel(2, "stand_pt_2", nil),
				channel(3, "stand.pt", nil),
			)
			Expect(names[1]).To(Equal("stand_pt"))
			Expect(names[2]).To(Equal("stand_pt_2"))
			Expect(names[3]).To(Equal("stand_pt_3"))
		})

	It("Should give each of several new names its own suffix", func(ctx SpecContext) {
		first, second := uuid.New(), uuid.New()
		names := migrate(
			ctx,
			channel(1, "a.b", nil),
			channel(2, "a.b", &first),
			channel(3, "a.b", &second),
		)
		Expect(names).To(Equal(map[v1.LocalKey]string{
			1: "a_b_3",
			2: "a_b_2",
			3: "a_b",
		}))
	})
})
