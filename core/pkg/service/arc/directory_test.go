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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Directories", func() {
	var (
		w    arc.Writer
		root directory.Key
	)
	parentsOf := func(ctx SpecContext, id resource.ID) []resource.ID {
		GinkgoHelper()
		var parents []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ParentsTraverser).
			ExcludeFieldData(true).
			Entries(&parents).
			Exec(ctx, tx)).To(Succeed())
		ids := make([]resource.ID, 0, len(parents))
		for _, p := range parents {
			ids = append(ids, p.ID)
		}
		return ids
	}
	createDirectory := func(
		ctx SpecContext,
		name string,
		parent directory.Key,
	) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name}
		Expect(w.CreateDirectory(ctx, &d, parent)).To(Succeed())
		return d
	}
	createArc := func(ctx SpecContext, name string, t arc.Type) arc.Arc {
		GinkgoHelper()
		a := newArc(
			t,
			arc.Base{Name: name, Source: arc.Source{Variant: arc.TextSource{}}},
		)
		Expect(w.Create(ctx, &a)).To(Succeed())
		return a
	}
	BeforeEach(func(ctx SpecContext) {
		w = svc.NewWriter(tx)
		var roots []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(ontology.RootID).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(resource.TypeArcDirectory).
			Entries(&roots).
			Exec(ctx, tx)).To(Succeed())
		Expect(roots).To(HaveLen(1))
		Expect(roots[0].Name).To(Equal("Arcs"))
		root = MustSucceed(directory.KeyFromResourceID(roots[0].ID))
	})

	Describe("Root", func() {
		It("Should reject renaming the root", func(ctx SpecContext) {
			Expect(w.RenameDirectory(ctx, root, "Automations")).
				To(MatchError(ContainSubstring("root Arcs directory cannot be renamed")))
		})
		It("Should reject moving the root", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			Expect(w.Move(ctx, root, lib.Key, directory.ResourceID(root))).
				To(MatchError(ContainSubstring("root Arcs directory cannot be moved")))
		})
		It("Should reject deleting the root", func(ctx SpecContext) {
			Expect(w.DeleteDirectory(ctx, root)).
				To(MatchError(ContainSubstring("root Arcs directory cannot be deleted")))
		})
	})

	Describe("Create", func() {
		It("Should create a directory under its parent", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			sub := createDirectory(ctx, "sub", lib.Key)
			Expect(parentsOf(ctx, sub.ResourceID())).To(ConsistOf(lib.ResourceID()))
		})
		It("Should reject a name that is not an identifier", func(ctx SpecContext) {
			d := directory.Directory{Name: "my lib"}
			Expect(w.CreateDirectory(ctx, &d, root)).To(MatchError(ContainSubstring(
				"must contain only letters, digits, and underscores",
			)))
		})
		It("Should reject a name used in the directory", func(ctx SpecContext) {
			createArc(ctx, "shared", arc.TypeModule)
			d := directory.Directory{Name: "shared"}
			Expect(w.CreateDirectory(ctx, &d, root)).
				To(MatchError(ContainSubstring(`name "shared" is already used`)))
		})
		It("Should allow one name in two directories", func(ctx SpecContext) {
			a := createDirectory(ctx, "a", root)
			b := createDirectory(ctx, "b", root)
			createDirectory(ctx, "same", a.Key)
			createDirectory(ctx, "same", b.Key)
		})
		It(
			"Should reject a standard module name directly under the root",
			func(ctx SpecContext) {
				d := directory.Directory{Name: "time"}
				Expect(w.CreateDirectory(ctx, &d, root)).
					To(MatchError(ContainSubstring("is a standard module name")))
			},
		)
		It("Should allow a standard module name in a directory", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			createDirectory(ctx, "time", lib.Key)
		})
	})

	Describe("Rename", func() {
		It("Should rename a directory", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			Expect(w.RenameDirectory(ctx, lib.Key, "core")).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(lib.ResourceID()).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("core"))
		})
		It("Should reject a name used in the directory", func(ctx SpecContext) {
			createDirectory(ctx, "taken", root)
			lib := createDirectory(ctx, "lib", root)
			Expect(w.RenameDirectory(ctx, lib.Key, "taken")).
				To(MatchError(ContainSubstring("already used")))
		})
	})

	Describe("Delete", func() {
		It("Should delete an empty directory", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			Expect(w.DeleteDirectory(ctx, lib.Key)).To(Succeed())
			Expect(otg.NewRetrieve().
				WhereIDs(lib.ResourceID()).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
		It("Should reject deleting a directory that is not empty", func(
			ctx SpecContext,
		) {
			lib := createDirectory(ctx, "lib", root)
			createDirectory(ctx, "sub", lib.Key)
			Expect(w.DeleteDirectory(ctx, lib.Key)).
				To(MatchError(ContainSubstring(`directory "lib" is not empty`)))
		})
	})

	Describe("Move", func() {
		It(
			"Should move an Arc into a directory with one parent",
			func(ctx SpecContext) {
				lib := createDirectory(ctx, "lib", root)
				m := createArc(ctx, "m", arc.TypeModule)
				Expect(w.Move(ctx, root, lib.Key, m.ResourceID())).To(Succeed())
				Expect(parentsOf(ctx, m.ResourceID())).To(ConsistOf(lib.ResourceID()))
			},
		)
		It("Should move a directory with its entries", func(ctx SpecContext) {
			a := createDirectory(ctx, "a", root)
			b := createDirectory(ctx, "b", root)
			m := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, root, a.Key, m.ResourceID())).To(Succeed())
			Expect(w.Move(ctx, root, b.Key, a.ResourceID())).To(Succeed())
			Expect(parentsOf(ctx, a.ResourceID())).To(ConsistOf(b.ResourceID()))
			Expect(parentsOf(ctx, m.ResourceID())).To(ConsistOf(a.ResourceID()))
		})
		It("Should leave an entry moved into its own directory", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, root, root, m.ResourceID())).To(Succeed())
			Expect(parentsOf(ctx, m.ResourceID())).
				To(ConsistOf(directory.ResourceID(root)))
		})
		It("Should reject an entry that is not in the source", func(
			ctx SpecContext,
		) {
			a := createDirectory(ctx, "a", root)
			b := createDirectory(ctx, "b", root)
			m := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, a.Key, b.Key, m.ResourceID())).
				To(MatchError(ContainSubstring(`"m" is not in the source directory`)))
			Expect(parentsOf(ctx, m.ResourceID())).
				To(ConsistOf(directory.ResourceID(root)))
		})
		It("Should reject a batch with one entry outside the source", func(
			ctx SpecContext,
		) {
			a := createDirectory(ctx, "a", root)
			b := createDirectory(ctx, "b", root)
			inRoot := createArc(ctx, "in_root", arc.TypeModule)
			inA := createArc(ctx, "in_a", arc.TypeModule)
			Expect(w.Move(ctx, root, a.Key, inA.ResourceID())).To(Succeed())
			Expect(w.Move(
				ctx, root, b.Key, inRoot.ResourceID(), inA.ResourceID(),
			)).To(MatchError(ContainSubstring(`"in_a" is not in the source directory`)))
			Expect(parentsOf(ctx, inRoot.ResourceID())).
				To(ConsistOf(directory.ResourceID(root)))
			Expect(parentsOf(ctx, inA.ResourceID())).To(ConsistOf(a.ResourceID()))
		})
		It("Should reject a move that duplicates a name", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			first := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, root, lib.Key, first.ResourceID())).To(Succeed())
			second := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, root, lib.Key, second.ResourceID())).
				To(MatchError(ContainSubstring(`name "m" is already used`)))
		})
		It("Should reject two legacy entries of one name moved together", func(
			ctx SpecContext,
		) {
			lib := createDirectory(ctx, "lib", root)
			// Stored without the name rules, as Arcs migrated from a release before
			// the rules are.
			first := newAutomation(arc.Base{Key: uuid.New(), Name: "dup"})
			second := newAutomation(arc.Base{Key: uuid.New(), Name: "dup"})
			first.ApplyDefaults()
			second.ApplyDefaults()
			Expect(gorp.NewCreate[arc.Key, arc.Arc]().
				Entries(&[]arc.Arc{first, second}).
				Exec(ctx, tx)).To(Succeed())
			Expect(otg.NewWriter(tx).DefineResources(
				ctx, first.ResourceID(), second.ResourceID(),
			)).To(Succeed())
			Expect(w.Move(ctx, root, lib.Key, first.ResourceID(), second.ResourceID())).
				To(MatchError(ContainSubstring(`name "dup" is already used`)))
		})
		It("Should reject moving a directory into itself", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			Expect(w.Move(ctx, root, lib.Key, lib.ResourceID())).
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject moving a directory below itself", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			sub := createDirectory(ctx, "sub", lib.Key)
			Expect(w.Move(ctx, root, sub.Key, lib.ResourceID())).
				To(MatchError(graph.ErrCyclicDependency))
		})
		It("Should reject moving a resource of another type", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			other := resource.ID{Type: resource.TypeChannel, Key: "1"}
			Expect(w.Move(ctx, root, lib.Key, other)).
				To(MatchError(ContainSubstring("cannot go in an Arc directory")))
		})
	})

	Describe("Arc names", func() {
		It("Should reject a module whose name is not an identifier", func(
			ctx SpecContext,
		) {
			a := newModule(
				arc.Base{
					Name:   "my module",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(w.Create(ctx, &a)).To(MatchError(ContainSubstring(
				"must contain only letters, digits, and underscores",
			)))
		})
		It("Should accept any automation name", func(ctx SpecContext) {
			createArc(ctx, "My Automation", arc.TypeAutomation)
		})
		It("Should reject a new Arc whose name is used in the root", func(
			ctx SpecContext,
		) {
			createDirectory(ctx, "lib", root)
			a := newModule(
				arc.Base{Name: "lib", Source: arc.Source{Variant: arc.TextSource{}}},
			)
			Expect(w.Create(ctx, &a)).
				To(MatchError(ContainSubstring(`name "lib" is already used`)))
		})
		It("Should reject a rename by upsert that duplicates a name", func(
			ctx SpecContext,
		) {
			createArc(ctx, "taken", arc.TypeModule)
			a := createArc(ctx, "free", arc.TypeModule)
			b := a.Base()
			b.Name = "taken"
			a.SetBase(b)
			Expect(w.Create(ctx, &a)).To(MatchError(ContainSubstring("already used")))
		})
		It("Should reject a dispatched rename that duplicates a name", func(
			ctx SpecContext,
		) {
			taken := newAutomation(
				arc.Base{
					Name:   "dispatch_taken",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(writer.Create(ctx, &taken)).To(Succeed())
			renamed := newAutomation(
				arc.Base{
					Name:   "dispatch_free",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(writer.Create(ctx, &renamed)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(
					writer.Delete(ctx, taken.Base().Key, renamed.Base().Key),
				).To(Succeed())
			})
			Expect(svc.Dispatch(ctx, renamed.Base().Key, "", []arc.Action{
				arc.NewRenameAction(arc.RenamePayload{Name: "dispatch_taken"}),
			})).To(MatchError(ContainSubstring("already used")))
		})
	})

	Describe("Ownership", func() {
		It("Should refuse a generic writer moving an Arc", func(ctx SpecContext) {
			g := MustSucceed(groupSvc.NewWriter(tx).Create(ctx, "g", ontology.RootID))
			a := createArc(ctx, "a", arc.TypeAutomation)
			Expect(otg.NewWriter(tx).DefineRelationships(
				ctx, g.ResourceID(), ontology.RelationshipTypeParentOf, a.ResourceID(),
			)).To(MatchError(validate.ErrValidation))
		})
		It("Should refuse a group created in a directory", func(ctx SpecContext) {
			Expect(groupSvc.NewWriter(tx).Create(
				ctx, "g", directory.ResourceID(root),
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
