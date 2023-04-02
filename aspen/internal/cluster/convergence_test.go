// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster_test

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/rand"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type newConvergenceVars struct {
	clusterSize          int
	convergenceThreshold time.Duration
	gossipInterval       time.Duration
	peerAddrCount        int
}

var progressiveNewConvergence = []newConvergenceVars{
	{
		clusterSize:          4,
		convergenceThreshold: time.Second * 1,
		gossipInterval:       time.Millisecond * 10,
		peerAddrCount:        1,
	},
	{
		clusterSize:          10,
		convergenceThreshold: time.Second * 3,
		gossipInterval:       time.Millisecond * 50,
		peerAddrCount:        3,
	},
}

var _ = Describe("Convergence", func() {
	var (
		gossipNet  *fmock.Network[gossip.Message, gossip.Message]
		pledgeNet  *fmock.Network[pledge.Request, pledge.Response]
		logger     *zap.SugaredLogger
		exp        alamos.Instrumentation
		clusterCtx signal.Context
		shutdown   context.CancelFunc
	)

	BeforeEach(func() {
		clusterCtx, shutdown = signal.WithCancel(ctx)
		gossipNet = fmock.NewNetwork[gossip.Message, gossip.Message]()
		pledgeNet = fmock.NewNetwork[pledge.Request, pledge.Response]()
		logger = zap.NewNop().Sugar()
	})

	AfterEach(func() {
		shutdown()
		Expect(errors.Is(clusterCtx.Wait(), context.Canceled)).To(BeTrue())
	})

	Context("Serial PledgeServer", func() {

		p := alamos.NewParametrize(alamos.IterVars(progressiveNewConvergence))
		p.Template(func(i int, values newConvergenceVars) {

			It(fmt.Sprintf("Should converge a cluster size of %v in %v "+
				"at an interval of %v seconds and a peer address count of %v",
				values.clusterSize, values.convergenceThreshold,
				values.gossipInterval, values.peerAddrCount), func() {
				var (
					clusters  []cluster.Cluster
					addresses []address.Address
				)
				subExp := alamos.Sub(exp, fmt.Sprintf("convergence_test_%v", i))
				for i := 0; i < values.clusterSize; i++ {
					gossipT := gossipNet.UnaryServer("")
					pledgeT := pledgeNet.UnaryServer(gossipT.Address)
					peerAddresses := rand.SubSlice(addresses, values.peerAddrCount)
					cluster, err := cluster.Join(
						clusterCtx,
						cluster.Config{
							HostAddress: gossipT.Address,
							Logger:      logger,
							Pledge: pledge.Config{
								Peers:           peerAddresses,
								TransportServer: pledgeT,
								TransportClient: pledgeNet.UnaryClient(),
								RetryInterval:   values.gossipInterval,
								RetryScale:      1,
							},
							Gossip: gossip.Config{
								TransportServer: gossipT,
								TransportClient: gossipNet.UnaryClient(),
								Interval:        values.gossipInterval,
							},
							Storage:    memkv.New(),
							Experiment: alamos.Sub(subExp, fmt.Sprintf("cluster_%v", i)),
						},
					)
					Expect(err).ToNot(HaveOccurred())
					addresses = append(addresses, gossipT.Address)
					clusters = append(clusters, cluster)
				}
				Expect(len(clusters)).To(Equal(values.clusterSize))
				for j, cluster_ := range clusters {
					Expect(cluster_.HostID()).To(Equal(node.ID(j + 1)))
				}
				for _, cluster_ := range clusters {
					Eventually(cluster_.Nodes, values.convergenceThreshold).Should(HaveLen(values.clusterSize))
				}
			})

		})
		p.Construct()
	})
})
