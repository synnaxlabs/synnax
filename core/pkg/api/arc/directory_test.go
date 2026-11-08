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
	"github.com/synnaxlabs/gorp"
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
	// newDirectory creates a directory under the root as the Core, deleted on cleanup.
	newDirectory := func(ctx SpecContext) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: uniqueName()}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).CreateDirectory(ctx, &d, root)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).DeleteDirectory(ctx, d.Key)
			})).To(Succeed())
		})
		return d
	}
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
			d := newDirectory(ctx)
			grantOn(ctx, author.ResourceID(), access.ActionRetrieve, d.ResourceID())
			Expect(retrieve(ctx)).To(ContainElement(d))
		})
		It("Should deny a caller without retrieve access", func(ctx SpecContext) {
			newDirectory(ctx)
			Expect(retrieve(ctx)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("CreateDirectory", func() {
		createIn := func(
			ctx SpecContext,
			parent directory.Key,
		) (apiarc.CreateDirectoryResponse, error) {
			return createDirectory(
				AuthedCtx(ctx, author),
				apiarc.CreateDirectoryRequest{
					Parent:    parent,
					Directory: directory.Directory{Name: uniqueName()},
				},
			)
		}
		It(
			"Should create a directory with create access and update on its parent",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionCreate,
					resource.ID{Type: resource.TypeArcDirectory},
				)
				grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
				res := MustSucceed(createIn(ctx, root))
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return arcSvc.NewWriter(tx).
							DeleteDirectory(ctx, res.Directory.Key)
					})).
						To(Succeed())
				})
				Expect(parentsOf(ctx, res.Directory.ResourceID())).
					To(ConsistOf(directory.ResourceID(root)))
			},
		)
		It("Should deny a caller without create access", func(ctx SpecContext) {
			grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
			Expect(createIn(ctx, root)).Error().To(MatchError(access.ErrDenied))
		})
		It(
			"Should deny a caller without update access to the parent",
			func(ctx SpecContext) {
				parent := newDirectory(ctx)
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionCreate,
					resource.ID{Type: resource.TypeArcDirectory},
				)
				Expect(createIn(ctx, parent.Key)).Error().
					To(denied(author, access.ActionUpdate, parent.ResourceID()))
			},
		)
	})

	Describe("RenameDirectory", func() {
		rename := func(ctx SpecContext, d directory.Directory, name string) error {
			_, err := renameDirectory(
				AuthedCtx(ctx, author),
				apiarc.RenameDirectoryRequest{Key: d.Key, Name: name},
			)
			return err
		}
		It("Should rename a directory with update access to it", func(ctx SpecContext) {
			d := newDirectory(ctx)
			grantUpdateOn(ctx, author.ResourceID(), d.ResourceID())
			name := uniqueName()
			Expect(rename(ctx, d, name)).To(Succeed())
			Expect(retrieveDirectory(ctx, d.Key).Name).To(Equal(name))
		})
		It("Should deny a caller without update access", func(ctx SpecContext) {
			d := newDirectory(ctx)
			Expect(rename(ctx, d, uniqueName())).
				To(denied(author, access.ActionUpdate, d.ResourceID()))
			Expect(retrieveDirectory(ctx, d.Key).Name).To(Equal(d.Name))
		})
		It("Should reject a rename of the root directory", func(ctx SpecContext) {
			grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
			Expect(rename(ctx, retrieveDirectory(ctx, root), uniqueName())).
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("DeleteDirectory", func() {
		deleteDir := func(ctx SpecContext, d directory.Directory) error {
			_, err := deleteDirectory(
				AuthedCtx(ctx, author),
				apiarc.DeleteDirectoryRequest{Keys: []directory.Key{d.Key}},
			)
			return err
		}
		It(
			"Should delete a directory with delete access and update on its parent",
			func(ctx SpecContext) {
				d := directory.Directory{Name: uniqueName()}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).CreateDirectory(ctx, &d, root)
				})).
					To(Succeed())
				grantOn(ctx, author.ResourceID(), access.ActionDelete, d.ResourceID())
				grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
				Expect(deleteDir(ctx, d)).To(Succeed())
				var res directory.Directory
				Expect(arcSvc.NewDirectoryRetrieve().
					Where(directory.MatchKeys(d.Key)).
					Entry(&res).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			},
		)
		It("Should deny a caller without delete access", func(ctx SpecContext) {
			d := newDirectory(ctx)
			grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
			Expect(deleteDir(ctx, d)).
				To(denied(author, access.ActionDelete, d.ResourceID()))
			Expect(retrieveDirectory(ctx, d.Key).Name).To(Equal(d.Name))
		})
		It(
			"Should deny a caller without update access to the parent",
			func(ctx SpecContext) {
				d := newDirectory(ctx)
				grantOn(ctx, author.ResourceID(), access.ActionDelete, d.ResourceID())
				Expect(deleteDir(ctx, d)).
					To(denied(author, access.ActionUpdate, directory.ResourceID(root)))
				Expect(retrieveDirectory(ctx, d.Key).Name).To(Equal(d.Name))
			},
		)
	})

	Describe("Move", func() {
		// createArcIn creates an Arc in the directory d, deleted on cleanup.
		createArcIn := func(ctx SpecContext, d directory.Directory) arc.Arc {
			GinkgoHelper()
			a := createArc(ctx, uniqueName())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Delete(ctx, a.Base().Key)
				})).To(Succeed())
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).Move(ctx, root, d.Key, a.ResourceID())
			})).
				To(Succeed())
			return a
		}
		move := func(
			ctx SpecContext,
			from, to directory.Directory,
			ids ...resource.ID,
		) error {
			_, err := move(
				AuthedCtx(ctx, author),
				apiarc.MoveRequest{From: from.Key, To: to.Key, IDs: ids},
			)
			return err
		}
		It(
			"Should move an Arc with update access to it and both directories",
			func(ctx SpecContext) {
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
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
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
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
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
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
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
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
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
				sub := newDirectory(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Move(
						ctx, root, src.Key, sub.ResourceID(),
					)
				})).To(Succeed())
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
			"Should move an Arc and a directory in one request",
			func(ctx SpecContext) {
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
				sub := newDirectory(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Move(
						ctx, root, src.Key, sub.ResourceID(),
					)
				})).To(Succeed())
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					a.ResourceID(),
					sub.ResourceID(),
					src.ResourceID(),
					dst.ResourceID(),
				)
				Expect(move(ctx, src, dst, a.ResourceID(), sub.ResourceID())).
					To(Succeed())
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(dst.ResourceID()))
				Expect(parentsOf(ctx, sub.ResourceID())).
					To(ConsistOf(dst.ResourceID()))
			},
		)
		It(
			"Should move neither entry when the caller cannot edit one of them",
			func(ctx SpecContext) {
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
				sub := newDirectory(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Move(
						ctx, root, src.Key, sub.ResourceID(),
					)
				})).To(Succeed())
				a := createArcIn(ctx, src)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					a.ResourceID(),
					src.ResourceID(),
					dst.ResourceID(),
				)
				Expect(move(ctx, src, dst, a.ResourceID(), sub.ResourceID())).
					To(denied(author, access.ActionUpdate, sub.ResourceID()))
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(src.ResourceID()))
				Expect(parentsOf(ctx, sub.ResourceID())).
					To(ConsistOf(src.ResourceID()))
			},
		)
		It(
			"Should reject a source that does not hold the entry",
			func(ctx SpecContext) {
				src := newDirectory(ctx)
				named := newDirectory(ctx)
				dst := newDirectory(ctx)
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
				src := newDirectory(ctx)
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
