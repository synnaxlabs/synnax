// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	create := func(ctx SpecContext, name string) v1.Range {
		GinkgoHelper()
		r := v1.Range{
			Key:       uuid.New(),
			Name:      name,
			TimeRange: telem.NewRangeSeconds(1, 2),
		}
		Expect(gorp.WrapWriter[v1.Key, v1.Range](db).Set(ctx, r)).To(Succeed())
		return r
	}
	rangeID := func(k uuid.UUID) resource.ID {
		return resource.ID{Type: resource.TypeRange, Key: k.String()}
	}
	labelID := func(k uuid.UUID) resource.ID {
		return resource.ID{Type: resource.TypeLabel, Key: k.String()}
	}
	relate := func(ctx SpecContext, rels ...ontology.Relationship) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[string, ontology.Relationship](db).
			Set(ctx, rels...)).To(Succeed())
	}
	migrate := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Range",
			Migrations: v2.Migrations,
		})).To(Succeed())
	}
	get := func(ctx SpecContext, key uuid.UUID) v2.Range {
		GinkgoHelper()
		var r v2.Range
		Expect(gorp.NewRetrieve[v2.Key, v2.Range]().
			Where(gorp.MatchKeys[v2.Key, v2.Range](key)).
			Entry(&r).
			Exec(ctx, db)).To(Succeed())
		return r
	}
	relationships := func(ctx SpecContext) []ontology.Relationship {
		GinkgoHelper()
		var rels []ontology.Relationship
		Expect(gorp.NewRetrieve[string, ontology.Relationship]().
			Entries(&rels).
			Exec(ctx, db)).To(Succeed())
		return rels
	}

	It("Should move a parent relationship into the parent", func(ctx SpecContext) {
		parent, child := create(ctx, "Test"), create(ctx, "Burn")
		relate(ctx, ontology.Relationship{
			From: rangeID(parent.Key),
			Type: ontology.RelationshipTypeParentOf,
			To:   rangeID(child.Key),
		})
		migrate(ctx)
		Expect(get(ctx, child.Key).Parent).To(HaveValue(Equal(parent.Key)))
		Expect(get(ctx, parent.Key).Parent).To(BeNil())
		Expect(relationships(ctx)).To(BeEmpty())
	})

	It(
		"Should drop a parent relationship to a range that is gone",
		func(ctx SpecContext) {
			child := create(ctx, "Burn")
			relate(ctx, ontology.Relationship{
				From: rangeID(uuid.New()),
				Type: ontology.RelationshipTypeParentOf,
				To:   rangeID(child.Key),
			})
			migrate(ctx)
			Expect(get(ctx, child.Key).Parent).To(BeNil())
			Expect(relationships(ctx)).To(BeEmpty())
		},
	)

	It(
		"Should drop a parent relationship from a range to itself",
		func(ctx SpecContext) {
			r := create(ctx, "Burn")
			relate(ctx, ontology.Relationship{
				From: rangeID(r.Key),
				Type: ontology.RelationshipTypeParentOf,
				To:   rangeID(r.Key),
			})
			migrate(ctx)
			Expect(get(ctx, r.Key).Parent).To(BeNil())
			Expect(relationships(ctx)).To(BeEmpty())
		},
	)

	It(
		"Should keep the parent of a range that is also its own parent",
		func(ctx SpecContext) {
			child, parent := create(ctx, "Burn"), create(ctx, "Test")
			// Relationships migrate in key order, so the self relationship goes first.
			if parent.Key.String() < child.Key.String() {
				child, parent = parent, child
			}
			relate(
				ctx,
				ontology.Relationship{
					From: rangeID(child.Key),
					Type: ontology.RelationshipTypeParentOf,
					To:   rangeID(child.Key),
				},
				ontology.Relationship{
					From: rangeID(parent.Key),
					Type: ontology.RelationshipTypeParentOf,
					To:   rangeID(child.Key),
				},
			)
			migrate(ctx)
			Expect(get(ctx, child.Key).Parent).To(HaveValue(Equal(parent.Key)))
			Expect(relationships(ctx)).To(BeEmpty())
		},
	)

	It("Should clear one parent in a cycle of ranges", func(ctx SpecContext) {
		a, b := create(ctx, "Test"), create(ctx, "Burn")
		relate(
			ctx,
			ontology.Relationship{
				From: rangeID(a.Key),
				Type: ontology.RelationshipTypeParentOf,
				To:   rangeID(b.Key),
			},
			ontology.Relationship{
				From: rangeID(b.Key),
				Type: ontology.RelationshipTypeParentOf,
				To:   rangeID(a.Key),
			},
		)
		migrate(ctx)
		parents := []*uuid.UUID{get(ctx, a.Key).Parent, get(ctx, b.Key).Parent}
		Expect(parents).To(ConsistOf(BeNil(), Not(BeNil())))
		if parents[0] == nil {
			Expect(parents[1]).To(HaveValue(Equal(a.Key)))
		} else {
			Expect(parents[0]).To(HaveValue(Equal(b.Key)))
		}
	})

	It("Should keep every parent in a chain of ranges", func(ctx SpecContext) {
		a, b, c := create(ctx, "Test"), create(ctx, "Burn"), create(ctx, "Hold")
		relate(
			ctx,
			ontology.Relationship{
				From: rangeID(a.Key),
				Type: ontology.RelationshipTypeParentOf,
				To:   rangeID(b.Key),
			},
			ontology.Relationship{
				From: rangeID(b.Key),
				Type: ontology.RelationshipTypeParentOf,
				To:   rangeID(c.Key),
			},
		)
		migrate(ctx)
		Expect(get(ctx, a.Key).Parent).To(BeNil())
		Expect(get(ctx, b.Key).Parent).To(HaveValue(Equal(a.Key)))
		Expect(get(ctx, c.Key).Parent).To(HaveValue(Equal(b.Key)))
	})

	It("Should move labels from relationships into the labels", func(ctx SpecContext) {
		r := create(ctx, "Burn")
		kept, gone := uuid.New(), uuid.New()
		Expect(gorp.WrapWriter[string, ontology.Resource](db).
			Set(ctx, ontology.Resource{ID: labelID(kept)})).To(Succeed())
		relate(
			ctx,
			ontology.Relationship{
				From: rangeID(r.Key),
				Type: ontology.RelationshipTypeLabeledBy,
				To:   labelID(kept),
			},
			ontology.Relationship{
				From: rangeID(r.Key),
				Type: ontology.RelationshipTypeLabeledBy,
				To:   labelID(gone),
			},
		)
		migrate(ctx)
		Expect(get(ctx, r.Key).Labels).To(Equal([]uuid.UUID{kept}))
		Expect(relationships(ctx)).To(BeEmpty())
	})

	It("Should keep a relationship it does not convert", func(ctx SpecContext) {
		r := create(ctx, "Burn")
		group := resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()}
		rel := ontology.Relationship{
			From: group,
			Type: ontology.RelationshipTypeParentOf,
			To:   rangeID(r.Key),
		}
		relate(ctx, rel)
		migrate(ctx)
		Expect(get(ctx, r.Key).Parent).To(BeNil())
		Expect(relationships(ctx)).To(ConsistOf(rel))
	})
})
