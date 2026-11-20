// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Unique names", Ordered, func() {
	var (
		db     *gorp.DB
		svc    *channel.Service
		folder group.Group
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, groupSvc := serviceConfig(ctx, node, node.DB)
		// The index applies whatever the old name validation flag says.
		cfg.ValidateNames = new(false)
		svc = MustOpen(channel.OpenService(ctx, cfg))
		folder = MustSucceed(
			groupSvc.CreateOrRetrieve(ctx, "Names", resource.TypeChannel),
		)
	})
	create := func(ctx SpecContext, ch *channel.Channel) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, ch)
		})
	}
	virtual := func(name string) channel.Channel {
		return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
	}
	// storeName writes name to the channel with key k past the writer, as a Core that
	// ran with the old flag could store a duplicate.
	storeName := func(ctx SpecContext, k channel.Key, name string) {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		ch.Name = name
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, ch)).
			To(Succeed())
	}
	rename := func(ctx SpecContext, k channel.Key, name string) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Rename(ctx, k, name, false)
		})
	}
	move := func(ctx SpecContext, k channel.Key, g *group.Key) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetGroup(ctx, g, k)
		})
	}

	It("Should refuse a name that a channel in another folder holds", func(
		ctx SpecContext,
	) {
		first := virtual("pt_unique_01")
		Expect(create(ctx, &first)).To(Succeed())
		second := virtual("pt_unique_01")
		second.Group = &folder.Key
		Expect(create(ctx, &second)).To(MatchError(
			ContainSubstring("channel with name 'pt_unique_01' already exists"),
		))
		other := virtual(UniqueChannelName())
		Expect(create(ctx, &other)).To(Succeed())
		Expect(rename(ctx, other.Key(), "pt_unique_01")).To(MatchError(
			ContainSubstring("channel with name 'pt_unique_01' already exists"),
		))
	})

	It("Should refuse two channels with one name in one create", func(
		ctx SpecContext,
	) {
		chs := []channel.Channel{virtual("pt_batch_01"), virtual("pt_batch_01")}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(MatchError(
			ContainSubstring("duplicate channel name 'pt_batch_01' in request"),
		))
	})

	It("Should keep each holder of a stored duplicate writable", func(ctx SpecContext) {
		a, b := virtual(UniqueChannelName()), virtual(UniqueChannelName())
		Expect(create(ctx, &a)).To(Succeed())
		Expect(create(ctx, &b)).To(Succeed())
		storeName(ctx, a.Key(), "dup_flag")
		storeName(ctx, b.Key(), "dup_flag")
		Expect(move(ctx, b.Key(), &folder.Key)).To(Succeed())
		Expect(rename(ctx, b.Key(), "dup_flag_2")).To(Succeed())
		Expect(rename(ctx, b.Key(), "dup_flag")).To(MatchError(
			ContainSubstring("channel with name 'dup_flag' already exists"),
		))
	})
})
