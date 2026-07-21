// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/types"
	rangerv0 "github.com/synnaxlabs/synnax/pkg/service/ranger/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/types/v1"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", func() {
	var (
		db        *gorp.DB
		otg       *ontology.Ontology
		gSvc      *group.Service
		searchIdx *search.Index
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx = MustOpen(search.OpenIndex())
		gSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
	})
	runMigrations := func(ctx context.Context) {
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Range",
			Migrations: types.NewMigrations(types.MigrationsConfig{
				Ontology: otg,
				Group:    gSvc,
			}),
		})).To(Succeed())
	}
	It("should migrate subgroups to parent ranges and delete groups", func(ctx SpecContext) {
		// Open a bare Range table with only the codec transition migration.
		// This simulates the state of the DB before the range_groups migration
		// was added.
		bareTable := MustSucceed(gorp.OpenTable(
			ctx, gorp.TableConfig[uuid.UUID, rangerv0.Range]{DB: db},
		))

		tx := db.OpenTx()

		// Create the "Ranges" group and a subgroup.
		tlg := MustSucceed(gSvc.NewWriter(tx).Create(ctx, "Ranges", ontology.RootID))
		subGroup := MustSucceed(
			gSvc.NewWriter(tx).Create(ctx, "Subgroup", tlg.OntologyID()),
		)

		// Create two ranges and their ontology resources under the subgroup.
		r1 := rangerv0.Range{
			Key:  uuid.New(),
			Name: "Range1",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(10 * telem.Second),
				End:   telem.TimeStamp(20 * telem.Second),
			},
		}
		r2 := rangerv0.Range{
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
		Expect(otgWriter.DefineResources(ctx, rangerv0.OntologyID(r1.Key))).To(Succeed())
		Expect(otgWriter.DefineResources(ctx, rangerv0.OntologyID(r2.Key))).To(Succeed())
		Expect(otgWriter.DefineRelationships(
			ctx,
			subGroup.OntologyID(),
			ontology.RelationshipTypeParentOf,
			rangerv0.OntologyID(r1.Key),
		)).To(Succeed())
		Expect(otgWriter.DefineRelationships(
			ctx,
			subGroup.OntologyID(),
			ontology.RelationshipTypeParentOf,
			rangerv0.OntologyID(r2.Key),
		)).To(Succeed())

		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(bareTable.Close()).To(Succeed())

		runMigrations(ctx)

		// The "Ranges" group and "Subgroup" should be deleted.
		var g group.Group
		Expect(gSvc.NewRetrieve().Where(group.MatchNames("Ranges")).Entry(&g).Exec(ctx, nil)).
			To(MatchError(query.ErrNotFound))
		Expect(gSvc.NewRetrieve().Where(group.MatchNames("Subgroup")).Entry(&g).Exec(ctx, nil)).
			To(MatchError(query.ErrNotFound))

		// There should be a new parent range named "Subgroup" whose time range
		// is the union of r1 and r2.
		var parentRange v1.Range
		Expect(gorp.NewRetrieve[uuid.UUID, v1.Range]().
			Where(gorp.Match[uuid.UUID, v1.Range](
				func(_ gorp.Context, r *v1.Range) (bool, error) {
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
			WhereIDs(parentRange.OntologyID()).
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
			rangerv0.OntologyID(r1.Key),
			rangerv0.OntologyID(r2.Key),
		))
	})

	It("should re-encode value-color ranges as nullable pointer color", func(ctx SpecContext) {
		bareTable := MustSucceed(gorp.OpenTable(
			ctx, gorp.TableConfig[uuid.UUID, rangerv0.Range]{DB: db},
		))
		tr := telem.TimeRange{
			Start: telem.TimeStamp(telem.Second),
			End:   telem.TimeStamp(2 * telem.Second),
		}
		colored := rangerv0.Range{
			Key:       uuid.New(),
			Name:      "Colored",
			TimeRange: tr,
			Color:     color.Color{R: 255, G: 128, B: 0, A: 1},
		}
		// A zero value-color denoted "no color" under the old layout, so it must
		// migrate to a nil pointer rather than a pointer to the zero color.
		uncolored := rangerv0.Range{Key: uuid.New(), Name: "Uncolored", TimeRange: tr}

		tx := db.OpenTx()
		Expect(bareTable.NewCreate().Entry(&colored).Exec(ctx, tx)).To(Succeed())
		Expect(bareTable.NewCreate().Entry(&uncolored).Exec(ctx, tx)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(bareTable.Close()).To(Succeed())

		runMigrations(ctx)

		var got v1.Range
		Expect(gorp.NewRetrieve[uuid.UUID, v1.Range]().
			Where(gorp.MatchKeys[uuid.UUID, v1.Range](colored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Color).ToNot(BeNil())
		Expect(*got.Color).To(Equal(color.Color{R: 255, G: 128, B: 0, A: 1}))

		Expect(gorp.NewRetrieve[uuid.UUID, v1.Range]().
			Where(gorp.MatchKeys[uuid.UUID, v1.Range](uncolored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Color).To(BeNil())
	})
})
