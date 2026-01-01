// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gossip_test

import (
	"context"
	"time"

	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
)

var _ = Describe("OperationSender", func() {
	var (
		net *fmock.Network[gossip.Message, gossip.Message]
	)
	BeforeEach(func() {
		net = fmock.NewNetwork[gossip.Message, gossip.Message]()
	})
	Describe("Two Node", func() {
		var (
			t1, t2, t3      *fmock.UnaryServer[gossip.Message, gossip.Message]
			nodes, nodesTwo node.Group
			sOne            store.Store
			g1              *gossip.Gossip
			gossipCtx       signal.Context
			cancel          context.CancelFunc
		)
		BeforeEach(func() {
			t1, t2, t3 = net.UnaryServer(""), net.UnaryServer(""), net.UnaryServer("")
			nodes = node.Group{1: {Key: 1, Address: t1.Address}, 2: {Key: 2, Address: t2.Address}}
			sOne = store.New(ctx)
			sOne.SetState(ctx, store.State{Nodes: nodes, HostKey: 1})
			nodesTwo = nodes.Copy()
			nodesTwo[3] = node.Node{Key: 3, Address: t3.Address, State: node.StateDead}
			sTwo := store.New(ctx)
			sTwo.SetState(ctx, store.State{Nodes: nodesTwo, HostKey: 2})
			gossipCtx, cancel = signal.WithCancel(ctx)
			var err error
			g1, err = gossip.New(gossip.Config{
				Instrumentation: PanicLogger(),
				Store:           sOne,
				TransportClient: net.UnaryClient(),
				TransportServer: t1,
				Interval:        5 * time.Millisecond,
			})
			Expect(err).ToNot(HaveOccurred())
			_, err = gossip.New(gossip.Config{
				Instrumentation: PanicLogger(),
				Store:           sTwo,
				TransportClient: net.UnaryClient(),
				TransportServer: t2,
				Interval:        5 * time.Millisecond,
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
		It("Should DPanic when an invalid message is received", func() {
			Expect(func() {
				_, _ = net.UnaryClient().Send(context.Background(), t2.Address, gossip.Message{})
			}).To(Panic())
		})
	})
})
