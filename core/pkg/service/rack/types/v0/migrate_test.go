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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/rack/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var (
		db   *gorp.DB
		stat *status.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
	})

	runMigration := func(ctx context.Context) {
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Rack",
			Migrations: []migrate.Migration{v0.NewMigration(v0.MigrationConfig{
				HostProvider: mock.NewStaticHostProvider(1),
				Status:       stat,
			})},
		})).To(Succeed())
	}

	retrieveRack := func(ctx context.Context, key v0.Key) v0.Rack {
		var r v0.Rack
		Expect(gorp.NewRetrieve[v0.Key, v0.Rack]().
			Where(gorp.MatchKeys[v0.Key, v0.Rack](key)).
			Entry(&r).
			Exec(ctx, db)).To(Succeed())
		return r
	}

	countRacks := func(ctx context.Context) int {
		return MustSucceed(gorp.NewRetrieve[v0.Key, v0.Rack]().Count(ctx, db))
	}

	It("Should create unknown statuses for racks missing them", func(ctx SpecContext) {
		r := v0.Rack{
			Key:  v0.Key(1<<16 | 50),
			Name: "rack without status",
		}
		Expect(gorp.NewCreate[v0.Key, v0.Rack]().
			Entry(&r).
			Exec(ctx, db)).To(Succeed())

		runMigration(ctx)

		var restoredStatus status.Status[v0.StatusDetails]
		Expect(status.NewRetrieve[v0.StatusDetails](stat).
			Where(status.MatchKeys[v0.StatusDetails](r.Key.OntologyID().String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
		Expect(restoredStatus.Message).To(Equal("Status unknown"))
		Expect(restoredStatus.Details.Rack).To(Equal(r.Key))
	})

	It("Should rename the host's embedded rack", func(ctx SpecContext) {
		legacyEmbeddedRack := v0.Rack{
			Key:  65538,
			Name: "sy_node_1_rack",
		}
		Expect(gorp.NewCreate[v0.Key, v0.Rack]().
			Entry(&legacyEmbeddedRack).
			Exec(ctx, db)).To(Succeed())

		runMigration(ctx)

		embeddedRack := retrieveRack(ctx, v0.Key(65538))
		Expect(embeddedRack.Embedded).To(BeTrue())
		Expect(embeddedRack.Name).To(Equal("Node 1 Embedded Driver"))
		Expect(countRacks(ctx)).To(Equal(1))
	})

	It("Should not match an embedded rack with a mismatched name", func(ctx SpecContext) {
		mismatchedRack := v0.Rack{
			Key:      65538,
			Name:     "Some Other Embedded Rack",
			Embedded: true,
		}
		Expect(gorp.NewCreate[v0.Key, v0.Rack]().
			Entry(&mismatchedRack).
			Exec(ctx, db)).To(Succeed())

		runMigration(ctx)

		got := retrieveRack(ctx, mismatchedRack.Key)
		Expect(got.Embedded).To(BeTrue())
		Expect(got.Name).To(Equal("Some Other Embedded Rack"))
		Expect(countRacks(ctx)).To(Equal(1))
	})

	It("Should leave an already-renamed embedded rack untouched", func(ctx SpecContext) {
		existingRack := v0.Rack{
			Key:      65538,
			Name:     "Node 1 Embedded Driver",
			Embedded: true,
		}
		Expect(gorp.NewCreate[v0.Key, v0.Rack]().
			Entry(&existingRack).
			Exec(ctx, db)).To(Succeed())

		runMigration(ctx)

		embeddedRack := retrieveRack(ctx, existingRack.Key)
		Expect(embeddedRack.Embedded).To(BeTrue())
		Expect(embeddedRack.Name).To(Equal("Node 1 Embedded Driver"))
		Expect(countRacks(ctx)).To(Equal(1))
	})
})
