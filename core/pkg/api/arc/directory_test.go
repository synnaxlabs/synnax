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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Directories", func() {
	var root directory.Key
	uniqueName := func() string {
		return "d_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
	}
	parentsOf := func(ctx SpecContext, id resource.ID) []resource.ID {
		GinkgoHelper()
		var parents []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ParentsTraverser).
			ExcludeFieldData(true).
			Entries(&parents).
			Exec(ctx, db)).To(Succeed())
		ids := make([]resource.ID, 0, len(parents))
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
			WhereTypes(resource.TypeArcDirectory).
			ExcludeFieldData(true).
			Entries(&roots).
			Exec(ctx, db)).To(Succeed())
		Expect(roots).To(HaveLen(1))
		root = MustSucceed(directory.KeyFromResourceID(roots[0].ID))
	})

	Describe("CreateDirectory", func() {
		It("Should create a directory under its parent", func(ctx SpecContext) {
			grantOn(
				ctx,
				author.ResourceID(),
				access.ActionCreate,
				resource.ID{Type: resource.TypeArcDirectory},
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
			Expect(parentsOf(ctx, res.Directory.ResourceID())).
				To(ConsistOf(directory.ResourceID(root)))
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
			grantUpdateOn(ctx, author.ResourceID(), d.ResourceID())
			name := uniqueName()
			MustSucceed(apiSvc.RenameDirectory(
				AuthedCtx(ctx, author),
				db,
				apiarc.RenameDirectoryRequest{Key: d.Key, Name: name},
			))
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(d.ResourceID()).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Name).To(Equal(name))
		})
	})

	Describe("DeleteDirectory", func() {
		It("Should delete a directory", func(ctx SpecContext) {
			d := directory.Directory{Name: uniqueName()}
			Expect(arcSvc.NewWriter(nil).CreateDirectory(ctx, &d, root)).To(Succeed())
			grantOn(ctx, author.ResourceID(), access.ActionDelete, d.ResourceID())
			MustSucceed(apiSvc.DeleteDirectory(
				AuthedCtx(ctx, author),
				db,
				apiarc.DeleteDirectoryRequest{Keys: []directory.Key{d.Key}},
			))
			Expect(otg.NewRetrieve().
				WhereIDs(d.ResourceID()).
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
			Expect(arcSvc.NewWriter(nil).Move(ctx, root, d.Key, a.ResourceID())).
				To(Succeed())
			return a
		}
		move := func(
			ctx SpecContext,
			from, to directory.Directory,
			ids ...resource.ID,
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
					author.ResourceID(),
					a.ResourceID(),
					src.ResourceID(),
					dst.ResourceID(),
				)
				Expect(move(ctx, src, dst, a.ResourceID())).To(Succeed())
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(dst.ResourceID()))
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
					author.ResourceID(),
					a.ResourceID(),
					dst.ResourceID(),
				)
				Expect(move(ctx, src, dst, a.ResourceID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(src.ResourceID()))
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
					author.ResourceID(),
					a.ResourceID(),
					src.ResourceID(),
				)
				Expect(move(ctx, src, dst, a.ResourceID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(src.ResourceID()))
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
					author.ResourceID(),
					src.ResourceID(),
					dst.ResourceID(),
				)
				Expect(move(ctx, src, dst, a.ResourceID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(src.ResourceID()))
			},
		)
		It(
			"Should deny moving a directory out of a source the caller cannot edit",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				dst := createDirectory(ctx)
				sub := createDirectory(ctx)
				Expect(arcSvc.NewWriter(nil).Move(
					ctx, root, src.Key, sub.ResourceID(),
				)).To(Succeed())
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					sub.ResourceID(),
					dst.ResourceID(),
				)
				Expect(move(ctx, src, dst, sub.ResourceID())).
					To(MatchError(access.ErrDenied))
				Expect(parentsOf(ctx, sub.ResourceID())).
					To(ConsistOf(src.ResourceID()))
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
					author.ResourceID(),
					a.ResourceID(),
					named.ResourceID(),
					dst.ResourceID(),
				)
				Expect(move(ctx, named, dst, a.ResourceID())).
					To(MatchError(validate.ErrValidation))
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(src.ResourceID()))
			},
		)
		It(
			"Should move within one directory with update access to it",
			func(ctx SpecContext) {
				src := createDirectory(ctx)
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					a.ResourceID(),
					src.ResourceID(),
				)
				Expect(move(ctx, src, src, a.ResourceID())).To(Succeed())
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(src.ResourceID()))
			},
		)
		It("Should refuse the generic ontology move of an Arc", func(ctx SpecContext) {
			d := createDirectory(ctx)
			a := createArc(ctx, uniqueName())
			DeferCleanup(func(ctx SpecContext) {
				Expect(arcSvc.NewWriter(nil).Delete(ctx, a.Base().Key)).To(Succeed())
			})
			rootID := directory.ResourceID(root)
			grantUpdateOn(
				ctx,
				author.ResourceID(),
				a.ResourceID(),
				d.ResourceID(),
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
					To:       d.ResourceID(),
					Children: []resource.ID{arc.ResourceID(a.Base().Key)},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
