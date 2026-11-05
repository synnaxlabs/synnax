// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ancestors", func() {
	parentOf := func(from, to resource.ID) ontology.Relationship {
		return ontology.Relationship{
			From: from,
			Type: ontology.RelationshipTypeParentOf,
			To:   to,
		}
	}
	var (
		top, mid, side, leaf resource.ID
		w                    ontology.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		top = newSampleType("an-top")
		mid = newSampleType("an-mid")
		side = newSampleType("an-side")
		leaf = newSampleType("an-leaf")
		w = otg.NewWriter(tx)
		Expect(w.DefineResources(ctx, top, mid, side, leaf)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, ontology.RootID, ontology.RelationshipTypeParentOf, top,
		)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, top, ontology.RelationshipTypeParentOf, mid,
		)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, mid, ontology.RelationshipTypeParentOf, leaf,
		)).To(Succeed())
	})

	It("Should return every parent relationship up to the ontology root",
		func() {
			Expect(otg.Ancestors(tx, resource.ID{}, leaf)).To(ConsistOf(
				parentOf(mid, leaf),
				parentOf(top, mid),
				parentOf(ontology.RootID, top),
			))
		})

	It("Should stop the walk at root", func() {
		Expect(otg.Ancestors(tx, top, leaf)).To(ConsistOf(
			parentOf(mid, leaf),
			parentOf(top, mid),
		))
	})

	It("Should return every path of a resource with several parents",
		func(ctx SpecContext) {
			Expect(w.DefineRelationships(
				ctx, side, ontology.RelationshipTypeParentOf, leaf,
			)).To(Succeed())
			Expect(otg.Ancestors(tx, top, leaf)).To(ConsistOf(
				parentOf(mid, leaf),
				parentOf(side, leaf),
				parentOf(top, mid),
			))
		})

	It("Should return a shared relationship once", func(ctx SpecContext) {
		other := newSampleType("an-other")
		Expect(w.DefineResources(ctx, other)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, mid, ontology.RelationshipTypeParentOf, other,
		)).To(Succeed())
		Expect(otg.Ancestors(tx, top, leaf, other)).To(ConsistOf(
			parentOf(mid, leaf),
			parentOf(mid, other),
			parentOf(top, mid),
		))
	})

	It("Should ignore relationships that are not parent relationships",
		func(ctx SpecContext) {
			Expect(w.DefineRelationships(ctx, side, "labeled_by", leaf)).To(Succeed())
			Expect(otg.Ancestors(tx, top, leaf)).To(ConsistOf(
				parentOf(mid, leaf),
				parentOf(top, mid),
			))
		})

	It("Should return nothing for a resource with no parents", func() {
		Expect(otg.Ancestors(tx, resource.ID{}, side)).To(BeEmpty())
	})

	It("Should read only committed state when tx is nil", func() {
		Expect(MustSucceed(otg.Ancestors(nil, resource.ID{}, leaf))).To(BeEmpty())
	})
})
