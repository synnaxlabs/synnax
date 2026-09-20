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
	v0 "github.com/synnaxlabs/synnax/pkg/service/device/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var (
		db        *gorp.DB
		statusSvc *status.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
	})

	runMigration := func(ctx context.Context) {
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Device",
			Migrations: v0.NewMigrations(v0.MigrationConfig{Status: statusSvc}),
		})).To(Succeed())
	}

	It(
		"Should create unknown statuses for devices missing them",
		func(ctx SpecContext) {
			d := v0.Device{
				Key:      "migration-device",
				Rack:     1<<16 | 1,
				Location: "loc",
				Name:     "Migration Test Device",
			}
			Expect(gorp.NewCreate[v0.Key, v0.Device]().
				Entry(&d).
				Exec(ctx, db)).To(Succeed())

			runMigration(ctx)

			var restoredStatus status.Status[v0.StatusDetails]
			Expect(statusSvc.NewRetrieve[v0.StatusDetails]().
				Where(status.MatchKeys[v0.StatusDetails](d.OntologyID().String())).
				Entry(&restoredStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
			Expect(
				restoredStatus.Message,
			).To(Equal("Migration Test Device state unknown"))
			Expect(restoredStatus.Details.Device).To(Equal(d.Key))
			Expect(restoredStatus.Details.Rack).To(Equal(d.Rack))
		},
	)

	It(
		"Should not create statuses for devices that already have them",
		func(ctx SpecContext) {
			d := v0.Device{
				Key:      "existing-status-device",
				Rack:     1<<16 | 1,
				Location: "loc",
				Name:     "Device With Status",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(gorp.NewCreate[v0.Key, v0.Device]().
				Entry(&d).
				Exec(ctx, db)).To(Succeed())
			existing := status.Status[v0.StatusDetails]{
				Key:     d.OntologyID().String(),
				Name:    d.Name,
				Variant: status.VariantSuccess,
				Message: "Device With Status is configured",
				Time:    telem.Now(),
				Details: v0.StatusDetails{Rack: d.Rack, Device: d.Key},
			}
			Expect(statusSvc.NewWriter(nil).Set(ctx, &existing)).To(Succeed())

			runMigration(ctx)

			var deviceStatus status.Status[v0.StatusDetails]
			Expect(statusSvc.NewRetrieve[v0.StatusDetails]().
				Where(status.MatchKeys[v0.StatusDetails](d.OntologyID().String())).
				Entry(&deviceStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(deviceStatus.Variant).To(Equal(status.VariantSuccess))
			Expect(deviceStatus.Message).To(ContainSubstring("Device With Status"))
		},
	)
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a Device row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			e := v0.Device{Key: "dev-1", Rack: 1, Name: "DAQ"}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Device",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Device]{
				DB:         db,
				Migrations: []migrate.Migration{v0.NormalizeKeys},
			}))
			var res v0.Device
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[v0.Key, v0.Device](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
