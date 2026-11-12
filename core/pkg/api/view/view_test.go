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
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apiview "github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

func createView(ctx SpecContext) view.View {
	GinkgoHelper()
	v := view.View{Variant: view.RangeView{Name: "Before"}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.View.NewWriter(tx).Create(ctx, &v)
	})).To(Succeed())
	return v
}

// exists returns whether a view has key.
func exists(ctx SpecContext, key view.Key) bool {
	GinkgoHelper()
	return MustSucceed(svc.View.NewRetrieve().
		Where(view.MatchKeys(key)).
		Exists(ctx, db))
}

func retrieveView(ctx SpecContext, key view.Key) view.View {
	GinkgoHelper()
	var v view.View
	Expect(svc.View.NewRetrieve().
		Where(view.MatchKeys(key)).
		Entry(&v).
		Exec(ctx, db)).To(Succeed())
	return v
}

var _ = Describe("Create", func() {
	newView := func() view.View {
		return view.View{Variant: view.RangeView{Name: "Created"}}.WithKey(uuid.New())
	}
	It("Should create the view with create on views", func(ctx SpecContext) {
		u := createUser(ctx)
		v := newView()
		grantOn(ctx, u.ResourceID(), access.ActionCreate, resource.ID{
			Type: resource.TypeView,
		})
		res := MustSucceed(create(AuthedCtx(ctx, u), apiview.CreateRequest{
			Views: []view.View{v},
		}))
		Expect(res.Views).To(HaveLen(1))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("Created"))
	})
	It("Should reject the request without create on the view", func(ctx SpecContext) {
		u := createUser(ctx)
		v := newView()
		Expect(create(AuthedCtx(ctx, u), apiview.CreateRequest{
			Views: []view.View{v},
		})).Error().To(denied(u, access.ActionCreate, v.ResourceID()))
		Expect(exists(ctx, v.GorpKey())).To(BeFalse())
	})
	It("Should refuse a view whose key exists", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionCreate, resource.ID{
			Type: resource.TypeView,
		})
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, v.ResourceID())
		v.SetBase(view.Base{Key: v.GorpKey(), Name: "After"})
		Expect(create(AuthedCtx(ctx, u), apiview.CreateRequest{
			Views: []view.View{v},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("view %s already exists", v.GorpKey())),
		))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("Before"))
	})
})

var _ = Describe("Delete", func() {
	It("Should delete the view with delete on it", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionDelete, v.ResourceID())
		MustSucceed(del(AuthedCtx(ctx, u), apiview.DeleteRequest{
			Keys: []view.Key{v.GorpKey()},
		}))
		Expect(exists(ctx, v.GorpKey())).To(BeFalse())
	})
	It("Should reject the request without delete on the view", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		Expect(del(AuthedCtx(ctx, u), apiview.DeleteRequest{
			Keys: []view.Key{v.GorpKey()},
		})).Error().To(denied(u, access.ActionDelete, v.ResourceID()))
		Expect(exists(ctx, v.GorpKey())).To(BeTrue())
	})
	It("Should need no access to delete a key that matches no view", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		MustSucceed(del(AuthedCtx(ctx, u), apiview.DeleteRequest{
			Keys: []view.Key{uuid.New()},
		}))
	})
})

var _ = Describe("Dispatch", func() {
	rename := []view.Action{view.NewSetAction(view.SetPayload{
		Path:  []string{"name"},
		Value: "After",
	})}
	It("Should reject the request without update on the view", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apiview.DispatchRequest{Key: v.GorpKey(), Actions: rename},
		)).Error().To(denied(u, access.ActionUpdate, v.ResourceID()))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("Before"))
	})
	It("Should apply the actions with update on the view", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		grantOn(ctx, u.ResourceID(), access.ActionUpdate, v.ResourceID())
		MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apiview.DispatchRequest{Key: v.GorpKey(), Actions: rename},
		))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("After"))
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
