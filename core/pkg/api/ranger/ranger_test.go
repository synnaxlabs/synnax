// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createRange(ctx SpecContext) ranger.Range {
	GinkgoHelper()
	r := ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
	Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
	return r
}

func retrieveRange(ctx SpecContext, key ranger.Key) ranger.Range {
	GinkgoHelper()
	var r ranger.Range
	Expect(svc.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
	return r
}

var _ = Describe("Dispatch", func() {
	rename := []ranger.Action{ranger.NewSetAction(ranger.SetPayload{
		Path:  []string{"name"},
		Value: "Cold flow",
	})}
	It("Should apply the actions when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
		MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Cold flow"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Hot fire"))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), ranger.ResourceID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
				Type: resource.TypeRange,
			})
			rec := &Recorder[ranger.Key, ranger.Action]{}
			DeferCleanup(svc.Ranger.OnAction(rec.Record))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, u), nil, apiranger.CreateRequest{
					Ranges: []ranger.Range{{
						Name:      "Created",
						TimeRange: telem.NewRangeSeconds(1, 5),
					}},
					DispatchKey: "dk-create",
				}),
			)
			Expect(res.Ranges).To(HaveLen(1))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)
})

var _ = Describe("Retrieve", func() {
	authed := func(ctx SpecContext) context.Context {
		GinkgoHelper()
		u := createUser(ctx)
		grant(
			ctx,
			access.ActionRetrieve,
			u.ResourceID(),
			resource.ID{Type: resource.TypeRange},
			resource.ID{Type: resource.TypeLabel},
		)
		return AuthedCtx(ctx, u)
	}
	keysOf := func(ranges []apiranger.Range) []ranger.Key {
		return lo.Map(
			ranges,
			func(r apiranger.Range, _ int) ranger.Key { return r.Key },
		)
	}

	It("Should retrieve the children of the given parents", func(ctx SpecContext) {
		parent := createRange(ctx)
		child := ranger.Range{
			Name:      "Burn",
			TimeRange: telem.NewRangeSeconds(1, 2),
			Parent:    &parent.Key,
		}
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &child)).To(Succeed())
		createRange(ctx)
		res := MustSucceed(apiSvc.Retrieve(authed(ctx), apiranger.RetrieveRequest{
			Parents: []ranger.Key{parent.Key},
		}))
		Expect(keysOf(res.Ranges)).To(Equal([]ranger.Key{child.Key}))
		Expect(res.Ranges[0].Parent).To(HaveValue(Equal(parent.Key)))
	})

	It("Should retrieve the ranges that hold any of the labels", func(ctx SpecContext) {
		a, b := label.Label{Name: "A"}, label.Label{Name: "B"}
		Expect(svc.Label.NewWriter(nil).Create(ctx, &a)).To(Succeed())
		Expect(svc.Label.NewWriter(nil).Create(ctx, &b)).To(Succeed())
		withA, withB := createRange(ctx), createRange(ctx)
		createRange(ctx)
		Expect(
			svc.Ranger.NewWriter(nil).
				AddLabels(ctx, withA.Key, []label.Key{a.Key}),
		).
			To(Succeed())
		Expect(
			svc.Ranger.NewWriter(nil).
				AddLabels(ctx, withB.Key, []label.Key{b.Key}),
		).
			To(Succeed())
		res := MustSucceed(apiSvc.Retrieve(authed(ctx), apiranger.RetrieveRequest{
			HasLabels: []label.Key{a.Key, b.Key},
		}))
		Expect(keysOf(res.Ranges)).To(ConsistOf(withA.Key, withB.Key))
	})
})
