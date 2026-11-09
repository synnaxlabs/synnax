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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rename", Ordered, func() {
	var n testutil.Node

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		n = testutil.NewNode(ctx)
	})

	It("Should rename the storage channel for a host key", func(ctx SpecContext) {
		out := MustSucceed(n.CreateChannels(ctx, []channel.Channel{
			{Name: "old-name", DataType: telem.TimestampT, IsIndex: true},
		}))
		key := out[0].Key()
		Expect(
			n.Channel.Rename(ctx, map[channel.Key]string{key: "new-name"}),
		).To(Succeed())
		stored := MustSucceed(n.Storage.TS.RetrieveChannel(ctx, key.StorageKey()))
		Expect(stored.Name).To(Equal("new-name"))
	})
	It("Should skip storage renames for a free channel", func(ctx SpecContext) {
		out := MustSucceed(n.CreateChannels(ctx, []channel.Channel{
			{
				Name:        "free-rename",
				DataType:    telem.Float32T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			},
		}))
		key := out[0].Key()
		Expect(n.Channel.Rename(
			ctx, map[channel.Key]string{key: "free-renamed"},
		)).To(Succeed())
		Expect(n.Storage.TS.RetrieveChannel(ctx, key.StorageKey())).Error().To(
			MatchError(query.ErrNotFound),
		)
	})
	It(
		"Should return an error when a key's leaseholder is not the host node",
		func(ctx SpecContext) {
			Expect(n.Channel.Rename(
				ctx,
				map[channel.Key]string{channel.NewKey(node.Key(99), 1): "unresolvable"},
			)).To(MatchError(query.ErrNotFound))
		},
	)
})
