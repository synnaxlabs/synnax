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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	var (
		db       *gorp.DB
		groupSvc *group.Service
		rackSvc  *rack.Service
		stat     *status.Service
		svc      *device.Service
		rec      *Recorder[device.Key, device.Action]
		dev      device.Device
	)
	write := func(ctx context.Context, fn func(device.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(svc.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	set := func(path string, value any) device.Action {
		return device.NewSetAction(
			device.SetPayload{Path: []string{path}, Value: value},
		)
	}
	retrieve := func(ctx context.Context) device.Device {
		GinkgoHelper()
		var res device.Device
		Expect(svc.NewRetrieve().
			Where(device.MatchKeys(dev.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	retrieveStatus := func(ctx context.Context) device.Status {
		GinkgoHelper()
		var res device.Status
		Expect(stat.NewRetrieve[device.StatusDetails]().
			Where(status.MatchOwners[device.StatusDetails](dev.ResourceID())).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roles := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roles)),
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
			Rack:   rackSvc,
		}))
		rec = &Recorder[device.Key, device.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		dev = device.Device{
			Key:        "dev",
			Rack:       rackSvc.EmbeddedKey,
			Location:   "dev1",
			Name:       "Pump",
			Make:       "Make",
			Model:      "Model",
			Properties: msgpack.EncodedJSON{},
		}
		Expect(write(ctx, func(w device.Writer) error {
			return w.Create(ctx, &dev)
		})).To(Succeed())
	})

	It("Should attach a create that holds the device and its status", func() {
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].DispatchKey).To(Equal("dk"))
		Expect(frames[0].Actions).To(Equal([]device.Action{
			device.NewCreateAction(device.CreatePayload{Device: dev}),
		}))
		Expect(dev.Status).ToNot(BeNil())
	})

	DescribeTable("Should store an editable field", func(
		ctx SpecContext,
		path string,
		value any,
		expect func(*device.Device),
	) {
		act := set(path, value)
		Expect(write(ctx, func(w device.Writer) error {
			return w.Apply(ctx, dev.Key, act)
		})).To(Succeed())
		expected := dev
		expected.Status = nil
		expect(&expected)
		Expect(retrieve(ctx)).To(Equal(expected))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(2))
		Expect(frames[1].DispatchKey).To(Equal("dk"))
		Expect(frames[1].Actions).To(Equal([]device.Action{act}))
	},
		Entry("name", "name", "Valve", func(d *device.Device) { d.Name = "Valve" }),
		Entry("model", "model", "Model 2", func(d *device.Device) {
			d.Model = "Model 2"
		}),
		Entry("location", "location", "dev2", func(d *device.Device) {
			d.Location = "dev2"
		}),
		Entry("configured", "configured", true, func(d *device.Device) {
			d.Configured = true
		}),
		Entry("properties", "properties", map[string]any{"port": 2.0}, func(
			d *device.Device,
		) {
			d.Properties = msgpack.EncodedJSON{"port": 2.0}
		}),
	)

	DescribeTable("Should refuse a path outside the editable fields", func(
		ctx SpecContext,
		path string,
		value any,
	) {
		Expect(write(ctx, func(w device.Writer) error {
			return w.Apply(ctx, dev.Key, set(path, value))
		})).To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx).Make).To(Equal("Make"))
		Expect(rec.Snapshot()).To(HaveLen(1))
	},
		Entry("key", "key", "other"),
		Entry("make", "make", "Other"),
		Entry("rack", "rack", 7),
		Entry("parent", "parent", "other"),
	)

	It("Should refuse an empty model", func(ctx SpecContext) {
		Expect(write(ctx, func(w device.Writer) error {
			return w.Apply(ctx, dev.Key, set("model", ""))
		})).To(MatchError(ContainSubstring("model: required")))
		Expect(retrieve(ctx).Model).To(Equal(dev.Model))
		Expect(rec.Snapshot()).To(HaveLen(1))
	})

	It("Should reject a create", func(ctx SpecContext) {
		Expect(write(ctx, func(w device.Writer) error {
			return w.Apply(
				ctx,
				dev.Key,
				device.NewCreateAction(device.CreatePayload{Device: dev}),
			)
		})).To(MatchError(ContainSubstring("only the Core dispatches a device create")))
	})

	It("Should rename the status with the device", func(ctx SpecContext) {
		Expect(write(ctx, func(w device.Writer) error {
			return w.Rename(ctx, dev.Key, "Valve")
		})).To(Succeed())
		Expect(retrieveStatus(ctx).Name).To(Equal("Valve"))
		Expect(
			rec.Snapshot()[1].Actions,
		).To(Equal([]device.Action{set("name", "Valve")}))
	})

	Describe("Group", func() {
		var g group.Group
		BeforeEach(func(ctx SpecContext) {
			g = group.Group{
				Name:   "Pumps",
				Parent: new(rackSvc.EmbeddedKey.ResourceID()),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
		})

		It("Should attach a set of the group for each device", func(ctx SpecContext) {
			Expect(write(ctx, func(w device.Writer) error {
				return w.SetGroup(ctx, &g.Key, dev.Key)
			})).To(Succeed())
			Expect(retrieve(ctx).Group).To(Equal(&g.Key))
			Expect(rec.Snapshot()[1].Actions).To(Equal([]device.Action{
				set("group", &g.Key),
			}))
		})

		It("Should refuse a group that does not sit under the rack", func(
			ctx SpecContext,
		) {
			top := group.Group{Name: "Top", Scope: "device"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &top)
			})).To(Succeed())
			Expect(write(ctx, func(w device.Writer) error {
				return w.Apply(ctx, dev.Key, set("group", top.Key.String()))
			})).To(MatchError(ContainSubstring("does not sit under rack")))
			Expect(retrieve(ctx).Group).To(BeNil())
		})

		It("Should move the device to a rack and out of its group", func(
			ctx SpecContext,
		) {
			other := rack.Rack{Name: "Other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := rackSvc.NewWriter(tx).Create(ctx, &other); err != nil {
					return err
				}
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, dev.Key)
			})).To(Succeed())
			Expect(write(ctx, func(w device.Writer) error {
				return w.Apply(
					ctx,
					dev.Key,
					device.NewSetRackAction(device.SetRackPayload{Rack: other.Key}),
				)
			})).To(Succeed())
			moved := retrieve(ctx)
			Expect(moved.Rack).To(Equal(other.Key))
			Expect(moved.Group).To(BeNil())
			Expect(retrieveStatus(ctx).Details.Rack).To(Equal(other.Key))
		})

		It("Should move the device under a parent and out of its group", func(
			ctx SpecContext,
		) {
			parent := device.Device{
				Key:        "parent",
				Rack:       rackSvc.EmbeddedKey,
				Location:   "parent1",
				Name:       "Parent",
				Make:       "Make",
				Model:      "Model",
				Properties: msgpack.EncodedJSON{},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).Create(ctx, &parent); err != nil {
					return err
				}
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, dev.Key)
			})).To(Succeed())
			setRack := device.NewSetRackAction(device.SetRackPayload{
				Rack:   rackSvc.EmbeddedKey,
				Parent: &parent.Key,
			})
			Expect(write(ctx, func(w device.Writer) error {
				return w.Apply(ctx, dev.Key, setRack)
			})).To(Succeed())
			moved := retrieve(ctx)
			Expect(moved.Rack).To(Equal(rackSvc.EmbeddedKey))
			Expect(moved.Parent).To(Equal(&parent.Key))
			Expect(moved.Group).To(BeNil())
			frames := rec.Snapshot()
			Expect(frames[len(frames)-1].Actions).To(Equal([]device.Action{setRack}))
		})
	})
	Describe("Properties", func() {
		at := func(path ...string) []string {
			return append([]string{"properties", "read"}, path...)
		}
		setAt := func(value any, path ...string) device.Action {
			return device.NewSetAction(
				device.SetPayload{Path: at(path...), Value: value},
			)
		}
		stored := func() msgpack.EncodedJSON {
			return msgpack.EncodedJSON{
				"host": "10.0.0.1",
				"read": map[string]any{
					"channels": map[string]any{"a": 1.0},
					"nodes":    []any{map[string]any{"key": "n1", "port": 1.0}},
				},
			}
		}
		apply := func(ctx context.Context, acts ...device.Action) error {
			return write(ctx, func(w device.Writer) error {
				return w.Apply(ctx, dev.Key, acts...)
			})
		}
		BeforeEach(func(ctx SpecContext) {
			Expect(apply(ctx, set("properties", map[string]any(stored())))).
				To(Succeed())
		})

		It("Should keep each key that the sets of one dispatch add to a map", func(
			ctx SpecContext,
		) {
			Expect(apply(
				ctx,
				setAt(2, "channels", "b"),
				setAt(3, "channels", "c"),
			)).To(Succeed())
			expected := stored()
			expected["read"].(map[string]any)["channels"] = map[string]any{
				"a": 1.0, "b": 2.0, "c": 3.0,
			}
			Expect(retrieve(ctx).Properties).To(Equal(expected))
		})

		It("Should remove a key inside the properties", func(ctx SpecContext) {
			Expect(apply(ctx, device.NewRemoveAction(
				device.RemovePayload{Path: at("channels"), Key: "a"},
			))).To(Succeed())
			expected := stored()
			expected["read"].(map[string]any)["channels"] = map[string]any{}
			Expect(retrieve(ctx).Properties).To(Equal(expected))
		})

		It("Should set a value inside a list entry", func(ctx SpecContext) {
			Expect(apply(ctx, setAt(2, "nodes", "n1", "port"))).To(Succeed())
			expected := stored()
			expected["read"].(map[string]any)["nodes"] = []any{
				map[string]any{"key": "n1", "port": 2.0},
			}
			Expect(retrieve(ctx).Properties).To(Equal(expected))
		})

		It("Should apply path actions to a copy of the properties", func() {
			prev := device.Device{Properties: stored()}
			next := MustSucceed(device.Reduce(
				prev,
				setAt(2, "channels", "b"),
				device.NewRemoveAction(
					device.RemovePayload{Path: at("channels"), Key: "a"},
				),
				device.NewInsertAction(device.InsertPayload{
					Path:  at("nodes"),
					Value: map[string]any{"key": "n2"},
				}),
				device.NewMoveAction(device.MovePayload{Path: at("nodes"), Key: "n1"}),
			))
			Expect(prev.Properties).To(Equal(stored()))
			expected := stored()
			expected["read"] = map[string]any{
				"channels": map[string]any{"b": 2.0},
				"nodes": []any{
					map[string]any{"key": "n1", "port": 1.0},
					map[string]any{"key": "n2"},
				},
			}
			Expect(next.Properties).To(Equal(expected))
		})

		DescribeTable("Should apply a set only when its guard on the model holds", func(
			ctx SpecContext,
			model string,
			applied bool,
		) {
			Expect(apply(ctx, device.NewSetAction(device.SetPayload{
				Path:   at("channels", "b"),
				Value:  2,
				Guards: []patch.Guard{{Path: []string{"model"}, Value: model}},
			}))).To(Succeed())
			expected := stored()
			if applied {
				expected["read"].(map[string]any)["channels"] = map[string]any{
					"a": 1.0, "b": 2.0,
				}
			}
			Expect(retrieve(ctx).Properties).To(Equal(expected))
		},
			Entry("holds", "Model", true),
			Entry("does not hold", "Other", false),
		)

		It("Should apply a set whose guard on the group holds", func() {
			key := uuid.New()
			next := MustSucceed(device.Reduce(
				device.Device{Group: &key, Properties: stored()},
				device.NewSetAction(device.SetPayload{
					Path:  at("channels", "b"),
					Value: 2,
					Guards: []patch.Guard{
						{Path: []string{"group"}, Value: key.String()},
					},
				}),
			))
			Expect(next.Properties["read"]).To(HaveKeyWithValue(
				"channels",
				map[string]any{"a": 1.0, "b": 2.0},
			))
		})

		It("Should refuse properties that are not an object", func(ctx SpecContext) {
			Expect(apply(ctx, set("properties", "text"))).
				To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx).Properties).To(Equal(stored()))
		})
	})
})
