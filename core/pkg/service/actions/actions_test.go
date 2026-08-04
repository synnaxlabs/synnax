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
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scoped", func() {
	It("Should marshal to snake_case JSON with all fields", func() {
		key := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		scoped := actions.Scoped[uuid.UUID, testAction]{
			Key:         key,
			DispatchKey: "batch-7",
			Seq:         42,
			Actions:     []testAction{{Type: "rename", Payload: "next"}},
		}
		b := MustSucceed(json.Marshal(scoped))
		var decoded map[string]any
		Expect(json.Unmarshal(b, &decoded)).To(Succeed())
		Expect(decoded).To(HaveKeyWithValue("key", key.String()))
		Expect(decoded).To(HaveKeyWithValue("dispatch_key", "batch-7"))
		Expect(decoded).To(HaveKeyWithValue("seq", float64(42)))
		Expect(decoded).To(HaveKey("actions"))
	})

	It("Should round-trip an action sequence through JSON", func() {
		key := uuid.New()
		original := actions.Scoped[uuid.UUID, testAction]{
			Key:         key,
			DispatchKey: "dk",
			Seq:         3,
			Actions: []testAction{
				{Type: "a", Payload: "one"},
				{Type: "b", Payload: "two"},
			},
		}
		b := MustSucceed(json.Marshal(original))
		var decoded actions.Scoped[uuid.UUID, testAction]
		Expect(json.Unmarshal(b, &decoded)).To(Succeed())
		Expect(decoded).To(Equal(original))
	})
})

var _ = Describe("DispatchRequest", func() {
	It("Should marshal to the same wire shape as Scoped without Seq", func() {
		req := actions.DispatchRequest[uuid.UUID, testAction]{
			Key:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			DispatchKey: "dk",
			Actions:     []testAction{{Type: "x", Payload: "y"}},
		}
		b := MustSucceed(json.Marshal(req))
		var decoded map[string]any
		Expect(json.Unmarshal(b, &decoded)).To(Succeed())
		Expect(decoded).To(HaveKeyWithValue("dispatch_key", "dk"))
		Expect(decoded).NotTo(HaveKey("seq"))
	})
})

var _ = Describe("State", func() {
	It("Should hand out a usable Dispatcher", func(ctx SpecContext) {
		state := actions.NewState[uuid.UUID, testAction]()
		received := make(chan actions.Scoped[uuid.UUID, testAction], 1)
		state.OnAction(
			func(_ context.Context, sa actions.Scoped[uuid.UUID, testAction]) {
				received <- sa
			},
		)
		key := uuid.New()
		state.Dispatcher().Notify(ctx, key, "dk", []testAction{{Type: "ping"}})
		var got actions.Scoped[uuid.UUID, testAction]
		Eventually(received).Should(Receive(&got))
		Expect(got.Key).To(Equal(key))
		Expect(got.DispatchKey).To(Equal("dk"))
		Expect(got.Seq).To(Equal(uint64(1)))
		Expect(got.Actions).To(HaveLen(1))
	})

	It(
		"Should stamp a monotonically increasing Seq across dispatches",
		func(ctx SpecContext) {
			state := actions.NewState[uuid.UUID, testAction]()
			seqs := make(chan uint64, 3)
			state.OnAction(
				func(_ context.Context, sa actions.Scoped[uuid.UUID, testAction]) {
					seqs <- sa.Seq
				},
			)
			d := state.Dispatcher()
			key := uuid.New()
			for range 3 {
				d.Notify(ctx, key, "", nil)
			}
			var got [3]uint64
			for i := range got {
				Eventually(seqs).Should(Receive(&got[i]))
			}
			Expect(got).To(Equal([3]uint64{1, 2, 3}))
		},
	)

	It(
		"Should share the Seq counter across Dispatchers handed out by the same State",
		func(ctx SpecContext) {
			state := actions.NewState[uuid.UUID, testAction]()
			var maxSeq atomic.Uint64
			state.OnAction(
				func(_ context.Context, sa actions.Scoped[uuid.UUID, testAction]) {
					for {
						prev := maxSeq.Load()
						if sa.Seq <= prev || maxSeq.CompareAndSwap(prev, sa.Seq) {
							break
						}
					}
				},
			)
			d1 := state.Dispatcher()
			d2 := state.Dispatcher()
			key := uuid.New()
			d1.Notify(ctx, key, "", nil)
			d2.Notify(ctx, key, "", nil)
			d1.Notify(ctx, key, "", nil)
			Eventually(maxSeq.Load).Should(Equal(uint64(3)))
		},
	)

	It("Should serialize Seq under concurrent Notify", func(ctx SpecContext) {
		state := actions.NewState[uuid.UUID, testAction]()
		const N = 200
		var mu sync.Mutex
		seen := make([]uint64, 0, N)
		state.OnAction(
			func(_ context.Context, sa actions.Scoped[uuid.UUID, testAction]) {
				mu.Lock()
				seen = append(seen, sa.Seq)
				mu.Unlock()
			},
		)
		d := state.Dispatcher()
		var wg sync.WaitGroup
		wg.Add(N)
		for range N {
			go func() {
				defer wg.Done()
				d.Notify(ctx, uuid.New(), "", nil)
			}()
		}
		wg.Wait()
		mu.Lock()
		defer mu.Unlock()
		Expect(seen).To(HaveLen(N))
		uniq := set.New(seen...)
		Expect(uniq).To(HaveLen(N))
		maxSeq := uint64(0)
		for s := range uniq {
			if s > maxSeq {
				maxSeq = s
			}
		}
		Expect(maxSeq).To(Equal(uint64(N)))
	})

	It(
		"Should fan out to every subscriber registered via OnAction",
		func(ctx SpecContext) {
			state := actions.NewState[uuid.UUID, testAction]()
			a := make(chan actions.Scoped[uuid.UUID, testAction], 1)
			b := make(chan actions.Scoped[uuid.UUID, testAction], 1)
			state.OnAction(
				func(_ context.Context, sa actions.Scoped[uuid.UUID, testAction]) { a <- sa },
			)
			state.OnAction(
				func(_ context.Context, sa actions.Scoped[uuid.UUID, testAction]) { b <- sa },
			)
			state.Dispatcher().Notify(ctx, uuid.New(), "", []testAction{{Type: "fan"}})
			Eventually(a).Should(Receive())
			Eventually(b).Should(Receive())
		},
	)

	It("Should stop notifying a subscriber after Disconnect", func(ctx SpecContext) {
		state := actions.NewState[uuid.UUID, testAction]()
		var hits atomic.Uint64
		disconnect := state.OnAction(
			func(_ context.Context, _ actions.Scoped[uuid.UUID, testAction]) {
				hits.Add(1)
			},
		)
		key := uuid.New()
		state.Dispatcher().Notify(ctx, key, "", nil)
		Eventually(hits.Load).Should(Equal(uint64(1)))
		disconnect()
		state.Dispatcher().Notify(ctx, key, "", nil)
		Consistently(hits.Load).Should(Equal(uint64(1)))
	})

	It(
		"Should carry the action sequence verbatim onto the Scoped envelope",
		func(ctx SpecContext) {
			state := actions.NewState[uuid.UUID, testAction]()
			received := make(chan actions.Scoped[uuid.UUID, testAction], 1)
			state.OnAction(
				func(_ context.Context, sa actions.Scoped[uuid.UUID, testAction]) {
					received <- sa
				},
			)
			actionSeq := []testAction{
				{Type: "a", Payload: "1"},
				{Type: "b", Payload: "2"},
				{Type: "c", Payload: "3"},
			}
			state.Dispatcher().Notify(ctx, uuid.New(), "dk", actionSeq)
			var got actions.Scoped[uuid.UUID, testAction]
			Eventually(received).Should(Receive(&got))
			Expect(got.Actions).To(Equal(actionSeq))
		},
	)
})
