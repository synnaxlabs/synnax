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
	"io"
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

// nextActionsName returns a distinct name, since the mock cluster keeps the set
// channel of every earlier spec.
func nextActionsName() string {
	return fmt.Sprintf("signals_actions_test_%d", actionsNameCounter.Add(1))
}

var _ = Describe("Actions", func() {
	Describe("ActionsConfig", func() {
		full := func() signals.ActionsConfig[uuid.UUID, action] {
			return signals.ActionsConfig[uuid.UUID, action]{
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
				func(clear func(*signals.ActionsConfig[uuid.UUID, action]), field string) {
					cfg := full()
					clear(&cfg)
					Expect(cfg.Validate()).To(MatchError(ContainSubstring(field)))
				},
				Entry(
					"observable",
					func(c *signals.ActionsConfig[uuid.UUID, action]) {
						c.Observable = nil
					},
					"observable",
				),
				Entry(
					"name",
					func(c *signals.ActionsConfig[uuid.UUID, action]) { c.Name = "" },
					"name",
				),
			)
		})
	})

	Describe("PublishActions", func() {
		var (
			state         *actions.State[uuid.UUID, action]
			name          string
			ch            channel.Channel
			closer        io.Closer
			responses     confluence.Outlet[framer.StreamerResponse]
			requests      confluence.Inlet[framer.StreamerRequest]
			closeStreamer io.Closer
		)
		BeforeEach(func(ctx SpecContext) {
			sigs := MustSucceed(signals.New(signals.Config{
				DB:      db,
				Channel: channelSvc,
				Framer:  framerSvc,
			}))
			state = actions.NewState[uuid.UUID, action](nil)
			name = nextActionsName()
			closer = MustSucceed(sigs.PublishActions(
				ctx,
				signals.ActionsConfig[uuid.UUID, action]{Observable: state, Name: name},
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
			requests, responses = confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			closeStreamer = signal.NewHardShutdown(sCtx, cancel)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(responses.Outlet()).Should(Receive())
		})
		AfterEach(func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(closeStreamer.Close()).To(Succeed())
			Expect(closer.Close()).To(Succeed())
		})

		decode := func(g Gomega, res framer.StreamerResponse) []scoped {
			var decoded []scoped
			for sample := range res.Frame.SeriesAt(0).Samples() {
				var sa scoped
				g.Expect(json.Unmarshal(sample, &sa)).To(Succeed())
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
			state.Dispatcher().Notify(ctx, key, "dk-1", acts)
			var res framer.StreamerResponse
			Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(ConsistOf(ch.Key()))
			Expect(decode(Default, res)).To(ConsistOf(scoped{
				Key:         key,
				DispatchKey: "dk-1",
				Seq:         1,
				Actions:     acts,
			}))
		})

		It("Should publish increasing sequence numbers", func(ctx SpecContext) {
			key := uuid.New()
			d := state.Dispatcher()
			d.Notify(ctx, key, "", []action{{Type: "a"}})
			d.Notify(ctx, key, "", []action{{Type: "b"}})
			d.Notify(ctx, key, "", []action{{Type: "c"}})
			var seqs []uint64
			Eventually(func(g Gomega) []uint64 {
				var res framer.StreamerResponse
				g.Eventually(responses.Outlet()).Should(Receive(&res))
				for _, sa := range decode(g, res) {
					seqs = append(seqs, sa.Seq)
				}
				return seqs
			}, 5*time.Second).Should(Equal([]uint64{1, 2, 3}))
		})
	})
})
