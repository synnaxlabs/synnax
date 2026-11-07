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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// createChannel creates a virtual channel leased to the given node.
func createChannel(
	ctx context.Context,
	n testutil.Node,
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
	n testutil.Node,
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
	Describe("ServiceConfig", func() {
		full := func(ctx context.Context) control.ServiceConfig {
			GinkgoHelper()
			n := testutil.NewNode(ctx)
			return control.ServiceConfig{TS: n.Storage.TS}
		}

		It("Should open when all required fields are set", func(ctx SpecContext) {
			Expect(full(ctx).Validate()).To(Succeed())
		})

		It("Should fail to open without a time-series database", func(ctx SpecContext) {
			Expect(control.OpenService(ctx, control.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("ts: must be non-nil")))
		})
	})

	Describe("Retrieve", func() {
		It("Should return the state of a channel the host arbitrates", func(
			ctx SpecContext,
		) {
			host := testutil.NewNode(ctx)
			ch := createChannel(ctx, host, "gateway", 1)
			w := openWriter(ctx, host, "philadelphia", ch.Key())
			Expect(MustSucceed(host.Control.Retrieve(ctx, ch.Key()))).To(ConsistOf(
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
			Expect(host.Control.Retrieve(ctx, ch.Key())).To(BeEmpty())
		})

		It("Should omit channels that no subject controls", func(ctx SpecContext) {
			host := testutil.NewNode(ctx)
			held := createChannel(ctx, host, "held", 1)
			free := createChannel(ctx, host, "unheld", 1)
			w := openWriter(ctx, host, "denver", held.Key())
			DeferClose(w)
			Expect(holders(MustSucceed(host.Control.Retrieve(
				ctx,
				held.Key(),
				free.Key(),
			)))).To(Equal(map[channel.Key]string{held.Key(): "denver"}))
		})

		It("Should return the state of every channel when no keys are given", func(
			ctx SpecContext,
		) {
			host := testutil.NewNode(ctx)
			first := createChannel(ctx, host, "first", 1)
			second := createChannel(ctx, host, "second", 1)
			DeferClose(openWriter(ctx, host, "boulder", first.Key()))
			DeferClose(openWriter(ctx, host, "seattle", second.Key()))
			Expect(holders(MustSucceed(host.Control.Retrieve(ctx)))).To(Equal(
				map[channel.Key]string{first.Key(): "boulder", second.Key(): "seattle"},
			))
		})
	})

	Describe("OnChange", func() {
		It("Should notify handlers when the host acquires and releases control", func(
			ctx SpecContext,
		) {
			host := testutil.NewNode(ctx)
			ch := createChannel(ctx, host, "gateway", 1)
			updates := collect(host.Control)
			w := openWriter(ctx, host, "chicago", ch.Key())
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
	})
})
