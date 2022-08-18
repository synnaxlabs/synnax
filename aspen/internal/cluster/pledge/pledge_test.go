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

var _ = Describe("Pledge", Unit, func() {
	var logger *zap.SugaredLogger

	BeforeEach(func() {
		//l, err := zap.NewProduction()
		//Expect(err).ToNot(HaveOccurred())
		logger = zap.NewNop().Sugar()
	})

	Describe("Pledge", func() {

		Context("No nodes Responding", func() {

			It("Should submit round robin propose requests at scaled intervals", func() {
				var (
					addresses     []address.Address
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
					addresses = append(addresses, t.Address)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					addresses,
					func() node.Group { return node.Group{} },
					pledge.Config{
						RequestTimeout: 1 * time.Millisecond,
						Transport:      t1,
						RetryScale:     1,
						RetryInterval:  1 * time.Millisecond,
						Logger:         logger,
					},
				)
				Expect(err).To(HaveOccurredAs(context.DeadlineExceeded))
				Expect(id).To(Equal(node.ID(0)))
				for i, entry := range net.Entries {
					Expect(entry.Target).To(Equal(addresses[i%4]))
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
					cfg := pledge.Config{
						Transport: t, Logger: logger,
						RetryInterval: 1 * time.Millisecond,
					}
					Expect(pledge.Arbitrate(candidates, cfg)).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: node.ID(i), Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					nodes.Addresses(),
					candidates,
					pledge.Config{
						Transport:     t1,
						Logger:        logger,
						RetryScale:    1,
						RetryInterval: 1 * time.Millisecond,
					},
				)
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
					cfg := pledge.Config{Transport: t, Logger: logger}
					if i != 0 {
						Expect(pledge.Arbitrate(allCandidates, cfg)).To(Succeed())
					} else {
						Expect(pledge.Arbitrate(responsibleCandidates, cfg)).To(Succeed())
					}
					id := node.ID(i)
					nodes[id] = node.Node{ID: node.ID(i), Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					[]address.Address{allCandidates()[0].Address},
					responsibleCandidates,
					pledge.Config{
						Transport:     t1,
						Logger:        logger,
						RetryInterval: 1 * time.Millisecond,
						RetryScale:    1,
					},
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
					cfg := pledge.Config{Transport: t, Logger: logger}
					if (i % 2) == 0 {
						Expect(pledge.Arbitrate(allCandidates, cfg)).To(Succeed())
					} else {
						Expect(pledge.Arbitrate(extraCandidates, cfg)).To(Succeed())
					}
					id := node.ID(i)
					nodes[id] = node.Node{ID: node.ID(i), Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(
					ctx,
					[]address.Address{allCandidates()[0].Address},
					extraCandidates,
					pledge.Config{
						Transport:     t1,
						Logger:        logger,
						RetryInterval: 1 * time.Millisecond,
						RetryScale:    1,
					},
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
					var state node.State
					if (i % 2) == 0 {
						state = node.StateHealthy
					} else {
						state = node.StateDead
					}
					cfg := pledge.Config{Transport: t, Logger: logger}
					Expect(pledge.Arbitrate(candidates, cfg)).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: node.ID(i), Address: t.Address, State: state}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()
				id, err := pledge.Pledge(ctx,
					[]address.Address{candidates()[0].Address},
					candidates,
					pledge.Config{Transport: t1, Logger: logger},
				)
				Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
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
					cfg := pledge.Config{
						Transport:     t,
						Logger:        logger,
						RetryInterval: 1 * time.Millisecond,
					}
					Expect(pledge.Arbitrate(candidates, cfg)).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: node.ID(i), Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				id, err := pledge.Pledge(ctx, nodes.Addresses(), candidates, pledge.Config{Transport: t1, Logger: logger})
				Expect(err).To(HaveOccurredAs(context.Canceled))
				Expect(id).To(Equal(node.ID(0)))
			})
		})
		Context("Concurrent Pledges", func() {
			It("Should assign unique IDs to all pledges", func() {
				var (
					nodes         = make(node.Group)
					candidates    = func() node.Group { return nodes }
					net           = fmock.NewNetwork[node.ID, node.ID]()
					t1            = net.RouteUnary("")
					numCandidates = 10
					numPledges    = 2
				)
				for i := 0; i < numCandidates; i++ {
					t := net.RouteUnary("")
					cfg := pledge.Config{Transport: t, Logger: logger}
					Expect(pledge.Arbitrate(candidates, cfg)).To(Succeed())
					id := node.ID(i)
					nodes[id] = node.Node{ID: id, Address: t.Address, State: node.StateHealthy}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				wg := sync.WaitGroup{}
				ids := make([]node.ID, numPledges)
				for i := 0; i < numPledges; i++ {
					wg.Add(1)
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						id, err := pledge.Pledge(ctx, nodes.Addresses(), candidates, pledge.Config{
							Transport:     t1,
							Logger:        logger,
							RetryScale:    1,
							RetryInterval: 1 * time.Millisecond,
						})
						Expect(err).ToNot(HaveOccurred())
						ids[i] = id
					}(i)
				}
				wg.Wait()
				Expect(len(lo.Uniq(ids))).To(Equal(numPledges))
			})

		})

		Context("No peer addresses provided to pledge", func() {

			It("Should return an ErrNoPeers", func() {
				id, err := pledge.Pledge(context.Background(), []address.Address{}, func() node.Group { return nil }, pledge.Config{})
				Expect(err).To(HaveOccurred())
				Expect(id).To(Equal(node.ID(0)))
			})

		})

	})

})
