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
	apiarcdirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
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
	retrieveDirectory := func(ctx SpecContext, key directory.Key) directory.Directory {
		GinkgoHelper()
		var d directory.Directory
		Expect(arcSvc.NewDirectoryRetrieve().
			Where(directory.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, nil)).To(Succeed())
		return d
	}
	parentsOf := func(ctx SpecContext, id resource.ID) []resource.ID {
		GinkgoHelper()
		key := MustSucceed(resource.ParseUUIDKey(id))
		if id.Type != resource.TypeArc {
			return []resource.ID{
				directory.ResourceID(*retrieveDirectory(ctx, key).Parent),
			}
		}
		var a arc.Arc
		Expect(arcSvc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&a).
			Exec(ctx, nil)).To(Succeed())
		return []resource.ID{directory.ResourceID(*a.Base().Directory)}
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
		var r directory.Directory
		Expect(arcSvc.NewDirectoryRetrieve().
			Where(directory.MatchNested(false)).
			Entry(&r).
			Exec(ctx, nil)).To(Succeed())
		root = r.Key
	})

	Describe("RetrieveDirectories", func() {
		retrieve := func(ctx SpecContext) ([]directory.Directory, error) {
			res, err := apiSvc.RetrieveDirectories(
				AuthedCtx(ctx, author),
				apiarcdirectory.RetrieveRequest{
					Where: &directory.Where{Parents: []directory.Key{root}},
				},
			)
			return res.Directories, err
		}
		It("Should retrieve the directories a directory holds", func(ctx SpecContext) {
			d := createDirectory(ctx)
			grantOn(ctx, author.ResourceID(), access.ActionRetrieve, d.ResourceID())
			Expect(retrieve(ctx)).To(ContainElement(d))
		})
		It("Should deny a caller without retrieve access", func(ctx SpecContext) {
			createDirectory(ctx)
			Expect(retrieve(ctx)).Error().To(MatchError(access.ErrDenied))
		})
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
			Expect(retrieveDirectory(ctx, d.Key).Name).To(Equal(name))
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
			var res directory.Directory
			Expect(arcSvc.NewDirectoryRetrieve().
				Where(directory.MatchKeys(d.Key)).
				Entry(&res).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
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
	})
})
