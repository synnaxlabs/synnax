// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	devicev0 "github.com/synnaxlabs/synnax/pkg/service/device/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	rackv0 "github.com/synnaxlabs/synnax/pkg/service/rack/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Device", func() {
	var (
		otg      *ontology.Ontology
		groupSvc *group.Service
		rackSvc  *rack.Service
		svc      *device.Service
		stat     *status.Service
		tx       gorp.Tx
		w        device.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.Open())
		groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		label := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    label,
			Search:   searchIdx,
		}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
			Search:       searchIdx,
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Status:   stat,
			Rack:     rackSvc,
			Search:   searchIdx,
		}))
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should correctly create a device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev1",
				Name:     "Dog",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It("Should correctly create an ontology resource for the device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device2",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev2",
				Name:     "Cat",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			var res ontology.Resource
			Expect(
				otg.NewRetrieve().WhereIDs(d.OntologyID()).Entry(&res).Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))
		})
		It("Should create a parent-child relationship between chassis and module", func(ctx SpecContext) {
			chassis := device.Device{
				Key:      "chassis1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "Slot 1",
				Name:     "cDAQ-9178",
				Make:     "NI",
				Model:    "cDAQ-9178",
			}
			Expect(w.Create(ctx, &chassis)).To(Succeed())

			chassisOntologyID := chassis.OntologyID()
			child := device.Device{
				Key:      "module1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "Mod 1",
				Name:     "NI-9205",
				Make:     "NI",
				Model:    "NI-9205",
				Parent:   &chassisOntologyID,
			}
			Expect(w.Create(ctx, &child)).To(Succeed())

			var children []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(chassis.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&children).
				Exec(ctx, tx)).To(Succeed())
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID).To(Equal(child.OntologyID()))
		})
		It("Should correctly create an ontology relationship between the device and the rack", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device3",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev3",
				Name:     "Bird",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rackSvc.EmbeddedKey.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))
		})
		It("Should not recreate the device in the ontology if it already exists", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device3",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev3",
				Name:     "Bird",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(otg.NewWriter(tx).DeleteRelationship(
				ctx,
				rackSvc.EmbeddedKey.OntologyID(),
				ontology.RelationshipTypeParentOf,
				d.OntologyID(),
			)).To(Succeed())
			d2 := device.Device{
				Key:      "device3",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev3",
				Name:     "Bird",
			}
			Expect(w.Create(ctx, &d2)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rackSvc.EmbeddedKey.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})
		It("Should redefine ontology relationships if a device has moved racks", func(ctx SpecContext) {
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
			Expect(w.Create(ctx, &d)).To(Succeed())

			// Step 2 - create a new rack
			rack2 := rack.Rack{Name: "Rack 2"}
			Expect(rw.Create(ctx, &rack2)).To(Succeed())

			// Step 3 - re-create device with different rack key
			d2 := device.Device{
				Key:      "moving-device",
				Rack:     rack2.Key,
				Location: "original-loc",
				Name:     "Mover",
			}
			Expect(w.Create(ctx, &d2)).To(Succeed())

			// Step 4 - assert that device is a child of new rack
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rack2.Key.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d2.OntologyID()))

			// Step 5 - verify device is no longer a child of the original rack
			var nRes ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rack1.Key.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&nRes).
				Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})
		It("Should update the status name when renaming a device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "rename-device",
				Rack:     rackSvc.EmbeddedKey,
				Location: "loc",
				Name:     "Original Name",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())

			d2 := device.Device{
				Key:      "rename-device",
				Rack:     rackSvc.EmbeddedKey,
				Location: "loc",
				Name:     "New Name",
			}
			Expect(w.Create(ctx, &d2)).To(Succeed())

			var deviceStatus device.Status
			Expect(status.NewRetrieve[device.StatusDetails](stat).
				WhereKeys(device.OntologyID(d.Key).String()).
				Entry(&deviceStatus).
				Exec(ctx, tx)).To(Succeed())
			Expect(deviceStatus.Name).To(Equal("New Name"))
			Expect(deviceStatus.Message).To(ContainSubstring("New Name"))
		})

		It("Should use the provided status when creating a device", func(ctx SpecContext) {
			providedStatus := &device.Status{
				Variant:     xstatus.VariantSuccess,
				Time:        telem.Now(),
				Message:     "Device is connected",
				Description: "Custom device description",
			}
			d := device.Device{
				Key:      "device-with-status",
				Rack:     rackSvc.EmbeddedKey,
				Location: "loc-status",
				Name:     "Device with custom status",
				Status:   providedStatus,
			}
			Expect(w.Create(ctx, &d)).To(Succeed())

			var deviceStatus device.Status
			Expect(status.NewRetrieve[device.StatusDetails](stat).
				WhereKeys(device.OntologyID(d.Key).String()).
				Entry(&deviceStatus).
				Exec(ctx, tx)).To(Succeed())
			Expect(deviceStatus.Variant).To(Equal(xstatus.VariantSuccess))
			Expect(deviceStatus.Message).To(Equal("Device is connected"))
			Expect(deviceStatus.Description).To(Equal("Custom device description"))
			// Key should be auto-assigned
			Expect(deviceStatus.Key).To(Equal(device.OntologyID(d.Key).String()))
			// Name should be auto-filled
			Expect(deviceStatus.Name).To(Equal(d.Name))
			// Details should be auto-filled
			Expect(deviceStatus.Details.Device).To(Equal(d.Key))
			Expect(deviceStatus.Details.Rack).To(Equal(d.Rack))
		})

		It("Should return a validation error if provided status has empty variant", func(ctx SpecContext) {
			providedStatus := &device.Status{
				Message: "Status with no variant",
				Time:    telem.Now(),
			}
			d := device.Device{
				Key:      "device-invalid-status",
				Rack:     rackSvc.EmbeddedKey,
				Location: "loc-invalid",
				Name:     "Device with invalid status",
				Status:   providedStatus,
			}
			Expect(w.Create(ctx, &d)).Error().To(MatchError(ContainSubstring("variant")))
		})
		It("Should populate Status and Parent on the device after Create", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device-status-parent",
				Rack:     rackSvc.EmbeddedKey,
				Location: "loc",
				Name:     "Populated Device",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(d.Status).ToNot(BeNil())
			Expect(d.Status.Variant).To(Equal(xstatus.VariantWarning))
			Expect(d.Status.Name).To(Equal("Populated Device"))
			Expect(d.Parent).ToNot(BeNil())
			Expect(*d.Parent).To(Equal(rackSvc.EmbeddedKey.OntologyID()))
		})
		It("Should populate Status and Parent with explicit parent after Create", func(ctx SpecContext) {
			chassis := device.Device{
				Key:      "pop-chassis",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-0",
				Name:     "Pop Chassis",
			}
			Expect(w.Create(ctx, &chassis)).To(Succeed())

			chassisID := device.OntologyID("pop-chassis")
			module := device.Device{
				Key:      "pop-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Pop Module",
				Parent:   &chassisID,
			}
			Expect(w.Create(ctx, &module)).To(Succeed())
			Expect(module.Status).ToNot(BeNil())
			Expect(module.Parent).ToNot(BeNil())
			Expect(*module.Parent).To(Equal(chassisID))
		})
	})
	Describe("Parent Ontology Relationship", func() {
		It("Should parent a device to another device via ontology ID", func(ctx SpecContext) {
			chassis := device.Device{
				Key:      "pd-chassis-1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-0",
				Name:     "Chassis 1",
			}
			Expect(w.Create(ctx, &chassis)).To(Succeed())

			module := device.Device{
				Key:      "pd-module-1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Module 1",
			}
			chassisParent := device.OntologyID("pd-chassis-1")
			module.Parent = &chassisParent
			Expect(w.Create(ctx, &module)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(device.OntologyID("pd-chassis-1")).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(module.OntologyID()))
		})

		It("Should default to rack when no parent is provided", func(ctx SpecContext) {
			d := device.Device{
				Key:      "pd-no-parent",
				Rack:     rackSvc.EmbeddedKey,
				Location: "loc",
				Name:     "Standalone",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rackSvc.EmbeddedKey.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))
		})

		It("Should re-parent a device when parent changes", func(ctx SpecContext) {
			chassisA := device.Device{
				Key:      "pd-chassis-a",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-a",
				Name:     "Chassis A",
			}
			chassisB := device.Device{
				Key:      "pd-chassis-b",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-b",
				Name:     "Chassis B",
			}
			Expect(w.Create(ctx, &chassisA)).To(Succeed())
			Expect(w.Create(ctx, &chassisB)).To(Succeed())

			chassisAParent := device.OntologyID("pd-chassis-a")
			module := device.Device{
				Key:      "pd-moving-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Moving Module",
				Parent:   &chassisAParent,
			}
			Expect(w.Create(ctx, &module)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(device.OntologyID("pd-chassis-a")).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(module.OntologyID()))

			// Re-parent to chassis B
			chassisBParent := device.OntologyID("pd-chassis-b")
			module.Parent = &chassisBParent
			Expect(w.Create(ctx, &module)).To(Succeed())

			Expect(otg.NewRetrieve().
				WhereIDs(device.OntologyID("pd-chassis-b")).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(module.OntologyID()))

			var nRes ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(device.OntologyID("pd-chassis-a")).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&nRes).
				Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})

		It("Should converge when parent is created after the child", func(ctx SpecContext) {
			module := device.Device{
				Key:      "pd-converge-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Converge Module",
			}
			Expect(w.Create(ctx, &module)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(rackSvc.EmbeddedKey.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(module.OntologyID()))

			chassis := device.Device{
				Key:      "pd-converge-chassis",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-0",
				Name:     "Converge Chassis",
			}
			Expect(w.Create(ctx, &chassis)).To(Succeed())

			// Re-send module with parent set (scanner would do this)
			convergeParent := device.OntologyID("pd-converge-chassis")
			module.Parent = &convergeParent
			Expect(w.Create(ctx, &module)).To(Succeed())

			var childRes ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(device.OntologyID("pd-converge-chassis")).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&childRes).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(childRes.ID).To(Equal(module.OntologyID()))
		})

		It("Should not skip update when an explicit parent is provided", func(ctx SpecContext) {
			chassis := device.Device{
				Key:      "pd-skip-chassis",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-0",
				Name:     "Skip Chassis",
			}
			Expect(w.Create(ctx, &chassis)).To(Succeed())

			d := device.Device{
				Key:      "pd-skip-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Skip Module",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())

			// Update: same rack, but now with an explicit parent
			skipParent := device.OntologyID("pd-skip-chassis")
			d.Parent = &skipParent
			Expect(w.Create(ctx, &d)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(device.OntologyID("pd-skip-chassis")).
				TraverseTo(ontology.ChildrenTraverser).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res.ID).To(Equal(d.OntologyID()))
		})
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device4",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev4", Name: "Fish",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It("Should retrieve devices by their model", func(ctx SpecContext) {
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
			Expect(w.Create(ctx, &d1)).To(Succeed())
			Expect(w.Create(ctx, &d2a)).To(Succeed())
			Expect(w.Create(ctx, &d2b)).To(Succeed())
			// Clear fields populated by Create that aren't stored in gorp.
			d2a.Status, d2a.Parent = nil, nil
			d2b.Status, d2b.Parent = nil, nil
			var res []device.Device
			Expect(svc.NewRetrieve().WhereModels("B").Entries(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res).To(ConsistOf(d2a, d2b))
		})
		It("Should retrieve devices by their make", func(ctx SpecContext) {
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
			Expect(w.Create(ctx, &d1)).To(Succeed())
			Expect(w.Create(ctx, &d2a)).To(Succeed())
			Expect(w.Create(ctx, &d2b)).To(Succeed())
			d2a.Status, d2a.Parent = nil, nil
			d2b.Status, d2b.Parent = nil, nil
			var res []device.Device
			Expect(svc.NewRetrieve().WhereMakes("B").Entries(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res).To(ConsistOf(d2a, d2b))
		})
		It("Should retrieve devices by their location", func(ctx SpecContext) {
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
			Expect(w.Create(ctx, &d1)).To(Succeed())
			Expect(w.Create(ctx, &d2a)).To(Succeed())
			Expect(w.Create(ctx, &d2b)).To(Succeed())
			d2a.Status, d2a.Parent = nil, nil
			var res []device.Device
			Expect(
				svc.NewRetrieve().WhereLocations("dev12").Entries(&res).Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(ConsistOf(d2a))
		})
	})
	Describe("Delete", func() {
		It("Should correctly delete a device and its associated status", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device14",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev14",
				Name:     "Fish",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			var res device.Device
			Expect(svc.NewRetrieve().WhereKeys(d.Key).Entry(&res).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			var deletedStatus device.Status
			Expect(status.NewRetrieve[device.StatusDetails](stat).
				WhereKeys(device.OntologyID(d.Key).String()).
				Entry(&deletedStatus).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
		It("Should correctly delete an ontology resource for the device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device15",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev15",
				Name:     "Fish",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			var res ontology.Resource
			Expect(
				otg.NewRetrieve().WhereIDs(d.OntologyID()).Entry(&res).Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})
	})
	Describe("Suspect Rack", func() {
		It("Should propagate rack warning status to devices on that rack", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.Open())
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
				DB:                  db,
				Ontology:            otg,
				Group:               groupSvc,
				HostProvider:        mock.StaticHostKeyProvider(1),
				Status:              stat,
				HealthCheckInterval: 10 * telem.Millisecond,
				Search:              searchIdx,
			}))
			svc := MustOpen(device.OpenService(ctx, device.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				Status:   stat,
				Rack:     rackSvc,
				Search:   searchIdx,
			}))

			r := rack.Rack{Name: "suspect rack"}
			Expect(rackSvc.NewWriter(nil).Create(ctx, &r)).To(Succeed())

			d := device.Device{
				Key:      "suspect-device",
				Rack:     r.Key,
				Location: "loc1",
				Name:     "Test Device",
			}
			Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())

			Eventually(func(g Gomega) {
				var deviceStatus device.Status
				g.Expect(status.NewRetrieve[device.StatusDetails](stat).
					WhereKeys(device.OntologyID(d.Key).String()).
					Entry(&deviceStatus).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(deviceStatus.Variant).To(Equal(xstatus.VariantWarning))
				g.Expect(deviceStatus.Message).To(ContainSubstring("not running"))
				g.Expect(deviceStatus.Details.Device).To(Equal(d.Key))
				g.Expect(deviceStatus.Details.Rack).To(Equal(r.Key))
			}).Should(Succeed())
		})
	})
	Describe("Migration", func() {
		It("Should create unknown statuses for devices missing them", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.Open())
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
				HostProvider: mock.StaticHostKeyProvider(1),
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
				WhereKeys(device.OntologyID(d.Key).String()).
				Entry(&restoredStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(restoredStatus.Variant).To(Equal(xstatus.VariantWarning))
			Expect(restoredStatus.Message).To(Equal("Migration Test Device state unknown"))
			Expect(restoredStatus.Details.Device).To(Equal(d.Key))
			Expect(restoredStatus.Details.Rack).To(Equal(rackSvc.EmbeddedKey))
		})

		It("Should not create statuses for devices that already have them", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.Open())
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
				HostProvider: mock.StaticHostKeyProvider(1),
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
			}
			Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())

			var deviceStatus device.Status
			Expect(status.NewRetrieve[device.StatusDetails](stat).
				WhereKeys(device.OntologyID(d.Key).String()).
				Entry(&deviceStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(deviceStatus.Variant).To(Equal(xstatus.VariantWarning))
			Expect(deviceStatus.Message).To(ContainSubstring("Device With Status"))
		})
	})
})
