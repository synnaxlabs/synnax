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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var (
		db   *gorp.DB
		otg  *ontology.Ontology
		gSvc *group.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		gSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
	})
	runMigration := func(ctx context.Context) {
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Range",
			Migrations: v0.NewMigrations(v0.MigrationConfig{
				Ontology: otg,
				Group:    gSvc,
			}),
		})).To(Succeed())
	}
	It("should migrate subgroups to parent ranges and delete groups", func(ctx SpecContext) {
		// Open a bare Range table with only the codec transition migration. This
		// simulates the state of the DB before the range_groups migration was added.
		bareTable := MustSucceed(gorp.OpenTable(
			ctx, gorp.TableConfig[v0.Key, v0.Range]{DB: db},
		))

		tx := db.OpenTx()

		// Create the "Ranges" group and a subgroup.
		tlg := MustSucceed(gSvc.NewWriter(tx).Create(ctx, "Ranges", ontology.RootID))
		subGroup := MustSucceed(
			gSvc.NewWriter(tx).Create(ctx, "Subgroup", tlg.OntologyID()),
		)

		// Create two ranges and their ontology resources under the subgroup.
		r1 := v0.Range{
			Key:  uuid.New(),
			Name: "Range1",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(10 * telem.Second),
				End:   telem.TimeStamp(20 * telem.Second),
			},
		}
		r2 := v0.Range{
			Key:  uuid.New(),
			Name: "Range2",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(15 * telem.Second),
				End:   telem.TimeStamp(25 * telem.Second),
			},
		}
		Expect(bareTable.NewCreate().Entry(&r1).Exec(ctx, tx)).To(Succeed())
		Expect(bareTable.NewCreate().Entry(&r2).Exec(ctx, tx)).To(Succeed())

		otgWriter := otg.NewWriter(tx)
		Expect(otgWriter.DefineResources(ctx, v0.OntologyID(r1.Key))).To(Succeed())
		Expect(otgWriter.DefineResources(ctx, v0.OntologyID(r2.Key))).To(Succeed())
		Expect(otgWriter.DefineRelationships(
			ctx,
			subGroup.OntologyID(),
			ontology.RelationshipTypeParentOf,
			v0.OntologyID(r1.Key),
		)).To(Succeed())
		Expect(otgWriter.DefineRelationships(
			ctx,
			subGroup.OntologyID(),
			ontology.RelationshipTypeParentOf,
			v0.OntologyID(r2.Key),
		)).To(Succeed())

		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(bareTable.Close()).To(Succeed())

		runMigration(ctx)

		// The "Ranges" group and "Subgroup" should be deleted.
		var g group.Group
		Expect(gSvc.NewRetrieve().Where(group.MatchNames("Ranges")).Entry(&g).Exec(ctx, nil)).
			To(MatchError(query.ErrNotFound))
		Expect(gSvc.NewRetrieve().Where(group.MatchNames("Subgroup")).Entry(&g).Exec(ctx, nil)).
			To(MatchError(query.ErrNotFound))

		// There should be a new parent range named "Subgroup" whose time range
		// is the union of r1 and r2.
		var parentRange v0.Range
		Expect(gorp.NewRetrieve[v0.Key, v0.Range]().
			Where(gorp.Match(
				func(_ gorp.Context, r *v0.Range) (bool, error) {
					return r.Name == "Subgroup", nil
				},
			)).
			Entry(&parentRange).Exec(ctx, db)).To(Succeed())
		Expect(parentRange.TimeRange).To(Equal(telem.TimeRange{
			Start: telem.TimeStamp(10 * telem.Second),
			End:   telem.TimeStamp(25 * telem.Second),
		}))

		// The parent range should have r1 and r2 as children in the ontology.
		var children []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(v0.OntologyID(parentRange.Key)).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypeRange).
			ExcludeFieldData(true).
			Entries(&children).
			Exec(ctx, nil)).To(Succeed())
		var childIDs []ontology.ID
		for _, c := range children {
			childIDs = append(childIDs, c.ID)
		}
		Expect(childIDs).To(ConsistOf(
			v0.OntologyID(r1.Key),
			v0.OntologyID(r2.Key),
		))
	})
})
