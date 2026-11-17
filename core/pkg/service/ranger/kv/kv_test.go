// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("KV", Ordered, func() {
	var (
		db        *gorp.DB
		rangerSvc *ranger.Service
		kvSvc     *kv.Service
		tx        gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
		}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		rangerSvc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:    db,
			Group: g,
		}))
		kvSvc = MustOpen(kv.OpenService(ctx, kv.ServiceConfig{
			DB: db,
		}))
	})
	BeforeEach(func() {
		tx = DeferClose(db.OpenTx())
	})

	It("Should be able to store key-value pairs in a range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(
			kvSvc.NewWriter(tx).
				SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
		).To(Succeed())
	})

	It("Should retrieve key-value pairs through a snapshot", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(
			kvSvc.NewWriter(tx).
				SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
		).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		var pairs []kv.Pair
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			pairs, err = kvSvc.NewReader(snap).GetMany(ctx, r.Key, []string{"key"})
			return err
		})).To(Succeed())
		Expect(pairs).To(Equal([]kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}))
	})

	It("Should list key-value pairs through a snapshot", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(
			kvSvc.NewWriter(tx).
				SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
		).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		var pairs []kv.Pair
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			pairs, err = kvSvc.NewReader(snap).List(ctx, r.Key)
			return err
		})).To(Succeed())
		Expect(pairs).To(Equal([]kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}))
	})

	It(
		"Should be able to retrieve key-value pairs from a range",
		func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(
				kvSvc.NewWriter(tx).
					SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
			).To(Succeed())
			Expect(kvSvc.NewReader(tx).GetMany(ctx, r.Key, []string{"key"})).
				To(Equal([]kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}))
		},
	)

	It("Should be able to delete key-value pairs from a range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(
			kvSvc.NewWriter(tx).
				SetMany(ctx, []kv.Pair{{Range: r.Key, Key: "key", Value: "value"}}),
		).To(Succeed())
		Expect(kvSvc.NewWriter(tx).Delete(ctx, r.Key, "key")).To(Succeed())
		Expect(kvSvc.NewReader(tx).List(ctx, r.Key)).To(BeEmpty())
	})

	It("Should set many key-value pairs on the range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).SetMany(ctx, []kv.Pair{
			{Range: r.Key, Key: "key1", Value: "value1"},
			{Range: r.Key, Key: "key2", Value: "value2"},
		})).To(Succeed())
		Expect(
			kvSvc.NewReader(tx).GetMany(ctx, r.Key, []string{"key1", "key2"}),
		).To(ConsistOf(
			kv.Pair{Range: r.Key, Key: "key1", Value: "value1"},
			kv.Pair{Range: r.Key, Key: "key2", Value: "value2"},
		))
	})

	It("Should reject a pair on a range that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		err := kvSvc.NewWriter(tx).SetMany(ctx, []kv.Pair{
			{Range: missing, Key: "key", Value: "value"},
		})
		Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
		Expect(err).To(MatchError(
			"range: range " + missing.String() + " does not exist: validation error",
		))
	})

	It("Should delete the pairs of a deleted range", func(ctx SpecContext) {
		r := &ranger.Range{Name: "Range", TimeRange: telem.NewRangeSeconds(5, 10)}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).SetMany(ctx, []kv.Pair{
			{Range: r.Key, Key: "key", Value: "value"},
		})).To(Succeed())
		Expect(rangerSvc.NewWriter(tx).Delete(ctx, r.Key)).To(Succeed())
		Expect(kvSvc.NewReader(tx).List(ctx, r.Key)).To(BeEmpty())
	})

	It("Should be able to list all key-value pairs in a range", func(ctx SpecContext) {
		r := &ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
		kvW := kvSvc.NewWriter(tx)
		Expect(kvW.SetMany(ctx, []kv.Pair{
			{Range: r.Key, Key: "key1", Value: "value1"},
			{Range: r.Key, Key: "key2", Value: "value2"},
		})).To(Succeed())
		pairs := MustSucceed(kvSvc.NewReader(tx).List(ctx, r.Key))
		Expect(pairs).To(Equal([]kv.Pair{
			{Range: r.Key, Key: "key1", Value: "value1"},
			{Range: r.Key, Key: "key2", Value: "value2"},
		}))
		Expect(kvW.Delete(ctx, r.Key, "key1")).To(Succeed())
		pairs = MustSucceed(kvSvc.NewReader(tx).List(ctx, r.Key))
		Expect(pairs).To(Equal([]kv.Pair{
			{Range: r.Key, Key: "key2", Value: "value2"},
		}))
	})
	It("Should list only the pairs of the range", func(ctx SpecContext) {
		tr := telem.TimeRange{
			Start: telem.TimeStamp(5 * telem.Second),
			End:   telem.TimeStamp(10 * telem.Second),
		}
		first := &ranger.Range{Name: "First", TimeRange: tr}
		second := &ranger.Range{Name: "Second", TimeRange: tr}
		w := rangerSvc.NewWriter(tx)
		Expect(w.Create(ctx, first)).To(Succeed())
		Expect(w.Create(ctx, second)).To(Succeed())
		Expect(kvSvc.NewWriter(tx).SetMany(ctx, []kv.Pair{
			{Range: first.Key, Key: "key", Value: "first"},
			{Range: second.Key, Key: "key", Value: "second"},
		})).To(Succeed())
		Expect(kvSvc.NewReader(tx).List(ctx, first.Key)).To(Equal([]kv.Pair{
			{Range: first.Key, Key: "key", Value: "first"},
		}))
	})
})

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[string, kv.Action]
	var (
		db    *gorp.DB
		kvSvc *kv.Service
		r     ranger.Range
		pair  kv.Pair
	)
	write := func(ctx context.Context, fn func(kv.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(kvSvc.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		rangerSvc := MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:    db,
			Group: g,
		}))
		kvSvc = MustOpen(kv.OpenService(ctx, kv.ServiceConfig{DB: db}))
		r = ranger.Range{Name: "Range", TimeRange: telem.NewRangeSeconds(5, 10)}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rangerSvc.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		pair = kv.Pair{Range: r.Key, Key: "key", Value: "before"}
		Expect(write(ctx, func(w kv.Writer) error {
			return w.SetMany(ctx, []kv.Pair{pair})
		})).To(Succeed())
	})
	set := func(path string, value any) kv.Action {
		return kv.NewSetAction(kv.SetPayload{Path: []string{path}, Value: value})
	}
	apply := func(ctx context.Context, key string, acts ...kv.Action) error {
		return write(ctx, func(w kv.Writer) error {
			return w.Apply(ctx, key, acts...)
		})
	}
	retrieve := func(ctx context.Context) []kv.Pair {
		GinkgoHelper()
		return MustSucceed(kvSvc.NewReader(db).GetMany(ctx, r.Key, []string{"key"}))
	}

	It("Should store the edited value", func(ctx SpecContext) {
		Expect(apply(ctx, pair.GorpKey(), set("value", "after"))).To(Succeed())
		Expect(retrieve(ctx)).
			To(Equal([]kv.Pair{{Range: r.Key, Key: "key", Value: "after"}}))
	})

	It("Should refuse a path outside the editable fields", func(ctx SpecContext) {
		err := apply(ctx, pair.GorpKey(), set("key", "other"))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).
			To(MatchError(ContainSubstring(`unknown object member name "key"`)))
		Expect(retrieve(ctx)).To(Equal([]kv.Pair{pair}))
	})

	It("Should refuse a pair that does not exist", func(ctx SpecContext) {
		other := kv.Pair{Range: r.Key, Key: "other"}
		Expect(apply(ctx, other.GorpKey(), set("value", "after"))).
			To(MatchError(query.ErrNotFound))
	})

	It("Should broadcast a set and an applied batch", func(ctx SpecContext) {
		rec := &Recorder[string, kv.Action]{}
		DeferCleanup(kvSvc.Actions().OnChange(rec.Record))
		created := []kv.Pair{
			{Range: r.Key, Key: "a", Value: "1"},
			{Range: r.Key, Key: "b", Value: "2"},
		}
		Expect(write(ctx, func(w kv.Writer) error {
			return w.SetMany(ctx, created)
		})).To(Succeed())
		Expect(apply(ctx, pair.GorpKey(), set("value", "after"))).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{
				Key:         created[0].GorpKey(),
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions: []kv.Action{
					kv.NewCreateAction(kv.CreatePayload{Pair: created[0]}),
				},
			},
			{
				Key:         created[1].GorpKey(),
				DispatchKey: "dk",
				Seq:         frames[1].Seq,
				Actions: []kv.Action{
					kv.NewCreateAction(kv.CreatePayload{Pair: created[1]}),
				},
			},
			{
				Key:         pair.GorpKey(),
				DispatchKey: "dk",
				Seq:         frames[2].Seq,
				Actions:     []kv.Action{set("value", "after")},
			},
		}))
	})
})
