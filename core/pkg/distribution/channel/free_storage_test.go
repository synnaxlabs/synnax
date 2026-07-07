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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagemock "github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// These tests cover the guarantee that free channels are registered transiently in
// every node's local storage engine: on the bootstrapper at create, on a requesting
// node when the create is routed through the bootstrapper, and at startup for
// channels already present in the channel table.
var _ = Describe("Free Storage Registration", func() {
	newFreeChannel := func() channel.Channel {
		return channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
	}

	It("Should register a free channel on the requesting node when the create routes through the bootstrapper", func(ctx SpecContext) {
		c := mock.NewCluster(ctx, 2)
		ch := newFreeChannel()
		Expect(c.Nodes[2].Channel.Create(ctx, &ch)).To(Succeed())
		bootstrapperCh := MustSucceed(
			c.Nodes[1].Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
		)
		Expect(bootstrapperCh.Transient).To(BeTrue())
		requesterCh := MustSucceed(
			c.Nodes[2].Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
		)
		Expect(requesterCh.Transient).To(BeTrue())
	})

	It("Should register a free channel retrieved by name on the requesting node", func(ctx SpecContext) {
		c := mock.NewCluster(ctx, 2)
		ch := newFreeChannel()
		Expect(c.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
		retrieved := channel.Channel{
			Name:        ch.Name,
			DataType:    ch.DataType,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
		Expect(c.Nodes[2].Channel.Create(
			ctx, &retrieved, channel.RetrieveIfNameExists(),
		)).To(Succeed())
		Expect(retrieved.Key()).To(Equal(ch.Key()))
		requesterCh := MustSucceed(
			c.Nodes[2].Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
		)
		Expect(requesterCh.Transient).To(BeTrue())
	})

	It("Should register free channels found in the channel table at startup", func(ctx SpecContext) {
		freshStorageCluster := DeferClose(storagemock.NewCluster())
		c := mock.NewCluster(ctx, 1)
		n := c.Nodes[1]
		ch := newFreeChannel()
		Expect(n.Channel.Create(ctx, &ch)).To(Succeed())

		// Restart the node's distribution layer with the same key-value store but a
		// fresh time-series engine, mirroring a process restart: the channel table
		// survives on disk while transient channel registrations do not.
		Expect(n.Layer.Close()).To(Succeed())
		freshStorage := freshStorageCluster.Provision(ctx)
		c.Provision(ctx, distribution.LayerConfig{
			Storage: &storage.Layer{KV: n.Storage.KV, TS: freshStorage.TS},
		})
		stored := MustSucceed(
			freshStorage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
		)
		Expect(stored.Transient).To(BeTrue())
	})
})
