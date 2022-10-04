package pledge_test

import (
	"context"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"sync"
	"time"
)

func baseConfig(n *fmock.Network[node.ID, node.ID], logger *zap.SugaredLogger) pledge.Config {
	cfg, _ := baseConfigWithAddr(n, logger)
	return cfg
}

func allCandidates(nodes node.Group) func() node.Group {
	return func() node.Group { return nodes }
}

func baseConfigWithAddr(n *fmock.Network[node.ID, node.ID], logger *zap.SugaredLogger) (pledge.Config, address.Address) {
	server := n.UnaryServer("")
	cfg := pledge.Config{
		TransportServer: server,
		TransportClient: n.UnaryClient(),
		Logger:          logger,
	}
	return cfg, server.Address
}

func provisionCandidates(
	n int,
	net *fmock.Network[node.ID, node.ID],
	nodes node.Group,
	candidates func(i int) func() node.Group,
	nodeState func(i int) node.State,
	logger *zap.SugaredLogger,
) node.Group {
	if candidates == nil {
		candidates = func(i int) func() node.Group {
			return func() node.Group { return nodes }
		}
	}
	if nodeState == nil {
		nodeState = func(i int) node.State { return node.StateHealthy }
	}
	for i := 0; i < n; i++ {
		cfg, addr := baseConfigWithAddr(net, logger)
		Expect(pledge.Arbitrate(cfg, pledge.Config{Candidates: candidates(i)})).To(Succeed())
		id := node.ID(i)
		nodes[id] = node.Node{ID: node.ID(i), Address: addr, State: nodeState(i)}
	}
	return nodes
}

var _ = Describe("PledgeServer", func() {
	var (
		logger *zap.SugaredLogger
		net    *fmock.Network[node.ID, node.ID]
	)

	BeforeEach(func() {
		//logger = lo.Must(zap.NewDevelopment()).Sugar()
		logger = zap.NewNop().Sugar()
		net = fmock.NewNetwork[node.ID, node.ID]()
	})

	Describe("PledgeServer", func() {

		Context("No nodes Responding", func() {
			It("Should submit round robin proposals at scaled intervals", func() {
				var (
					peers         []address.Address
					numTransports = 4
					handler       = func(ctx context.Context, id node.ID) (node.ID, error) {
						return 0, errors.New("pledge failed")
					}
				)
				for i := 0; i < numTransports; i++ {
					t := net.UnaryServer("")
					t.BindHandler(handler)
					peers = append(peers, t.Address)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(ctx, baseConfig(net, logger), pledge.Config{
					Peers:      peers,
					Candidates: func() node.Group { return node.Group{} },
				}, pledge.BlazingFastConfig)
				Expect(err).To(HaveOccurredAs(context.DeadlineExceeded))
				Expect(id).To(Equal(node.ID(0)))
				for i, entry := range net.Entries {
					Expect(entry.Target).To(Equal(peers[i%4]))
				}
				Expect(net.Entries).ToNot(HaveLen(0))
			})
		})
	})

	Describe("Responsible", func() {
		Context("Cluster State is Synchronized", func() {
			It("Should correctly assign an ID", func() {
				var (
					nodes         = make(node.Group)
					numCandidates = 10
				)
				provisionCandidates(numCandidates, net, nodes, nil, nil, logger)
				candidates := allCandidates(nodes)
				ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(ctx, baseConfig(net, logger), pledge.Config{
					Peers:      nodes.Addresses(),
					Candidates: candidates,
				}, pledge.BlazingFastConfig)
				Expect(err).To(BeNil())
				Expect(id).To(Equal(node.ID(10)))
			})
		})
		Context("Responsible is Missing UniqueNodeIDs", func() {
			It("Should correctly assign an ID", func() {
				var (
					nodes      = make(node.Group)
					candidates = func(i int) func() node.Group {
						return func() node.Group {
							if i == 0 {
								return nodes.Where(func(id node.ID, _ node.Node) bool {
									return !lo.Contains([]node.ID{8, 9, 10}, id)
								})
							}
							return nodes
						}
					}
				)
				nodes = provisionCandidates(10, net, nodes, candidates, nil, logger)
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					baseConfig(net, logger),
					pledge.Config{Peers: []address.Address{nodes[0].Address}},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(BeNil())
				Expect(id).To(Equal(node.ID(10)))
			})
		})
		Context("One juror are aware of a new node", func() {
			It("Should assign the correct ID", func() {
				var (
					nodes           = make(node.Group)
					allCandidates   = func() node.Group { return nodes }
					extraCandidates = func() node.Group {
						n := nodes.Copy()
						n[10] = node.Node{ID: 10, Address: "localhost:10", State: node.StateHealthy}
						return n
					}
					net = fmock.NewNetwork[node.ID, node.ID]()
				)
				provisionCandidates(10, net, nodes, func(i int) func() node.Group {
					return lo.Ternary(i%2 == 0, extraCandidates, allCandidates)
				}, nil, logger)
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					baseConfig(net, logger),
					pledge.Config{
						Peers:      []address.Address{allCandidates()[0].Address},
						Candidates: extraCandidates,
					},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(BeNil())
				Expect(id).To(BeNumerically(">=", node.ID(11)))
			})
		})
		Context("Too Few Healthy UniqueNodeIDs To Form a Quorum", func() {
			It("Should return an errQuorumUnreachable", func() {
				var (
					numCandidates = 10
					nodes         = make(node.Group)
				)
				provisionCandidates(numCandidates, net, nodes, nil, func(i int) node.State {
					return lo.Ternary(i%2 == 0, node.StateHealthy, node.StateDead)
				}, logger)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					baseConfig(net, logger),
					pledge.Config{
						Peers:      []address.Address{nodes[1].Address},
						Candidates: allCandidates(nodes),
					},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(HaveOccurredAs(context.DeadlineExceeded))
				Expect(id).To(Equal(node.ID(0)))
			})
		})
		Describe("Cancelling a pledge", func() {
			It("Should stop all operations and return a cancellation error", func() {
				var (
					numCandidates = 10
					nodes         = make(node.Group)
				)
				provisionCandidates(numCandidates, net, nodes, nil, nil, logger)
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				id, err := pledge.Pledge(ctx, baseConfig(net, logger), pledge.Config{
					Peers:      nodes.Addresses(),
					Candidates: allCandidates(nodes),
				})
				Expect(err).To(HaveOccurredAs(context.Canceled))
				Expect(id).To(Equal(node.ID(0)))
			})
		})

		Context("Concurrent Pledges", func() {
			It("Should assign unique IDs to all pledges", func() {
				var (
					mu         sync.Mutex
					nodes      = make(node.Group)
					candidates = func(i int) func() node.Group {
						mu.Lock()
						defer mu.Unlock()
						return func() node.Group { return nodes.Copy() }
					}
					numCandidates = 10
					numPledges    = 2
				)
				provisionCandidates(numCandidates, net, nodes, candidates, nil, logger)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				var wg sync.WaitGroup
				ids := make([]node.ID, numPledges)
				for i := 0; i < numPledges; i++ {
					wg.Add(1)
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						cfg, addr := baseConfigWithAddr(net, logger)
						id, err := pledge.Pledge(
							ctx,
							cfg,
							pledge.Config{
								Candidates: candidates(0),
								Peers:      nodes.Addresses(),
							},
							pledge.BlazingFastConfig,
						)
						Expect(err).ToNot(HaveOccurred())
						ids[i] = id
						mu.Lock()
						defer mu.Unlock()
						nodes[id] = node.Node{ID: id, Address: addr, State: node.StateHealthy}
					}(i)
				}
				wg.Wait()
				Expect(len(lo.Uniq(ids))).To(Equal(numPledges))
			})

		})

	})

})
