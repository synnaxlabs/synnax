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
	"encoding/json"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

const labeledBy ontology.RelationshipType = "labeled_by"

var _ = Describe("MigrateKey", func() {
	DescribeTable("Should give a status owned by its key a UUIDv5 key and an owner",
		func(owner resource.ID) {
			key, got := v2.MigrateKey(owner.String())
			Expect(uuid.Parse(key)).Error().ToNot(HaveOccurred())
			Expect(got).To(HaveValue(Equal(owner)))
		},
		Entry("rack", resource.ID{Type: resource.TypeRack, Key: "3"}),
		Entry("task", resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}),
		Entry("device", resource.ID{Type: resource.TypeDevice, Key: "dev-1"}),
		Entry("channel", resource.ID{Type: resource.TypeChannel, Key: "5"}),
	)

	It("Should derive the same key from the same old key", func() {
		a, _ := v2.MigrateKey("rack:3")
		b, _ := v2.MigrateKey("rack:3")
		c, _ := v2.MigrateKey("rack:4")
		Expect(a).To(Equal(b))
		Expect(a).ToNot(Equal(c))
	})

	DescribeTable("Should keep a key that no resource owns",
		func(key string) {
			got, owner := v2.MigrateKey(key)
			Expect(got).To(Equal(key))
			Expect(owner).To(BeNil())
		},
		Entry("a custom key", "pressure-alarm"),
		Entry("a UUID", uuid.New().String()),
		Entry("a resource that cannot own a status", "label:"+uuid.New().String()),
	)
})

var _ = Describe("Migration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	write := func(ctx SpecContext, statuses ...v1.Status[json.RawMessage]) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[string, v1.Status[json.RawMessage]](db).
			Set(ctx, statuses...)).To(Succeed())
	}
	define := func(ctx SpecContext, ids ...resource.ID) {
		GinkgoHelper()
		for _, id := range ids {
			Expect(gorp.WrapWriter[string, ontology.Resource](db).
				Set(ctx, ontology.Resource{ID: id})).To(Succeed())
		}
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
			Namespace:  "Status",
			Migrations: []gorp.Migration{v2.Migration},
		})).To(Succeed())
	}
	get := func(ctx SpecContext, key string) v2.Status[json.RawMessage] {
		GinkgoHelper()
		var s v2.Status[json.RawMessage]
		Expect(gorp.NewRetrieve[string, v2.Status[json.RawMessage]]().
			Where(gorp.MatchKeys[string, v2.Status[json.RawMessage]](key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}
	missing := func(ctx SpecContext, key string) {
		GinkgoHelper()
		Expect(gorp.NewRetrieve[string, v2.Status[json.RawMessage]]().
			Where(gorp.MatchKeys[string, v2.Status[json.RawMessage]](key)).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	}
	stored := func(name, key, details string) v1.Status[json.RawMessage] {
		return v1.Status[json.RawMessage]{
			Key:     key,
			Name:    name,
			Variant: v1.VariantSuccess,
			Time:    telem.Now(),
			Details: json.RawMessage(details),
		}
	}
	statusID := func(key string) resource.ID {
		return resource.ID{Type: resource.TypeStatus, Key: key}
	}

	It("Should move an owned status to its new key and owner", func(ctx SpecContext) {
		owner := resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}
		define(ctx, owner)
		write(ctx, stored("Read", owner.String(), `{"running":true}`))
		migrate(ctx)
		key, _ := v2.MigrateKey(owner.String())
		got := get(ctx, key)
		Expect(got.Owner).To(HaveValue(Equal(owner)))
		Expect(got.Name).To(Equal("Read"))
		missing(ctx, owner.String())
	})

	It("Should keep a status that no resource owns", func(ctx SpecContext) {
		write(ctx, stored("Alarm", "pressure-alarm", `null`))
		migrate(ctx)
		got := get(ctx, "pressure-alarm")
		Expect(got.Owner).To(BeNil())
	})

	It("Should remove the owner's key from the details", func(ctx SpecContext) {
		owner := resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}
		define(ctx, owner)
		details := `{"task":"` + owner.Key + `","rack":3,"running":true}`
		write(ctx, stored("Read", owner.String(), details))
		migrate(ctx)
		key, _ := v2.MigrateKey(owner.String())
		Expect(get(ctx, key).Details).To(MatchJSON(`{"rack":3,"running":true}`))
	})

	It("Should move labels from relationships into the labels", func(ctx SpecContext) {
		kept, gone := uuid.New(), uuid.New()
		keptID := resource.ID{Type: resource.TypeLabel, Key: kept.String()}
		goneID := resource.ID{Type: resource.TypeLabel, Key: gone.String()}
		define(ctx, keptID)
		write(ctx, stored("Alarm", "alarm", `null`))
		relate(ctx,
			ontology.Relationship{From: statusID("alarm"), Type: labeledBy, To: keptID},
			ontology.Relationship{From: statusID("alarm"), Type: labeledBy, To: goneID},
		)
		migrate(ctx)
		Expect(get(ctx, "alarm").Labels).To(Equal([]uuid.UUID{kept}))
		var rels []ontology.Relationship
		Expect(gorp.NewRetrieve[string, ontology.Relationship]().
			Entries(&rels).
			Exec(ctx, db)).To(Succeed())
		Expect(rels).To(BeEmpty())
	})

	It("Should delete a status whose owner is gone", func(ctx SpecContext) {
		owner := resource.ID{Type: resource.TypeRack, Key: "7"}
		parent := resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()}
		define(ctx, statusID(owner.String()))
		write(ctx, stored("Rack", owner.String(), `null`))
		relate(ctx, ontology.Relationship{
			From: parent,
			Type: ontology.RelationshipTypeParentOf,
			To:   statusID(owner.String()),
		})
		migrate(ctx)
		missing(ctx, owner.String())
		key, _ := v2.MigrateKey(owner.String())
		missing(ctx, key)
		Expect(gorp.NewRetrieve[string, ontology.Resource]().
			Where(gorp.MatchKeys[string, ontology.Resource](
				statusID(owner.String()).String(),
			)).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		var rels []ontology.Relationship
		Expect(gorp.NewRetrieve[string, ontology.Relationship]().
			Entries(&rels).
			Exec(ctx, db)).To(Succeed())
		Expect(rels).To(BeEmpty())
	})

	It("Should move the ontology resource and relationships of a moved status",
		func(ctx SpecContext) {
			owner := resource.ID{Type: resource.TypeDevice, Key: "dev-1"}
			parent := resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()}
			oldID := statusID(owner.String())
			define(ctx, owner, oldID)
			write(ctx, stored("Device", owner.String(), `{"device":"dev-1"}`))
			relate(ctx, ontology.Relationship{
				From: parent,
				Type: ontology.RelationshipTypeParentOf,
				To:   oldID,
			})
			migrate(ctx)
			key, _ := v2.MigrateKey(owner.String())
			newID := statusID(key)
			var res ontology.Resource
			Expect(gorp.NewRetrieve[string, ontology.Resource]().
				Where(gorp.MatchKeys[string, ontology.Resource](newID.String())).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.ID).To(Equal(newID))
			Expect(gorp.NewRetrieve[string, ontology.Resource]().
				Where(gorp.MatchKeys[string, ontology.Resource](oldID.String())).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			var rels []ontology.Relationship
			Expect(gorp.NewRetrieve[string, ontology.Relationship]().
				Entries(&rels).
				Exec(ctx, db)).To(Succeed())
			Expect(rels).To(ConsistOf(ontology.Relationship{
				From: parent,
				Type: ontology.RelationshipTypeParentOf,
				To:   newID,
			}))
		},
	)
})
