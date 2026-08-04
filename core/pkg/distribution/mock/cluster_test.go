// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Cluster", func() {
	Describe("Provision", func() {
		It(
			"Should open a three node memory backed distribution layer",
			func(ctx SpecContext) {
				cluster := mock.NewCluster(ctx, 3)
				nodeOne := cluster.Nodes[node.Key(1)]
				nodeTwo := cluster.Nodes[node.Key(2)]
				nodeThree := cluster.Nodes[node.Key(3)]

				Expect(nodeOne.Cluster.HostKey()).To(Equal(node.Key(1)))
				Expect(nodeTwo.Cluster.HostKey()).To(Equal(node.Key(2)))
				Expect(nodeThree.Cluster.HostKey()).To(Equal(node.Key(3)))

				ch := channel.Channel{
					Name:        "SG_01",
					DataType:    telem.Float64T,
					Virtual:     true,
					Leaseholder: 1,
				}

				ch = MustSucceed(nodeOne.Channel.Create(ctx, []channel.Channel{ch}))[0]
				Expect(ch.Key().Lease()).To(Equal(node.Key(1)))

				chs := MustSucceed(nodeOne.Storage.TS.RetrieveChannels(
					ctx, ch.Key().StorageKey(),
				))
				Expect(chs).To(HaveLen(1))
				Expect(chs[0].Key).To(Equal(ch.Key().StorageKey()))
			},
		)
	})
})
