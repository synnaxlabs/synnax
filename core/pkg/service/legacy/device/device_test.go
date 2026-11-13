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
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Device", func() {
	var (
		groupSvc *group.Service
		rackSvc  *rack.Service
		svc      *device.Service
		stat     *status.Service
		tx       gorp.Tx
		w        device.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
		}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
			Rack:   rackSvc,
		}))
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	})
	Describe("Create", func() {
		It("Should correctly create a device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev1",
				Name:     "Dog",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			var res device.Device
			Expect(
				svc.NewRetrieve().
					Where(device.MatchKeys(d.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).
				To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It(
			"Should refuse a device on a rack that does not exist",
			func(ctx SpecContext) {
				d := device.Device{
					Key:      "no-rack-device",
					Rack:     rack.NewKey(1, 999),
					Location: "loc",
					Name:     "Orphan",
					Make:     "Test Make",
					Model:    "Test Model",
				}
				Expect(w.Create(ctx, &d)).To(MatchError(fmt.Sprintf(
					"rack: rack %d does not exist: validation error",
					rack.NewKey(1, 999),
				)))
				Expect(svc.NewRetrieve().
					Where(device.MatchKeys(d.Key)).
					Exists(ctx, tx)).To(BeFalse())
			},
		)
		It("Should refuse a device whose key exists", func(ctx SpecContext) {
			d := device.Device{
				Key:      "existing-device",
				Rack:     rackSvc.EmbeddedKey,
				Location: "loc",
				Name:     "Original Name",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			again := device.Device{
				Key:      d.Key,
				Rack:     d.Rack,
				Location: d.Location,
				Name:     "New Name",
				Make:     d.Make,
				Model:    d.Model,
			}
			Expect(w.Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("device existing-device already exists")),
			))
			var res device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("Original Name"))
			var deviceStatus device.Status
			Expect(stat.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
				Entry(&deviceStatus).
				Exec(ctx, tx)).To(Succeed())
			Expect(deviceStatus.Name).To(Equal("Original Name"))
		})

		It(
			"Should use the provided status when creating a device",
			func(ctx SpecContext) {
				providedStatus := &device.Status{
					Variant:     status.VariantSuccess,
					Time:        telem.Now(),
					Message:     "Device is connected",
					Description: "Custom device description",
				}
				d := device.Device{
					Key:      "device-with-status",
					Rack:     rackSvc.EmbeddedKey,
					Location: "loc-status",
					Name:     "Device with custom status",
					Make:     "Test Make",
					Model:    "Test Model",
					Status:   providedStatus,
				}
				Expect(w.Create(ctx, &d)).To(Succeed())

				var deviceStatus device.Status
				Expect(stat.NewRetrieve[device.StatusDetails]().
					Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
					Entry(&deviceStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(deviceStatus.Variant).To(Equal(status.VariantSuccess))
				Expect(deviceStatus.Message).To(Equal("Device is connected"))
				Expect(deviceStatus.Description).To(Equal("Custom device description"))
				Expect(deviceStatus.Owner).To(HaveValue(Equal(d.ResourceID())))
				// Name should be auto-filled
				Expect(deviceStatus.Name).To(Equal(d.Name))
				// Details should be auto-filled
				Expect(deviceStatus.Details.Rack).To(Equal(d.Rack))
			},
		)

		It(
			"Should return a validation error if provided status has empty variant",
			func(ctx SpecContext) {
				providedStatus := &device.Status{
					Message: "Status with no variant",
					Time:    telem.Now(),
				}
				d := device.Device{
					Key:      "device-invalid-status",
					Rack:     rackSvc.EmbeddedKey,
					Location: "loc-invalid",
					Name:     "Device with invalid status",
					Make:     "Test Make",
					Model:    "Test Model",
					Status:   providedStatus,
				}
				Expect(
					w.Create(ctx, &d),
				).Error().
					To(MatchError(ContainSubstring("variant")))
			},
		)
		It(
			"Should populate Status on the device after Create",
			func(ctx SpecContext) {
				d := device.Device{
					Key:      "device-status-parent",
					Rack:     rackSvc.EmbeddedKey,
					Location: "loc",
					Name:     "Populated Device",
					Make:     "Test Make",
					Model:    "Test Model",
				}
				Expect(w.Create(ctx, &d)).To(Succeed())
				Expect(d.Status).ToNot(BeNil())
				Expect(d.Status.Variant).To(Equal(status.VariantWarning))
				Expect(d.Status.Name).To(Equal("Populated Device"))
				Expect(d.Parent).To(BeNil())
			},
		)
		It(
			"Should not write a provided status over the status of another device",
			func(ctx SpecContext) {
				retrieve := func(d device.Device) device.Status {
					GinkgoHelper()
					var s device.Status
					Expect(stat.NewRetrieve[device.StatusDetails]().
						Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
						Entry(&s).
						Exec(ctx, tx)).To(Succeed())
					return s
				}
				a := device.Device{
					Key:      "status-owner-a",
					Rack:     rackSvc.EmbeddedKey,
					Location: "a",
					Name:     "A",
					Make:     "Test Make",
					Model:    "Test Model",
				}
				Expect(w.Create(ctx, &a)).To(Succeed())
				aKey := retrieve(a).Key
				b := device.Device{
					Key:      "status-owner-b",
					Rack:     rackSvc.EmbeddedKey,
					Location: "b",
					Name:     "B",
					Make:     "Test Make",
					Model:    "Test Model",
					Status: &device.Status{
						Key:     aKey,
						Variant: status.VariantSuccess,
						Message: "forged",
						Time:    telem.Now(),
					},
				}
				Expect(w.Create(ctx, &b)).To(Succeed())
				Expect(retrieve(a).Message).To(Equal("Device status unknown"))
				bStatus := retrieve(b)
				Expect(bStatus.Message).To(Equal("forged"))
				Expect(bStatus.Key).ToNot(Equal(aKey))
			},
		)

		DescribeTable(
			"Should keep the live status when an apply renames or moves the device",
			func(
				ctx SpecContext,
				change func(context.Context, device.Device) []device.Action,
			) {
				d := device.Device{
					Key:      "keep-status-device",
					Rack:     rackSvc.EmbeddedKey,
					Location: "keep-loc",
					Name:     "Keep Device",
					Make:     "Test Make",
					Model:    "Test Model",
				}
				Expect(w.Create(ctx, &d)).To(Succeed())
				live := device.Status{
					Owner:   new(d.ResourceID()),
					Name:    d.Name,
					Time:    telem.Now(),
					Variant: status.VariantSuccess,
					Message: "Device is connected",
					Details: device.StatusDetails{Rack: d.Rack},
				}
				Expect(stat.NewWriter(tx).Set(ctx, &live)).To(Succeed())
				Expect(w.Apply(ctx, d.Key, change(ctx, d)...)).To(Succeed())
				var updated device.Device
				Expect(svc.NewRetrieve().
					Where(device.MatchKeys(d.Key)).
					Entry(&updated).
					Exec(ctx, tx)).To(Succeed())
				var stored device.Status
				Expect(stat.NewRetrieve[device.StatusDetails]().
					Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
					Entry(&stored).
					Exec(ctx, tx)).To(Succeed())
				Expect(stored.Variant).To(Equal(status.VariantSuccess))
				Expect(stored.Message).To(Equal("Device is connected"))
				Expect(stored.Time).To(Equal(live.Time))
				Expect(stored.Name).To(Equal(updated.Name))
				Expect(stored.Details.Rack).To(Equal(updated.Rack))
			},
			Entry(
				"renamed",
				func(context.Context, device.Device) []device.Action {
					return []device.Action{device.NewSetAction(device.SetPayload{
						Path:  []string{"name"},
						Value: "Renamed Device",
					})}
				},
			),
			Entry(
				"moved under another device",
				func(ctx context.Context, d device.Device) []device.Action {
					GinkgoHelper()
					return []device.Action{device.NewSetRackAction(
						device.SetRackPayload{
							Rack:   d.Rack,
							Parent: createChassis(ctx, w, d.Rack),
						},
					)}
				},
			),
			Entry(
				"renamed and moved under another device",
				func(ctx context.Context, d device.Device) []device.Action {
					GinkgoHelper()
					return []device.Action{
						device.NewSetAction(device.SetPayload{
							Path:  []string{"name"},
							Value: "Renamed Device",
						}),
						device.NewSetRackAction(device.SetRackPayload{
							Rack:   d.Rack,
							Parent: createChassis(ctx, w, d.Rack),
						}),
					}
				},
			),
			Entry(
				"moved to another rack",
				func(ctx context.Context, _ device.Device) []device.Action {
					GinkgoHelper()
					r := rack.Rack{Name: "Destination Rack"}
					Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
					return []device.Action{
						device.NewSetRackAction(device.SetRackPayload{Rack: r.Key}),
					}
				},
			),
		)

		It(
			"Should refuse a status whose device does not exist",
			func(ctx SpecContext) {
				d := device.Device{
					Key:  "early-status-device",
					Rack: rackSvc.EmbeddedKey,
				}
				Expect(stat.NewWriter(tx).Set(ctx, &device.Status{
					Owner:   new(d.ResourceID()),
					Name:    "Early Device",
					Time:    telem.Now(),
					Variant: status.VariantSuccess,
					Message: "Device is connected",
					Details: device.StatusDetails{Rack: d.Rack},
				})).To(MatchError(ContainSubstring(
					"owner: device early-status-device does not exist",
				)))
			},
		)

		It(
			"Should populate Status and Parent with an explicit parent after Create",
			func(ctx SpecContext) {
				chassis := device.Device{
					Key:      "pop-chassis",
					Rack:     rackSvc.EmbeddedKey,
					Location: "slot-0",
					Name:     "Pop Chassis",
					Make:     "Test Make",
					Model:    "Test Model",
				}
				Expect(w.Create(ctx, &chassis)).To(Succeed())

				chassisID := chassis.Key
				module := device.Device{
					Key:      "pop-module",
					Rack:     rackSvc.EmbeddedKey,
					Location: "slot-1",
					Name:     "Pop Module",
					Make:     "Test Make",
					Model:    "Test Model",
					Parent:   &chassisID,
				}
				Expect(w.Create(ctx, &module)).To(Succeed())
				Expect(module.Status).ToNot(BeNil())
				Expect(module.Parent).ToNot(BeNil())
				Expect(*module.Parent).To(Equal(chassisID))
			},
		)
	})
	Describe("CreateMany", func() {
		It("Should create multiple devices", func(ctx SpecContext) {
			devices := []device.Device{
				{
					Key:      "device-many-1",
					Rack:     rackSvc.EmbeddedKey,
					Location: "loc-1",
					Name:     "Device 1",
					Make:     "Test Make",
					Model:    "Test Model",
				},
				{
					Key:      "device-many-2",
					Rack:     rackSvc.EmbeddedKey,
					Location: "loc-2",
					Name:     "Device 2",
					Make:     "Test Make",
					Model:    "Test Model",
				},
			}
			Expect(w.CreateMany(ctx, &devices)).To(Succeed())

			var retrieved []device.Device
			Expect(svc.NewRetrieve().Where(device.MatchKeys(
				devices[0].Key,
				devices[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("Parent", func() {
		It("Should find the devices that a device holds", func(ctx SpecContext) {
			chassis := createChassis(ctx, w, rackSvc.EmbeddedKey)
			module := device.Device{
				Key:      "pd-module-1",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Module 1",
				Make:     "Test Make",
				Model:    "Test Model",
				Parent:   chassis,
			}
			Expect(w.Create(ctx, &module)).To(Succeed())
			var held []device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchParents(*chassis)).
				Entries(&held).
				Exec(ctx, tx)).To(Succeed())
			Expect(held).To(HaveLen(1))
			Expect(held[0].Key).To(Equal(module.Key))
			Expect(*held[0].Parent).To(Equal(*chassis))
		})

		It("Should filter devices by whether they are nested", func(ctx SpecContext) {
			chassis := createChassis(ctx, w, rackSvc.EmbeddedKey)
			module := device.Device{
				Key:      "pd-nested-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Nested Module",
				Make:     "Test Make",
				Model:    "Test Model",
				Parent:   chassis,
			}
			Expect(w.Create(ctx, &module)).To(Succeed())
			var nested, top []device.Device
			Expect(svc.NewRetrieve().
				Where(device.And(
					device.MatchRacks(rackSvc.EmbeddedKey),
					device.MatchHasParent(true),
				)).
				Entries(&nested).
				Exec(ctx, tx)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.And(
					device.MatchRacks(rackSvc.EmbeddedKey),
					device.MatchHasParent(false),
				)).
				Entries(&top).
				Exec(ctx, tx)).To(Succeed())
			Expect(nested).To(HaveLen(1))
			Expect(nested[0].Key).To(Equal(module.Key))
			Expect(top).To(HaveLen(1))
			Expect(top[0].Key).To(Equal(*chassis))
		})

		It("Should move a device to another parent", func(ctx SpecContext) {
			chassisA := createChassis(ctx, w, rackSvc.EmbeddedKey)
			chassisB := createChassis(ctx, w, rackSvc.EmbeddedKey)
			module := device.Device{
				Key:      "pd-moving-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Moving Module",
				Make:     "Test Make",
				Model:    "Test Model",
				Parent:   chassisA,
			}
			Expect(w.Create(ctx, &module)).To(Succeed())
			Expect(w.Apply(
				ctx,
				module.Key,
				device.NewSetRackAction(device.SetRackPayload{
					Rack:   module.Rack,
					Parent: chassisB,
				}),
			)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.MatchParents(*chassisA)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(svc.NewRetrieve().
				Where(device.MatchParents(*chassisB)).
				Exists(ctx, tx)).To(BeTrue())
		})

		It(
			"Should clear the parent of a device moved without one",
			func(ctx SpecContext) {
				chassis := createChassis(ctx, w, rackSvc.EmbeddedKey)
				module := device.Device{
					Key:      "pd-cleared-module",
					Rack:     rackSvc.EmbeddedKey,
					Location: "slot-1",
					Name:     "Cleared Module",
					Make:     "Test Make",
					Model:    "Test Model",
					Parent:   chassis,
				}
				Expect(w.Create(ctx, &module)).To(Succeed())
				Expect(w.Apply(
					ctx,
					module.Key,
					device.NewSetRackAction(device.SetRackPayload{Rack: module.Rack}),
				)).To(Succeed())
				var res device.Device
				Expect(svc.NewRetrieve().
					Where(device.MatchKeys(module.Key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.Parent).To(BeNil())
			},
		)

		It("Should refuse a parent that does not exist", func(ctx SpecContext) {
			missing := device.Key("pd-missing-chassis")
			module := device.Device{
				Key:      "pd-orphan-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Orphan Module",
				Make:     "Test Make",
				Model:    "Test Model",
				Parent:   &missing,
			}
			Expect(w.Create(ctx, &module)).
				To(MatchError(
					"parent: device pd-missing-chassis does not exist: validation error",
				))
		})

		It("Should refuse a device that holds itself", func(ctx SpecContext) {
			chassis := createChassis(ctx, w, rackSvc.EmbeddedKey)
			module := device.Device{
				Key:      "pd-loop-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Loop Module",
				Make:     "Test Make",
				Model:    "Test Model",
				Parent:   chassis,
			}
			Expect(w.Create(ctx, &module)).To(Succeed())
			Expect(w.Apply(
				ctx,
				*chassis,
				device.NewSetRackAction(device.SetRackPayload{
					Rack:   rackSvc.EmbeddedKey,
					Parent: &module.Key,
				}),
			)).To(MatchError(fmt.Sprintf(
				"parent: device %s would form a cycle: validation error",
				*chassis,
			)))
		})

		It("Should delete the devices a deleted device holds", func(ctx SpecContext) {
			chassis := createChassis(ctx, w, rackSvc.EmbeddedKey)
			module := device.Device{
				Key:      "pd-cascade-module",
				Rack:     rackSvc.EmbeddedKey,
				Location: "slot-1",
				Name:     "Cascade Module",
				Make:     "Test Make",
				Model:    "Test Model",
				Parent:   chassis,
			}
			Expect(w.Create(ctx, &module)).To(Succeed())
			Expect(w.Delete(ctx, *chassis)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(*chassis, module.Key)).
				Exists(ctx, tx)).To(BeFalse())
		})
	})
	Describe("Group", func() {
		var d device.Device
		createGroup := func(ctx context.Context, r rack.Key) group.Key {
			GinkgoHelper()
			parent := r.ResourceID()
			g := group.Group{Name: "Folder", Parent: &parent}
			Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
			return g.Key
		}
		BeforeEach(func(ctx SpecContext) {
			r := rack.Rack{Name: "Grouped Rack"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			d = device.Device{
				Key:      "grouped-device",
				Rack:     r.Key,
				Location: "loc",
				Name:     "Grouped Device",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
		})
		retrieve := func(ctx context.Context) device.Device {
			GinkgoHelper()
			var res device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			return res
		}

		It("Should move a device into a group under its rack", func(ctx SpecContext) {
			g := createGroup(ctx, d.Rack)
			Expect(w.SetGroup(ctx, &g, d.Key)).To(Succeed())
			Expect(retrieve(ctx).Group).To(Equal(&g))
		})

		It("Should refuse a group under another rack", func(ctx SpecContext) {
			g := createGroup(ctx, rackSvc.EmbeddedKey)
			Expect(w.SetGroup(ctx, &g, d.Key)).
				To(MatchError(ContainSubstring("does not sit under rack")))
			Expect(retrieve(ctx).Group).To(BeNil())
		})

		It("Should keep the group of a renamed device", func(ctx SpecContext) {
			g := createGroup(ctx, d.Rack)
			Expect(w.SetGroup(ctx, &g, d.Key)).To(Succeed())
			Expect(w.Rename(ctx, d.Key, "Renamed Device")).To(Succeed())
			Expect(retrieve(ctx).Group).To(Equal(&g))
		})

		It(
			"Should refuse to move a device with a parent into a group",
			func(ctx SpecContext) {
				g := createGroup(ctx, d.Rack)
				d.Parent = createChassis(ctx, w, d.Rack)
				Expect(w.Apply(
					ctx,
					d.Key,
					device.NewSetRackAction(device.SetRackPayload{
						Rack:   d.Rack,
						Parent: d.Parent,
					}),
				)).To(Succeed())
				Expect(w.SetGroup(ctx, &g, d.Key)).To(MatchError(fmt.Sprintf(
					"group: a device held by device %s cannot sit in a group: "+
						"validation error",
					*d.Parent,
				)))
				Expect(retrieve(ctx).Group).To(BeNil())
			},
		)

		It(
			"Should refuse a new device with a parent and a group",
			func(ctx SpecContext) {
				g := createGroup(ctx, d.Rack)
				module := device.Device{
					Key:      "grouped-module",
					Rack:     d.Rack,
					Location: "slot-1",
					Name:     "Grouped Module",
					Make:     "Test Make",
					Model:    "Test Model",
					Parent:   &d.Key,
					Group:    &g,
				}
				Expect(w.Create(ctx, &module)).To(MatchError(fmt.Sprintf(
					"group: a device held by device %s cannot sit in a group: "+
						"validation error",
					d.Key,
				)))
			},
		)
	})
	Describe("Rack", func() {
		It("Should refuse to delete a rack that holds a device", func(ctx SpecContext) {
			r := rack.Rack{Name: "Held Rack"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			d := device.Device{
				Key:      "rack-held-device",
				Rack:     r.Key,
				Location: "loc",
				Name:     "Held Device",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, r.Key)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(fmt.Sprintf(
					"cannot delete rack %d because device rack-held-device references "+
						"it through rack",
					r.Key,
				))),
			))
			Expect(rackSvc.NewRetrieve().
				Where(rack.MatchKeys(r.Key)).
				Exists(ctx, tx)).To(BeTrue())
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, r.Key)).To(Succeed())
		})
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "device4",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev4",
				Name:     "Fish",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			var res device.Device
			Expect(
				svc.NewRetrieve().
					Where(device.MatchKeys(d.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).
				To(Succeed())
			Expect(res.Key).To(Equal(d.Key))
		})
		It("Should retrieve devices by their model", func(ctx SpecContext) {
			d1 := device.Device{
				Key:      "device5",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev5",
				Name:     "Fish",
				Make:     "Test Make",
				Model:    "A",
			}
			d2a := device.Device{
				Key:      "device6",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev6",
				Name:     "Fish",
				Make:     "Test Make",
				Model:    "B",
			}
			d2b := device.Device{
				Key:      "device7",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev7",
				Name:     "Fish",
				Make:     "Test Make",
				Model:    "B",
			}
			Expect(w.Create(ctx, &d1)).To(Succeed())
			Expect(w.Create(ctx, &d2a)).To(Succeed())
			Expect(w.Create(ctx, &d2b)).To(Succeed())
			// Clear fields populated by Create that aren't stored in gorp.
			d2a.Status, d2a.Parent = nil, nil
			d2b.Status, d2b.Parent = nil, nil
			// A stored device reads back with an allocated, empty Properties.
			d2a.Properties = msgpack.EncodedJSON{}
			d2b.Properties = msgpack.EncodedJSON{}
			var res []device.Device
			Expect(
				svc.NewRetrieve().
					Where(device.MatchModels("B")).
					Entries(&res).
					Exec(ctx, tx),
			).
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
				Model:    "Test Model",
			}
			d2a := device.Device{
				Key:      "device9",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev9",
				Name:     "Fish",
				Make:     "B",
				Model:    "Test Model",
			}
			d2b := device.Device{
				Key:      "device10",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev10",
				Name:     "Fish",
				Make:     "B",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d1)).To(Succeed())
			Expect(w.Create(ctx, &d2a)).To(Succeed())
			Expect(w.Create(ctx, &d2b)).To(Succeed())
			d2a.Status, d2a.Parent = nil, nil
			d2b.Status, d2b.Parent = nil, nil
			// A stored device reads back with an allocated, empty Properties.
			d2a.Properties = msgpack.EncodedJSON{}
			d2b.Properties = msgpack.EncodedJSON{}
			var res []device.Device
			Expect(
				svc.NewRetrieve().
					Where(device.MatchMakes("B")).
					Entries(&res).
					Exec(ctx, tx),
			).
				To(Succeed())
			Expect(res).To(ConsistOf(d2a, d2b))
		})
		It("Should retrieve devices by their location", func(ctx SpecContext) {
			d1 := device.Device{
				Key:      "device11",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev11",
				Name:     "Fish",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			d2a := device.Device{
				Key:      "device12",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev12",
				Name:     "Fish",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			d2b := device.Device{
				Key:      "device13",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev13",
				Name:     "Fish",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d1)).To(Succeed())
			Expect(w.Create(ctx, &d2a)).To(Succeed())
			Expect(w.Create(ctx, &d2b)).To(Succeed())
			d2a.Status, d2a.Parent = nil, nil
			// A stored device reads back with an allocated, empty Properties.
			d2a.Properties = msgpack.EncodedJSON{}
			var res []device.Device
			Expect(
				svc.NewRetrieve().
					Where(device.MatchLocations("dev12")).
					Entries(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(ConsistOf(d2a))
		})
	})
	Describe("Rename", func() {
		It(
			"Should rename a device and its status, keeping every other field",
			func(ctx SpecContext) {
				d := device.Device{
					Key:        "rename-device",
					Rack:       rackSvc.EmbeddedKey,
					Location:   "dev-rename",
					Name:       "Before",
					Make:       "Test Make",
					Model:      "Test Model",
					Configured: true,
					Properties: msgpack.EncodedJSON{"host": "10.0.0.1"},
					Status: &device.Status{
						Variant: status.VariantSuccess,
						Message: "Connected",
						Time:    telem.Now(),
					},
				}
				Expect(w.Create(ctx, &d)).To(Succeed())
				Expect(w.Rename(ctx, d.Key, "After")).To(Succeed())
				var res device.Device
				Expect(svc.NewRetrieve().
					Where(device.MatchKeys(d.Key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.Name).To(Equal("After"))
				Expect(res.Location).To(Equal(d.Location))
				Expect(res.Make).To(Equal(d.Make))
				Expect(res.Model).To(Equal(d.Model))
				Expect(res.Rack).To(Equal(d.Rack))
				Expect(res.Configured).To(BeTrue())
				Expect(res.Properties).To(Equal(d.Properties))
				var renamed device.Status
				Expect(stat.NewRetrieve[device.StatusDetails]().
					Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
					Entry(&renamed).
					Exec(ctx, tx)).To(Succeed())
				Expect(renamed.Name).To(Equal("After"))
				Expect(renamed.Variant).To(Equal(status.VariantSuccess))
				Expect(renamed.Message).To(Equal("Connected"))
				Expect(renamed.Owner).To(HaveValue(Equal(d.ResourceID())))
			},
		)
		It("Should return a validation error for an empty name", func(ctx SpecContext) {
			d := device.Device{
				Key:      "rename-device-empty",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev-rename-empty",
				Name:     "Before",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(w.Rename(ctx, d.Key, "")).
				To(MatchError(ContainSubstring("name: required")))
		})
		It(
			"Should return a not found error for a missing device",
			func(ctx SpecContext) {
				Expect(w.Rename(ctx, "missing-device", "After")).
					To(MatchError(query.ErrNotFound))
			},
		)
		It("Should write the default status when the device has none", func(
			ctx SpecContext,
		) {
			d := device.Device{
				Key:      "rename-device-no-status",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev-rename-no-status",
				Name:     "Before",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			sw := stat.NewWriter(tx)
			Expect(sw.DeleteOwned(ctx, d.ResourceID())).
				To(Succeed())
			Expect(w.Rename(ctx, d.Key, "After")).To(Succeed())
			var s device.Status
			Expect(stat.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
				Entry(&s).
				Exec(ctx, tx)).To(Succeed())
			Expect(s.Name).To(Equal("After"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Device status unknown"))
			Expect(s.Details.Rack).To(Equal(rackSvc.EmbeddedKey))
		})
	})
	Describe("Group", func() {
		var underRack group.Group
		BeforeEach(func(ctx SpecContext) {
			underRack = group.Group{
				Name:   "Under rack",
				Parent: new(rackSvc.EmbeddedKey.ResourceID()),
			}
			Expect(groupSvc.NewWriter(tx).Create(ctx, &underRack)).To(Succeed())
		})
		createDevice := func(ctx context.Context, key string, g *group.Key) error {
			return w.Create(ctx, &device.Device{
				Key:      key,
				Rack:     rackSvc.EmbeddedKey,
				Location: key,
				Name:     key,
				Make:     "Test Make",
				Model:    "Test Model",
				Group:    g,
			})
		}
		retrieveGroup := func(ctx context.Context, key string) *group.Key {
			GinkgoHelper()
			var d device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(key)).
				Entry(&d).
				Exec(ctx, tx)).To(Succeed())
			return d.Group
		}

		It(
			"Should move a device into a group under its rack and back out",
			func(ctx SpecContext) {
				Expect(createDevice(ctx, "grouped", nil)).To(Succeed())
				Expect(w.SetGroup(ctx, &underRack.Key, "grouped")).To(Succeed())
				Expect(retrieveGroup(ctx, "grouped")).To(Equal(&underRack.Key))
				Expect(w.SetGroup(ctx, nil, "grouped")).To(Succeed())
				Expect(retrieveGroup(ctx, "grouped")).To(BeNil())
			},
		)

		It(
			"Should reject a group that does not sit under the device's rack",
			func(ctx SpecContext) {
				top := group.Group{Name: "Top", Scope: resource.TypeDevice}
				Expect(groupSvc.NewWriter(tx).Create(ctx, &top)).To(Succeed())
				Expect(createDevice(ctx, "elsewhere", nil)).To(Succeed())
				Expect(w.SetGroup(ctx, &top.Key, "elsewhere")).To(MatchError(
					"group: group Top does not sit under rack " +
						rackSvc.EmbeddedKey.ResourceID().Key + ": validation error",
				))
				Expect(retrieveGroup(ctx, "elsewhere")).To(BeNil())
			},
		)

		It("Should create a device in a group under its rack", func(ctx SpecContext) {
			Expect(createDevice(ctx, "created", &underRack.Key)).To(Succeed())
			Expect(retrieveGroup(ctx, "created")).To(Equal(&underRack.Key))
		})

		It(
			"Should reject a created device in a group of another scope",
			func(ctx SpecContext) {
				channels := group.Group{Name: "Channels", Scope: resource.TypeChannel}
				Expect(groupSvc.NewWriter(tx).Create(ctx, &channels)).To(Succeed())
				Expect(createDevice(ctx, "mismatch", &channels.Key)).To(MatchError(
					"group: a device cannot sit in a channel group: validation error",
				))
			},
		)
	})
	Describe("Delete", func() {
		It(
			"Should correctly delete a device and its associated status",
			func(ctx SpecContext) {
				d := device.Device{
					Key:      "device14",
					Rack:     rackSvc.EmbeddedKey,
					Location: "dev14",
					Name:     "Fish",
					Make:     "Test Make",
					Model:    "Test Model",
				}
				Expect(w.Create(ctx, &d)).To(Succeed())
				Expect(w.Delete(ctx, d.Key)).To(Succeed())
				var res device.Device
				Expect(
					svc.NewRetrieve().
						Where(device.MatchKeys(d.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(MatchError(query.ErrNotFound))
				Expect(stat.NewRetrieve[any]().
					Where(status.MatchOwners[any](d.ResourceID())).
					Entry(new(status.Status[any])).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("Observe", func() {
		It("Should notify when a device is created", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB: db,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				DB: db,
			}))
			roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
				DB: db,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:      db,
				HostKey: 1,
				Status:  stat,
				DriverRole: MustSucceed(
					racktestutil.CreateDriverRole(ctx, db, roleSvc),
				),
			}))
			svc := MustOpen(device.OpenService(ctx, device.ServiceConfig{
				DB:     db,
				Status: stat,
				Rack:   rackSvc,
			}))
			var created atomic.Value
			disconnect := svc.Observe().OnChange(
				func(ctx context.Context, r gorp.TxReader[device.Key, device.Device]) {
					for change := range r {
						if change.Kind != gorp.ChangeDelete {
							created.Store(change.Value.Name)
						}
					}
				},
			)
			defer disconnect()
			d := device.Device{
				Key:      "observed",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev1",
				Name:     "Observed",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &d)
			})).To(Succeed())
			Eventually(created.Load).Should(Equal("Observed"))
		})
	})
	Describe("Suspect Rack", func() {
		It(
			"Should propagate rack warning status to devices on that rack",
			func(ctx SpecContext) {
				db := DeferClose(gorp.Wrap(memkv.New()))
				MustOpen(label.OpenService(ctx, label.ServiceConfig{
					DB: db,
				}))
				stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
					DB: db,
				}))
				roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
					DB: db,
				}))
				rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
					DB:                  db,
					HostKey:             1,
					Status:              stat,
					HealthCheckInterval: 10 * telem.Millisecond,
					DriverRole: MustSucceed(
						racktestutil.CreateDriverRole(ctx, db, roleSvc),
					),
				}))
				svc := MustOpen(device.OpenService(ctx, device.ServiceConfig{
					DB:     db,
					Status: stat,
					Rack:   rackSvc,
				}))

				r := rack.Rack{Name: "suspect rack"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return rackSvc.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())

				d := device.Device{
					Key:      "suspect-device",
					Rack:     r.Key,
					Location: "loc1",
					Name:     "Test Device",
					Make:     "Test Make",
					Model:    "Test Model",
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &d)
				})).To(Succeed())

				Eventually(func(g Gomega) {
					var deviceStatus device.Status
					g.Expect(stat.NewRetrieve[device.StatusDetails]().
						Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
						Entry(&deviceStatus).
						Exec(ctx, db)).To(Succeed())
					g.Expect(deviceStatus.Variant).To(Equal(status.VariantWarning))
					g.Expect(deviceStatus.Message).To(Equal("Driver not running"))
					g.Expect(deviceStatus.Owner).To(HaveValue(Equal(d.ResourceID())))
					g.Expect(deviceStatus.Details.Rack).To(Equal(r.Key))
				}).Should(Succeed())
			},
		)
	})

	Describe("Search", func() {
		create := func(ctx SpecContext, key, name string) resource.ID {
			GinkgoHelper()
			d := device.Device{
				Key:      key,
				Rack:     rackSvc.EmbeddedKey,
				Name:     name,
				Make:     "acme",
				Model:    "usb6000",
				Location: "bay7",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			return device.ResourceID(d.Key)
		}

		It(
			"Should return the matching devices, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "a", "turbopump test")
				b := create(ctx, "b", "turbopump")
				q := search.Query[resource.ID]{Term: "turbopump"}
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b, a}))
				q.Limit = 1
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b}))
			},
		)

		DescribeTable("Should match a device by its",
			func(ctx SpecContext, term string) {
				id := create(ctx, "igniter", "igniter")
				q := search.Query[resource.ID]{Term: term}
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{id}))
			},
			Entry("name", "igniter"),
			Entry("make", "acme"),
			Entry("model", "usb6000"),
			Entry("location", "bay7"),
		)

		It(
			"Should not match a staged device outside its transaction",
			func(ctx SpecContext) {
				create(ctx, "injector", "injector")
				q := search.Query[resource.ID]{Term: "injector"}
				Expect(
					MustSucceed(svc.Resources().Search(ctx, db, q)).Keys,
				).To(BeEmpty())
			},
		)
	})
	Describe("Resources", func() {
		It("Should name each device by its name", func(ctx SpecContext) {
			d := device.Device{
				Key:      "resource-device",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev1",
				Name:     "Dog",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{d.ResourceID()})).
				To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Name}}))
		})

		It("Should leave out an ID that names no device", func(ctx SpecContext) {
			d := device.Device{
				Key:      "present-device",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev1",
				Name:     "Cat",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(svc.Resources().Retrieve(
				ctx,
				tx,
				[]resource.ID{device.ResourceID("missing-device"), d.ResourceID()},
			)).To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Name}}))
		})

		It("Should list each device by its name", func(ctx SpecContext) {
			d := device.Device{
				Key:      "listed-device",
				Rack:     rackSvc.EmbeddedKey,
				Location: "dev1",
				Name:     "Bird",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(w.Create(ctx, &d)).To(Succeed())
			Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
				resource.Resource{ID: d.ResourceID(), Name: d.Name},
			))
		})
	})
})

// createChassis creates a device on rackKey and returns its key.
func createChassis(ctx context.Context, w device.Writer, rackKey rack.Key) *device.Key {
	GinkgoHelper()
	d := device.Device{
		Key:      device.Key(uuid.New().String()),
		Rack:     rackKey,
		Location: "slot-0",
		Name:     "Chassis",
		Make:     "Test Make",
		Model:    "Test Model",
	}
	Expect(w.Create(ctx, &d)).To(Succeed())
	return &d.Key
}

var _ = Describe("Writer in concurrent transactions", func() {
	It(
		"Should keep the group that a move writes during concurrent applies",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB: db,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				DB: db,
			}))
			roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
				DB: db,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:      db,
				HostKey: 1,
				Status:  stat,
				DriverRole: MustSucceed(
					racktestutil.CreateDriverRole(ctx, db, roleSvc),
				),
			}))
			svc := MustOpen(device.OpenService(ctx, device.ServiceConfig{
				DB:     db,
				Status: stat,
				Rack:   rackSvc,
			}))
			groups := make([]group.Group, 50)
			for i := range groups {
				groups[i] = group.Group{
					Name:   fmt.Sprintf("Concurrent %d", i),
					Parent: new(rackSvc.EmbeddedKey.ResourceID()),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return groupSvc.NewWriter(tx).Create(ctx, &groups[i])
				})).To(Succeed())
			}
			d := device.Device{
				Key:      "concurrent",
				Rack:     rackSvc.EmbeddedKey,
				Location: "concurrent",
				Name:     "concurrent",
				Make:     "Test Make",
				Model:    "Test Model",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &d)
			})).To(Succeed())
			retrieveGroup := func() *group.Key {
				GinkgoHelper()
				var stored device.Device
				Expect(svc.NewRetrieve().
					Where(device.MatchKeys(d.Key)).
					Entry(&stored).
					Exec(ctx, db)).To(Succeed())
				return stored.Group
			}
			var (
				wg    sync.WaitGroup
				moved atomic.Bool
			)
			for range 4 {
				wg.Go(func() {
					defer GinkgoRecover()
					for !moved.Load() {
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Rename(ctx, d.Key, "renamed")
						})).To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).SetGroup(ctx, &g.Key, d.Key)
					})).
						To(Succeed())
					Expect(retrieveGroup()).To(Equal(&g.Key))
				}
			})
			wg.Wait()
			Expect(retrieveGroup()).To(Equal(&groups[len(groups)-1].Key))
		},
	)
})

var _ = Describe("References", func() {
	var (
		db      *gorp.DB
		labels  *label.Service
		rackSvc *rack.Service
		svc     *device.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		labels = MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
			Rack:   rackSvc,
			References: []gorp.Reference[device.Key, device.Device]{
				gorp.NewKeyReference(
					gorp.KeyReferenceConfig[device.Key, device.Device, label.Key]{
						Name:     "properties.label",
						Table:    "label",
						OnDelete: gorp.Restrict,
						Kind:     gorp.Use,
						Targets: func(d *device.Device) []label.Key {
							var props struct {
								Label label.Key `json:"label"`
							}
							if err := d.Properties.Unmarshal(&props); err != nil ||
								props.Label == uuid.Nil() {
								return nil
							}
							return []label.Key{props.Label}
						},
					},
				),
			},
		}))
	})

	create := func(ctx context.Context, rk rack.Key, lbl label.Key) error {
		d := device.Device{
			Key:        "referencing",
			Rack:       rk,
			Location:   "loc",
			Name:       "Referencing",
			Make:       "Test Make",
			Model:      "Test Model",
			Properties: msgpack.EncodedJSON{"label": lbl},
		}
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &d)
		})
	}

	It("Should create a device whose reference names a target", func(ctx SpecContext) {
		l := label.Label{Name: "Label"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return labels.NewWriter(tx).Create(ctx, &l)
		})).To(Succeed())
		Expect(create(ctx, rackSvc.EmbeddedKey, l.Key)).To(Succeed())
	})

	It("Should refuse a device whose reference names a missing target",
		func(ctx SpecContext) {
			key := uuid.NewV7()
			Expect(create(ctx, rackSvc.EmbeddedKey, key)).To(MatchError(fmt.Sprintf(
				"properties.label: label %s does not exist: validation error",
				key,
			)))
		})

	It("Should refuse a device on a rack that does not exist", func(ctx SpecContext) {
		Expect(create(ctx, rack.NewKey(1, 999), uuid.Nil())).To(MatchError(fmt.Sprintf(
			"rack: rack %d does not exist: validation error",
			rack.NewKey(1, 999),
		)))
	})
})
