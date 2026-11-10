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
	"encoding/json/v2"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apidevice "github.com/synnaxlabs/synnax/pkg/api/device"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var deviceType = resource.ID{Type: resource.TypeDevice}

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

// need is an action on an object.
type need struct {
	action access.Action
	object resource.ID
}

// grantAll grants each need to u.
func grantAll(ctx SpecContext, u user.User, needs ...need) {
	GinkgoHelper()
	for _, n := range needs {
		grant(ctx, n.action, u.ResourceID(), n.object)
	}
}

// newDeviceNeeds returns what a create of a new device d on the embedded rack needs:
// create on d and its status, and update on the rack, which d sits in. d is the device,
// or the device type for any device.
func newDeviceNeeds(d resource.ID) []need {
	return []need{
		{access.ActionCreate, d},
		{access.ActionCreate, resource.ID{Type: resource.TypeStatus}},
		{access.ActionUpdate, svc.Rack.EmbeddedKey.ResourceID()},
	}
}

// statusOf returns the status that d holds.
func statusOf(ctx SpecContext, d device.Device) resource.ID {
	GinkgoHelper()
	var st device.Status
	Expect(svc.Status.NewRetrieve[device.StatusDetails]().
		Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
		Entry(&st).
		Exec(ctx, db)).To(Succeed())
	return st.ResourceID()
}

func newDevice(key string) device.Device {
	return device.Device{
		Key:      key,
		Rack:     svc.Rack.EmbeddedKey,
		Location: "dev1",
		Name:     "Before",
		Make:     "Test Make",
		Model:    "Test Model",
	}
}

func createDevice(ctx SpecContext, key string) device.Device {
	GinkgoHelper()
	d := newDevice(key)
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Device.NewWriter(tx).Create(ctx, &d)
	})).To(Succeed())
	return d
}

func retrieveDevice(ctx SpecContext, key string) device.Device {
	GinkgoHelper()
	var d device.Device
	Expect(svc.Device.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&d).
		Exec(ctx, db)).To(Succeed())
	return d
}

func setLiveStatus(ctx SpecContext, d device.Device) {
	GinkgoHelper()
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Status.NewWriter(tx).Set(ctx, &device.Status{
			Owner:   new(d.ResourceID()),
			Name:    d.Name,
			Time:    telem.Now(),
			Variant: status.VariantSuccess,
			Message: "Device is running",
			Details: device.StatusDetails{Rack: d.Rack},
		})
	})).To(Succeed())
}

var _ = Describe("Create", func() {
	It(
		"Should leave out the status of each device the caller cannot retrieve",
		func(ctx SpecContext) {
			u := createUser(ctx)
			d := newDevice("create-only-hidden")
			grantAll(ctx, u, newDeviceNeeds(d.ResourceID())...)
			res := MustSucceed(create(
				AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			))
			Expect(res.Devices).To(HaveLen(1))
			Expect(res.Devices[0].Status).To(BeNil())
			Expect(retrieveDevice(ctx, d.Key).Key).To(Equal(d.Key))
		},
	)
	DescribeTable(
		"Should refuse a new device and store nothing",
		func(ctx SpecContext, refused int, key string) {
			u := createUser(ctx)
			d := newDevice(key)
			needs := newDeviceNeeds(d.ResourceID())
			grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
			Expect(create(
				AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			)).Error().To(denied(u, needs[refused].action, needs[refused].object))
			Expect(svc.Device.NewRetrieve().
				Where(device.MatchKeys(key)).
				Exists(ctx, db)).To(BeFalse())
		},
		Entry("without create on the device", 0, "create-denied-device"),
		Entry("without update on its rack", 2, "create-denied-rack"),
	)
	authedCtx := func(ctx SpecContext) context.Context {
		GinkgoHelper()
		u := createUser(ctx)
		grantAll(ctx, u, newDeviceNeeds(deviceType)...)
		grant(ctx, access.ActionUpdate, u.ResourceID(), deviceType)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), deviceType)
		return AuthedCtx(ctx, u)
	}
	It("Should check read access on each device in a batch", func(ctx SpecContext) {
		u := createUser(ctx)
		readable := newDevice("batch-readable")
		hidden := newDevice("batch-hidden")
		grantAll(ctx, u, newDeviceNeeds(deviceType)...)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), readable.ResourceID())
		res := MustSucceed(create(
			AuthedCtx(ctx, u),
			apidevice.CreateRequest{Devices: []device.Device{readable, hidden}},
		))
		Expect(res.Devices).To(HaveLen(2))
		Expect(res.Devices[0].Status.Owner).To(HaveValue(Equal(readable.ResourceID())))
		Expect(res.Devices[0].Status.Message).To(Equal("Device status unknown"))
		Expect(res.Devices[1].Status).To(BeNil())
	})
	It("Should refuse a device whose key exists", func(ctx SpecContext) {
		d := createDevice(ctx, "create-existing")
		setLiveStatus(ctx, d)
		edited := d
		edited.Status = nil
		edited.Name = "After"
		Expect(create(
			authedCtx(ctx),
			apidevice.CreateRequest{Devices: []device.Device{edited}},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("device create-existing already exists")),
		))
		Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Before"))
		var stored device.Status
		Expect(svc.Status.NewRetrieve[device.StatusDetails]().
			Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.Message).To(Equal("Device is running"))
	})
})

var _ = Describe("Dispatch rename", func() {
	renameTo := func(key device.Key, name string) apidevice.DispatchRequest {
		return apidevice.DispatchRequest{
			Key: key,
			Actions: []device.Action{device.NewSetAction(device.SetPayload{
				Path:  []string{"name"},
				Value: name,
			})},
		}
	}
	It("Should rename the device and its status with update on both", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-granted")
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			d.ResourceID(),
			statusOf(ctx, d),
		)
		MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			renameTo(d.Key, "After"),
		))
		renamed := retrieveDevice(ctx, d.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Location).To(Equal(d.Location))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-empty")
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			d.ResourceID(),
			statusOf(ctx, d),
		)
		Expect(dispatch(
			AuthedCtx(ctx, u),
			renameTo(d.Key, ""),
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing device", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), device.ResourceID("missing"))
		Expect(dispatch(
			AuthedCtx(ctx, u),
			renameTo("missing", "After"),
		)).Error().To(MatchError(query.ErrNotFound))
	})
	deviceID := func(_ SpecContext, d device.Device) resource.ID {
		return d.ResourceID()
	}
	DescribeTable(
		"Should refuse the rename and keep the name",
		func(
			ctx SpecContext,
			key string,
			granted func(SpecContext, device.Device) resource.ID,
			refused func(SpecContext, device.Device) resource.ID,
		) {
			u := createUser(ctx)
			d := createDevice(ctx, key)
			grant(ctx, access.ActionUpdate, u.ResourceID(), granted(ctx, d))
			Expect(dispatch(
				AuthedCtx(ctx, u),
				renameTo(d.Key, "After"),
			)).Error().To(denied(u, access.ActionUpdate, refused(ctx, d)))
			Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Before"))
		},
		Entry("without update on the device", "rename-denied", statusOf, deviceID),
		Entry(
			"without update on its status",
			"rename-denied-status",
			deviceID,
			statusOf,
		),
	)
})

var _ = Describe("Delete", func() {
	// deleteNeeds returns what a delete of d needs: delete on d and its status, and
	// update on the rack that d leaves.
	deleteNeeds := func(ctx SpecContext, d device.Device) []need {
		GinkgoHelper()
		return []need{
			{access.ActionDelete, d.ResourceID()},
			{access.ActionDelete, statusOf(ctx, d)},
			{access.ActionUpdate, d.Rack.ResourceID()},
		}
	}
	exists := func(ctx SpecContext, key string) bool {
		GinkgoHelper()
		return MustSucceed(svc.Device.NewRetrieve().
			Where(device.MatchKeys(key)).
			Exists(ctx, db))
	}
	It("Should delete the device and its status", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "delete-granted")
		grantAll(ctx, u, deleteNeeds(ctx, d)...)
		MustSucceed(del(
			AuthedCtx(ctx, u),
			apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
		))
		Expect(exists(ctx, d.Key)).To(BeFalse())
		Expect(svc.Status.NewRetrieve[device.StatusDetails]().
			Where(status.MatchOwners[device.StatusDetails](d.ResourceID())).
			Entry(&device.Status{}).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})
	DescribeTable(
		"Should refuse the delete and keep the device",
		func(ctx SpecContext, refused int, key string) {
			u := createUser(ctx)
			d := createDevice(ctx, key)
			needs := deleteNeeds(ctx, d)
			grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
			Expect(del(
				AuthedCtx(ctx, u),
				apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
			)).Error().To(denied(u, needs[refused].action, needs[refused].object))
			Expect(exists(ctx, d.Key)).To(BeTrue())
		},
		Entry("without delete on the device", 0, "delete-denied-device"),
		Entry("without delete on its status", 1, "delete-denied-status"),
		Entry("without update on its rack", 2, "delete-denied-rack"),
	)
})

var _ = Describe("Retrieve", func() {
	It("Should leave out the devices that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		granted := createDevice(ctx, "leave-out-granted")
		createDevice(ctx, "leave-out-denied")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apidevice.RetrieveRequest{}))
		Expect(res.Devices).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

	It("Should attach only each device's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := createDevice(ctx, "retrieve-with-status")
		withoutStatus := createDevice(ctx, "retrieve-without-status")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Status.NewWriter(tx).DeleteOwned(ctx, withoutStatus.ResourceID())
		})).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.ResourceID(),
			withStatus.ResourceID(), withoutStatus.ResourceID(),
		)
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{
				Where: &device.Where{Key: &filter.Set[device.Key]{
					In: []device.Key{withoutStatus.Key, withStatus.Key},
				}},
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
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Device.NewWriter(tx).Create(ctx, &module)
		})).To(Succeed())
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{
				Where: &device.Where{Parents: []device.Key{chassis.Key}},
			},
		))
		Expect(res.Devices).To(HaveLen(1))
		Expect(res.Devices[0].Key).To(Equal(module.Key))
		Expect(res.Devices[0].Parent).To(Equal(&chassis.Key))
		res = MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{Where: &device.Where{
				Key: &filter.Set[device.Key]{
					In: []device.Key{chassis.Key, module.Key},
				},
				Nested: new(false),
			}},
		))
		Expect(res.Devices).To(HaveLen(1))
		Expect(res.Devices[0].Key).To(Equal(chassis.Key))
	})

	It("Should treat a missing key as a non-match", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "present_device")
		grant(
			ctx,
			access.ActionRetrieve,
			u.ResourceID(),
			d.ResourceID(),
			device.ResourceID("missing_device"),
		)
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{
				Where: &device.Where{Key: &filter.Set[device.Key]{
					In: []device.Key{d.Key, "missing_device"},
				}},
			},
		))
		Expect(res.Devices).To(HaveLen(1))
		Expect(res.Devices[0].Key).To(Equal(d.Key))
	})

	It("Should return a path error for an invalid regex", func(ctx SpecContext) {
		u := createUser(ctx)
		Expect(retrieve(AuthedCtx(ctx, u), apidevice.RetrieveRequest{
			Where: &device.Where{Make: &filter.String{Regex: new("(a")}},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"where.make.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})

var _ = Describe("RetrieveRequest", func() {
	It("Should reject a request member that it does not declare", func() {
		var req apidevice.RetrieveRequest
		Expect(json.Unmarshal([]byte(`{"keys":["dev1"]}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "keys"`)),
		))
	})
})

// createRackGroup creates a group under the embedded rack, where its devices can sit.
func createRackGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: new(svc.Rack.EmbeddedKey.ResourceID())}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Group.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

var _ = Describe("Dispatch group move", func() {
	// denied matches an error that denies the subject the action on object.
	denied := func(
		subject resource.ID,
		action access.Action,
		object resource.ID,
	) types.GomegaMatcher {
		return SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring("%s cannot %s %s", subject, action, object)),
		)
	}
	move := func(
		ctx SpecContext,
		u user.User,
		g group.Group,
		d device.Device,
	) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apidevice.DispatchRequest{
			Key: d.Key,
			Actions: []device.Action{device.NewSetAction(device.SetPayload{
				Path:  []string{"group"},
				Value: g.Key.String(),
			})},
		})
	}
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			d := createDevice(ctx, "set-group-denied")
			g := createRackGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), d.ResourceID())
			Expect(move(ctx, u, g, d)).
				Error().To(denied(u.ResourceID(), access.ActionUpdate, g.ResourceID()))
			Expect(retrieveDevice(ctx, d.Key).Group).To(BeNil())
		},
	)
	It(
		"Should refuse the move without update on the source group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			d := createDevice(ctx, "set-group-source-denied")
			from := createRackGroup(ctx, "Set group source denied")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Device.NewWriter(tx).SetGroup(ctx, &from.Key, d.Key)
			})).
				To(Succeed())
			to := createRackGroup(ctx, "Set group destination granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				d.ResourceID(),
				to.ResourceID(),
			)
			Expect(move(ctx, u, to, d)).
				Error().
				To(denied(u.ResourceID(), access.ActionUpdate, from.ResourceID()))
			Expect(retrieveDevice(ctx, d.Key).Group).To(Equal(&from.Key))
		},
	)
	It("Should refuse the move without update on the device", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "set-group-device-denied")
		g := createRackGroup(ctx, "Set group device denied")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		Expect(move(ctx, u, g, d)).
			Error().To(denied(u.ResourceID(), access.ActionUpdate, d.ResourceID()))
		Expect(retrieveDevice(ctx, d.Key).Group).To(BeNil())
	})
	It(
		"Should move the device with update on the device and both groups",
		func(ctx SpecContext) {
			u := createUser(ctx)
			d := createDevice(ctx, "set-group-granted")
			from := createRackGroup(ctx, "Set group source granted")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Device.NewWriter(tx).SetGroup(ctx, &from.Key, d.Key)
			})).
				To(Succeed())
			to := createRackGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				d.ResourceID(),
				from.ResourceID(),
				to.ResourceID(),
			)
			MustSucceed(move(ctx, u, to, d))
			Expect(retrieveDevice(ctx, d.Key).Group).To(Equal(&to.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	inGroup := func(key string, g group.Group) device.Device {
		d := newDevice(key)
		d.Group = &g.Key
		return d
	}
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createRackGroup(ctx, "Create denied")
			grantAll(ctx, u, newDeviceNeeds(deviceType)...)
			Expect(create(
				AuthedCtx(ctx, u),
				apidevice.CreateRequest{
					Devices: []device.Device{inGroup("create-group-denied", g)},
				},
			)).Error().To(denied(u, access.ActionUpdate, g.ResourceID()))
		},
	)
	It("Should create the device with update on the group", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createRackGroup(ctx, "Create granted")
		grantAll(ctx, u, newDeviceNeeds(deviceType)...)
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		MustSucceed(create(
			AuthedCtx(ctx, u),
			apidevice.CreateRequest{
				Devices: []device.Device{inGroup("create-group-granted", g)},
			},
		))
		Expect(retrieveDevice(ctx, "create-group-granted").Group).To(Equal(&g.Key))
	})
})

var _ = Describe("Dispatch", func() {
	locate := apidevice.DispatchRequest{
		DispatchKey: "dk",
		Actions: []device.Action{device.NewSetAction(device.SetPayload{
			Path:  []string{"location"},
			Value: "Bay 2",
		})},
	}
	It("Should apply the actions with update on the device", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "dispatch-granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), d.ResourceID())
		req := locate
		req.Key = d.Key
		MustSucceed(dispatch(AuthedCtx(ctx, u), req))
		Expect(retrieveDevice(ctx, d.Key).Location).To(Equal("Bay 2"))
	})
	It("Should refuse the actions without update on the device", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		d := createDevice(ctx, "dispatch-denied")
		req := locate
		req.Key = d.Key
		Expect(dispatch(AuthedCtx(ctx, u), req)).
			Error().To(denied(u, access.ActionUpdate, d.ResourceID()))
		Expect(retrieveDevice(ctx, d.Key).Location).To(Equal(d.Location))
	})
})

var _ = Describe("Move to another rack", func() {
	var (
		u     user.User
		g     group.Group
		d     device.Device
		other rack.Rack
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		g = createRackGroup(ctx, "Rack move")
		d = createDevice(ctx, "rack-move-"+uuid.New().String())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Device.NewWriter(tx).SetGroup(ctx, &g.Key, d.Key)
		})).To(Succeed())
		other = rack.Rack{Name: "Rack move"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Rack.NewWriter(tx).Create(ctx, &other)
		})).To(Succeed())
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			d.ResourceID(),
			statusOf(ctx, d),
			d.Rack.ResourceID(),
			other.ResourceID(),
		)
	})
	setRack := func(ctx SpecContext) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apidevice.DispatchRequest{
			Key: d.Key,
			Actions: []device.Action{device.NewSetRackAction(
				device.SetRackPayload{Rack: other.Key},
			)},
		})
	}
	It("Should refuse to move a device without update on its group", func(
		ctx SpecContext,
	) {
		Expect(setRack(ctx)).
			Error().To(denied(u, access.ActionUpdate, g.ResourceID()))
		Expect(retrieveDevice(ctx, d.Key).Rack).To(Equal(d.Rack))
	})
	It("Should move the device and drop its group with update on it", func(
		ctx SpecContext,
	) {
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		MustSucceed(setRack(ctx))
		moved := retrieveDevice(ctx, d.Key)
		Expect(moved.Rack).To(Equal(other.Key))
		Expect(moved.Group).To(BeNil())
	})
})
