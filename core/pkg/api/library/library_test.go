// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilibrary "github.com/synnaxlabs/synnax/pkg/api/library"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var libraryType = ontology.ID{Type: ontology.ResourceTypeLibrary}

var _ = Describe("Library", func() {
	Describe("Create", func() {
		It("Should create a library when create is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionCreate, libraryType)
			res := MustSucceed(apiSvc.Create(
				AuthedCtx(ctx, u),
				nil,
				apilibrary.CreateRequest{Libraries: []library.Library{{Name: "Bus"}}},
			))
			Expect(res.Libraries).To(HaveLen(1))
			Expect(res.Libraries[0].Key).ToNot(BeZero())
		})

		It("Should reject a create that is not granted", func(ctx SpecContext) {
			Expect(apiSvc.Create(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apilibrary.CreateRequest{Libraries: []library.Library{{Name: "Bus"}}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve libraries by key when retrieve is granted", func(
			ctx SpecContext,
		) {
			l := createLibrary(ctx, "Engine")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionRetrieve, l.OntologyID())
			res := MustSucceed(apiSvc.Retrieve(
				AuthedCtx(ctx, u),
				apilibrary.RetrieveRequest{Keys: []library.Key{l.Key}},
			))
			Expect(res.Libraries).To(HaveExactElements(
				HaveField("Name", "Engine"),
			))
		})

		It("Should reject a retrieve that is not granted", func(ctx SpecContext) {
			l := createLibrary(ctx, "Hidden")
			Expect(apiSvc.Retrieve(
				AuthedCtx(ctx, createUser(ctx)),
				apilibrary.RetrieveRequest{Keys: []library.Key{l.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Rename", func() {
		It("Should rename a library when update is granted", func(ctx SpecContext) {
			l := createLibrary(ctx, "Old")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionUpdate, l.OntologyID())
			MustSucceed(apiSvc.Rename(
				AuthedCtx(ctx, u),
				nil,
				apilibrary.RenameRequest{Key: l.Key, Name: "New"},
			))
			var res library.Library
			Expect(libSvc.NewRetrieve().
				Where(library.MatchKeys(l.Key)).
				Entry(&res).
				Exec(ctx, nil)).To(Succeed())
			Expect(res.Name).To(Equal("New"))
		})

		It("Should reject a rename that is not granted", func(ctx SpecContext) {
			l := createLibrary(ctx, "Fixed")
			Expect(apiSvc.Rename(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apilibrary.RenameRequest{Key: l.Key, Name: "New"},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Import", func() {
		It("Should import a DBC file when update is granted", func(ctx SpecContext) {
			l := createLibrary(ctx, "Vehicle")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionUpdate, l.OntologyID())
			data := MustSucceed(
				os.ReadFile("../../service/library/icd/testdata/basic.dbc"),
			)
			res := MustSucceed(apiSvc.Import(
				AuthedCtx(ctx, u),
				nil,
				apilibrary.ImportRequest{Key: l.Key, Format: icd.FormatDBC, Data: data},
			))
			Expect(res.Library.Key).To(Equal(l.Key))
			Expect(res.Library.Entries).ToNot(BeEmpty())
		})

		It("Should reject an import that is not granted", func(ctx SpecContext) {
			l := createLibrary(ctx, "Locked")
			Expect(apiSvc.Import(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apilibrary.ImportRequest{Key: l.Key, Format: icd.FormatDBC},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It("Should delete a library when delete is granted", func(ctx SpecContext) {
			l := createLibrary(ctx, "Doomed")
			u := createUser(ctx)
			grantOn(ctx, u.OntologyID(), access.ActionDelete, l.OntologyID())
			MustSucceed(apiSvc.Delete(
				AuthedCtx(ctx, u),
				nil,
				apilibrary.DeleteRequest{Keys: []library.Key{l.Key}},
			))
			Expect(libSvc.NewRetrieve().
				Where(library.MatchKeys(l.Key)).
				Exists(ctx, nil)).To(BeFalse())
		})

		It("Should reject a delete that is not granted", func(ctx SpecContext) {
			l := createLibrary(ctx, "Kept")
			Expect(apiSvc.Delete(
				AuthedCtx(ctx, createUser(ctx)),
				nil,
				apilibrary.DeleteRequest{Keys: []library.Key{l.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
})
