// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", func() {
	var (
		db        *gorp.DB
		svc       *ranger.Service
		otg       *ontology.Ontology
		gSvc      *group.Service
		searchIdx *search.Index
		lab       *label.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx = MustOpen(search.Open())
		gSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		lab = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    gSvc,
			Search:   searchIdx,
		}))
	})
	It("should migrate subgroups to parent ranges and delete groups", func(ctx SpecContext) {
		// Open a bare Range table with only the codec transition migration.
		// This simulates the state of the DB before the range_groups migration
		// was added.
		bareTable := MustSucceed(gorp.OpenTable[uuid.UUID, ranger.Range](
			ctx, gorp.TableConfig[ranger.Range]{DB: db},
		))

		tx := db.OpenTx()

		// Create the "Ranges" group and a subgroup.
		tlg := MustSucceed(gSvc.NewWriter(tx).Create(ctx, "Ranges", ontology.RootID))
		subGroup := MustSucceed(
			gSvc.NewWriter(tx).Create(ctx, "Subgroup", tlg.OntologyID()),
		)

		// Create two ranges and their ontology resources under the subgroup.
		r1 := ranger.Range{
			Key:  uuid.New(),
			Name: "Range1",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(10 * telem.Second),
				End:   telem.TimeStamp(20 * telem.Second),
			},
		}
		r2 := ranger.Range{
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
		Expect(otgWriter.DefineResource(ctx, ranger.OntologyID(r1.Key))).To(Succeed())
		Expect(otgWriter.DefineResource(ctx, ranger.OntologyID(r2.Key))).To(Succeed())
		Expect(otgWriter.DefineRelationship(
			ctx,
			subGroup.OntologyID(),
			ontology.RelationshipTypeParentOf,
			ranger.OntologyID(r1.Key),
		)).To(Succeed())
		Expect(otgWriter.DefineRelationship(
			ctx,
			subGroup.OntologyID(),
			ontology.RelationshipTypeParentOf,
			ranger.OntologyID(r2.Key),
		)).To(Succeed())

		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(bareTable.Close()).To(Succeed())

		svc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:             db,
			Ontology:       otg,
			Group:          gSvc,
			Label:          lab,
			ForceMigration: new(true),
			Search:         searchIdx,
		}))

		// The "Ranges" group and "Subgroup" should be deleted.
		var g group.Group
		Expect(gSvc.NewRetrieve().WhereNames("Ranges").Entry(&g).Exec(ctx, nil)).
			To(MatchError(query.ErrNotFound))
		Expect(gSvc.NewRetrieve().WhereNames("Subgroup").Entry(&g).Exec(ctx, nil)).
			To(MatchError(query.ErrNotFound))

		// There should be a new parent range named "Subgroup" whose time range
		// is the union of r1 and r2.
		var parentRange ranger.Range
		Expect(svc.NewRetrieve().WhereNames("Subgroup").
			Entry(&parentRange).Exec(ctx, nil)).To(Succeed())
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
			Entries(&children).
			Exec(ctx, nil)).To(Succeed())
		var childNames []string
		for _, c := range children {
			childNames = append(childNames, c.Name)
		}
		Expect(childNames).To(ConsistOf("Range1", "Range2"))
	})
})
