// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateArc", func() {
	It("Should lift a v3 Arc as an automation", func(ctx SpecContext) {
		key := uuid.New()
		migrated := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
			Key:  key,
			Name: "legacy",
			Mode: v3.ModeText,
		}))
		Expect(migrated.Key).To(Equal(key))
		Expect(migrated.Name).To(Equal("legacy"))
		Expect(migrated.Mode).To(Equal(v4.ModeText))
		Expect(migrated.Type).To(Equal(v4.TypeAutomation))
	})
})

var _ = Describe("NewGroupMigration", func() {
	It("Should parent every Arc without a group under Arcs", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		arcs := MustSucceed(groups.CreateOrRetrieve(ctx, "Arcs", ontology.RootID))
		table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Arc]{
			DB: db,
		}))
		tx := db.OpenTx()
		elsewhere := MustSucceed(groups.NewWriter(tx).Create(
			ctx, "elsewhere", ontology.RootID,
		))
		orphan := v4.Arc{Key: uuid.New(), Name: "orphan", Type: v4.TypeAutomation}
		grouped := v4.Arc{Key: uuid.New(), Name: "grouped", Type: v4.TypeAutomation}
		Expect(table.NewCreate().Entry(&orphan).Exec(ctx, tx)).To(Succeed())
		Expect(table.NewCreate().Entry(&grouped).Exec(ctx, tx)).To(Succeed())
		w := otg.NewWriter(tx)
		Expect(w.DefineResources(
			ctx, orphan.OntologyID(), grouped.OntologyID(),
		)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx,
			elsewhere.OntologyID(),
			ontology.RelationshipTypeParentOf,
			grouped.OntologyID(),
		)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(table.Close()).To(Succeed())

		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Arc",
			Migrations: []gorp.Migration{
				v4.NewGroupMigration(v4.MigrationConfig{Ontology: otg, Group: arcs}),
			},
		})).To(Succeed())

		parentsOf := func(id ontology.ID) []ontology.ID {
			GinkgoHelper()
			var parents []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(id).
				TraverseTo(ontology.ParentsTraverser).
				ExcludeFieldData(true).
				Entries(&parents).
				Exec(ctx, nil)).To(Succeed())
			ids := make([]ontology.ID, 0, len(parents))
			for _, p := range parents {
				ids = append(ids, p.ID)
			}
			return ids
		}
		Expect(parentsOf(orphan.OntologyID())).To(ConsistOf(arcs.OntologyID()))
		Expect(parentsOf(grouped.OntologyID())).To(ConsistOf(elsewhere.OntologyID()))
	})
})
