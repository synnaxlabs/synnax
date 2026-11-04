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
	apiicd "github.com/synnaxlabs/synnax/pkg/api/icd"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var icdType = ontology.ID{Type: ontology.ResourceTypeICD}

var _ = Describe("ICD", func() {
	Describe("Create", func() {
		It("Should create an ICD when create is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionCreate, icdType)
			res := MustSucceed(apiSvc.Create(
				AuthedCtx(ctx, u),
				nil,
				apiicd.CreateRequest{ICDs: []icd.ICD{{Name: "Bus"}}},
			))
			Expect(res.ICDs).To(HaveLen(1))
			Expect(res.ICDs[0].Key).ToNot(BeZero())
			Expect(res.ICDs[0].Hash).ToNot(BeEmpty())
		})

		It("Should reject a create that is not granted", func(ctx SpecContext) {
			Expect(apiSvc.Create(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apiicd.CreateRequest{ICDs: []icd.ICD{{Name: "Bus"}}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject a save of an ICD that changed since it was read", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Shared")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionCreate, icdType)
			stale := ic
			stale.Hash = "0000000000000000"
			Expect(apiSvc.Create(
				AuthedCtx(ctx, u),
				nil,
				apiicd.CreateRequest{ICDs: []icd.ICD{stale}},
			)).Error().To(MatchError(icd.ErrStale))
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve ICDs by key when retrieve is granted", func(
			ctx SpecContext,
		) {
			ic := createICD(ctx, "Engine")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionRetrieve, ic.OntologyID())
			res := MustSucceed(apiSvc.Retrieve(
				AuthedCtx(ctx, u),
				apiicd.RetrieveRequest{Keys: []icd.Key{ic.Key}},
			))
			Expect(res.ICDs).To(HaveExactElements(HaveField("Name", "Engine")))
		})

		It("Should reject a retrieve that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Hidden")
			Expect(apiSvc.Retrieve(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.RetrieveRequest{Keys: []icd.Key{ic.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Check", func() {
		It("Should return every problem in an ICD", func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionRetrieve, icdType)
			res := MustSucceed(apiSvc.Check(
				AuthedCtx(ctx, u),
				apiicd.CheckRequest{ICD: icd.ICD{}},
			))
			Expect(res.Problems).To(Equal(validate.Problems{
				{Path: "name", Message: "required"},
			}))
		})

		It("Should return no problems for a valid ICD", func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionRetrieve, icdType)
			res := MustSucceed(apiSvc.Check(
				AuthedCtx(ctx, u),
				apiicd.CheckRequest{ICD: icd.ICD{Name: "Bus"}},
			))
			Expect(res.Problems).To(BeEmpty())
		})

		It("Should reject a check that is not granted", func(ctx SpecContext) {
			Expect(apiSvc.Check(
				AuthedCtx(ctx, createUser(ctx)),
				apiicd.CheckRequest{ICD: icd.ICD{Name: "Bus"}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Rename", func() {
		It("Should rename an ICD when update is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Old")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionUpdate, ic.OntologyID())
			MustSucceed(apiSvc.Rename(
				AuthedCtx(ctx, u),
				nil,
				apiicd.RenameRequest{Key: ic.Key, Name: "New"},
			))
			var res icd.ICD
			Expect(svc.ICD.NewRetrieve().
				Where(icd.MatchKeys(ic.Key)).
				Entry(&res).
				Exec(ctx, nil)).To(Succeed())
			Expect(res.Name).To(Equal("New"))
		})

		It("Should reject a rename that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Fixed")
			Expect(apiSvc.Rename(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apiicd.RenameRequest{Key: ic.Key, Name: "New"},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Import", func() {
		It("Should import a DBC file when update is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Vehicle")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionUpdate, ic.OntologyID())
			data := MustSucceed(os.ReadFile("../../service/icd/testdata/basic.dbc"))
			res := MustSucceed(apiSvc.Import(
				AuthedCtx(ctx, u),
				nil,
				apiicd.ImportRequest{Key: ic.Key, Format: icd.FormatDBC, Data: data},
			))
			Expect(res.ICD.Key).To(Equal(ic.Key))
			Expect(res.ICD.Messages).ToNot(BeEmpty())
		})

		It("Should reject an import that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Locked")
			Expect(apiSvc.Import(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apiicd.ImportRequest{Key: ic.Key, Format: icd.FormatDBC},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It("Should delete an ICD when delete is granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Doomed")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionDelete, ic.OntologyID())
			MustSucceed(apiSvc.Delete(
				AuthedCtx(ctx, u),
				nil,
				apiicd.DeleteRequest{Keys: []icd.Key{ic.Key}},
			))
			Expect(svc.ICD.NewRetrieve().
				Where(icd.MatchKeys(ic.Key)).
				Exists(ctx, nil)).To(BeFalse())
		})

		It("Should reject a delete that is not granted", func(ctx SpecContext) {
			ic := createICD(ctx, "Kept")
			Expect(apiSvc.Delete(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apiicd.DeleteRequest{Keys: []icd.Key{ic.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
})
