// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pledge_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

func baseConfig(n *fmock.Network[pledge.Request, pledge.Response]) pledge.Config {
	cfg, _ := baseConfigWithAddr(n)
	return cfg
}

func allCandidates(nodes node.Group) func() node.Group {
	return func() node.Group { return nodes }
}

func baseConfigWithAddr(
	n *fmock.Network[pledge.Request, pledge.Response],
) (pledge.Config, address.Address) {
	server := n.UnaryServer("")
	cfg := pledge.Config{
		TransportServer: server,
		TransportClient: n.UnaryClient(),
	}
	return cfg, server.Address
}

func provisionCandidates(
	n int,
	net *fmock.Network[pledge.Request, pledge.Response],
	nodes node.Group,
	candidates func(i int) func() node.Group,
	nodeState func(i int) node.State,
) node.Group {
	if candidates == nil {
		candidates = func(i int) func() node.Group {
			return func() node.Group { return nodes }
		}
	}
	if nodeState == nil {
		nodeState = func(i int) node.State { return node.StateHealthy }
	}
	for i := range n {
		cfg, addr := baseConfigWithAddr(net)
		Expect(pledge.Arbitrate(cfg, pledge.Config{
			Candidates: candidates(i),
		})).To(Succeed())
		id := node.Key(i)
		nodes[id] = node.Node{Key: node.Key(i), Address: addr, State: nodeState(i)}
	}
	return nodes
}

var _ = Describe("PledgeServer", func() {
	var net *fmock.Network[pledge.Request, pledge.Response]
	BeforeEach(func() {
		net = fmock.NewNetwork[pledge.Request, pledge.Response]()
	})

	Describe("PledgeServer", func() {

		Context("No nodes Responding", func() {
			It("Should submit round robin proposals at scaled intervals", func() {
				var (
					peers         []address.Address
					numTransports = 4
					handler       = func(ctx context.Context, req pledge.Request) (pledge.Response, error) {
						return req, errors.New("pledge failed")
					}
				)
				for range numTransports {
					t := net.UnaryServer("")
					t.BindHandler(handler)
					peers = append(peers, t.Address)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()
				res, err := pledge.Pledge(ctx, baseConfig(net), pledge.Config{
					Instrumentation: ins.Child("no-nodes-responding"),
					Peers:           peers,
					Candidates:      func() node.Group { return node.Group{} },
				}, pledge.BlazingFastConfig)
				Expect(err).To(HaveOccurredAs(context.DeadlineExceeded))
				Expect(res.Key).To(Equal(node.Key(0)))
				for i, entry := range net.Entries {
					Expect(entry.Target).To(Equal(peers[i%4]))
				}
				Expect(net.Entries).ToNot(HaveLen(0))
			})
		})
	})

	Describe("Responsible", func() {
		Context("Cluster State is Synchronized", func() {
			It("Should correctly assign an Name", func() {
				var (
					nodes         = make(node.Group)
					numCandidates = 10
				)
				provisionCandidates(numCandidates, net, nodes, nil, nil)
				candidates := allCandidates(nodes)
				ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
				defer cancel()
				res, err := pledge.Pledge(ctx, baseConfig(net), pledge.Config{
					Instrumentation: ins.Child("cluster-state-synchronized"),
					Peers:           nodes.Addresses(),
					Candidates:      candidates,
				}, pledge.BlazingFastConfig)
				Expect(err).To(BeNil())
				Expect(res.Key).To(Equal(node.Key(10)))
			})
		})
		Context("Responsible is Missing UniqueLeaseholders", func() {
			It("Should correctly assign an Name", func() {
				var (
					nodes      = make(node.Group)
					candidates = func(i int) func() node.Group {
						return func() node.Group {
							if i == 0 {
								return nodes.Where(func(key node.Key, _ node.Node) bool {
									return !lo.Contains([]node.Key{8, 9, 10}, key)
								})
							}
							return nodes
						}
					}
				)
				nodes = provisionCandidates(10, net, nodes, candidates, nil)
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				res, err := pledge.Pledge(
					ctx,
					baseConfig(net),
					pledge.Config{
						Candidates: func() node.Group { return nodes },
						Peers:      []address.Address{nodes[0].Address},
					},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(BeNil())
				Expect(res.Key).To(Equal(node.Key(10)))
			})
		})
		Context("One juror are aware of a new node", func() {
			It("Should assign the correct Name", func() {
				var (
					nodes           = make(node.Group)
					allCandidates   = func() node.Group { return nodes }
					extraCandidates = func() node.Group {
						n := nodes.Copy()
						n[10] = node.Node{Key: 10, Address: "localhost:10", State: node.StateHealthy}
						return n
					}
					net = fmock.NewNetwork[pledge.Request, pledge.Response]()
				)
				provisionCandidates(10, net, nodes, func(i int) func() node.Group {
					return lo.Ternary(i%2 == 0, extraCandidates, allCandidates)
				}, nil)
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				res, err := pledge.Pledge(
					ctx,
					baseConfig(net),
					pledge.Config{
						Instrumentation: ins.Child("one-juror-aware-of-new-node"),
						Peers:           []address.Address{allCandidates()[0].Address},
						Candidates:      extraCandidates,
					},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(BeNil())
				Expect(res.Key).To(BeNumerically(">=", node.Key(11)))
			})
		})
		Context("Too Few Healthy UniqueLeaseholders To Form a Quorum", func() {
			It("Should return an errQuorumUnreachable", func() {
				var (
					numCandidates = 10
					nodes         = make(node.Group)
				)
				provisionCandidates(numCandidates, net, nodes, nil, func(i int) node.State {
					return lo.Ternary(i%2 == 0, node.StateHealthy, node.StateDead)
				})
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()
				_, err := pledge.Pledge(
					ctx,
					baseConfig(net),
					pledge.Config{
						Peers:      []address.Address{nodes[1].Address},
						Candidates: allCandidates(nodes),
					},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(HaveOccurredAs(context.DeadlineExceeded))
			})
		})
		Describe("Cancelling a pledge", func() {
			It("Should stop all operations and return a cancellation error", func() {
				var (
					numCandidates = 10
					nodes         = make(node.Group)
				)
				provisionCandidates(numCandidates, net, nodes, nil, nil)
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				res, err := pledge.Pledge(ctx, baseConfig(net), pledge.Config{
					Peers:      nodes.Addresses(),
					Candidates: allCandidates(nodes),
				})
				Expect(err).To(HaveOccurredAs(context.Canceled))
				Expect(res.Key).To(Equal(node.Key(0)))
			})
		})

		Context("Concurrent Pledges", func() {
			It("Should assign unique keys to all pledges", func() {
				var (
					mu         sync.Mutex
					nodes      = make(node.Group)
					candidates = func(i int) func() node.Group {
						return func() node.Group {
							mu.Lock()
							defer mu.Unlock()
							return nodes.Copy()
						}
					}
					numCandidates = 10
					numPledges    = 2
				)
				provisionCandidates(numCandidates, net, nodes, candidates, nil)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				var wg sync.WaitGroup
				ids := make([]node.Key, numPledges)
				for i := range numPledges {
					wg.Add(1)
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						cfg, addr := baseConfigWithAddr(net)
						res, err := pledge.Pledge(
							ctx,
							cfg,
							pledge.Config{
								Instrumentation: ins.Child("concurrent-pledges"),
								Candidates:      candidates(0),
								Peers:           nodes.Addresses(),
							},
							pledge.BlazingFastConfig,
						)
						Expect(err).ToNot(HaveOccurred())
						ids[i] = res.Key
						mu.Lock()
						defer mu.Unlock()
						nodes[res.Key] = node.Node{Key: res.Key, Address: addr, State: node.StateHealthy}
					}(i)
				}
				wg.Wait()
				Expect(len(lo.Uniq(ids))).To(Equal(numPledges))
			})

		})

	})

})
