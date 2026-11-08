// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgorp "github.com/synnaxlabs/freighter/gorp"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apiview "github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

func createView(ctx SpecContext) view.View {
	GinkgoHelper()
	v := view.View{Variant: view.RangeView{Name: "Before"}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.View.NewWriter(tx).Create(ctx, &v)
	})).To(Succeed())
	return v
}

func retrieveView(ctx SpecContext, key view.Key) view.View {
	GinkgoHelper()
	var v view.View
	Expect(svc.View.NewRetrieve().
		Where(view.MatchKeys(key)).
		Entry(&v).
		Exec(ctx, nil)).To(Succeed())
	return v
}

var _ = Describe("Rename", func() {
	It("Should rename the view when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, v.ResourceID())
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Rename)(
			AuthedCtx(ctx, u),
			apiview.RenameRequest{Key: v.GorpKey(), Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveView(ctx, v.GorpKey())
		Expect(renamed.Base().Name).To(Equal("After"))
		Expect(renamed.Type()).To(Equal(view.TypeRange))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, v.ResourceID())
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Rename)(
			AuthedCtx(ctx, u),
			apiview.RenameRequest{Key: v.GorpKey(), Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing view", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, view.ResourceID(key))
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Rename)(
			AuthedCtx(ctx, u),
			apiview.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Rename)(
			AuthedCtx(ctx, u),
			apiview.RenameRequest{Key: v.GorpKey(), Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("Before"))
	})
})

var _ = Describe("Retrieve", func() {
	keysOf := func(res apiview.RetrieveResponse) []view.Key {
		keys := make([]view.Key, len(res.Views))
		for i, v := range res.Views {
			keys[i] = v.GorpKey()
		}
		return keys
	}
	byKeys := func(keys ...view.Key) apiview.RetrieveRequest {
		return apiview.RetrieveRequest{
			Where: &view.Where{Key: &filter.Set[view.Key]{In: keys}},
		}
	}

	It("Should return the named views that the subject can retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		v := createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, v.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), byKeys(v.GorpKey())))
		Expect(keysOf(res)).To(Equal([]view.Key{v.GorpKey()}))
	})

	It("Should deny a named view that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		allowed, denied := createView(ctx), createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, allowed.ResourceID())
		Expect(retrieve(
			AuthedCtx(ctx, u),
			byKeys(allowed.GorpKey(), denied.GorpKey()),
		)).Error().To(MatchError(access.Denied(
			u.ResourceID(),
			access.ActionRetrieve,
			denied.ResourceID(),
		).Error()))
	})

	It("Should leave out matched views that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		allowed, denied := createView(ctx), createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, allowed.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiview.RetrieveRequest{
			Where: &view.Where{Or: []view.Where{
				{Key: &filter.Set[view.Key]{Eq: new(allowed.GorpKey())}},
				{Key: &filter.Set[view.Key]{Eq: new(denied.GorpKey())}},
			}},
		}))
		Expect(keysOf(res)).To(Equal([]view.Key{allowed.GorpKey()}))
	})
})
