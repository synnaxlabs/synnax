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
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/types/v0"
	access "github.com/synnaxlabs/synnax/pkg/service/access/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	run := func(ctx SpecContext) {
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Policy",
			Migrations: []migrate.Migration{v0.Migration},
		})).To(Succeed())
	}

	newLegacy := func(subjects ...ontology.ID) v0.Policy {
		return v0.Policy{
			Key:      uuid.New(),
			Subjects: subjects,
			Objects:  []ontology.ID{{Type: "schematic"}},
			Actions:  []access.Action{"all"},
		}
	}

	It("Should extract legacy policies into the KV mapping and delete them", func(ctx SpecContext) {
		u1 := ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.NewString()}
		u2 := ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.NewString()}
		shared := newLegacy(u1, u2)
		single := newLegacy(u1)
		modern := v0.Policy{
			Key:     uuid.New(),
			Objects: []ontology.ID{{Type: "label"}},
			Actions: []access.Action{"retrieve"},
		}
		Expect(gorp.NewCreate[uuid.UUID, v0.Policy]().
			Entries(&[]v0.Policy{shared, single, modern}).Exec(ctx, db)).To(Succeed())

		run(ctx)

		mappings := MustSucceed(v0.ReadLegacyMappings(ctx, db))
		Expect(mappings).To(HaveLen(2))
		byUser := make(map[string][]v0.Policy, len(mappings))
		for _, m := range mappings {
			byUser[m.UserOntologyID.String()] = m.Policies
		}
		Expect(byUser[u1.String()]).To(ConsistOf(shared, single))
		Expect(byUser[u2.String()]).To(ConsistOf(shared))

		var remaining []v0.Policy
		Expect(gorp.NewRetrieve[uuid.UUID, v0.Policy]().
			Entries(&remaining).Exec(ctx, db)).To(Succeed())
		Expect(remaining).To(ConsistOf(modern))
	})

	It("Should write no mapping when no legacy policies exist", func(ctx SpecContext) {
		modern := v0.Policy{Key: uuid.New(), Actions: []access.Action{"all"}}
		Expect(gorp.NewCreate[uuid.UUID, v0.Policy]().
			Entry(&modern).Exec(ctx, db)).To(Succeed())
		run(ctx)
		Expect(v0.ReadLegacyMappings(ctx, db)).To(BeNil())
	})

	It("Should skip extraction when the pre-gorp migration flag is set", func(ctx SpecContext) {
		Expect(db.Set(
			ctx, []byte("sy_rbac_migration_performed"), []byte{1},
		)).To(Succeed())
		legacy := newLegacy(ontology.ID{Type: ontology.ResourceTypeUser, Key: "u1"})
		Expect(gorp.NewCreate[uuid.UUID, v0.Policy]().
			Entry(&legacy).Exec(ctx, db)).To(Succeed())
		run(ctx)
		Expect(v0.ReadLegacyMappings(ctx, db)).To(BeNil())
		var remaining []v0.Policy
		Expect(gorp.NewRetrieve[uuid.UUID, v0.Policy]().
			Entries(&remaining).Exec(ctx, db)).To(Succeed())
		Expect(remaining).To(ConsistOf(legacy))
	})
})

var _ = Describe("Legacy mappings", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	writeMapping := func(ctx SpecContext, mappings []v0.LegacyUserMapping) {
		raw := MustSucceed(json.Marshal(mappings))
		Expect(db.Set(ctx, []byte(v0.LegacyMappingKVKey), raw)).To(Succeed())
	}

	Describe("ReadLegacyMappings", func() {
		It("Should return nil when no mapping is stored", func(ctx SpecContext) {
			Expect(v0.ReadLegacyMappings(ctx, db)).To(BeNil())
		})
		It("Should read the persisted mapping", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeUser, Key: "u1"}
			writeMapping(ctx, []v0.LegacyUserMapping{{UserOntologyID: id}})
			got := MustSucceed(v0.ReadLegacyMappings(ctx, db))
			Expect(got).To(HaveLen(1))
			Expect(got[0].UserOntologyID).To(Equal(id))
		})
	})

	Describe("DeleteLegacyMappings", func() {
		It("Should remove the persisted mapping", func(ctx SpecContext) {
			writeMapping(ctx, []v0.LegacyUserMapping{{
				UserOntologyID: ontology.ID{Type: ontology.ResourceTypeUser, Key: "u1"},
			}})
			Expect(v0.DeleteLegacyMappings(ctx, db)).To(Succeed())
			Expect(v0.ReadLegacyMappings(ctx, db)).To(BeNil())
		})
	})
})
