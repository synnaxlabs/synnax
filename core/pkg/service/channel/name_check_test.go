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

var _ = Describe("Name checks", func() {
	var (
		db  *gorp.DB
		svc *channel.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	virtual := func(name string) channel.Channel {
		return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
	}

	It("Should find a channel by the decomposed form of its name", func(
		ctx SpecContext,
	) {
		decomposed := UniqueChannelName() + "_é"
		ch := virtual(decomposed)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		var found []channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchNames(decomposed)).
			Entries(&found).
			Exec(ctx, db)).To(Succeed())
		Expect(channel.KeysFromChannels(found)).To(ConsistOf(ch.Key()))
	})

	It(
		"Should find a stored name that the rule refuses when a create asks for it",
		func(
			ctx SpecContext,
		) {
			ch := virtual(UniqueChannelName())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			// A Core that ran with the old flag stored a dotted name.
			legacy := ch
			legacy.Name = "tank." + ch.Name
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, legacy)).
				To(Succeed())
			again := virtual(legacy.Name)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					Create(ctx, &again, channel.RetrieveIfNameExists())
			})).To(Succeed())
			Expect(again.Key()).To(Equal(ch.Key()))
		},
	)
})
