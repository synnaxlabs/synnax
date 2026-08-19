// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

func baseConfig(n *mock.Network[pledge.Request, pledge.Response]) pledge.Config {
	cfg, _ := baseConfigWithAddr(n)
	return cfg
}

func allCandidates(nodes node.Group) func() node.Group {
	return func() node.Group { return nodes }
}

func baseConfigWithAddr(
	n *mock.Network[pledge.Request, pledge.Response],
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
	net *mock.Network[pledge.Request, pledge.Response],
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

// maxProposals is the proposal budget the budget specs configure. Kept small so a
// miscounted round is cheap to detect.
const maxProposals = 3

// jurorFailure is the failure provisionFailingJuror returns on every proposal.
const jurorFailure = "juror unreachable"

// provisionFailingJuror returns a server that fails every proposal it receives. It
// stands in for a juror that is listed as a healthy candidate but is unreachable.
func provisionFailingJuror(
	n *mock.Network[pledge.Request, pledge.Response],
) *mock.UnaryServer[pledge.Request, pledge.Response] {
	server := n.UnaryServer("")
	server.BindHandler(
		func(context.Context, pledge.Request) (pledge.Response, error) {
			return pledge.Response{}, errors.New(jurorFailure)
		},
	)
	return server
}

// arbitrateResponsible registers a node that acts as responsible over candidates with a
// budget of maxProposals, and returns its address.
func arbitrateResponsible(
	n *mock.Network[pledge.Request, pledge.Response],
	ins alamos.Instrumentation,
	candidates func() node.Group,
) address.Address {
	GinkgoHelper()
	cfg, addr := baseConfigWithAddr(n)
	Expect(pledge.Arbitrate(cfg, pledge.Config{
		Instrumentation: ins,
		Candidates:      candidates,
		MaxProposals:    maxProposals,
	})).To(Succeed())
	return addr
}

// sendPledge asks the responsible at addr to run a pledge, bounding it so a responsible
// that never exhausts its budget fails the spec instead of hanging it.
func sendPledge(
	ctx context.Context,
	n *mock.Network[pledge.Request, pledge.Response],
	addr address.Address,
) (pledge.Response, error) {
	tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	DeferCleanup(cancel)
	return n.UnaryClient().Send(tCtx, addr, pledge.Request{Key: 0})
}

// countRequests returns the number of requests the network delivered to target.
func countRequests(
	n *mock.Network[pledge.Request, pledge.Response],
	target address.Address,
) int {
	count := 0
	for _, entry := range n.Entries() {
		if entry.Target == target {
			count++
		}
	}
	return count
}

var _ = Describe("PledgeServer", func() {
	var net *mock.Network[pledge.Request, pledge.Response]
	BeforeEach(func() {
		net = mock.NewNetwork[pledge.Request, pledge.Response]()
	})

	Describe("Invalid configuration", func() {
		missingRequiredFields := SatisfyAll(
			MatchError(ContainSubstring("transport_client: must be non-nil")),
			MatchError(ContainSubstring("transport_server: must be non-nil")),
			MatchError(ContainSubstring("candidates: must be non-nil")),
		)
		It("Should return a validation error from Pledge", func(ctx SpecContext) {
			Expect(pledge.Pledge(ctx, pledge.Config{})).Error().
				To(missingRequiredFields)
		})
		It("Should return a validation error from Arbitrate", func() {
			Expect(pledge.Arbitrate(pledge.Config{})).To(missingRequiredFields)
		})
	})

	Describe("PledgeServer", func() {
		Context("No nodes Responding", func() {
			It(
				"Should submit round robin proposals at scaled intervals",
				func(ctx SpecContext) {
					const numTransports = 4
					tCtx, cancel := context.WithCancel(ctx)
					defer cancel()
					var (
						peers []address.Address
						count int
						// Terminate once we've observed two full round-robin cycles
						// rather than on a wall-clock deadline. A short deadline races
						// with coarse timer resolution (notably on Windows), where the
						// retry ticker's first tick can fire after the deadline,
						// yielding zero attempts.
						handler = func(_ context.Context, req pledge.Request) (pledge.Response, error) {
							count++
							if count >= 2*numTransports {
								cancel()
							}
							return req, errors.New("pledge failed")
						}
					)
					for range numTransports {
						t := net.UnaryServer("")
						t.BindHandler(handler)
						peers = append(peers, t.Address)
					}
					Expect(pledge.Pledge(tCtx, baseConfig(net), pledge.Config{
						Instrumentation: ins.Child("no-nodes-responding"),
						Peers:           peers,
						Candidates:      func() node.Group { return node.Group{} },
					}, pledge.BlazingFastConfig)).Error().To(MatchError(context.Canceled))
					entries := net.Entries()
					Expect(len(entries)).To(BeNumerically(">=", 2*numTransports))
					for i, entry := range entries {
						Expect(entry.Target).To(Equal(peers[i%numTransports]))
					}
				},
			)
		})
	})

	Describe("Responsible", func() {
		Context("Cluster State is Synchronized", func() {
			It("Should correctly assign an Name", func(ctx SpecContext) {
				var (
					nodes         = make(node.Group)
					numCandidates = 10
				)
				provisionCandidates(numCandidates, net, nodes, nil, nil)
				candidates := allCandidates(nodes)
				tCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
				defer cancel()
				res := MustSucceed(pledge.Pledge(tCtx, baseConfig(net), pledge.Config{
					Instrumentation: ins.Child("cluster-state-synchronized"),
					Peers:           nodes.Addresses(),
					Candidates:      candidates,
				}, pledge.BlazingFastConfig))
				// The pledge algorithm may assign a marginally higher key than the
				// minimum when a quorum consultation hits a transient timeout and the
				// responsible retries with an incremented proposal (see pledge.go).
				Expect(res.Key).To(BeNumerically(">=", node.Key(10)))
			})
		})
		Context("Responsible is Missing UniqueLeaseholders", func() {
			It("Should correctly assign an Name", func(ctx SpecContext) {
				var (
					nodes      = make(node.Group)
					candidates = func(i int) func() node.Group {
						return func() node.Group {
							if i == 0 {
								return nodes.Where(
									func(key node.Key, _ node.Node) bool {
										return !lo.Contains([]node.Key{8, 9, 10}, key)
									},
								)
							}
							return nodes
						}
					}
				)
				nodes = provisionCandidates(10, net, nodes, candidates, nil)
				tCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				res := MustSucceed(pledge.Pledge(
					tCtx,
					baseConfig(net),
					pledge.Config{
						Candidates: func() node.Group { return nodes },
						Peers:      []address.Address{nodes[0].Address},
					},
					pledge.BlazingFastConfig,
				))
				Expect(res.Key).To(BeNumerically(">=", node.Key(10)))
			})
		})
		Context("One juror are aware of a new node", func() {
			It("Should assign the correct Name", func(ctx SpecContext) {
				var (
					nodes           = make(node.Group)
					allCandidates   = func() node.Group { return nodes }
					extraCandidates = func() node.Group {
						n := nodes.Copy()
						n[10] = node.Node{
							Key:     10,
							Address: "localhost:10",
							State:   node.StateHealthy,
						}
						return n
					}
					net = mock.NewNetwork[pledge.Request, pledge.Response]()
				)
				provisionCandidates(10, net, nodes, func(i int) func() node.Group {
					return lo.Ternary(i%2 == 0, extraCandidates, allCandidates)
				}, nil)
				tCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				res := MustSucceed(pledge.Pledge(
					tCtx,
					baseConfig(net),
					pledge.Config{
						Instrumentation: ins.Child("one-juror-aware-of-new-node"),
						Peers:           []address.Address{allCandidates()[0].Address},
						Candidates:      extraCandidates,
					},
					pledge.BlazingFastConfig,
				))
				Expect(res.Key).To(BeNumerically(">=", node.Key(11)))
			})
		})
		Context("Too Few Healthy UniqueLeaseholders To Form a Quorum", func() {
			It("Should return an errQuorumUnreachable", func(ctx SpecContext) {
				var (
					numCandidates = 10
					nodes         = make(node.Group)
				)
				provisionCandidates(
					numCandidates,
					net,
					nodes,
					nil,
					func(i int) node.State {
						return lo.Ternary(i%2 == 0, node.StateHealthy, node.StateDead)
					},
				)
				tCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
				_, err := pledge.Pledge(
					tCtx,
					baseConfig(net),
					pledge.Config{
						Peers:      []address.Address{nodes[1].Address},
						Candidates: allCandidates(nodes),
					},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(MatchError(context.DeadlineExceeded))
			})
		})
		Context("Proposal budget", func() {
			It("Should retry past rejected keys without consuming the proposal budget",
				func(ctx SpecContext) {
					nodes := make(node.Group)
					provisionCandidates(2, net, nodes, nil, nil)
					client := net.UnaryClient()
					// Approve keys 2 through 21 on the first juror, as a pledge that
					// failed partway would leave behind. The 20 keys exceed
					// MaxProposals, so counting the rejected re-proposals would exhaust
					// every retry's budget before it reaches a fresh key.
					for k := node.Key(2); k <= 21; k++ {
						Expect(client.Send(
							ctx,
							nodes[0].Address,
							pledge.Request{Key: k},
						)).To(Equal(pledge.Response{}))
					}
					tCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
					DeferCleanup(cancel)
					// A generous request timeout keeps the long climb over the
					// polluted keys from timing out under parallel suite load.
					res := MustSucceed(
						pledge.Pledge(tCtx, baseConfig(net), pledge.Config{
							Instrumentation: ins.Child("rejections-uncounted"),
							Peers:           nodes.Addresses(),
							Candidates:      allCandidates(nodes),
						}, pledge.BlazingFastConfig, pledge.Config{
							RequestTimeout: time.Second,
						}),
					)
					Expect(res.Key).To(BeNumerically(">=", node.Key(22)))
				},
			)
			It("Should consume the budget and return the error when a juror fails",
				func(ctx SpecContext) {
					juror := provisionFailingJuror(net)
					jurors := node.Group{1: {
						Key:     1,
						Address: juror.Address,
						State:   node.StateHealthy,
					}}
					addr := arbitrateResponsible(
						net,
						ins.Child("infra-failures-counted"),
						allCandidates(jurors),
					)
					Expect(sendPledge(ctx, net, addr)).Error().
						To(MatchError(ContainSubstring(jurorFailure)))
					Expect(countRequests(net, juror.Address)).To(Equal(maxProposals))
				},
			)
			It("Should let an infrastructure failure outweigh a concurrent rejection",
				func(ctx SpecContext) {
					// Every round consults both jurors: one rejects the stale key while
					// the other fails outright. The failure must decide the round,
					// otherwise the rejection makes it free and the climb never stops.
					nodes := make(node.Group)
					provisionCandidates(1, net, nodes, nil, nil)
					failing := provisionFailingJuror(net)
					nodes[1] = node.Node{
						Key:     1,
						Address: failing.Address,
						State:   node.StateHealthy,
					}
					client := net.UnaryClient()
					for k := node.Key(2); k <= 2+node.Key(maxProposals); k++ {
						Expect(client.Send(
							ctx,
							nodes[0].Address,
							pledge.Request{Key: k},
						)).To(Equal(pledge.Response{}))
					}
					addr := arbitrateResponsible(
						net,
						ins.Child("infra-outweighs-rejection"),
						allCandidates(nodes),
					)
					Expect(sendPledge(ctx, net, addr)).Error().
						To(MatchError(ContainSubstring(jurorFailure)))
					Expect(countRequests(net, failing.Address)).To(Equal(maxProposals))
				},
			)
		})

		Describe("Cancelling a pledge", func() {
			It(
				"Should stop all operations and return a cancellation error",
				func(ctx SpecContext) {
					var (
						numCandidates = 10
						nodes         = make(node.Group)
					)
					provisionCandidates(numCandidates, net, nodes, nil, nil)
					tCtx, cancel := context.WithCancel(ctx)
					cancel()
					res, err := pledge.Pledge(tCtx, baseConfig(net), pledge.Config{
						Peers:      nodes.Addresses(),
						Candidates: allCandidates(nodes),
					})
					Expect(err).To(MatchError(context.Canceled))
					Expect(res.Key).To(Equal(node.Key(0)))
				},
			)
		})

		Context("Concurrent Pledges", func() {
			It("Should assign unique keys to all pledges", func(ctx SpecContext) {
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
				tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				var wg sync.WaitGroup
				ids := make([]node.Key, numPledges)
				for i := range numPledges {
					wg.Add(1)
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						mu.Lock()
						addrs := nodes.Addresses()
						mu.Unlock()
						cfg, addr := baseConfigWithAddr(net)
						res := MustSucceed(pledge.Pledge(
							tCtx,
							cfg,
							pledge.Config{
								Instrumentation: ins.Child("concurrent-pledges"),
								Candidates:      candidates(0),
								Peers:           addrs,
							},
							pledge.BlazingFastConfig,
						))
						ids[i] = res.Key
						mu.Lock()
						defer mu.Unlock()
						nodes[res.Key] = node.Node{
							Key:     res.Key,
							Address: addr,
							State:   node.StateHealthy,
						}
					}(i)
				}
				wg.Wait()
				Expect(lo.Uniq(ids)).To(HaveLen(numPledges))
			})
		})
	})
})
