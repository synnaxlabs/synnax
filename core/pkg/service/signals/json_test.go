// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"encoding/json"
	"io"
	"math"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const jsonSetChannelName = "json_publisher_set"

type jsonPayload struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

var _ = Describe("JSON", func() {
	Describe("JSONPublisherConfig", func() {
		full := func() signals.JSONPublisherConfig[jsonPayload] {
			return signals.JSONPublisherConfig[jsonPayload]{
				Observable: observe.New[jsonPayload](),
				SetName:    jsonSetChannelName,
			}
		}

		Describe("Validate", func() {
			It("Should succeed when all required fields are set", func() {
				Expect(full().Validate()).To(Succeed())
			})

			DescribeTable(
				"Should fail when a required field is missing",
				func(clear func(*signals.JSONPublisherConfig[jsonPayload]), field string) {
					cfg := full()
					clear(&cfg)
					Expect(cfg.Validate()).To(MatchError(ContainSubstring(field)))
				},
				Entry(
					"observable",
					func(c *signals.JSONPublisherConfig[jsonPayload]) {
						c.Observable = nil
					},
					"observable",
				),
				Entry(
					"set name",
					func(c *signals.JSONPublisherConfig[jsonPayload]) { c.SetName = "" },
					"set_name",
				),
			)
		})
		Describe("Override", func() {
			It("Should default the pipeline name to the set channel name", func() {
				cfg := signals.JSONPublisherConfig[jsonPayload]{}.
					Override(signals.JSONPublisherConfig[jsonPayload]{
						SetName: jsonSetChannelName,
					})
				Expect(cfg.Name).To(Equal(jsonSetChannelName))
			})
			It("Should keep an explicitly provided pipeline name", func() {
				cfg := signals.JSONPublisherConfig[jsonPayload]{Name: "custom"}.
					Override(signals.JSONPublisherConfig[jsonPayload]{
						SetName: jsonSetChannelName,
					})
				Expect(cfg.Name).To(Equal("custom"))
			})
		})
	})

	Describe("PublishJSON", Serial, func() {
		var (
			obs           observe.Observer[jsonPayload]
			ch            channel.Channel
			closer        io.Closer
			responses     confluence.Outlet[framer.StreamerResponse]
			requests      confluence.Inlet[framer.StreamerRequest]
			closeStreamer io.Closer
		)
		BeforeEach(func(ctx SpecContext) {
			sigs := MustSucceed(signals.New(signals.Config{
				Channel: channelSvc,
				Framer:  framerSvc,
			}))
			obs = observe.New[jsonPayload]()
			closer = MustSucceed(
				sigs.PublishJSON(ctx, signals.JSONPublisherConfig[jsonPayload]{
					Observable: obs,
					SetName:    jsonSetChannelName,
				}),
			)
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(jsonSetChannelName)).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
				Keys:        channel.Keys{ch.Key()},
				SendOpenAck: true,
			}))
			requests, responses = confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			closeStreamer = signal.NewHardShutdown(sCtx, cancel)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			// The open ack is the barrier: once it arrives, the relay has applied the
			// streamer's demand, so any update after it reaches the outlet.
			Eventually(responses.Outlet()).Should(Receive())
		})
		AfterEach(func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(closeStreamer.Close()).To(Succeed())
			Expect(closer.Close()).To(Succeed())
		})
		It(
			"Should create the set channel as a free, internal, virtual JSON channel",
			func() {
				Expect(ch.Virtual).To(BeTrue())
				Expect(ch.Free()).To(BeTrue())
				Expect(ch.Internal).To(BeTrue())
				Expect(ch.DataType).To(Equal(telem.JSONT))
			},
		)
		It("Should publish an emitted value as a single JSON sample", func(
			ctx SpecContext,
		) {
			v := jsonPayload{Name: "cat", Value: 12.5}
			obs.Notify(ctx, v)
			var res framer.StreamerResponse
			Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(ConsistOf(ch.Key()))
			samples := make([]jsonPayload, 0, 1)
			for sample := range res.Frame.SeriesAt(0).Samples() {
				var decoded jsonPayload
				Expect(json.Unmarshal(sample, &decoded)).To(Succeed())
				samples = append(samples, decoded)
			}
			Expect(samples).To(ConsistOf(v))
		})
		It("Should drop a value that cannot be marshalled", func(ctx SpecContext) {
			obs.Notify(ctx, jsonPayload{Name: "cat", Value: math.Inf(1)})
			Consistently(responses.Outlet()).ShouldNot(Receive())
		})
	})
})
