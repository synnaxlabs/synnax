// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Dispatch", func() {
	var (
		setChannel channel.Channel
		requests   confluence.Inlet[framer.StreamerRequest]
		responses  confluence.Outlet[framer.StreamerResponse]
		shutdown   io.Closer
	)
	BeforeEach(func(ctx SpecContext) {
		Expect(dist.Channel.NewRetrieve().
			Where(channel.MatchNames("sy_arc_set")).
			Entry(&setChannel).
			Exec(ctx, nil)).To(Succeed())
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
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

	insert := func(replica uint32, counter uint64, char rune) arc.Action {
		return arc.NewInsertCharAction(arc.InsertCharPayload{Op: crdt.Insert{
			ID:   crdt.ID{Replica: replica, Counter: counter},
			Side: crdt.SideRight,
			Char: char,
		}})
	}

	It("Should broadcast dispatched actions on the arc set channel", func(ctx SpecContext) {
		key := uuid.New()
		seq := []arc.Action{
			insert(1, 1, 'h'),
			arc.NewDeleteCharAction(arc.DeleteCharPayload{Op: crdt.Delete{
				ID: crdt.ID{Replica: 1, Counter: 1},
			}}),
		}
		Expect(svc.NewWriter(nil).Dispatch(ctx, key, "dk-1", seq)).To(Succeed())
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), time.Second*5).Should(Receive(&res))
		var decoded []actions.Scoped[arc.Key, arc.Action]
		for sample := range res.Frame.SeriesAt(0).Samples() {
			var sa actions.Scoped[arc.Key, arc.Action]
			Expect(json.Unmarshal(sample, &sa)).To(Succeed())
			decoded = append(decoded, sa)
		}
		Expect(decoded).To(HaveLen(1))
		Expect(decoded[0].Key).To(Equal(key))
		Expect(decoded[0].DispatchKey).To(Equal("dk-1"))
		Expect(decoded[0].Actions).To(Equal(seq))
	})

	It("Should scope each broadcast to the dispatched arc key", func(ctx SpecContext) {
		keyA, keyB := uuid.New(), uuid.New()
		Expect(svc.NewWriter(nil).Dispatch(ctx, keyA, "", []arc.Action{insert(1, 1, 'a')})).To(Succeed())
		Expect(svc.NewWriter(nil).Dispatch(ctx, keyB, "", []arc.Action{insert(2, 1, 'b')})).To(Succeed())
		var keys []arc.Key
		Eventually(func(g Gomega) []arc.Key {
			select {
			case res := <-responses.Outlet():
				for sample := range res.Frame.SeriesAt(0).Samples() {
					var sa actions.Scoped[arc.Key, arc.Action]
					g.Expect(json.Unmarshal(sample, &sa)).To(Succeed())
					keys = append(keys, sa.Key)
				}
			default:
			}
			return keys
		}, time.Second*5).Should(ConsistOf(keyA, keyB))
	})
})
