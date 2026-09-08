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
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/control"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// createChannel creates a virtual channel with the given name.
func createChannel(ctx SpecContext, name string) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{Name: name, DataType: telem.Float64T, Virtual: true}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

// openWriter opens a writer that acquires control over keys under the given subject
// name.
func openWriter(
	ctx SpecContext,
	subject string,
	keys ...channel.Key,
) *framer.Writer {
	GinkgoHelper()
	return MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:           keys,
		Start:          telem.SecondTS,
		Sync:           new(true),
		ControlSubject: xcontrol.Subject{Key: subject, Name: subject},
	}))
}

// openControlStreamer streams the control channel and returns its response outlet.
func openControlStreamer(ctx SpecContext) confluence.Outlet[framer.StreamerResponse] {
	GinkgoHelper()
	var ch channel.Channel
	Expect(channelSvc.NewRetrieve().
		Where(channel.MatchNames("sy_control")).
		Entry(&ch).
		Exec(ctx, nil),
	).To(Succeed())
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        channel.Keys{ch.Key()},
		SendOpenAck: true,
	}))
	requests, responses := confluence.Attach(streamer, 10)
	sCtx, cancel := signal.Isolated()
	shutdown := signal.NewHardShutdown(sCtx, cancel)
	DeferCleanup(func() {
		requests.Close()
		confluence.Drain(responses)
		Expect(shutdown.Close()).To(Succeed())
	})
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	// The open ack is the barrier: once it arrives, the relay has applied the
	// streamer's demand, so any transfer after it reaches the outlet.
	Eventually(responses.Outlet()).Should(Receive())
	return responses
}

// receiveUpdates decodes every control update carried by the next streamer response.
func receiveUpdates(
	responses confluence.Outlet[framer.StreamerResponse],
) []control.Update {
	GinkgoHelper()
	var res framer.StreamerResponse
	Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
	var updates []control.Update
	for sample := range res.Frame.SeriesAt(0).Samples() {
		var u control.Update
		Expect(json.Unmarshal(sample, &u)).To(Succeed())
		updates = append(updates, u)
	}
	return updates
}

var _ = Describe("Control", func() {
	Describe("Control channel", func() {
		It("Should be created as a free, internal, virtual JSON channel", func(
			ctx SpecContext,
		) {
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames("sy_control")).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Virtual).To(BeTrue())
			Expect(ch.Free()).To(BeTrue())
			Expect(ch.Internal).To(BeTrue())
			Expect(ch.DataType).To(Equal(telem.JSONT))
		})
	})

	Describe("Retrieve", func() {
		It("Should return the state of a controlled channel", func(ctx SpecContext) {
			ch := createChannel(ctx, "retrieve_held")
			w := openWriter(ctx, "philadelphia", ch.Key())
			Expect(MustSucceed(controlSvc.Retrieve(ctx, ch.Key()))).To(ConsistOf(
				control.State{
					Subject: control.Subject{
						Key:  "philadelphia",
						Name: "philadelphia",
					},
					Resource:  ch.Key(),
					Authority: xcontrol.AuthorityAbsolute,
				},
			))
			Expect(w.Close()).To(Succeed())
			Expect(controlSvc.Retrieve(ctx, ch.Key())).To(BeEmpty())
		})

		It("Should omit channels that no subject controls", func(ctx SpecContext) {
			ch := createChannel(ctx, "retrieve_unheld")
			Expect(controlSvc.Retrieve(ctx, ch.Key())).To(BeEmpty())
		})
	})

	Describe("Publishing", func() {
		It("Should publish an acquire and a release on the control channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, "publish_transfers")
			responses := openControlStreamer(ctx)
			w := openWriter(ctx, "chicago", ch.Key())
			acquire := receiveUpdates(responses)
			Expect(acquire).To(HaveLen(1))
			Expect(acquire[0].Transfers).To(HaveLen(1))
			Expect(acquire[0].Transfers[0].From).To(BeNil())
			Expect(acquire[0].Transfers[0].To.Resource).To(Equal(ch.Key()))
			Expect(acquire[0].Transfers[0].To.Subject.Name).To(Equal("chicago"))
			Expect(w.Close()).To(Succeed())
			release := receiveUpdates(responses)
			Expect(release).To(HaveLen(1))
			Expect(release[0].Transfers).To(HaveLen(1))
			Expect(release[0].Transfers[0].To).To(BeNil())
			Expect(release[0].Transfers[0].From.Resource).To(Equal(ch.Key()))
		})
	})
})
