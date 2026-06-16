// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package actions_test

import (
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	channelmock "github.com/synnaxlabs/synnax/pkg/service/channel/mock"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var serviceNameCounter atomic.Uint64

// nextServiceName returns a distinct lowercased identifier so the set channel
// created by each spec does not collide with previous runs in the same
// process. The mock cluster persists channels across specs.
func nextServiceName() string {
	return fmt.Sprintf("actions_test_%d", serviceNameCounter.Add(1))
}

var _ = Describe("PublishSignals", func() {
	var (
		state       *actions.State[uuid.UUID, testAction]
		serviceName string
		closer      io.Closer
		setChannel  channel.Channel
		streamer    framer.Streamer
		requests    confluence.Inlet[framer.StreamerRequest]
		responses   confluence.Outlet[framer.StreamerResponse]
		shutdown    io.Closer
	)
	BeforeEach(func(ctx SpecContext) {
		state = actions.NewState[uuid.UUID, testAction]()
		serviceName = nextServiceName()
		closer = MustSucceed(actions.PublishSignals(ctx, actions.SignalsConfig[uuid.UUID, testAction]{
			Provider: sigs,
			State:    state,
			Name:     serviceName,
		}))
		DeferCleanup(func() { Expect(closer.Close()).To(Succeed()) })
		Expect(channelmock.ChannelService(dist).NewRetrieve().
			Where(channel.MatchNames(fmt.Sprintf("sy_%s_set", serviceName))).
			Entry(&setChannel).
			Exec(ctx, nil),
		).To(Succeed())
		streamer = MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: channel.Keys{setChannel.Key()},
		}))
		requests, responses = confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		shutdown = signal.NewHardShutdown(sCtx, cancel)
		DeferCleanup(func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(shutdown.Close()).To(Succeed())
		})
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		time.Sleep(10 * time.Millisecond)
	})

	It("Should create the set channel as a free, internal JSON channel", func() {
		Expect(setChannel.Name).To(Equal(fmt.Sprintf("sy_%s_set", serviceName)))
		Expect(setChannel.Virtual).To(BeTrue())
		Expect(setChannel.Free()).To(BeTrue())
		Expect(setChannel.Internal).To(BeTrue())
	})

	It("Should broadcast a Notify call as a JSON-encoded Scoped frame", func(ctx SpecContext) {
		key := uuid.New()
		actionSeq := []testAction{{Type: "rename", Payload: "next-name"}}
		state.Dispatcher().Notify(ctx, key, "dk-1", actionSeq)
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), time.Second*5).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(ConsistOf(setChannel.Key()))
		samples := res.Frame.SeriesAt(0).Samples()
		var decoded []actions.Scoped[uuid.UUID, testAction]
		for sample := range samples {
			var sa actions.Scoped[uuid.UUID, testAction]
			Expect(json.Unmarshal(sample, &sa)).To(Succeed())
			decoded = append(decoded, sa)
		}
		Expect(decoded).To(HaveLen(1))
		Expect(decoded[0].Key).To(Equal(key))
		Expect(decoded[0].DispatchKey).To(Equal("dk-1"))
		Expect(decoded[0].Seq).To(Equal(uint64(1)))
		Expect(decoded[0].Actions).To(Equal(actionSeq))
	})

	It("Should stamp monotonically increasing Seq across successive Notify calls", func(ctx SpecContext) {
		key := uuid.New()
		d := state.Dispatcher()
		d.Notify(ctx, key, "", []testAction{{Type: "a"}})
		d.Notify(ctx, key, "", []testAction{{Type: "b"}})
		d.Notify(ctx, key, "", []testAction{{Type: "c"}})
		var seqs []uint64
		Eventually(func(g Gomega) []uint64 {
			select {
			case res := <-responses.Outlet():
				for sample := range res.Frame.SeriesAt(0).Samples() {
					var sa actions.Scoped[uuid.UUID, testAction]
					g.Expect(json.Unmarshal(sample, &sa)).To(Succeed())
					seqs = append(seqs, sa.Seq)
				}
			default:
			}
			return seqs
		}, time.Second*5).Should(Equal([]uint64{1, 2, 3}))
	})
})

var _ = Describe("SignalsConfig.Validate", func() {
	DescribeTable("Should reject configs missing required fields",
		func(mutate func(*actions.SignalsConfig[uuid.UUID, testAction]), wantField string) {
			state := actions.NewState[uuid.UUID, testAction]()
			cfg := actions.SignalsConfig[uuid.UUID, testAction]{
				Provider: sigs,
				State:    state,
				Name:     "actions_validate",
			}
			mutate(&cfg)
			Expect(cfg.Validate()).To(MatchError(ContainSubstring(wantField)))
		},
		Entry("missing provider", func(c *actions.SignalsConfig[uuid.UUID, testAction]) {
			c.Provider = nil
		}, "provider"),
		Entry("missing state", func(c *actions.SignalsConfig[uuid.UUID, testAction]) {
			c.State = nil
		}, "state"),
		Entry("empty name", func(c *actions.SignalsConfig[uuid.UUID, testAction]) {
			c.Name = ""
		}, "name"),
	)

	It("Should accept a fully-populated config", func() {
		Expect(actions.SignalsConfig[uuid.UUID, testAction]{
			Provider: sigs,
			State:    actions.NewState[uuid.UUID, testAction](),
			Name:     "ok",
		}.Validate()).To(Succeed())
	})
})
