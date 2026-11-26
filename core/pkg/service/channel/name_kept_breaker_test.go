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
	"fmt"

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

var _ = Describe("Kept names", func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		var cfg channel.ServiceConfig
		// A default Core: name validation on.
		cfg, groupSvc = serviceConfig(ctx, node, node.DB)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	virtual := func(name string) channel.Channel {
		return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
	}
	create := func(
		ctx SpecContext,
		ch *channel.Channel,
		opts ...channel.CreateOption,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, ch, opts...)
		})
	}
	// storeName writes name past the writer, as a v0.62 Core that ran with name
	// validation off stored it. The upgrade keeps it.
	storeName := func(ctx SpecContext, ch channel.Channel, name string) {
		GinkgoHelper()
		ch.Name = name
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, ch)).
			To(Succeed())
	}

	It("Should refuse a name kept in two folders, in Arc and in a create that "+
		"retrieves it", func(ctx SpecContext) {
		rigs := MustSucceed(
			groupSvc.CreateOrRetrieve(ctx, "Rigs", resource.TypeChannel),
		)
		lower := virtual("kept_pt")
		lower.Group = &rigs.Key
		Expect(create(ctx, &lower)).To(Succeed())
		higher := virtual(UniqueChannelName())
		Expect(create(ctx, &higher)).To(Succeed())
		Expect(higher.Key()).To(BeNumerically(">", lower.Key()))
		storeName(ctx, higher, "kept_pt")
		ambiguous := MatchError(ContainSubstring("2 channels are named 'kept_pt'"))
		Expect(svc.NewArcSymbolResolver(db).Resolve(ctx, "kept_pt")).Error().
			To(ambiguous)
		again := virtual("kept_pt")
		Expect(create(ctx, &again, channel.RetrieveIfNameExists())).To(ambiguous)
	})

	It("Should take a rename to the exact name that a kept name holds",
		func(ctx SpecContext) {
			ch := virtual(UniqueChannelName())
			Expect(create(ctx, &ch)).To(Succeed())
			storeName(ctx, ch, "ö_kept")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, ch.Key(), "ö_kept", false)
			})).To(Succeed())
			var stored channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(fmt.Sprintf("%+q", stored.Name)).
				To(Equal(fmt.Sprintf("%+q", "\u00F6_kept")))
		})
})
