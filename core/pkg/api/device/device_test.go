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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apidevice "github.com/synnaxlabs/synnax/pkg/api/device"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createDevice(ctx SpecContext, key string) device.Device {
	GinkgoHelper()
	d := device.Device{
		Key:      key,
		Rack:     svc.Rack.EmbeddedKey,
		Location: "dev1",
		Name:     "Before",
		Make:     "Test Make",
		Model:    "Test Model",
	}
	Expect(svc.Device.NewWriter(nil).Create(ctx, &d)).To(Succeed())
	return d
}

func retrieveDevice(ctx SpecContext, key string) device.Device {
	GinkgoHelper()
	var d device.Device
	Expect(svc.Device.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&d).
		Exec(ctx, nil)).To(Succeed())
	return d
}

func setLiveStatus(ctx SpecContext, d device.Device) {
	GinkgoHelper()
	Expect(svc.Status.NewWriter(nil).Set(ctx, &device.Status{
		Owner:   new(d.ResourceID()),
		Name:    d.Name,
		Time:    telem.Now(),
		Variant: status.VariantSuccess,
		Message: "Device is running",
		Details: device.StatusDetails{Rack: d.Rack},
	})).To(Succeed())
}

var _ = Describe("Create", func() {
	deviceType := resource.ID{Type: resource.TypeDevice}
	authedCtx := func(ctx SpecContext) context.Context {
		GinkgoHelper()
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), deviceType)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), deviceType)
		return AuthedCtx(ctx, u)
	}
	createStripped := func(ctx SpecContext, key string) device.Device {
		GinkgoHelper()
		d := createDevice(ctx, key)
		setLiveStatus(ctx, d)
		d.Status = nil
		return d
	}
	It(
		"Should leave out each status when create is the only grant",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.ResourceID(), deviceType)
			hidden := createStripped(ctx, "create-only-hidden")
			fresh := hidden
			fresh.Key = "create-only-fresh"
			res := MustSucceed(apiSvc.Create(
				AuthedCtx(ctx, u), nil,
				apidevice.CreateRequest{Devices: []device.Device{fresh, hidden}},
			))
			Expect(res.Devices).To(HaveLen(2))
			Expect(res.Devices[0].Status).To(BeNil())
			Expect(res.Devices[1].Status).To(BeNil())
			Expect(retrieveDevice(ctx, fresh.Key).Key).To(Equal(fresh.Key))
		},
	)
	It("Should leave out the status when read is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		hidden := createStripped(ctx, "create-update-hidden")
		grant(ctx, access.ActionCreate, u.ResourceID(), deviceType)
		grant(ctx, access.ActionUpdate, u.ResourceID(), hidden.ResourceID())
		res := MustSucceed(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apidevice.CreateRequest{Devices: []device.Device{hidden}},
		))
		Expect(res.Devices).To(HaveLen(1))
		Expect(res.Devices[0].Status).To(BeNil())
	})
	It("Should reject a caller with only update access", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createStripped(ctx, "update-only")
		grant(ctx, access.ActionUpdate, u.ResourceID(), d.ResourceID())
		Expect(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apidevice.CreateRequest{Devices: []device.Device{d}},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should check read access on each device in a batch", func(ctx SpecContext) {
		u := createUser(ctx)
		readable := createStripped(ctx, "batch-readable")
		hidden := createStripped(ctx, "batch-hidden")
		grant(ctx, access.ActionCreate, u.ResourceID(), deviceType)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), readable.ResourceID())
		res := MustSucceed(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apidevice.CreateRequest{Devices: []device.Device{readable, hidden}},
		))
		Expect(res.Devices).To(HaveLen(2))
		Expect(res.Devices[0].Status.Message).To(Equal("Device is running"))
		Expect(res.Devices[1].Status).To(BeNil())
	})
	It("Should reply with the stored status of each device", func(ctx SpecContext) {
		existing := createStripped(ctx, "create-existing")
		fresh := existing
		fresh.Key = "create-fresh"
		res := MustSucceed(apiSvc.Create(
			authedCtx(ctx), nil,
			apidevice.CreateRequest{Devices: []device.Device{fresh, existing}},
		))
		Expect(res.Devices).To(HaveLen(2))
		Expect(res.Devices[0].Status.Owner).To(HaveValue(Equal(fresh.ResourceID())))
		Expect(res.Devices[0].Status.Message).To(Equal("Device status unknown"))
		Expect(res.Devices[1].Status.Owner).To(HaveValue(Equal(existing.ResourceID())))
		Expect(res.Devices[1].Status.Message).To(Equal("Device is running"))
	})
	It(
		"Should keep the live status when a reply is created again",
		func(ctx SpecContext) {
			d := createStripped(ctx, "create-round-trip")
			authed := authedCtx(ctx)
			res := MustSucceed(apiSvc.Create(
				authed, nil, apidevice.CreateRequest{Devices: []device.Device{d}},
			))
			Expect(apiSvc.Create(
				authed, nil, apidevice.CreateRequest(res),
			)).Error().ToNot(HaveOccurred())
			var stored device.Status
			Expect(svc.Status.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
				Entry(&stored).
				Exec(ctx, nil)).To(Succeed())
			Expect(stored.Message).To(Equal("Device is running"))
		},
	)
})

var _ = Describe("Rename", func() {
	It("Should rename the device when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), d.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: d.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveDevice(ctx, d.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Location).To(Equal(d.Location))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-empty")
		grant(ctx, access.ActionUpdate, u.ResourceID(), d.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: d.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing device", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), device.ResourceID("missing"))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: "missing", Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-denied")
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: d.Key, Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Before"))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should attach only each device's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := createDevice(ctx, "retrieve-with-status")
		withoutStatus := createDevice(ctx, "retrieve-without-status")
		w := svc.Status.NewWriter(nil)
		Expect(w.DeleteOwned(ctx, withoutStatus.ResourceID())).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.ResourceID(),
			withStatus.ResourceID(), withoutStatus.ResourceID(),
		)
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{
				Keys:          []device.Key{withoutStatus.Key, withStatus.Key},
				IncludeStatus: true,
			},
		))
		Expect(res.Devices).To(HaveLen(2))
		Expect(res.Devices[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Devices[0].Status).To(BeNil())
		Expect(res.Devices[1].Key).To(Equal(withStatus.Key))
		Expect(
			res.Devices[1].Status.Owner,
		).To(HaveValue(Equal(withStatus.ResourceID())))
	})

	It("Should filter devices by parent and by nesting", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), resource.ID{
			Type: resource.TypeDevice,
		})
		chassis := createDevice(ctx, "retrieve-chassis")
		module := device.Device{
			Key:      "retrieve-module",
			Rack:     svc.Rack.EmbeddedKey,
			Location: "slot-1",
			Name:     "Module",
			Make:     "Test Make",
			Model:    "Test Model",
			Parent:   &chassis.Key,
		}
		Expect(svc.Device.NewWriter(nil).Create(ctx, &module)).To(Succeed())
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{Parents: []device.Key{chassis.Key}},
		))
		Expect(res.Devices).To(HaveLen(1))
		Expect(res.Devices[0].Key).To(Equal(module.Key))
		Expect(res.Devices[0].Parent).To(Equal(&chassis.Key))
		res = MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{
				Keys:   []device.Key{chassis.Key, module.Key},
				Nested: new(false),
			},
		))
		Expect(res.Devices).To(HaveLen(1))
		Expect(res.Devices[0].Key).To(Equal(chassis.Key))
	})
})

// createRackGroup creates a group under the embedded rack, where its devices can sit.
func createRackGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: new(svc.Rack.EmbeddedKey.ResourceID())}
	Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			d := createDevice(ctx, "set-group-denied")
			g := createRackGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), d.ResourceID())
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
				apidevice.SetGroupRequest{Group: &g.Key, Keys: []device.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveDevice(ctx, d.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the device with update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			d := createDevice(ctx, "set-group-granted")
			g := createRackGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				d.ResourceID(),
				g.ResourceID(),
			)
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
				apidevice.SetGroupRequest{Group: &g.Key, Keys: []device.Key{d.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveDevice(ctx, d.Key).Group).To(Equal(&g.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	newDevice := func(key string, g group.Group) device.Device {
		return device.Device{
			Key:      key,
			Rack:     svc.Rack.EmbeddedKey,
			Location: key,
			Name:     key,
			Make:     "Test Make",
			Model:    "Test Model",
			Group:    &g.Key,
		}
	}
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createRackGroup(ctx, "Create denied")
			grant(
				ctx,
				access.ActionCreate,
				u.ResourceID(),
				resource.ID{Type: resource.TypeDevice},
			)
			Expect(apiSvc.Create(
				AuthedCtx(ctx, u), nil,
				apidevice.CreateRequest{
					Devices: []device.Device{newDevice("create-group-denied", g)},
				},
			)).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the device with update on the group", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createRackGroup(ctx, "Create granted")
		grant(
			ctx,
			access.ActionCreate,
			u.ResourceID(),
			resource.ID{Type: resource.TypeDevice},
		)
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		Expect(apiSvc.Create(
			AuthedCtx(ctx, u), nil,
			apidevice.CreateRequest{
				Devices: []device.Device{newDevice("create-group-granted", g)},
			},
		)).Error().ToNot(HaveOccurred())
		Expect(retrieveDevice(ctx, "create-group-granted").Group).To(Equal(&g.Key))
	})
})

var _ = Describe("Create on another rack", func() {
	It(
		"Should drop the group of a moved device without rights on the group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createRackGroup(ctx, "Rack move")
			d := createDevice(ctx, "rack-move")
			Expect(svc.Device.NewWriter(nil).SetGroup(ctx, &g.Key, d.Key)).To(Succeed())
			other := rack.Rack{Name: "Rack move"}
			Expect(svc.Rack.NewWriter(nil).Create(ctx, &other)).To(Succeed())
			grant(
				ctx,
				access.ActionCreate,
				u.ResourceID(),
				resource.ID{Type: resource.TypeDevice},
			)
			grant(ctx, access.ActionUpdate, u.ResourceID(), d.ResourceID())
			d.Rack, d.Parent = other.Key, nil
			Expect(apiSvc.Create(
				AuthedCtx(ctx, u), nil,
				apidevice.CreateRequest{Devices: []device.Device{d}},
			)).Error().ToNot(HaveOccurred())
			moved := retrieveDevice(ctx, d.Key)
			Expect(moved.Rack).To(Equal(other.Key))
			Expect(moved.Group).To(BeNil())
		},
	)
})
