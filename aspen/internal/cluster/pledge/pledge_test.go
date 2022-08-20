package pledge_test

import (
	"context"
	"github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/address"
	. "github.com/arya-analytics/x/testutil"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"sync"
	"time"
)

var _ = Describe("Pledge", func() {
	var logger *zap.SugaredLogger

	BeforeEach(func() { logger = zap.NewNop().Sugar() })

	Describe("Pledge", func() {

		Context("No nodes Responding", func() {
			It("Should submit round robin proposals at scaled intervals", func() {
				var (
					peers         []address.Address
					numTransports = 4
					net           = fmock.NewNetwork[node.ID, node.ID]()
					handler       = func(ctx context.Context, id node.ID) (node.ID, error) {
						return 0, errors.New("pledge failed")
					}
					t1 = net.RouteUnary("")
				)
				for i := 0; i < numTransports; i++ {
					t := net.RouteUnary("")
					t.BindHandler(handler)
					peers = append(peers, t.Address)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(ctx, pledge.Config{
					Peers:      peers,
					Transport:  t1,
					Candidates: func() node.Group { return node.Group{} },
					Logger:     logger,
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
					candidates    = func() node.Group { return nodes }
					net           = fmock.NewNetwork[node.ID, node.ID]()
					t1            = net.RouteUnary("")
					numCandidates = 10
				)
				for i := 0; i < numCandidates; i++ {
					t := net.RouteUnary("")
					Expect(pledge.Arbitrate(pledge.Config{Candidates: candidates, Transport: t})).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: id, Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(ctx, pledge.Config{
					Peers:      nodes.Addresses(),
					Candidates: candidates,
					Transport:  t1,
				}, pledge.BlazingFastConfig)
				Expect(err).To(BeNil())
				Expect(id).To(Equal(node.ID(10)))
			})
		})
		Context("Responsible is Missing UniqueNodeIDs", func() {
			It("Should correctly assign an ID", func() {
				var (
					nodes                 = make(node.Group)
					allCandidates         = func() node.Group { return nodes }
					responsibleCandidates = func() node.Group {
						return allCandidates().Where(func(id node.ID, _ node.Node) bool {
							return !lo.Contains([]node.ID{8, 9, 10}, id)
						})
					}
					net = fmock.NewNetwork[node.ID, node.ID]()
					t1  = net.RouteUnary("")
				)
				for i := 0; i < 10; i++ {
					t := net.RouteUnary("")
					Expect(pledge.Arbitrate(pledge.Config{
						Logger:     logger,
						Transport:  t,
						Candidates: lo.Ternary(i == 0, responsibleCandidates, allCandidates),
					})).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: node.ID(i), Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					pledge.Config{
						Peers:      []address.Address{allCandidates()[0].Address},
						Transport:  t1,
						Logger:     logger,
						Candidates: responsibleCandidates,
					},
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
						return node.Group{10: node.Node{ID: 10, Address: "localhost:10", State: node.StateHealthy}}
					}
					net = fmock.NewNetwork[node.ID, node.ID]()
					t1  = net.RouteUnary("")
				)
				for i := 0; i < 10; i++ {
					t := net.RouteUnary("")
					Expect(pledge.Arbitrate(pledge.Config{
						Logger:     logger,
						Transport:  t,
						Candidates: lo.Ternary(i%2 == 0, allCandidates, extraCandidates),
					})).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: id, Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					pledge.Config{
						Peers:      []address.Address{allCandidates()[0].Address},
						Candidates: extraCandidates,
						Transport:  t1,
						Logger:     logger,
					},
					pledge.BlazingFastConfig,
				)
				Expect(err).To(BeNil())
				Expect(id).To(Equal(node.ID(11)))
			})
		})
		Context("Too Few Healthy UniqueNodeIDs SinkTarget Form a Quorum", func() {
			It("Should return an errQuorumUnreachable", func() {
				var (
					nodes         = make(node.Group)
					candidates    = func() node.Group { return nodes }
					net           = fmock.NewNetwork[node.ID, node.ID]()
					t1            = net.RouteUnary("")
					numCandidates = 10
				)
				for i := 0; i < numCandidates; i++ {
					t := net.RouteUnary("")
					Expect(pledge.Arbitrate(pledge.Config{
						Logger:     logger,
						Transport:  t,
						Candidates: candidates,
					})).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{
						ID:      id,
						Address: t.Address,
						State:   lo.Ternary(i%2 == 0, node.StateHealthy, node.StateDead),
					}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					pledge.Config{
						Peers:      []address.Address{candidates()[0].Address},
						Candidates: candidates,
						Transport:  t1,
						Logger:     logger,
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
					nodes         = make(node.Group)
					candidates    = func() node.Group { return nodes }
					net           = fmock.NewNetwork[node.ID, node.ID]()
					t1            = net.RouteUnary("")
					numCandidates = 10
				)
				for i := 0; i < numCandidates; i++ {
					t := net.RouteUnary("")
					Expect(pledge.Arbitrate(pledge.Config{
						Logger:     logger,
						Candidates: candidates,
						Transport:  t,
					})).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: id, Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				id, err := pledge.Pledge(ctx, pledge.Config{
					Peers:      nodes.Addresses(),
					Candidates: candidates,
					Transport:  t1,
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
					candidates = func() node.Group {
						mu.Lock()
						defer mu.Unlock()
						return nodes.Copy()
					}
					net           = fmock.NewNetwork[node.ID, node.ID]()
					numCandidates = 10
					numPledges    = 2
				)

				for i := 0; i < numCandidates; i++ {
					t := net.RouteUnary("")
					Expect(pledge.Arbitrate(pledge.Config{
						Transport:  t,
						Candidates: candidates,
					})).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: id, Address: t.Address, State: node.StateHealthy}
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				var wg sync.WaitGroup
				ids := make([]node.ID, numPledges)
				for i := 0; i < numPledges; i++ {
					wg.Add(1)
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						t := net.RouteUnary("")
						id, err := pledge.Pledge(
							ctx,
							pledge.Config{
								Candidates: candidates,
								Peers:      nodes.Addresses(),
								Transport:  t,
							},
							pledge.BlazingFastConfig,
						)
						Expect(err).ToNot(HaveOccurred())
						ids[i] = id
						mu.Lock()
						defer mu.Unlock()
						nodes[id] = node.Node{ID: id, Address: t.Address, State: node.StateHealthy}
					}(i)
				}
				wg.Wait()
				Expect(len(lo.Uniq(ids))).To(Equal(numPledges))
			})

		})

	})

})
