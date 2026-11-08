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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	gorptestutil "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/x/errors"
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

var _ = Describe("DispatchResponse", func() {
	It("Should marshal its actions under the actions key", func() {
		res := actions.DispatchResponse[testAction]{
			Actions: []testAction{{Type: "x", Payload: "y"}},
		}
		b := MustSucceed(json.Marshal(res))
		var decoded map[string]any
		Expect(json.Unmarshal(b, &decoded)).To(Succeed())
		Expect(decoded).To(HaveKeyWithValue("actions", HaveLen(1)))
	})
})

var _ = Describe("State", func() {
	var (
		db    *gorp.DB
		state *actions.State[uuid.UUID, testAction]
		rec   *Recorder[uuid.UUID, testAction]
	)
	BeforeEach(func() {
		db = DeferClose(gorptestutil.OpenGorpMsgpackDB())
		state = actions.NewState[uuid.UUID, testAction](db)
		rec = &Recorder[uuid.UUID, testAction]{}
		DeferCleanup(state.OnChange(rec.Record))
	})

	attach := func(
		ctx SpecContext,
		d actions.Dispatcher[uuid.UUID, testAction],
		key uuid.UUID,
		acts ...testAction,
	) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			d.Attach(tx, key, acts...)
			return nil
		})).To(Succeed())
	}

	It("Should deliver an attached batch with its commit", func(ctx SpecContext) {
		key := uuid.New()
		attach(ctx, state.Dispatcher(), key, testAction{Type: "ping"})
		Expect(rec.Snapshot()).To(Equal([]actions.Scoped[uuid.UUID, testAction]{{
			Key:     key,
			Seq:     1,
			Actions: []testAction{{Type: "ping"}},
		}}))
	})

	It("Should give each batch a higher Seq than the batches before it", func(
		ctx SpecContext,
	) {
		key := uuid.New()
		for range 3 {
			attach(ctx, state.Dispatcher(), key)
		}
		Expect(rec.Snapshot()).To(HaveExactElements(
			HaveField("Seq", uint64(1)),
			HaveField("Seq", uint64(2)),
			HaveField("Seq", uint64(3)),
		))
	})

	It(
		"Should deliver the batches of one transaction in order with rising Seq values",
		func(ctx SpecContext) {
			a, b := uuid.New(), uuid.New()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				state.Dispatcher().
					WithDispatchKey("dk").
					Attach(tx, a, testAction{Type: "a"})
				state.Dispatcher().Attach(tx, b, testAction{Type: "b"})
				return nil
			})).To(Succeed())
			Expect(rec.Snapshot()).To(Equal([]actions.Scoped[uuid.UUID, testAction]{
				{Key: a, DispatchKey: "dk", Seq: 1, Actions: []testAction{{Type: "a"}}},
				{Key: b, Seq: 2, Actions: []testAction{{Type: "b"}}},
			}))
		},
	)

	It(
		"Should number two batches for one key in one commit so a client keeps both",
		func(ctx SpecContext) {
			key := uuid.New()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				state.Dispatcher().
					WithDispatchKey("dk").
					Attach(tx, key, testAction{Type: "edit"})
				state.Dispatcher().Attach(tx, key, testAction{Type: "sweep"})
				return nil
			})).To(Succeed())
			var (
				last uint64
				kept []string
			)
			for _, b := range rec.Snapshot() {
				// A client drops a batch whose Seq does not exceed the last it applied.
				if b.Seq <= last {
					continue
				}
				last = b.Seq
				kept = append(kept, b.Actions[0].Type)
			}
			Expect(kept).To(Equal([]string{"edit", "sweep"}))
		},
	)

	It(
		"Should skip the numbers that an aborted transaction took",
		func(ctx SpecContext) {
			txErr := errors.New("abort")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				state.Dispatcher().Attach(tx, uuid.New())
				return txErr
			})).To(MatchError(txErr))
			attach(ctx, state.Dispatcher(), uuid.New())
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField("Seq", uint64(2))))
		},
	)

	It("Should deliver nothing when the transaction aborts", func(ctx SpecContext) {
		txErr := errors.New("abort")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			state.Dispatcher().Attach(tx, uuid.New(), testAction{Type: "a"})
			return txErr
		})).To(MatchError(txErr))
		Expect(rec.Snapshot()).To(BeEmpty())
	})

	It("Should keep the batches of two States apart", func(ctx SpecContext) {
		other := actions.NewState[uuid.UUID, testAction](db)
		otherRec := &Recorder[uuid.UUID, testAction]{}
		DeferCleanup(other.OnChange(otherRec.Record))
		attach(ctx, other.Dispatcher(), uuid.New(), testAction{Type: "other"})
		Expect(rec.Snapshot()).To(BeEmpty())
		Expect(otherRec.Snapshot()).To(HaveLen(1))
	})

	It("Should stop calling a handler after Disconnect", func(ctx SpecContext) {
		var hits int
		disconnect := state.OnChange(
			func(context.Context, actions.Scoped[uuid.UUID, testAction]) { hits++ },
		)
		attach(ctx, state.Dispatcher(), uuid.New())
		disconnect()
		attach(ctx, state.Dispatcher(), uuid.New())
		Expect(hits).To(Equal(1))
	})

	Describe("Of", func() {
		It("Should return the batches of a commit", func(ctx SpecContext) {
			var commits []gorp.Commit
			DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
				commits = append(commits, c)
			}))
			key := uuid.New()
			attach(ctx, state.Dispatcher(), key, testAction{Type: "a"})
			Expect(commits).To(HaveLen(1))
			Expect(
				state.Of(commits[0]),
			).To(Equal([]actions.Scoped[uuid.UUID, testAction]{
				{Key: key, Seq: 1, Actions: []testAction{{Type: "a"}}},
			}))
		})
	})
})

var _ = Describe("Dispatcher", func() {
	var (
		db    *gorp.DB
		state *actions.State[uuid.UUID, testAction]
		rec   *Recorder[uuid.UUID, testAction]
	)
	BeforeEach(func() {
		db = DeferClose(gorptestutil.OpenGorpMsgpackDB())
		state = actions.NewState[uuid.UUID, testAction](db)
		rec = &Recorder[uuid.UUID, testAction]{}
		DeferCleanup(state.OnChange(rec.Record))
	})

	Describe("WithDispatchKey", func() {
		It("Should stamp the dispatch key onto each batch", func(ctx SpecContext) {
			d := state.Dispatcher().WithDispatchKey("dk")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				d.Attach(tx, uuid.New())
				d.Attach(tx, uuid.New())
				return nil
			})).To(Succeed())
			Expect(rec.Snapshot()).To(HaveExactElements(
				HaveField("DispatchKey", "dk"),
				HaveField("DispatchKey", "dk"),
			))
		})

		It("Should leave the original Dispatcher unchanged", func(ctx SpecContext) {
			d := state.Dispatcher()
			d.WithDispatchKey("dk")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				d.Attach(tx, uuid.New())
				return nil
			})).To(Succeed())
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField("DispatchKey", "")))
		})
	})

	Describe("Attach", func() {
		It("Should panic when the transaction is a DB", func() {
			Expect(func() { state.Dispatcher().Attach(db, uuid.New()) }).To(Panic())
		})
	})
})
