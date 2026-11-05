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
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apiview "github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/query"
)

func createView(ctx SpecContext) view.View {
	GinkgoHelper()
	v := view.View{Variant: view.RangeView{Name: "Before"}}
	Expect(svc.View.NewWriter(nil).Create(ctx, &v)).To(Succeed())
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
		grantUpdateOn(ctx, u.OntologyID(), v.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apiview.RenameRequest{Key: v.GorpKey(), Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveView(ctx, v.GorpKey())
		Expect(renamed.Base().Name).To(Equal("After"))
		Expect(renamed.Type()).To(Equal(view.TypeRange))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		grantUpdateOn(ctx, u.OntologyID(), v.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apiview.RenameRequest{Key: v.GorpKey(), Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing view", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grantUpdateOn(ctx, u.OntologyID(), view.OntologyID(key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apiview.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		v := createView(ctx)
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apiview.RenameRequest{Key: v.GorpKey(), Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveView(ctx, v.GorpKey()).Base().Name).To(Equal("Before"))
	})
})
