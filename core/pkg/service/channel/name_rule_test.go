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

var _ = Describe("Name rule", Ordered, func() {
	var (
		db       *gorp.DB
		groupSvc *group.Service
		svc      *channel.Service
		folder   group.Group
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, gs := serviceConfig(ctx, node, node.DB)
		groupSvc = gs
		// The rule and the index apply whatever the old name validation flag says.
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
	stored := func(ctx SpecContext, k channel.Key) string {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch.Name
	}
	// storeName writes name to the channel with key k past the writer, as a Core that
	// ran with the old flag stored it.
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

	It("Should store a created name in NFC", func(ctx SpecContext) {
		ch := virtual("n\u0303ame_nfc")
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(ch.Name).To(Equal("\u00F1ame_nfc"))
		Expect(stored(ctx, ch.Key())).To(Equal("\u00F1ame_nfc"))
	})

	It("Should store a new name in NFC", func(ctx SpecContext) {
		ch := virtual(UniqueChannelName())
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(rename(ctx, ch.Key(), "n\u0303ame_renamed")).To(Succeed())
		Expect(stored(ctx, ch.Key())).To(Equal("\u00F1ame_renamed"))
	})

	It("Should find a stored channel by the decomposed form of its name", func(
		ctx SpecContext,
	) {
		stored := virtual(UniqueChannelName() + "_\u00E9")
		Expect(create(ctx, &stored)).To(Succeed())
		again := virtual(stored.Name[:len(stored.Name)-2] + "e\u0301")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &again, channel.RetrieveIfNameExists())
		})).To(Succeed())
		Expect(again.Key()).To(Equal(stored.Key()))
	})

	It("Should refuse a name that breaks the rule", func(ctx SpecContext) {
		ch := virtual("stand.pt")
		Expect(create(ctx, &ch)).
			To(MatchError(ContainSubstring("no '.' or invisible character")))
	})

	It(
		"Should refuse a name in NFD that a channel holds in NFC",
		func(ctx SpecContext) {
			first := virtual("\u00E9_clash")
			Expect(create(ctx, &first)).To(Succeed())
			second := virtual("e\u0301_clash")
			Expect(create(ctx, &second)).
				To(MatchError(ContainSubstring("already exists")))
		},
	)

	It("Should keep a stored name that breaks the rule writable", func(
		ctx SpecContext,
	) {
		ch := virtual(UniqueChannelName())
		Expect(create(ctx, &ch)).To(Succeed())
		storeName(ctx, ch.Key(), "stand.pt.flag")
		Expect(move(ctx, ch.Key(), &folder.Key)).To(Succeed())
		Expect(stored(ctx, ch.Key())).To(Equal("stand.pt.flag"))
		Expect(rename(ctx, ch.Key(), "stand_pt_flag")).To(Succeed())
	})

	It("Should keep a stored name that is not in NFC when a write keeps it", func(
		ctx SpecContext,
	) {
		ch := virtual(UniqueChannelName())
		Expect(create(ctx, &ch)).To(Succeed())
		storeName(ctx, ch.Key(), "o\u0308_flag")
		Expect(move(ctx, ch.Key(), &folder.Key)).To(Succeed())
		Expect(stored(ctx, ch.Key())).To(Equal("o\u0308_flag"))
	})
})
