package gossip_test

import (
	"context"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/signal"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/internal/node"
	"go.uber.org/zap"
	"time"

	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
)

var _ = Describe("OperationSender", func() {
	var (
		net    *fmock.Network[gossip.Message, gossip.Message]
		logger *zap.SugaredLogger
	)
	BeforeEach(func() {
		net = fmock.NewNetwork[gossip.Message, gossip.Message]()
		logger = zap.NewNop().Sugar()
	})
	Describe("Two Node", func() {
		var (
			t1, t2, t3      *fmock.Unary[gossip.Message, gossip.Message]
			nodes, nodesTwo node.Group
			sOne            store.Store
			g1              *gossip.Gossip
			gossipCtx       signal.Context
			cancel          context.CancelFunc
		)
		BeforeEach(func() {
			t1, t2, t3 = net.RouteUnary(""), net.RouteUnary(""), net.RouteUnary("")
			nodes = node.Group{1: {ID: 1, Address: t1.Address}, 2: {ID: 2, Address: t2.Address}}
			sOne = store.New()
			sOne.SetState(store.State{Nodes: nodes, HostID: 1})
			nodesTwo = nodes.Copy()
			nodesTwo[3] = node.Node{ID: 3, Address: t3.Address, State: node.StateDead}
			sTwo := store.New()
			sTwo.SetState(store.State{Nodes: nodesTwo, HostID: 2})
			gossipCtx, cancel = signal.WithCancel(ctx)
			var err error
			g1, err = gossip.New(gossip.Config{
				Store:     sOne,
				Transport: t1,
				Logger:    logger,
				Interval:  5 * time.Millisecond,
			})
			Expect(err).ToNot(HaveOccurred())
			_, err = gossip.New(gossip.Config{
				Store:     sTwo,
				Transport: t2,
				Logger:    logger,
				Interval:  5 * time.Millisecond,
			})
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should converge after a single exchange", func() {
			Expect(g1.GossipOnce(gossipCtx)).To(Succeed())
			Expect(sOne.CopyState().Nodes).To(HaveLen(3))
			Expect(sOne.CopyState().Nodes[1].Heartbeat.Version).To(Equal(uint32(1)))
			Expect(sOne.CopyState().Nodes[3].State).To(Equal(node.StateDead))
			Expect(sOne.CopyState().Nodes[2].Heartbeat.Version).To(Equal(uint32(0)))
			cancel()
		})
		It("Should gossip at the correct interval", func() {
			g1.GoGossip(gossipCtx)
			defer func() {
				cancel()
				Expect(errors.Is(gossipCtx.Wait(), context.Canceled)).To(BeTrue())
			}()
			Eventually(func(g Gomega) {
				g.Expect(sOne.CopyState().Nodes).To(HaveLen(3))
				g.Expect(sOne.CopyState().Nodes[1].Heartbeat.Version).To(BeNumerically(">", uint32(2)))
				g.Expect(sOne.CopyState().Nodes[3].State).To(Equal(node.StateDead))
				g.Expect(sOne.CopyState().Nodes[2].Heartbeat.Version).To(Equal(uint32(0)))
			}).Should(Succeed())
		})
		It("Should return an error when an invalid message is received", func() {
			_, err := t1.Send(context.Background(), t2.Address, gossip.Message{})
			Expect(err).To(HaveOccurred())
		})
	})
})
