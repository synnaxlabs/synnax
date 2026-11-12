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
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	apiarcdirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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
			Exec(ctx, db)).To(Succeed())
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
			Exec(ctx, db)).To(Succeed())
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
			res, err := retrieveDirectories(
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
		It(
			"Should leave out the directories that the caller cannot retrieve",
			func(ctx SpecContext) {
				d := newDirectory(ctx)
				Expect(retrieve(ctx)).ToNot(ContainElement(d))
			},
		)
		It(
			"Should deny a named directory that the caller cannot retrieve",
			func(ctx SpecContext) {
				d := newDirectory(ctx)
				Expect(retrieveDirectories(
					AuthedCtx(ctx, author),
					apiarcdirectory.RetrieveRequest{
						Where: &directory.Where{
							Key: &filter.Set[directory.Key]{Eq: new(d.Key)},
						},
					},
				)).Error().To(MatchError(access.Denied(
					author.ResourceID(),
					access.ActionRetrieve,
					d.ResourceID(),
				).Error()))
			},
		)
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

	Describe("DispatchDirectory", func() {
		set := func(path string, value any) directory.Action {
			return directory.NewSetAction(
				directory.SetPayload{Path: []string{path}, Value: value},
			)
		}
		apply := func(
			ctx SpecContext,
			d directory.Directory,
			acts ...directory.Action,
		) error {
			_, err := dispatchDirectory(
				AuthedCtx(ctx, author),
				apiarc.DispatchDirectoryRequest{Key: d.Key, Actions: acts},
			)
			return err
		}
		It("Should rename a directory with update access to it", func(ctx SpecContext) {
			d := newDirectory(ctx)
			grantUpdateOn(ctx, author.ResourceID(), d.ResourceID())
			name := uniqueName()
			Expect(apply(ctx, d, set("name", name))).To(Succeed())
			Expect(retrieveDirectory(ctx, d.Key).Name).To(Equal(name))
		})
		It("Should deny a caller without update access", func(ctx SpecContext) {
			d := newDirectory(ctx)
			Expect(apply(ctx, d, set("name", uniqueName()))).
				To(denied(author, access.ActionUpdate, d.ResourceID()))
			Expect(retrieveDirectory(ctx, d.Key).Name).To(Equal(d.Name))
		})
		It(
			"Should move a directory with update access to it and both parents",
			func(ctx SpecContext) {
				dst := newDirectory(ctx)
				d := newDirectory(ctx)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					d.ResourceID(),
					directory.ResourceID(root),
					dst.ResourceID(),
				)
				Expect(apply(ctx, d, set("parent", dst.Key))).To(Succeed())
				Expect(parentsOf(ctx, d.ResourceID())).To(ConsistOf(dst.ResourceID()))
			},
		)
		It(
			"Should deny a move without update access to the new parent",
			func(ctx SpecContext) {
				dst := newDirectory(ctx)
				d := newDirectory(ctx)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					d.ResourceID(),
					directory.ResourceID(root),
				)
				Expect(apply(ctx, d, set("parent", dst.Key))).
					To(denied(author, access.ActionUpdate, dst.ResourceID()))
				Expect(parentsOf(ctx, d.ResourceID())).
					To(ConsistOf(directory.ResourceID(root)))
			},
		)
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
					Exec(ctx, db)).To(MatchError(query.ErrNotFound))
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

	Describe("Dispatch set directory", func() {
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
		setDirectory := func(ctx SpecContext, a arc.Arc, to directory.Directory) error {
			_, err := dispatch(AuthedCtx(ctx, author), apiarc.DispatchRequest{
				Key: a.Base().Key,
				Actions: []arc.Action{
					arc.NewSetDirectoryAction(
						arc.SetDirectoryPayload{Directory: to.Key},
					),
				},
			})
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
				Expect(setDirectory(ctx, a, dst)).To(Succeed())
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(dst.ResourceID()))
			},
		)
		DescribeTable(
			"Should refuse the move and keep the Arc in its directory",
			func(ctx SpecContext, refusedOn func(a arc.Arc, src, dst directory.Directory) resource.ID) {
				src := newDirectory(ctx)
				dst := newDirectory(ctx)
				a := createArcIn(ctx, src)
				refused := refusedOn(a, src, dst)
				grantUpdateOn(ctx, author.ResourceID(), lo.Without(
					[]resource.ID{a.ResourceID(), src.ResourceID(), dst.ResourceID()},
					refused,
				)...)
				Expect(setDirectory(ctx, a, dst)).
					To(denied(author, access.ActionUpdate, refused))
				Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(src.ResourceID()))
			},
			Entry(
				"without update on the source",
				func(_ arc.Arc, src, _ directory.Directory) resource.ID {
					return src.ResourceID()
				},
			),
			Entry(
				"without update on the destination",
				func(_ arc.Arc, _, dst directory.Directory) resource.ID {
					return dst.ResourceID()
				},
			),
			Entry(
				"without update on the Arc",
				func(a arc.Arc, _, _ directory.Directory) resource.ID {
					return a.ResourceID()
				},
			),
		)
	})
})
