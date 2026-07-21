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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	devicev0 "github.com/synnaxlabs/synnax/pkg/service/device/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	rackv0 "github.com/synnaxlabs/synnax/pkg/service/rack/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	It("Should create unknown statuses for devices missing them", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
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
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			HostProvider: mock.NewStaticHostProvider(1),
			Status:       stat,
			Search:       searchIdx,
		}))

		d := devicev0.Device{
			Key:      "migration-device",
			Rack:     rackv0.Key(rackSvc.EmbeddedKey),
			Location: "loc",
			Name:     "Migration Test Device",
		}
		Expect(gorp.NewCreate[string, devicev0.Device]().
			Entry(&d).
			Exec(ctx, db)).To(Succeed())

		MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Status:   stat,
			Rack:     rackSvc,
			Search:   searchIdx,
		}))

		var restoredStatus device.Status
		Expect(status.NewRetrieve[device.StatusDetails](stat).
			Where(status.MatchKeys[device.StatusDetails](device.OntologyID(d.Key).String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
		Expect(restoredStatus.Message).To(Equal("Migration Test Device state unknown"))
		Expect(restoredStatus.Details.Device).To(Equal(d.Key))
		Expect(restoredStatus.Details.Rack).To(Equal(rackSvc.EmbeddedKey))
	})

	It("Should not create statuses for devices that already have them", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
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
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			HostProvider: mock.NewStaticHostProvider(1),
			Status:       stat,
			Search:       searchIdx,
		}))
		svc := MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Status:   stat,
			Rack:     rackSvc,
			Search:   searchIdx,
		}))

		d := device.Device{
			Key:      "existing-status-device",
			Rack:     rackSvc.EmbeddedKey,
			Location: "loc",
			Name:     "Device With Status",
			Make:     "Test Make",
			Model:    "Test Model",
		}
		Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())

		var deviceStatus device.Status
		Expect(status.NewRetrieve[device.StatusDetails](stat).
			Where(status.MatchKeys[device.StatusDetails](device.OntologyID(d.Key).String())).
			Entry(&deviceStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(deviceStatus.Variant).To(Equal(status.VariantWarning))
		Expect(deviceStatus.Message).To(ContainSubstring("Device With Status"))
	})
})
