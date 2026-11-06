// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// openTargets opens the device and status services on a fresh DB and returns them,
// with the embedded rack and the Targets that create through them.
func openTargets(ctx SpecContext) (
	*device.Service,
	*status.Service,
	rack.Key,
	*tasktestutil.Targets,
) {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	roles := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           db,
		Ontology:     otg,
		HostProvider: testutil.NewStaticHostProvider(1),
		Status:       statuses,
		DriverRole:   MustSucceed(racktestutil.CreateDriverRole(ctx, roles)),
	}))
	devices := MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Status:   statuses,
		Rack:     racks,
	}))
	return devices, statuses, racks.EmbeddedKey, tasktestutil.NewTargets(
		tasktestutil.TargetsConfig{
			Device: devices,
			Status: statuses,
			Rack:   racks.EmbeddedKey,
		},
	)
}

var _ = Describe("Tasks", func() {
	var (
		svc    *arctask.Service
		tasks  *tasktestutil.Tasks
		arcKey arc.Key
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		a := arc.Arc{Key: uuid.New(), Name: "arc"}
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		arcKey = a.Key
		svc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: svc.Stores(),
		}))
	})

	It("Should create a task whose config lives in the store", func(ctx SpecContext) {
		key := MustSucceed(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
			"arc_key": arcKey.String(),
		}))
		data := MustSucceed(svc.Config.Read(ctx, nil, key))
		Expect(data["arc_key"]).To(Equal(arcKey.String()))
		Expect(data["performance"]).To(Equal("auto"))
	})

	It("Should return the error of a config the store rejects", func(ctx SpecContext) {
		Expect(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
			"arc_key":     uuid.New().String(),
			"performance": "BOGUS",
		})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
	})
})

var _ = Describe("CreateDevice", func() {
	It("Should create a device on the configured rack", func(ctx SpecContext) {
		devices, _, rackKey, targets := openTargets(ctx)
		Expect(targets.CreateDevice(ctx, "dev-1")).To(Succeed())
		var d device.Device
		Expect(devices.NewRetrieve().
			Where(device.MatchKeys("dev-1")).
			Entry(&d).
			Exec(ctx, nil)).To(Succeed())
		Expect(d.Rack).To(Equal(rackKey))
		Expect(d.Name).To(Equal("dev-1"))
	})
})

var _ = Describe("CreateStatus", func() {
	It("Should create an info status named after its key", func(ctx SpecContext) {
		_, statuses, _, targets := openTargets(ctx)
		Expect(targets.CreateStatus(ctx, "status-1")).To(Succeed())
		var s status.Status[any]
		Expect(statuses.NewRetrieve[any]().
			Where(status.MatchKeys[any]("status-1")).
			Entry(&s).
			Exec(ctx, nil)).To(Succeed())
		Expect(s.Name).To(Equal("status-1"))
		Expect(s.Variant).To(Equal(status.VariantInfo))
	})
})
