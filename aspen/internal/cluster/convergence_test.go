// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Convergence", func() {
	var (
		gossipNet *freightertestutil.Network[gossip.Message, gossip.Message]
		pledgeNet *freightertestutil.Network[pledge.Request, pledge.Response]
	)

	BeforeEach(func() {
		gossipNet = freightertestutil.NewNetwork[gossip.Message, gossip.Message]()
		pledgeNet = freightertestutil.NewNetwork[pledge.Request, pledge.Response]()
	})

	Context("Serial PledgeServer", func() {
		DescribeTable(
			"Should converge a cluster",
			func(
				ctx SpecContext,
				clusterSize int,
				threshold, gossipInterval time.Duration,
				peerAddrCount int,
			) {
				var (
					clusters  []*cluster.Cluster
					addresses []address.Address
				)
				for range clusterSize {
					gossipT := gossipNet.UnaryServer("")
					pledgeT := pledgeNet.UnaryServer(gossipT.Address())
					peerAddresses := lo.Samples(addresses, peerAddrCount)
					cluster := MustOpen(cluster.Open(
						ctx,
						cluster.Config{
							HostAddress: gossipT.Address(),
							Pledge: pledge.Config{
								Peers:           peerAddresses,
								TransportServer: pledgeT,
								TransportClient: pledgeNet.UnaryClient(),
								RetryInterval:   gossipInterval,
								RetryScale:      1,
							},
							Gossip: gossip.Config{
								TransportServer: gossipT,
								TransportClient: gossipNet.UnaryClient(),
								Interval:        gossipInterval,
							},
							Storage: DeferClose(memkv.New()),
						},
					))
					addresses = append(addresses, gossipT.Address())
					clusters = append(clusters, cluster)
					// A responsible node proposes the highest key it knows plus one, so
					// the cluster has to agree on its membership before the next node
					// pledges. Joining faster than gossip converges leaves keys the
					// algorithm never promised, and can hand one key to two nodes.
					for _, c := range clusters {
						Eventually(
							c.Nodes,
							threshold,
						).Should(HaveLen(len(clusters)))
					}
				}
				Expect(clusters).To(HaveLen(clusterSize))
				for j, c := range clusters {
					Expect(c.HostKey()).To(Equal(node.Key(j + 1)))
				}
			},
			Entry(
				"of 4 nodes in 1s at a 10ms interval with 1 peer address",
				4, time.Second, 10*time.Millisecond, 1,
			),
			Entry(
				"of 10 nodes in 3s at a 50ms interval with 3 peer addresses",
				10, 3*time.Second, 50*time.Millisecond, 3,
			),
		)
	})
})
