// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// createChannel creates a virtual channel leased to the given node.
func createChannel(
	ctx context.Context,
	n mock.Node,
	name string,
	leaseholder node.Key,
) channel.Channel {
	GinkgoHelper()
	return MustSucceed(n.Channel.Create(ctx, []channel.Channel{{
		Name:        name,
		DataType:    telem.Float64T,
		Virtual:     true,
		Leaseholder: leaseholder,
	}}))[0]
}

// openWriter opens a writer on n that acquires control over keys under the given
// subject name.
func openWriter(
	ctx context.Context,
	n mock.Node,
	subject string,
	keys ...channel.Key,
) *writer.Writer {
	GinkgoHelper()
	return MustSucceed(n.Framer.OpenWriter(ctx, writer.Config{
		Keys:           keys,
		Start:          telem.SecondTS,
		Sync:           new(true),
		ControlSubject: xcontrol.Subject{Key: subject, Name: subject},
	}))
}

// collect registers a handler on svc that funnels every update into the returned
// channel.
func collect(svc *control.Service) chan control.Update {
	updates := make(chan control.Update, 20)
	DeferCleanup(svc.OnChange(func(_ context.Context, u control.Update) {
		updates <- u
	}))
	return updates
}

// holders reduces states to a map of channel key to the name of the controlling
// subject.
func holders(states []control.State) map[channel.Key]string {
	out := make(map[channel.Key]string, len(states))
	for _, s := range states {
		out[s.Resource] = s.Subject.Name
	}
	return out
}

var _ = Describe("Control", func() {
	Describe("Retrieve", func() {
		It("Should return the state of a channel the host arbitrates", func(
			ctx SpecContext,
		) {
			gw := mock.NewCluster(ctx, 1).Nodes[1]
			ch := createChannel(ctx, gw, "gateway", 1)
			w := openWriter(ctx, gw, "philadelphia", ch.Key())
			Expect(MustSucceed(gw.Control.Retrieve(ctx, ch.Key()))).To(ConsistOf(
				control.State{
					Subject: xcontrol.Subject{
						Key:  "philadelphia",
						Name: "philadelphia",
					},
					Resource:  ch.Key(),
					Authority: xcontrol.AuthorityAbsolute,
				},
			))
			Expect(w.Close()).To(Succeed())
			Expect(gw.Control.Retrieve(ctx, ch.Key())).To(BeEmpty())
		})

		It("Should omit channels that no subject controls", func(ctx SpecContext) {
			gw := mock.NewCluster(ctx, 1).Nodes[1]
			held := createChannel(ctx, gw, "held", 1)
			free := createChannel(ctx, gw, "unheld", 1)
			w := openWriter(ctx, gw, "denver", held.Key())
			DeferClose(w)
			Expect(holders(MustSucceed(gw.Control.Retrieve(
				ctx,
				held.Key(),
				free.Key(),
			)))).To(Equal(map[channel.Key]string{held.Key(): "denver"}))
		})

		It("Should return the state of every channel when no keys are given", func(
			ctx SpecContext,
		) {
			cluster := mock.NewCluster(ctx, 2)
			gw, peer := cluster.Nodes[1], cluster.Nodes[2]
			local := createChannel(ctx, gw, "local", 1)
			remote := createChannel(ctx, gw, "remote", 2)
			gwWriter := openWriter(ctx, gw, "boulder", local.Key())
			DeferClose(gwWriter)
			peerWriter := openWriter(ctx, peer, "seattle", remote.Key())
			DeferClose(peerWriter)
			Expect(holders(MustSucceed(gw.Control.Retrieve(ctx)))).To(Equal(
				map[channel.Key]string{local.Key(): "boulder", remote.Key(): "seattle"},
			))
		})

		It("Should route to the leaseholder of a channel held by a peer", func(
			ctx SpecContext,
		) {
			cluster := mock.NewCluster(ctx, 2)
			gw, peer := cluster.Nodes[1], cluster.Nodes[2]
			ch := createChannel(ctx, gw, "remote", 2)
			w := openWriter(ctx, peer, "austin", ch.Key())
			DeferClose(w)
			Expect(holders(MustSucceed(gw.Control.Retrieve(ctx, ch.Key())))).To(Equal(
				map[channel.Key]string{ch.Key(): "austin"},
			))
		})
	})

	Describe("OnChange", func() {
		It("Should notify handlers when the host acquires and releases control", func(
			ctx SpecContext,
		) {
			gw := mock.NewCluster(ctx, 1).Nodes[1]
			ch := createChannel(ctx, gw, "gateway", 1)
			updates := collect(gw.Control)
			w := openWriter(ctx, gw, "chicago", ch.Key())
			var acquire control.Update
			Eventually(updates).Should(Receive(&acquire))
			Expect(acquire.Transfers).To(HaveLen(1))
			Expect(acquire.Transfers[0].From).To(BeNil())
			Expect(acquire.Transfers[0].To.Resource).To(Equal(ch.Key()))
			Expect(acquire.Transfers[0].To.Subject.Name).To(Equal("chicago"))
			Expect(w.Close()).To(Succeed())
			var release control.Update
			Eventually(updates).Should(Receive(&release))
			Expect(release.Transfers).To(HaveLen(1))
			Expect(release.Transfers[0].To).To(BeNil())
			Expect(release.Transfers[0].From.Resource).To(Equal(ch.Key()))
		})

		It("Should notify handlers when a peer acquires and releases control", func(
			ctx SpecContext,
		) {
			cluster := mock.NewCluster(ctx, 2)
			gw, peer := cluster.Nodes[1], cluster.Nodes[2]
			ch := createChannel(ctx, gw, "remote", 2)
			updates := collect(gw.Control)
			w := openWriter(ctx, peer, "portland", ch.Key())
			Eventually(updates).Should(Receive(WithTransform(
				func(u control.Update) []control.Transfer { return u.Transfers },
				ConsistOf(control.Transfer{To: &control.State{
					Subject:   xcontrol.Subject{Key: "portland", Name: "portland"},
					Resource:  ch.Key(),
					Authority: xcontrol.AuthorityAbsolute,
				}}),
			)))
			Expect(w.Close()).To(Succeed())
			Eventually(updates).Should(Receive(WithTransform(
				func(u control.Update) []control.Transfer { return u.Transfers },
				ConsistOf(control.Transfer{From: &control.State{
					Subject:   xcontrol.Subject{Key: "portland", Name: "portland"},
					Resource:  ch.Key(),
					Authority: xcontrol.AuthorityAbsolute,
				}}),
			)))
		})
	})
})
