// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type member struct {
	Key    string
	Parent string
}

func (m member) GorpKey() string { return m.Key }

func (member) SetOptions() []any { return nil }

var _ = Describe("NewFieldMigration", func() {
	var db *gorp.DB
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
	})

	var (
		parent  = resource.ID{Type: resource.TypeGroup, Key: "parent"}
		child   = resource.ID{Type: resource.TypeChannel, Key: "child"}
		missing = resource.ID{Type: resource.TypeChannel, Key: "missing"}
		label   = resource.ID{Type: resource.TypeLabel, Key: "label"}
	)
	migration := v0.NewFieldMigration(v0.FieldConfig[string, member]{
		Name: "member_parent",
		Converts: func(rel v0.Relationship) bool {
			return rel.Type == v0.RelationshipTypeParentOf
		},
		Holder: func(rel v0.Relationship) string { return rel.To.Key },
		Set: func(_ context.Context, _ gorp.Tx, m *member, rel v0.Relationship) (
			bool,
			error,
		) {
			m.Parent = rel.From.Key
			return true, nil
		},
	})

	It(
		"Should move each converted relationship into its holder",
		func(ctx SpecContext) {
			labeled := v0.Relationship{
				From: child,
				Type: v0.RelationshipTypeLabeledBy,
				To:   label,
			}
			Expect(gorp.WrapWriter[string, member](db).
				Set(ctx, member{Key: child.Key})).To(Succeed())
			stored := []v0.Relationship{
				{From: parent, Type: v0.RelationshipTypeParentOf, To: child},
				{From: parent, Type: v0.RelationshipTypeParentOf, To: missing},
				labeled,
			}
			Expect(gorp.WrapWriter[string, v0.Relationship](db).
				Set(ctx, stored...)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "member",
				Migrations: []gorp.Migration{migration},
			})).To(Succeed())
			var m member
			Expect(gorp.NewRetrieve[string, member]().
				Where(gorp.MatchKeys[string, member](child.Key)).
				Entry(&m).
				Exec(ctx, db)).To(Succeed())
			Expect(m.Parent).To(Equal(parent.Key))
			rels := MustSucceed(v0.RetrieveRelationships(
				ctx,
				db,
				func(v0.Relationship) bool { return true },
			))
			Expect(rels).To(ConsistOf(labeled))
		},
	)
})

var _ = Describe("Graph", func() {
	var (
		db             *gorp.DB
		a, b, c        resource.ID
		aToB, bToC     v0.Relationship
		resources      func(ctx context.Context) []v0.Resource
		relationships  func(ctx context.Context) []v0.Relationship
		allRelationsOf = func(v0.Relationship) bool { return true }
	)
	BeforeEach(func(ctx SpecContext) {
		db = gorp.Wrap(memkv.New())
		DeferCleanup(db.Close)
		a = resource.ID{Type: resource.TypeGroup, Key: "a"}
		b = resource.ID{Type: resource.TypeGroup, Key: "b"}
		c = resource.ID{Type: resource.TypeChannel, Key: "1"}
		aToB = v0.Relationship{From: a, Type: v0.RelationshipTypeParentOf, To: b}
		bToC = v0.Relationship{From: b, Type: v0.RelationshipTypeParentOf, To: c}
		Expect(v0.DefineResources(ctx, db, a, b, c)).To(Succeed())
		Expect(gorp.WrapWriter[string, v0.Relationship](db).Set(ctx, aToB, bToC)).
			To(Succeed())
		resources = func(ctx context.Context) []v0.Resource {
			GinkgoHelper()
			var out []v0.Resource
			Expect(gorp.NewRetrieve[string, v0.Resource]().
				Entries(&out).
				Exec(ctx, db)).To(Succeed())
			return out
		}
		relationships = func(ctx context.Context) []v0.Relationship {
			GinkgoHelper()
			return MustSucceed(v0.RetrieveRelationships(ctx, db, allRelationsOf))
		}
	})

	Describe("DefineResources", func() {
		It("Should store a resource for each ID", func(ctx SpecContext) {
			Expect(resources(ctx)).To(ConsistOf(
				v0.Resource{ID: a},
				v0.Resource{ID: b},
				v0.Resource{ID: c},
			))
		})
	})

	Describe("RetrieveRelationships", func() {
		It("Should return the relationships keep accepts", func(ctx SpecContext) {
			Expect(v0.RetrieveRelationships(
				ctx,
				db,
				func(r v0.Relationship) bool { return r.To == c },
			)).To(ConsistOf(bToC))
		})
	})

	Describe("DeleteRelationships", func() {
		It(
			"Should delete the relationships and skip missing ones",
			func(ctx SpecContext) {
				missing := v0.Relationship{
					From: c,
					Type: v0.RelationshipTypeParentOf,
					To:   a,
				}
				Expect(v0.DeleteRelationships(ctx, db, aToB, missing)).To(Succeed())
				Expect(relationships(ctx)).To(ConsistOf(bToC))
			},
		)
	})

	Describe("DeleteResources", func() {
		It(
			"Should delete the resources and every relationship from or to them",
			func(ctx SpecContext) {
				Expect(v0.DeleteResources(ctx, db, b)).To(Succeed())
				Expect(resources(ctx)).To(ConsistOf(
					v0.Resource{ID: a},
					v0.Resource{ID: c},
				))
				Expect(relationships(ctx)).To(BeEmpty())
			},
		)
	})

	Describe("Drop", func() {
		It("Should delete every resource and relationship", func(ctx SpecContext) {
			Expect(v0.Drop(ctx, db)).To(Succeed())
			Expect(resources(ctx)).To(BeEmpty())
			Expect(relationships(ctx)).To(BeEmpty())
		})
	})
})
