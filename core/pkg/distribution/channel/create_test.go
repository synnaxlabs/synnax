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
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Allocate", Ordered, func() {
	var (
		n    testutil.Node
		host node.Key
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		n = testutil.NewNode(ctx)
		host = n.Identity.HostKey
	})

	It(
		"Should assign a local key to a host channel and create no storage channel",
		func(ctx SpecContext) {
			out := MustSucceed(n.Channel.Allocate(ctx, []channel.Channel{
				{
					Name:        "host",
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: n.Identity.HostKey,
				},
			}))
			Expect(out[0].LocalKey).ToNot(BeZero())
			Expect(out[0].Key().Lease()).To(Equal(host))
			Expect(n.Storage.TS.RetrieveChannel(ctx, uint32(out[0].Key()))).
				Error().To(MatchError(query.ErrNotFound))
		},
	)
	It(
		"Should default an unspecified leaseholder to the host node",
		func(ctx SpecContext) {
			out := MustSucceed(n.Channel.Allocate(ctx, []channel.Channel{
				{Name: "defaulted", DataType: telem.TimestampT, IsIndex: true},
			}))
			Expect(out[0].Leaseholder).To(Equal(host))
		},
	)
	It("Should assign sequential local keys across a batch", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Allocate(ctx, []channel.Channel{
			{Name: "batch-a", DataType: telem.TimestampT, IsIndex: true},
			{Name: "batch-b", DataType: telem.TimestampT, IsIndex: true},
			{Name: "batch-c", DataType: telem.TimestampT, IsIndex: true},
		}))
		Expect(out[1].LocalKey).To(Equal(out[0].LocalKey + 1))
		Expect(out[2].LocalKey).To(Equal(out[1].LocalKey + 1))
	})
	It(
		"Should set the local index of an index channel to its own local key",
		func(ctx SpecContext) {
			out := MustSucceed(n.Channel.Allocate(ctx, []channel.Channel{
				{Name: "index", DataType: telem.TimestampT, IsIndex: true},
			}))
			Expect(out[0].LocalIndex).To(Equal(out[0].LocalKey))
		},
	)
	It("Should assign a key to a free virtual channel", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Allocate(ctx, []channel.Channel{
			{
				Name:        "free",
				DataType:    telem.Float32T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			},
		}))
		Expect(out[0].LocalKey).ToNot(BeZero())
		Expect(out[0].Key().Lease()).To(Equal(node.KeyFree))
	})
	It(
		"Should return an error when the leaseholder is not the host node",
		func(ctx SpecContext) {
			Expect(n.Channel.Allocate(ctx, []channel.Channel{
				{
					Name:        "unresolvable",
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: node.Key(99),
				},
			})).Error().To(MatchError(query.ErrNotFound))
		},
	)
})

var _ = Describe("Issued", Ordered, func() {
	var (
		n      testutil.Node
		leased channel.Channel
		free   channel.Channel
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		n = testutil.NewNode(ctx)
		out := MustSucceed(n.Channel.Allocate(ctx, []channel.Channel{
			{Name: "leased", DataType: telem.TimestampT, IsIndex: true},
			{
				Name:        "free",
				DataType:    telem.Float32T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			},
		}))
		leased, free = out[0], out[1]
	})

	DescribeTable(
		"Should report whether a key was issued",
		func(key func() channel.Key, issued bool) {
			Expect(n.Channel.Issued(key())).To(Equal(issued))
		},
		Entry(
			"an allocated host key",
			func() channel.Key { return leased.Key() },
			true,
		),
		Entry("an allocated free key", func() channel.Key { return free.Key() }, true),
		Entry("a host key above the counter", func() channel.Key {
			return channel.NewKey(leased.Leaseholder, leased.LocalKey+1)
		}, false),
		Entry("a free key above the counter", func() channel.Key {
			return channel.NewKey(node.KeyFree, free.LocalKey+1)
		}, false),
		Entry("a zero local key", func() channel.Key {
			return channel.NewKey(leased.Leaseholder, 0)
		}, false),
		Entry("a key leased to another node", func() channel.Key {
			return channel.NewKey(node.Key(99), leased.LocalKey+1000)
		}, true),
	)
})

var _ = Describe("Free key counter", func() {
	It("Should not reuse a free key after the Core reopens", func(ctx SpecContext) {
		store := DeferClose(storagetestutil.OpenLayer(ctx))
		create := func() channel.LocalKey {
			GinkgoHelper()
			n := testutil.OpenNode(ctx, distribution.LayerConfig{Storage: store.Layer})
			defer func() { Expect(n.Close()).To(Succeed()) }()
			out := MustSucceed(n.Channel.Allocate(ctx, []channel.Channel{{
				Name:        "free",
				DataType:    telem.Float32T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			}}))
			return out[0].LocalKey
		}
		first := create()
		Expect(create()).To(BeNumerically(">", first))
	})
})
