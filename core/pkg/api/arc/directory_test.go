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
		// createArcIn creates an Arc in the directory d, deleted on cleanup.
		createArcIn := func(ctx SpecContext, d directory.Directory) arc.Arc {
			GinkgoHelper()
			a := createArc(ctx, uniqueName())
			DeferCleanup(func(ctx SpecContext) {
				Expect(arcSvc.NewWriter(nil).Delete(ctx, a.Base().Key)).To(Succeed())
			})
			Expect(arcSvc.NewWriter(nil).Move(ctx, root, d.Key, a.OntologyID())).
				To(Succeed())
			return a
		}
		move := func(
			ctx SpecContext,
			from, to directory.Directory,
			ids ...ontology.ID,
		) error {
			_, err := apiSvc.Move(
				AuthedCtx(ctx, author),
				db,
				apiarc.MoveRequest{From: from.Key, To: to.Key, IDs: ids},
			)
			return err
		}
		It(
			"Should move an Arc with update access to it and both directories",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				dst := createDirectory(ctx)
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.OntologyID(),
					a.OntologyID(),
					src.OntologyID(),
					dst.OntologyID(),
				)
				Expect(move(ctx, src, dst, a.OntologyID())).To(Succeed())
				Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(dst.OntologyID()))
			},
		)
		It(
			"Should deny a caller without update access to the source",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				dst := createDirectory(ctx)
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.OntologyID(),
					a.OntologyID(),
					dst.OntologyID(),
				)
				Expect(move(ctx, src, dst, a.OntologyID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(src.OntologyID()))
			},
		)
		It(
			"Should deny a caller without update access to the destination",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				dst := createDirectory(ctx)
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.OntologyID(),
					a.OntologyID(),
					src.OntologyID(),
				)
				Expect(move(ctx, src, dst, a.OntologyID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(src.OntologyID()))
			},
		)
		It(
			"Should deny a caller without update access to the entry",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				dst := createDirectory(ctx)
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.OntologyID(),
					src.OntologyID(),
					dst.OntologyID(),
				)
				Expect(move(ctx, src, dst, a.OntologyID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(src.OntologyID()))
			},
		)
		It(
			"Should deny moving a directory out of a source the caller cannot edit",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				dst := createDirectory(ctx)
				sub := createDirectory(ctx)
				Expect(arcSvc.NewWriter(nil).Move(
					ctx, root, src.Key, sub.OntologyID(),
				)).To(Succeed())
				grantUpdateOn(
					ctx,
					author.OntologyID(),
					sub.OntologyID(),
					dst.OntologyID(),
				)
				Expect(move(ctx, src, dst, sub.OntologyID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, sub.OntologyID())).
					To(ConsistOf(src.OntologyID()))
			},
		)
		It(
			"Should reject a source that does not hold the entry",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				named := createDirectory(ctx)
				dst := createDirectory(ctx)
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.OntologyID(),
					a.OntologyID(),
					named.OntologyID(),
					dst.OntologyID(),
				)
				Expect(move(ctx, named, dst, a.OntologyID())).
					To(MatchError(validate.ErrValidation))
				Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(src.OntologyID()))
			},
		)
		It(
			"Should move within one directory with update access to it",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.OntologyID(),
					a.OntologyID(),
					src.OntologyID(),
				)
				Expect(move(ctx, src, src, a.OntologyID())).To(Succeed())
				Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(src.OntologyID()))
			},
		)
		It("Should refuse the generic ontology move of an Arc", func(ctx SpecContext) {
			d := createDirectory(ctx)
			a := createArc(ctx, uniqueName())
			DeferCleanup(func(ctx SpecContext) {
				Expect(arcSvc.NewWriter(nil).Delete(ctx, a.Base().Key)).To(Succeed())
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
					Children: []ontology.ID{arc.OntologyID(a.Base().Key)},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
