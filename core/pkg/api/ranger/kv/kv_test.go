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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apikv "github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createRange(ctx SpecContext) ranger.Range {
	GinkgoHelper()
	r := ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Ranger.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	return r
}

func listPairs(ctx SpecContext, r ranger.Range) []kv.Pair {
	GinkgoHelper()
	return MustSucceed(svc.KV.NewReader(db).List(ctx, r.Key))
}

var _ = Describe("Set", func() {
	var (
		u user.User
		r ranger.Range
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
	})
	setAs := func(ctx SpecContext, pairs ...kv.Pair) (struct{}, error) {
		return set(AuthedCtx(ctx, u), apikv.SetRequest{Pairs: pairs})
	}
	It("Should refuse the set without update on the range", func(ctx SpecContext) {
		Expect(setAs(ctx, kv.Pair{Range: r.Key, Key: "k", Value: "v"})).Error().
			To(denied(u, access.ActionUpdate, r.ResourceID()))
		Expect(listPairs(ctx, r)).To(BeEmpty())
	})
	It("Should set the pair with update on the range", func(ctx SpecContext) {
		grant(ctx, u, access.ActionUpdate, r.ResourceID())
		pair := kv.Pair{Range: r.Key, Key: "k", Value: "v"}
		MustSucceed(setAs(ctx, pair))
		Expect(listPairs(ctx, r)).To(Equal([]kv.Pair{pair}))
	})
	It(
		"Should refuse a pair on a range the subject cannot update",
		func(ctx SpecContext) {
			other := createRange(ctx)
			grant(ctx, u, access.ActionUpdate, r.ResourceID())
			Expect(setAs(ctx, kv.Pair{Range: other.Key, Key: "k", Value: "v"})).
				Error().To(denied(u, access.ActionUpdate, other.ResourceID()))
			Expect(listPairs(ctx, other)).To(BeEmpty())
		},
	)
})

var _ = Describe("Dispatch", func() {
	var (
		u    user.User
		r    ranger.Range
		pair kv.Pair
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
		pair = kv.Pair{Range: r.Key, Key: "k", Value: "stored"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.KV.NewWriter(tx).SetMany(ctx, []kv.Pair{pair})
		})).To(Succeed())
	})
	dispatchAs := func(ctx SpecContext, value string) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apikv.DispatchRequest{
			Key: pair.GorpKey(),
			Actions: []kv.Action{kv.NewSetAction(
				kv.SetPayload{Path: []string{"value"}, Value: value},
			)},
		})
	}
	It("Should change the value with update on the range", func(ctx SpecContext) {
		grant(ctx, u, access.ActionUpdate, r.ResourceID())
		MustSucceed(dispatchAs(ctx, "changed"))
		Expect(listPairs(ctx, r)).
			To(Equal([]kv.Pair{{Range: r.Key, Key: "k", Value: "changed"}}))
	})
	It("Should refuse a change without update on the range", func(ctx SpecContext) {
		Expect(dispatchAs(ctx, "changed")).Error().
			To(denied(u, access.ActionUpdate, r.ResourceID()))
		Expect(listPairs(ctx, r)).To(Equal([]kv.Pair{pair}))
	})
})

var _ = Describe("Delete", func() {
	var (
		u user.User
		r ranger.Range
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.KV.NewWriter(tx).SetMany(ctx, []kv.Pair{
				{Range: r.Key, Key: "k", Value: "v"},
			})
		})).To(Succeed())
	})
	deleteAs := func(ctx SpecContext, keys ...string) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apikv.DeleteRequest{Range: r.Key, Keys: keys})
	}
	It("Should refuse the delete without update on the range", func(ctx SpecContext) {
		Expect(deleteAs(ctx, "k")).Error().
			To(denied(u, access.ActionUpdate, r.ResourceID()))
		Expect(listPairs(ctx, r)).To(HaveLen(1))
	})
	It("Should delete the pair with update on the range", func(ctx SpecContext) {
		grant(ctx, u, access.ActionUpdate, r.ResourceID())
		MustSucceed(deleteAs(ctx, "k"))
		Expect(listPairs(ctx, r)).To(BeEmpty())
	})
	It("Should need no access to delete a key that matches no pair", func(
		ctx SpecContext,
	) {
		MustSucceed(deleteAs(ctx, "missing"))
		Expect(listPairs(ctx, r)).To(HaveLen(1))
	})
})

var _ = Describe("Get", func() {
	var (
		u    user.User
		r    ranger.Range
		pair kv.Pair
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
		pair = kv.Pair{Range: r.Key, Key: "k", Value: "v"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.KV.NewWriter(tx).SetMany(ctx, []kv.Pair{pair})
		})).To(Succeed())
	})
	It("Should refuse the get without retrieve on the range", func(ctx SpecContext) {
		Expect(get(AuthedCtx(ctx, u), apikv.GetRequest{Range: r.Key})).Error().
			To(denied(u, access.ActionRetrieve, r.ResourceID()))
	})
	DescribeTable("Should return the pairs with retrieve on the range", func(
		ctx SpecContext,
		keys []string,
	) {
		grant(ctx, u, access.ActionRetrieve, r.ResourceID())
		Expect(get(AuthedCtx(ctx, u), apikv.GetRequest{Range: r.Key, Keys: keys})).
			To(Equal(apikv.GetResponse{Pairs: []kv.Pair{pair}}))
	},
		Entry("for every key", nil),
		Entry("for the given keys", []string{"k"}),
	)
})
