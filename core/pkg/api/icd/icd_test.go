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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiicd "github.com/synnaxlabs/synnax/pkg/api/icd"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var icdType = resource.ID{Type: resource.TypeICD}

var _ = Describe("ICD", func() {
	Describe("Create", func() {
		It("Should create an ICD when create is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionCreate, icdType)
			res := MustSucceed(create(
				AuthedCtx(ctx, u),
				apiicd.CreateRequest{ICDs: []icd.ICD{{Name: "Bus"}}},
			))
			Expect(res.ICDs).To(HaveLen(1))
			Expect(res.ICDs[0].Key).ToNot(BeZero())
		})

		It("Should stamp the dispatch key onto the create broadcast", func(
			ctx SpecContext,
		) {
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionCreate, icdType)
			rec := &Recorder[icd.Key, icd.Action]{}
			DeferCleanup(svc.ICD.OnAction(rec.Record))
			MustSucceed(create(
				AuthedCtx(ctx, u),
				apiicd.CreateRequest{
					ICDs:        []icd.ICD{{Name: "Bus"}},
					DispatchKey: "dk-create",
				},
			))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		})

		It("Should reject a create that is not granted", func(ctx SpecContext) {
			Expect(create(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.CreateRequest{ICDs: []icd.ICD{{Name: "Bus"}}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve ICDs by key when retrieve is granted", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Engine")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, ic.ResourceID())
			res := MustSucceed(retrieve(
				AuthedCtx(ctx, u),
				apiicd.RetrieveRequest{
					Where: &icd.Where{Key: &filter.Set[icd.Key]{In: []icd.Key{ic.Key}}},
				},
			))
			Expect(res.ICDs).To(HaveExactElements(HaveField("Name", "Engine")))
		})

		It("Should leave out the ICDs that the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			shown := createICD(ctx, "Shown")
			createICD(ctx, "Unshown")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, shown.ResourceID())
			res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiicd.RetrieveRequest{}))
			Expect(res.ICDs).To(HaveExactElements(HaveField("Key", shown.Key)))
		})

		It("Should reject a retrieve that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Hidden")
			Expect(retrieve(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.RetrieveRequest{
					Where: &icd.Where{Key: &filter.Set[icd.Key]{In: []icd.Key{ic.Key}}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Check", func() {
		It("Should return every problem in an ICD", func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, icdType)
			res := MustSucceed(check(
				AuthedCtx(ctx, u),
				apiicd.CheckRequest{ICD: icd.ICD{}},
			))
			Expect(res.Problems).To(Equal(validate.Problems{
				{Path: "name", Message: "name is required"},
			}))
		})

		It("Should return no problems for a valid ICD", func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, icdType)
			res := MustSucceed(check(
				AuthedCtx(ctx, u),
				apiicd.CheckRequest{ICD: icd.ICD{Name: "Bus"}},
			))
			Expect(res.Problems).To(BeEmpty())
		})

		It("Should check an ICD that the subject can retrieve", func(ctx SpecContext) {
			ic := createICD(ctx, "Bus")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, ic.ResourceID())
			ic.Name = ""
			res := MustSucceed(check(AuthedCtx(ctx, u), apiicd.CheckRequest{ICD: ic}))
			Expect(res.Problems).To(Equal(validate.Problems{
				{Path: "name", Message: "name is required"},
			}))
		})

		It("Should reject a check of an ICD it cannot retrieve", func(ctx SpecContext) {
			granted := createICD(ctx, "Granted")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, granted.ResourceID())
			Expect(check(
				AuthedCtx(ctx, u),
				apiicd.CheckRequest{ICD: createICD(ctx, "Hidden")},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject a check that is not granted", func(ctx SpecContext) {
			Expect(check(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.CheckRequest{ICD: icd.ICD{Name: "Bus"}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Dispatch", func() {
		rename := []icd.Action{icd.NewSetAction(icd.SetPayload{
			Path:  []string{"name"},
			Value: "New",
		})}

		It("Should apply the actions when update is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Old")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, ic.ResourceID())
			MustSucceed(dispatch(
				AuthedCtx(ctx, u),
				apiicd.DispatchRequest{Key: ic.Key, DispatchKey: "dk", Actions: rename},
			))
			Expect(retrieveICD(ctx, ic.Key).Name).To(Equal("New"))
		})

		It("Should reject a dispatch that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Fixed")
			Expect(dispatch(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.DispatchRequest{Key: ic.Key, DispatchKey: "dk", Actions: rename},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveICD(ctx, ic.Key).Name).To(Equal("Fixed"))
		})

		It("Should return the dispatch error", func(ctx SpecContext) {
			u := createUser(ctx)
			key := uuid.New()
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, icd.ResourceID(key))
			Expect(dispatch(
				AuthedCtx(ctx, u),
				apiicd.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
			)).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Import", func() {
		It("Should import a DBC file when update is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Vehicle")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, ic.ResourceID())
			data := MustSucceed(os.ReadFile("../../service/icd/testdata/basic.dbc"))
			res := MustSucceed(importFile(
				AuthedCtx(ctx, u),
				apiicd.ImportRequest{Key: ic.Key, Format: icd.FormatDBC, Data: data},
			))
			Expect(res.ICD.Key).To(Equal(ic.Key))
			Expect(res.ICD.Messages).ToNot(BeEmpty())
		})

		It("Should stamp the dispatch key onto the create broadcast", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Echo")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionUpdate, ic.ResourceID())
			rec := &Recorder[icd.Key, icd.Action]{}
			DeferCleanup(svc.ICD.OnAction(rec.Record))
			data := MustSucceed(os.ReadFile("../../service/icd/testdata/basic.dbc"))
			res := MustSucceed(importFile(
				AuthedCtx(ctx, u),
				apiicd.ImportRequest{
					Key:         ic.Key,
					Format:      icd.FormatDBC,
					Data:        data,
					DispatchKey: "dk-import",
				},
			))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-import"))
			Expect(frames[0].Actions).To(Equal([]icd.Action{
				icd.NewCreateAction(icd.CreatePayload{ICD: res.ICD}),
			}))
		})

		It("Should reject an import that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Locked")
			Expect(importFile(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.ImportRequest{Key: ic.Key, Format: icd.FormatDBC},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It("Should delete an ICD when delete is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Doomed")
			u := createUser(ctx)
			grantOn(ctx, u.ResourceID(), access.ActionDelete, ic.ResourceID())
			MustSucceed(del(
				AuthedCtx(ctx, u),
				apiicd.DeleteRequest{Keys: []icd.Key{ic.Key}},
			))
			Expect(svc.ICD.NewRetrieve().
				Where(icd.MatchKeys(ic.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject a delete that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Kept")
			Expect(del(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.DeleteRequest{Keys: []icd.Key{ic.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
})
