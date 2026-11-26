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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Chained renames", Ordered, func() {
	var (
		db  *gorp.DB
		svc *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	// pair creates two virtual channels, the first with the lower key.
	pair := func(ctx SpecContext) (channel.Channel, channel.Channel) {
		GinkgoHelper()
		var out [2]channel.Channel
		for i := range out {
			out[i] = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &out[i])
			})).To(Succeed())
		}
		Expect(out[0].Key()).To(BeNumerically("<", out[1].Key()))
		return out[0], out[1]
	}
	renameMany := func(ctx SpecContext, keys []channel.Key, names []string) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).RenameMany(ctx, keys, names, false)
		})
	}

	It("Should give a chained rename one outcome in either direction", func(
		ctx SpecContext,
	) {
		a, b := pair(ctx)
		// a moves to a new name, and b takes the old name of a.
		intoLower := renameMany(
			ctx,
			[]channel.Key{a.Key(), b.Key()},
			[]string{UniqueChannelName(), a.Name},
		)
		c, d := pair(ctx)
		// c takes the old name of d, and d moves to a new name.
		intoHigher := renameMany(
			ctx,
			[]channel.Key{c.Key(), d.Key()},
			[]string{d.Name, UniqueChannelName()},
		)
		Expect(intoHigher == nil).To(
			Equal(intoLower == nil),
			"into the name of the lower key: %v; into the name of the higher key: %v",
			intoLower,
			intoHigher,
		)
	})
})
