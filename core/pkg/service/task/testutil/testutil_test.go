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
	"github.com/samber/lo"
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
	"github.com/synnaxlabs/synnax/pkg/service/task"
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
		db     *gorp.DB
		svc    *arctask.Service
		tasks  *tasktestutil.Tasks
		arcKey arc.Key
	)
	createArc := func(ctx SpecContext) arc.Key {
		GinkgoHelper()
		a := arc.Arc{
			Variant: arc.AutomationArc{Base: arc.Base{
				Key:    uuid.New(),
				Name:   "arc",
				Source: arc.Source{Variant: arc.TextSource{}},
			}},
		}
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		return a.Base().Key
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db, Ontology: otg}))
		arcKey = createArc(ctx)
		svc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:        db,
			Stores:    svc.Stores(),
			MatchArcs: svc.MatchArcs,
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

	It("Should filter the tasks by Arc", func(ctx SpecContext) {
		create := func(a arc.Key) task.Key {
			GinkgoHelper()
			return MustSucceed(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
				"arc_key": a.String(),
			}))
		}
		first, second := create(arcKey), create(arcKey)
		create(createArc(ctx))
		var matched []task.Task
		Expect(tasks.NewRetrieve().
			Where(task.MatchArcs(arcKey)).
			Entries(&matched).
			Exec(ctx, db)).To(Succeed())
		Expect(lo.Map(matched, func(t task.Task, _ int) task.Key { return t.Key })).
			To(ConsistOf(first, second))
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
