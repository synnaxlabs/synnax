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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
	parentsOf := func(ctx SpecContext, id ontology.ID) []ontology.ID {
		GinkgoHelper()
		var parents []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ParentsTraverser).
			ExcludeFieldData(true).
			Entries(&parents).
			Exec(ctx, tx)).To(Succeed())
		ids := make([]ontology.ID, 0, len(parents))
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
		a := arc.Arc{Name: name, Mode: arc.ModeText, Type: t}
		Expect(w.Create(ctx, &a)).To(Succeed())
		return a
	}
	BeforeEach(func(ctx SpecContext) {
		w = svc.NewWriter(tx)
		var roots []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(ontology.RootID).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypeArcDirectory).
			Entries(&roots).
			Exec(ctx, tx)).To(Succeed())
		Expect(roots).To(HaveLen(1))
		Expect(roots[0].Name).To(Equal("Arcs"))
		root = MustSucceed(directory.KeyFromOntologyID(roots[0].ID))
	})

	Describe("Root", func() {
		It("Should reject renaming the root", func(ctx SpecContext) {
			Expect(w.RenameDirectory(ctx, root, "Automations")).
				To(MatchError(ContainSubstring("root Arcs directory cannot be renamed")))
		})
		It("Should reject moving the root", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			Expect(w.Move(ctx, lib.Key, directory.OntologyID(root))).
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
			Expect(parentsOf(ctx, sub.OntologyID())).To(ConsistOf(lib.OntologyID()))
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
				WhereIDs(lib.OntologyID()).
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
				WhereIDs(lib.OntologyID()).
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
				Expect(w.Move(ctx, lib.Key, m.OntologyID())).To(Succeed())
				Expect(parentsOf(ctx, m.OntologyID())).To(ConsistOf(lib.OntologyID()))
			},
		)
		It("Should move a directory with its entries", func(ctx SpecContext) {
			a := createDirectory(ctx, "a", root)
			b := createDirectory(ctx, "b", root)
			m := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, a.Key, m.OntologyID())).To(Succeed())
			Expect(w.Move(ctx, b.Key, a.OntologyID())).To(Succeed())
			Expect(parentsOf(ctx, a.OntologyID())).To(ConsistOf(b.OntologyID()))
			Expect(parentsOf(ctx, m.OntologyID())).To(ConsistOf(a.OntologyID()))
		})
		It("Should leave an entry moved into its own directory", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, root, m.OntologyID())).To(Succeed())
			Expect(parentsOf(ctx, m.OntologyID())).
				To(ConsistOf(directory.OntologyID(root)))
		})
		It("Should reject a move that duplicates a name", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			first := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, lib.Key, first.OntologyID())).To(Succeed())
			second := createArc(ctx, "m", arc.TypeModule)
			Expect(w.Move(ctx, lib.Key, second.OntologyID())).
				To(MatchError(ContainSubstring(`name "m" is already used`)))
		})
		It("Should reject two entries of one name moved together", func(
			ctx SpecContext,
		) {
			a := createDirectory(ctx, "a", root)
			b := createDirectory(ctx, "b", root)
			c := createDirectory(ctx, "c", root)
			first := createDirectory(ctx, "dup", a.Key)
			second := createDirectory(ctx, "dup", b.Key)
			Expect(w.Move(ctx, c.Key, first.OntologyID(), second.OntologyID())).
				To(MatchError(ContainSubstring(`name "dup" is already used`)))
		})
		It("Should reject moving a directory into itself", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			Expect(w.Move(ctx, lib.Key, lib.OntologyID())).
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject moving a directory below itself", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			sub := createDirectory(ctx, "sub", lib.Key)
			Expect(w.Move(ctx, sub.Key, lib.OntologyID())).
				To(MatchError(graph.ErrCyclicDependency))
		})
		It("Should reject moving a resource of another type", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			other := ontology.ID{Type: ontology.ResourceTypeChannel, Key: "1"}
			Expect(w.Move(ctx, lib.Key, other)).
				To(MatchError(ContainSubstring("cannot go in an Arc directory")))
		})
	})

	Describe("Arc names", func() {
		It("Should reject a module whose name is not an identifier", func(
			ctx SpecContext,
		) {
			a := arc.Arc{Name: "my module", Mode: arc.ModeText, Type: arc.TypeModule}
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
			a := arc.Arc{Name: "lib", Mode: arc.ModeText, Type: arc.TypeModule}
			Expect(w.Create(ctx, &a)).
				To(MatchError(ContainSubstring(`name "lib" is already used`)))
		})
		It("Should reject a rename by upsert that duplicates a name", func(
			ctx SpecContext,
		) {
			createArc(ctx, "taken", arc.TypeModule)
			a := createArc(ctx, "free", arc.TypeModule)
			a.Name = "taken"
			Expect(w.Create(ctx, &a)).To(MatchError(ContainSubstring("already used")))
		})
		It("Should reject a dispatched rename that duplicates a name", func(
			ctx SpecContext,
		) {
			taken := arc.Arc{Name: "dispatch_taken", Mode: arc.ModeText}
			Expect(writer.Create(ctx, &taken)).To(Succeed())
			renamed := arc.Arc{Name: "dispatch_free", Mode: arc.ModeText}
			Expect(writer.Create(ctx, &renamed)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(writer.Delete(ctx, taken.Key, renamed.Key)).To(Succeed())
			})
			Expect(svc.Dispatch(ctx, renamed.Key, "", []arc.Action{
				arc.NewRenameAction(arc.RenamePayload{Name: "dispatch_taken"}),
			})).To(MatchError(ContainSubstring("already used")))
		})
	})

	Describe("Ownership", func() {
		It("Should refuse a generic writer moving an Arc", func(ctx SpecContext) {
			g := MustSucceed(groupSvc.NewWriter(tx).Create(ctx, "g", ontology.RootID))
			a := createArc(ctx, "a", arc.TypeAutomation)
			Expect(otg.NewWriter(tx).DefineRelationships(
				ctx, g.OntologyID(), ontology.RelationshipTypeParentOf, a.OntologyID(),
			)).To(MatchError(validate.ErrValidation))
		})
		It("Should refuse a group created in a directory", func(ctx SpecContext) {
			Expect(groupSvc.NewWriter(tx).Create(
				ctx, "g", directory.OntologyID(root),
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
