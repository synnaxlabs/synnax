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
	"encoding/json/v2"
	"fmt"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type action struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type scoped = actions.Scoped[uuid.UUID, action]

var actionsNameCounter atomic.Uint64

// nextActionsName returns a distinct name, since the mock cluster keeps the set channel
// of every earlier spec.
func nextActionsName() string {
	return fmt.Sprintf("signals_actions_test_%d", actionsNameCounter.Add(1))
}

var _ = Describe("Actions", func() {
	Describe("ActionPublisherConfig", func() {
		full := func() signals.ActionPublisherConfig[uuid.UUID, action] {
			return signals.ActionPublisherConfig[uuid.UUID, action]{
				Observable: actions.NewState[uuid.UUID, action](nil),
				Name:       "actions_validate",
			}
		}

		Describe("Validate", func() {
			It("Should succeed when all required fields are set", func() {
				Expect(full().Validate()).To(Succeed())
			})

			DescribeTable(
				"Should fail when a required field is missing",
				func(clear func(*signals.ActionPublisherConfig[uuid.UUID, action]), field string) {
					cfg := full()
					clear(&cfg)
					Expect(cfg.Validate()).To(MatchError(ContainSubstring(field)))
				},
				Entry(
					"observable",
					func(c *signals.ActionPublisherConfig[uuid.UUID, action]) {
						c.Observable = nil
					},
					"observable",
				),
				Entry(
					"name",
					func(c *signals.ActionPublisherConfig[uuid.UUID, action]) { c.Name = "" },
					"name",
				),
			)
		})
	})

	Describe("PublishActions", func() {
		var (
			state     *actions.State[uuid.UUID, action]
			ch        channel.Channel
			responses confluence.Outlet[framer.StreamerResponse]
		)
		BeforeEach(func(ctx SpecContext) {
			sigs := MustSucceed(signals.New(signals.Config{
				DB:      db,
				Channel: channelSvc,
				Framer:  framerSvc,
			}))
			state = actions.NewState[uuid.UUID, action](nil)
			name := nextActionsName()
			MustOpen(sigs.PublishActions(
				ctx,
				signals.ActionPublisherConfig[uuid.UUID, action]{
					Observable: state,
					Name:       name,
				},
			))
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(fmt.Sprintf("sy_%s_set", name))).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
				Keys:        channel.Keys{ch.Key()},
				SendOpenAck: true,
			}))
			var requests confluence.Inlet[framer.StreamerRequest]
			requests, responses = confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			closeStreamer := signal.NewHardShutdown(sCtx, cancel)
			DeferCleanup(func() {
				requests.Close()
				confluence.Drain(responses)
				Expect(closeStreamer.Close()).To(Succeed())
			})
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(responses.Outlet()).Should(Receive())
		})

		decode := func(res framer.StreamerResponse) []scoped {
			GinkgoHelper()
			var decoded []scoped
			for sample := range res.Frame.SeriesAt(0).Samples() {
				var sa scoped
				Expect(json.Unmarshal(sample, &sa)).To(Succeed())
				decoded = append(decoded, sa)
			}
			return decoded
		}

		It("Should create the set channel as a free, internal JSON channel", func() {
			Expect(ch.Virtual).To(BeTrue())
			Expect(ch.Free()).To(BeTrue())
			Expect(ch.Internal).To(BeTrue())
			Expect(ch.DataType).To(Equal(telem.JSONT))
		})

		It("Should publish a notified action sequence as one sample", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			acts := []action{{Type: "rename", Payload: "next-name"}}
			state.Dispatcher().WithDispatchKey("dk-1").Notify(ctx, key, acts...)
			var res framer.StreamerResponse
			Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(ConsistOf(ch.Key()))
			Expect(decode(res)).To(ConsistOf(scoped{
				Key:         key,
				DispatchKey: "dk-1",
				Seq:         1,
				Actions:     acts,
			}))
		})

		It("Should publish increasing sequence numbers", func(ctx SpecContext) {
			key := uuid.New()
			d := state.Dispatcher()
			d.Notify(ctx, key, action{Type: "a"})
			d.Notify(ctx, key, action{Type: "b"})
			d.Notify(ctx, key, action{Type: "c"})
			var seqs []uint64
			for len(seqs) < 3 {
				var res framer.StreamerResponse
				Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
				for _, sa := range decode(res) {
					seqs = append(seqs, sa.Seq)
				}
			}
			Expect(seqs).To(Equal([]uint64{1, 2, 3}))
		})
	})
})
