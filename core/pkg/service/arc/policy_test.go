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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Policy", func() {
	var (
		arcs   ontology.ID
		groups group.Writer
		otgW   ontology.Writer
	)

	BeforeEach(func(ctx SpecContext) {
		var g group.Group
		Expect(groupSvc.NewRetrieve().
			Where(group.MatchNames("Arcs")).
			Entry(&g).
			Exec(ctx, tx)).To(Succeed())
		arcs = g.OntologyID()
		groups = groupSvc.NewWriter(tx)
		otgW = otg.NewWriter(tx)
	})

	createArc := func(ctx SpecContext, name string) arc.Arc {
		GinkgoHelper()
		a := arc.Arc{Name: name, Mode: arc.ModeText}
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		return a
	}

	move := func(ctx SpecContext, from, to, child ontology.ID) error {
		GinkgoHelper()
		Expect(otgW.DeleteRelationships(ctx, ontology.Relationship{
			From: from, Type: ontology.RelationshipTypeParentOf, To: child,
		})).To(Succeed())
		return otgW.DefineRelationships(
			ctx, to, ontology.RelationshipTypeParentOf, child,
		)
	}

	Describe("Names", func() {
		It(
			"Should reject a group under Arcs whose name is not an identifier",
			func(ctx SpecContext) {
				Expect(groups.Create(ctx, "sy lib", arcs)).Error().
					To(MatchError(ContainSubstring("letters, digits, and underscores")))
			},
		)
		It(
			"Should accept an identifier under Arcs and any name elsewhere",
			func(ctx SpecContext) {
				MustSucceed(groups.Create(ctx, "sy_lib", arcs))
				MustSucceed(groups.Create(ctx, "sy lib", ontology.RootID))
			},
		)
		It(
			"Should reject a module whose name is not an identifier",
			func(ctx SpecContext) {
				m := arc.Arc{Name: "my lib", Type: arc.TypeModule, Mode: arc.ModeText}
				Expect(svc.NewWriter(tx).Create(ctx, &m)).
					To(MatchError(ContainSubstring("letters, digits, and underscores")))
				a := arc.Arc{Name: "my automation", Mode: arc.ModeText}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			},
		)
	})

	Describe("Name conflicts", func() {
		It(
			"Should reject two entries with one name in one directory",
			func(ctx SpecContext) {
				createArc(ctx, "dup")
				Expect(groups.Create(ctx, "dup", arcs)).Error().
					To(MatchError(ContainSubstring("already used")))
				a := arc.Arc{Name: "dup", Mode: arc.ModeText}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).
					To(MatchError(ContainSubstring("already used")))
			},
		)
		It("Should allow one name in two directories", func(ctx SpecContext) {
			sub := MustSucceed(groups.Create(ctx, "sy_lib", arcs))
			a := createArc(ctx, "m1")
			Expect(move(ctx, arcs, sub.OntologyID(), a.OntologyID())).To(Succeed())
			createArc(ctx, "m1")
		})
		It("Should reject a move that duplicates a name", func(ctx SpecContext) {
			sub := MustSucceed(groups.Create(ctx, "sy_lib", arcs))
			a := createArc(ctx, "m2")
			Expect(move(ctx, arcs, sub.OntologyID(), a.OntologyID())).To(Succeed())
			b := createArc(ctx, "m2")
			Expect(move(ctx, arcs, sub.OntologyID(), b.OntologyID())).
				To(MatchError(ContainSubstring("already used")))
		})
		It(
			"Should reject a group rename that duplicates a name",
			func(ctx SpecContext) {
				MustSucceed(groups.Create(ctx, "g1", arcs))
				g2 := MustSucceed(groups.Create(ctx, "g2", arcs))
				Expect(groups.Rename(ctx, g2.Key, "g1")).
					To(MatchError(ContainSubstring("already used")))
			},
		)
		It(
			"Should reject an Arc rename by upsert that duplicates a name",
			func(ctx SpecContext) {
				createArc(ctx, "r1")
				b := createArc(ctx, "r2")
				b.Name = "r1"
				Expect(svc.NewWriter(tx).Create(ctx, &b)).
					To(MatchError(ContainSubstring("already used")))
			},
		)
		It(
			"Should reject a dispatched rename that duplicates a name",
			func(ctx SpecContext) {
				w := svc.NewWriter(nil)
				a := arc.Arc{Name: "d1", Mode: arc.ModeText}
				b := arc.Arc{Name: "d2", Mode: arc.ModeText}
				Expect(w.Create(ctx, &a)).To(Succeed())
				Expect(w.Create(ctx, &b)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(w.Delete(ctx, a.Key, b.Key)).To(Succeed())
				})
				Expect(svc.Dispatch(ctx, b.Key, "", []arc.Action{
					arc.NewRenameAction(arc.RenamePayload{Name: "d1"}),
				})).To(MatchError(ContainSubstring("already used")))
			},
		)
	})

	Describe("Standard module names", func() {
		It(
			"Should reject a standard module name directly under Arcs",
			func(ctx SpecContext) {
				Expect(groups.Create(ctx, "time", arcs)).Error().
					To(MatchError(ContainSubstring("standard module")))
				a := arc.Arc{Name: "time", Mode: arc.ModeText}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).
					To(MatchError(ContainSubstring("standard module")))
			},
		)
		It(
			"Should allow a standard module name in a directory",
			func(ctx SpecContext) {
				sub := MustSucceed(groups.Create(ctx, "sy_lib", arcs))
				MustSucceed(groups.Create(ctx, "time", sub.OntologyID()))
			},
		)
	})

	Describe("Membership", func() {
		It(
			"Should reject a resource of another type under Arcs",
			func(ctx SpecContext) {
				Expect(otgW.DefineRelationships(
					ctx, arcs, ontology.RelationshipTypeParentOf, testRack.OntologyID(),
				)).To(MatchError(ContainSubstring("cannot go under Arcs")))
			},
		)
		It("Should reject an Arc outside Arcs", func(ctx SpecContext) {
			elsewhere := MustSucceed(groups.Create(ctx, "elsewhere", ontology.RootID))
			a := createArc(ctx, "out")
			Expect(otgW.DefineRelationships(
				ctx,
				elsewhere.OntologyID(),
				ontology.RelationshipTypeParentOf,
				a.OntologyID(),
			)).To(MatchError(ContainSubstring("only go under Arcs")))
		})
		It("Should reject a group holding an Arc outside Arcs", func(ctx SpecContext) {
			sub := MustSucceed(groups.Create(ctx, "sy_lib", arcs))
			inner := MustSucceed(groups.Create(ctx, "inner", sub.OntologyID()))
			a := createArc(ctx, "m")
			Expect(move(ctx, arcs, inner.OntologyID(), a.OntologyID())).To(Succeed())
			Expect(move(ctx, arcs, ontology.RootID, sub.OntologyID())).
				To(MatchError(ContainSubstring("holding an Arc")))
		})
		It("Should let a group without Arcs leave Arcs", func(ctx SpecContext) {
			sub := MustSucceed(groups.Create(ctx, "sy_lib", arcs))
			MustSucceed(groups.Create(ctx, "inner", sub.OntologyID()))
			Expect(move(ctx, arcs, ontology.RootID, sub.OntologyID())).To(Succeed())
		})
		It(
			"Should move a directory with its entries between directories",
			func(ctx SpecContext) {
				d1 := MustSucceed(groups.Create(ctx, "d1", arcs))
				d2 := MustSucceed(groups.Create(ctx, "d2", arcs))
				a := createArc(ctx, "m")
				Expect(move(ctx, arcs, d1.OntologyID(), a.OntologyID())).To(Succeed())
				Expect(move(ctx, arcs, d2.OntologyID(), d1.OntologyID())).To(Succeed())
			},
		)
		DescribeTable(
			"Should check every entry of a group placed under Arcs",
			func(ctx SpecContext, fill func(ctx SpecContext, g ontology.ID), msg string) {
				g := MustSucceed(groups.Create(ctx, "g", ontology.RootID))
				fill(ctx, g.OntologyID())
				Expect(move(ctx, ontology.RootID, arcs, g.OntologyID())).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("another type", func(ctx SpecContext, g ontology.ID) {
				GinkgoHelper()
				Expect(otgW.DefineRelationships(
					ctx, g, ontology.RelationshipTypeParentOf, testRack.OntologyID(),
				)).To(Succeed())
			}, "cannot go under Arcs"),
			Entry(
				"a name that is not an identifier",
				func(ctx SpecContext, g ontology.ID) {
					GinkgoHelper()
					MustSucceed(groups.Create(ctx, "sy lib", g))
				},
				"letters, digits, and underscores",
			),
			Entry("two entries of one name", func(ctx SpecContext, g ontology.ID) {
				GinkgoHelper()
				MustSucceed(groups.Create(ctx, "x", g))
				MustSucceed(groups.Create(ctx, "x", g))
			}, "already used"),
		)
	})
})
