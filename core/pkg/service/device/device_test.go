// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Device", func() {
	var (
		ctx      context.Context
		otg      *ontology.Ontology
		groupSvc *group.Service
		rackSvc  *rack.Service
		svc      *device.Service
		tx       gorp.Tx
		w        device.Writer
	)
	BeforeEach(func() {
		ctx := context.Background()
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		groupSvc = MustSucceed(
			group.OpenService(ctx, group.Config{DB: db, Ontology: otg}),
		)
		label := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
		}))
		stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    label,
		}))
		rackSvc = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
		}))
		svc = MustSucceed(device.OpenService(ctx, device.Config{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Status:   stat,
		}))
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
		Expect(svc.Close()).To(Succeed())
		Expect(rackSvc.Close()).To(Succeed())
		Expect(groupSvc.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should correctly create a device", func() {
			d := device.Device{
				Key:      "device1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev1",
				Name:     "Dog",
			}
			Expect(w.Create(ctx, d)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It("Should correctly create an ontology resource for the device", func() {
			d := device.Device{
				Key:      "device2",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev2",
				Name:     "Cat",
			}
			Expect(w.Create(ctx, d)).To(Succeed())
			var res ontology.Resource
			Expect(
				otg.NewRetrieve().WhereIDs(d.OntologyID()).Entry(&res).Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))
		})
		It("Should correctly create an ontology relationship between the device and the rack", func() {
			d := device.Device{
				Key:      "device3",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev3",
				Name:     "Bird",
			}
			Expect(w.Create(ctx, d)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rackSvc.EmbeddedKey.OntologyID()).
				TraverseTo(ontology.Children).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))
		})
		It("Should not recreate the device in the ontology if it already exists", func() {
			d := device.Device{
				Key:      "device3",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev3",
				Name:     "Bird",
			}
			Expect(w.Create(ctx, d)).To(Succeed())
			Expect(otg.NewWriter(tx).DeleteRelationship(
				ctx,
				rackSvc.EmbeddedKey.OntologyID(),
				ontology.ParentOf,
				d.OntologyID(),
			)).To(Succeed())
			Expect(w.Create(ctx, d)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rackSvc.EmbeddedKey.OntologyID()).
				TraverseTo(ontology.Children).
				Entry(&res).
				Exec(ctx, tx),
			).To(MatchError(query.NotFound))
		})
		It("Should redefine ontology relationships if a device has moved racks", func() {
			rw := rackSvc.NewWriter(tx)
			rack1 := rack.Rack{Name: "Rack 1"}
			Expect(rw.Create(ctx, &rack1)).To(Succeed())
			// Step 1 - create new device on embedded rack
			d := device.Device{
				Key:      "moving-device",
				Rack:     rack1.Key,
				Location: "original-loc",
				Name:     "Mover",
			}
			Expect(w.Create(ctx, d)).To(Succeed())

			// Step 2 - create a new rack
			rack2 := rack.Rack{Name: "Rack 2"}
			Expect(rw.Create(ctx, &rack2)).To(Succeed())

			// Step 3 - re-create device with different rack key
			d.Rack = rack2.Key
			Expect(w.Create(ctx, d)).To(Succeed())

			// Step 4 - assert that device is a child of new rack
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rack2.Key.OntologyID()).
				TraverseTo(ontology.Children).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))

			// Step 5 - verify device is no longer a child of the original rack
			var nRes ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rack1.Key.OntologyID()).
				TraverseTo(ontology.Children).
				Entry(&nRes).
				Exec(ctx, tx),
			).To(MatchError(query.NotFound))
		})
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a device", func() {
			d := device.Device{
				Key:      "device4",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev4", Name: "Fish",
			}
			Expect(w.Create(ctx, d)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It("Should retrieve devices by their model", func() {
			d1 := device.Device{
				Key:      "device5",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev5",
				Name:     "Fish",
				Model:    "A",
			}
			d2a := device.Device{
				Key:      "device6",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev6",
				Name:     "Fish",
				Model:    "B",
			}
			d2b := device.Device{
				Key:      "device7",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev7",
				Name:     "Fish",
				Model:    "B",
			}
			Expect(w.Create(ctx, d1)).To(Succeed())
			Expect(w.Create(ctx, d2a)).To(Succeed())
			Expect(w.Create(ctx, d2b)).To(Succeed())
			var res []device.Device
			Expect(svc.NewRetrieve().WhereModels("B").Entries(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res).To(ConsistOf(d2a, d2b))
		})
		It("Should retrieve devices by their make", func() {
			d1 := device.Device{
				Key:      "device8",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev8",
				Name:     "Fish",
				Make:     "A",
			}
			d2a := device.Device{
				Key:      "device9",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev9",
				Name:     "Fish",
				Make:     "B",
			}
			d2b := device.Device{
				Key:      "device10",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev10",
				Name:     "Fish",
				Make:     "B",
			}
			Expect(w.Create(ctx, d1)).To(Succeed())
			Expect(w.Create(ctx, d2a)).To(Succeed())
			Expect(w.Create(ctx, d2b)).To(Succeed())
			var res []device.Device
			Expect(svc.NewRetrieve().WhereMakes("B").Entries(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res).To(ConsistOf(d2a, d2b))
		})
		It("Should retrieve devices by their location", func() {
			d1 := device.Device{
				Key:      "device11",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev11",
				Name:     "Fish",
			}
			d2a := device.Device{
				Key:      "device12",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev12",
				Name:     "Fish",
			}
			d2b := device.Device{
				Key:      "device13",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev13",
				Name:     "Fish",
			}
			Expect(w.Create(ctx, d1)).To(Succeed())
			Expect(w.Create(ctx, d2a)).To(Succeed())
			Expect(w.Create(ctx, d2b)).To(Succeed())
			var res []device.Device
			Expect(
				svc.NewRetrieve().WhereLocations("dev12").Entries(&res).Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(ConsistOf(d2a))
		})
	})
	Describe("Delete", func() {
		It("Should correctly delete a device", func() {
			d := device.Device{
				Key:      "device14",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev14",
				Name:     "Fish",
			}
			Expect(w.Create(ctx, d)).To(Succeed())
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).
				To(MatchError(query.NotFound))
		})
		It("Should correctly delete an ontology resource for the device", func() {
			d := device.Device{
				Key:      "device15",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev15",
				Name:     "Fish",
			}
			Expect(w.Create(ctx, d)).To(Succeed())
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			var res ontology.Resource
			Expect(
				otg.NewRetrieve().WhereIDs(d.OntologyID()).Entry(&res).Exec(ctx, tx),
			).To(MatchError(query.NotFound))
		})
	})
})
