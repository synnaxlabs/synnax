// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	apiicd "github.com/synnaxlabs/synnax/pkg/api/icd"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var icdType = resource.ID{Type: resource.TypeICD}

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

// keyed returns a where filter that selects the ICD with key.
func keyed(key icd.Key) *icd.Where {
	return &icd.Where{Key: &filter.Set[icd.Key]{Eq: &key}}
}

var _ = Describe("ICD", func() {
	Describe("Create", func() {
		It("Should create an ICD when create is granted", func(ctx SpecContext) {
			u := newUser(ctx)
			grant(ctx, u, access.ActionCreate, icdType)
			res := MustSucceed(create(
				AuthedCtx(ctx, u),
				apiicd.CreateRequest{ICDs: []icd.ICD{{Name: "Bus"}}},
			))
			Expect(res.ICDs).To(HaveLen(1))
			Expect(res.ICDs[0].Key).ToNot(BeZero())
			Expect(res.ICDs[0].Hash).ToNot(BeEmpty())
		})

		It("Should reject a create that is not granted", func(ctx SpecContext) {
			u := newUser(ctx)
			res, err := create(
				AuthedCtx(ctx, u),
				apiicd.CreateRequest{ICDs: []icd.ICD{{Name: "Bus"}}},
			)
			Expect(err).To(MatchError(access.ErrDenied))
			Expect(res).To(BeZero())
		})

		It("Should reject a save of an ICD that changed since it was read", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Shared")
			u := newUser(ctx)
			grant(ctx, u, access.ActionUpdate, ic.ResourceID())
			stale := ic
			stale.Hash = "0000000000000000"
			Expect(create(
				AuthedCtx(ctx, u),
				apiicd.CreateRequest{ICDs: []icd.ICD{stale}},
			)).Error().To(MatchError(library.ErrStale))
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve ICDs by key when retrieve is granted", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Engine")
			u := newUser(ctx)
			grant(ctx, u, access.ActionRetrieve, ic.ResourceID())
			res := MustSucceed(retrieve(
				AuthedCtx(ctx, u),
				apiicd.RetrieveRequest{Where: keyed(ic.Key)},
			))
			Expect(res.ICDs).To(HaveExactElements(HaveField("Name", "Engine")))
		})

		It("Should reject a retrieve by key that is not granted", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Hidden")
			u := newUser(ctx)
			Expect(retrieve(
				AuthedCtx(ctx, u),
				apiicd.RetrieveRequest{Where: keyed(ic.Key)},
			)).Error().To(denied(u, access.ActionRetrieve, ic.ResourceID()))
		})

		It("Should leave out the ICDs the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			granted := createICD(ctx, "Granted")
			createICD(ctx, "Withheld")
			u := newUser(ctx)
			grant(ctx, u, access.ActionRetrieve, granted.ResourceID())
			res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiicd.RetrieveRequest{}))
			Expect(res.ICDs).To(HaveExactElements(HaveField("Key", granted.Key)))
		})
	})

	Describe("Check", func() {
		It("Should return every problem in an unsaved ICD", func(ctx SpecContext) {
			res := MustSucceed(check(
				AuthedCtx(ctx, newUser(ctx)),
				apiicd.CheckRequest{ICD: icd.ICD{}},
			))
			Expect(res.Problems).To(Equal(library.Problems{
				{Path: "name", Message: "required"},
			}))
		})

		It("Should return no problems for a valid stored ICD", func(ctx SpecContext) {
			ic := createICD(ctx, "Bus")
			u := newUser(ctx)
			grant(ctx, u, access.ActionRetrieve, ic.ResourceID())
			res := MustSucceed(check(
				AuthedCtx(ctx, u),
				apiicd.CheckRequest{ICD: ic},
			))
			Expect(res.Problems).To(BeEmpty())
		})

		It("Should reject a check of a stored ICD that is not granted", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Bus")
			u := newUser(ctx)
			Expect(check(
				AuthedCtx(ctx, u),
				apiicd.CheckRequest{ICD: ic},
			)).Error().To(denied(u, access.ActionRetrieve, ic.ResourceID()))
		})
	})

	Describe("Rename", func() {
		It("Should rename an ICD when update is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Old")
			u := newUser(ctx)
			grant(ctx, u, access.ActionUpdate, ic.ResourceID())
			MustSucceed(rename(
				AuthedCtx(ctx, u),
				apiicd.RenameRequest{Key: ic.Key, Name: "New"},
			))
			Expect(MustSucceed(retrieveICD(ctx, ic.Key)).Name).To(Equal("New"))
		})

		It("Should reject a rename that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Fixed")
			u := newUser(ctx)
			Expect(rename(
				AuthedCtx(ctx, u),
				apiicd.RenameRequest{Key: ic.Key, Name: "New"},
			)).Error().To(denied(u, access.ActionUpdate, ic.ResourceID()))
			Expect(MustSucceed(retrieveICD(ctx, ic.Key)).Name).To(Equal("Fixed"))
		})
	})

	Describe("Import", func() {
		It("Should import a DBC file when update is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Vehicle")
			u := newUser(ctx)
			grant(ctx, u, access.ActionUpdate, ic.ResourceID())
			data := MustSucceed(os.ReadFile("../../service/icd/testdata/basic.dbc"))
			res := MustSucceed(imp(
				AuthedCtx(ctx, u),
				apiicd.ImportRequest{Key: ic.Key, Format: icd.FormatDBC, Data: data},
			))
			Expect(res.ICD.Key).To(Equal(ic.Key))
			Expect(res.ICD.Messages).ToNot(BeEmpty())
		})

		It("Should reject an import that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Locked")
			u := newUser(ctx)
			data := MustSucceed(os.ReadFile("../../service/icd/testdata/basic.dbc"))
			Expect(imp(
				AuthedCtx(ctx, u),
				apiicd.ImportRequest{Key: ic.Key, Format: icd.FormatDBC, Data: data},
			)).Error().To(denied(u, access.ActionUpdate, ic.ResourceID()))
			Expect(MustSucceed(retrieveICD(ctx, ic.Key)).Messages).To(BeEmpty())
		})
	})

	Describe("Delete", func() {
		It("Should delete an ICD when delete is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Doomed")
			u := newUser(ctx)
			grant(ctx, u, access.ActionDelete, ic.ResourceID())
			MustSucceed(del(
				AuthedCtx(ctx, u),
				apiicd.DeleteRequest{Keys: []icd.Key{ic.Key}},
			))
			Expect(retrieveICD(ctx, ic.Key)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should reject a delete that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Kept")
			u := newUser(ctx)
			Expect(del(
				AuthedCtx(ctx, u),
				apiicd.DeleteRequest{Keys: []icd.Key{ic.Key}},
			)).Error().To(denied(u, access.ActionDelete, ic.ResourceID()))
			Expect(MustSucceed(retrieveICD(ctx, ic.Key)).Key).To(Equal(ic.Key))
		})
	})
})
