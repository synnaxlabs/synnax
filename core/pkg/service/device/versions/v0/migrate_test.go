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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
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

	It("Should create unknown statuses for devices missing them", func(ctx SpecContext) {
		d := v0.Device{
			Key:      "migration-device",
			Rack:     1<<16 | 1,
			Location: "loc",
			Name:     "Migration Test Device",
		}
		Expect(gorp.NewCreate[string, v0.Device]().
			Entry(&d).
			Exec(ctx, db)).To(Succeed())

		runMigration(ctx)

		var restoredStatus status.Status[v0.StatusDetails]
		Expect(status.NewRetrieve[v0.StatusDetails](statusSvc).
			Where(status.MatchKeys[v0.StatusDetails](ontology.ID{Type: ontology.ResourceTypeDevice, Key: d.Key}.String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
		Expect(restoredStatus.Message).To(Equal("Migration Test Device state unknown"))
		Expect(restoredStatus.Details.Device).To(Equal(d.Key))
		Expect(restoredStatus.Details.Rack).To(Equal(d.Rack))
	})

	It("Should not create statuses for devices that already have them", func(ctx SpecContext) {
		d := v0.Device{
			Key:      "existing-status-device",
			Rack:     1<<16 | 1,
			Location: "loc",
			Name:     "Device With Status",
			Make:     "Test Make",
			Model:    "Test Model",
		}
		Expect(gorp.NewCreate[string, v0.Device]().
			Entry(&d).
			Exec(ctx, db)).To(Succeed())
		existing := status.Status[v0.StatusDetails]{
			Key:     ontology.ID{Type: ontology.ResourceTypeDevice, Key: d.Key}.String(),
			Name:    d.Name,
			Variant: status.VariantSuccess,
			Message: "Device With Status is configured",
			Time:    telem.Now(),
			Details: v0.StatusDetails{Rack: d.Rack, Device: d.Key},
		}
		Expect(status.NewWriter[v0.StatusDetails](statusSvc, nil).
			Set(ctx, &existing)).To(Succeed())

		runMigration(ctx)

		var deviceStatus status.Status[v0.StatusDetails]
		Expect(status.NewRetrieve[v0.StatusDetails](statusSvc).
			Where(status.MatchKeys[v0.StatusDetails](ontology.ID{Type: ontology.ResourceTypeDevice, Key: d.Key}.String())).
			Entry(&deviceStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(deviceStatus.Variant).To(Equal(status.VariantSuccess))
		Expect(deviceStatus.Message).To(ContainSubstring("Device With Status"))
	})
})
