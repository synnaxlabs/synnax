// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	apiontology "github.com/synnaxlabs/synnax/pkg/api/ontology"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Directories", func() {
	var root directory.Key
	uniqueName := func() string {
		return "d_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
	}
	parentsOf := func(ctx SpecContext, id ontology.ID) []ontology.ID {
		GinkgoHelper()
		var parents []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ParentsTraverser).
			ExcludeFieldData(true).
			Entries(&parents).
			Exec(ctx, db)).To(Succeed())
		ids := make([]ontology.ID, 0, len(parents))
		for _, p := range parents {
			ids = append(ids, p.ID)
		}
		return ids
	}
	createDirectory := func(ctx SpecContext) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: uniqueName()}
		Expect(arcSvc.NewWriter(nil).CreateDirectory(ctx, &d, root)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(arcSvc.NewWriter(nil).DeleteDirectory(ctx, d.Key)).To(Succeed())
		})
		return d
	}
	BeforeEach(func(ctx SpecContext) {
		var roots []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(ontology.RootID).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypeArcDirectory).
			ExcludeFieldData(true).
			Entries(&roots).
			Exec(ctx, db)).To(Succeed())
		Expect(roots).To(HaveLen(1))
		root = MustSucceed(directory.KeyFromOntologyID(roots[0].ID))
	})

	Describe("CreateDirectory", func() {
		It("Should create a directory under its parent", func(ctx SpecContext) {
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypeArcDirectory},
			)
			res := MustSucceed(apiSvc.CreateDirectory(
				AuthedCtx(ctx, author),
				db,
				apiarc.CreateDirectoryRequest{
					Parent:    root,
					Directory: directory.Directory{Name: uniqueName()},
				},
			))
			DeferCleanup(func(ctx SpecContext) {
				Expect(arcSvc.NewWriter(nil).DeleteDirectory(ctx, res.Directory.Key)).
					To(Succeed())
			})
			Expect(parentsOf(ctx, res.Directory.OntologyID())).
				To(ConsistOf(directory.OntologyID(root)))
		})
		It("Should deny a caller without create access", func(ctx SpecContext) {
			Expect(apiSvc.CreateDirectory(
				AuthedCtx(ctx, author),
				db,
				apiarc.CreateDirectoryRequest{
					Parent:    root,
					Directory: directory.Directory{Name: uniqueName()},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("RenameDirectory", func() {
		It("Should rename a directory", func(ctx SpecContext) {
			d := createDirectory(ctx)
			grantUpdateOn(ctx, author.OntologyID(), d.OntologyID())
			name := uniqueName()
			MustSucceed(apiSvc.RenameDirectory(
				AuthedCtx(ctx, author),
				db,
				apiarc.RenameDirectoryRequest{Key: d.Key, Name: name},
			))
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(d.OntologyID()).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Name).To(Equal(name))
		})
	})

	Describe("DeleteDirectory", func() {
		It("Should delete a directory", func(ctx SpecContext) {
			d := directory.Directory{Name: uniqueName()}
			Expect(arcSvc.NewWriter(nil).CreateDirectory(ctx, &d, root)).To(Succeed())
			grantOn(ctx, author.OntologyID(), access.ActionDelete, d.OntologyID())
			MustSucceed(apiSvc.DeleteDirectory(
				AuthedCtx(ctx, author),
				db,
				apiarc.DeleteDirectoryRequest{Keys: []directory.Key{d.Key}},
			))
			Expect(otg.NewRetrieve().
				WhereIDs(d.OntologyID()).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Move", func() {
		It("Should move an Arc into a directory", func(ctx SpecContext) {
			d := createDirectory(ctx)
			a := createArc(ctx, uniqueName())
			DeferCleanup(func(ctx SpecContext) {
				Expect(arcSvc.NewWriter(nil).Delete(ctx, a.Key)).To(Succeed())
			})
			grantUpdateOn(ctx, author.OntologyID(), a.OntologyID(), d.OntologyID())
			MustSucceed(apiSvc.Move(
				AuthedCtx(ctx, author),
				db,
				apiarc.MoveRequest{To: d.Key, IDs: []ontology.ID{a.OntologyID()}},
			))
			Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(d.OntologyID()))
		})
		It("Should deny a caller without update access", func(ctx SpecContext) {
			d := createDirectory(ctx)
			a := createArc(ctx, uniqueName())
			DeferCleanup(func(ctx SpecContext) {
				Expect(arcSvc.NewWriter(nil).Delete(ctx, a.Key)).To(Succeed())
			})
			Expect(apiSvc.Move(
				AuthedCtx(ctx, author),
				db,
				apiarc.MoveRequest{To: d.Key, IDs: []ontology.ID{a.OntologyID()}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should refuse the generic ontology move of an Arc", func(ctx SpecContext) {
			d := createDirectory(ctx)
			a := createArc(ctx, uniqueName())
			DeferCleanup(func(ctx SpecContext) {
				Expect(arcSvc.NewWriter(nil).Delete(ctx, a.Key)).To(Succeed())
			})
			rootID := directory.OntologyID(root)
			grantUpdateOn(
				ctx,
				author.OntologyID(),
				a.OntologyID(),
				d.OntologyID(),
				rootID,
			)
			otgAPI := MustSucceed(apiontology.NewService(apiontology.ServiceConfig{
				Ontology: otg,
				RBAC:     rbacSvc,
			}))
			Expect(otgAPI.MoveChildren(
				AuthedCtx(ctx, author),
				db,
				apiontology.MoveChildrenRequest{
					From:     rootID,
					To:       d.OntologyID(),
					Children: []ontology.ID{arc.OntologyID(a.Key)},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
