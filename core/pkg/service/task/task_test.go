// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Task", Ordered, func() {
	var (
		db          *gorp.DB
		svc         *task.Service
		otg         *ontology.Ontology
		w           task.Writer
		tx          gorp.Tx
		rackService *rack.Service
		testRack    *rack.Rack
		stat        *status.Service
	)
	BeforeAll(func(ctx SpecContext) {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustSucceed(search.Open())
		DeferCleanup(func() {
			Expect(searchIdx.Close()).To(Succeed())
		})
		g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		stat = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackService = MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               g,
			HostProvider:        mock.StaticHostKeyProvider(1),
			Status:              stat,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		svc = MustSucceed(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackService,
			Status:   stat,
			Search:   searchIdx,
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(rackService.NewWriter(db).Create(ctx, testRack)).To(Succeed())
	})
	BeforeEach(func(ctx SpecContext) {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func(ctx SpecContext) {
		Expect(tx.Close()).To(Succeed())
	})
	AfterAll(func(ctx SpecContext) {
		Expect(svc.Close()).To(Succeed())
		Expect(rackService.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	Describe("Task", func() {
		It("Should construct and deconstruct a key from its components", func(ctx SpecContext) {
			rk := rack.NewKey(cluster.NodeKey(1), 1)
			k := task.NewKey(rk, 2)
			Expect(k.Rack()).To(Equal(rk))
			Expect(k.LocalKey()).To(Equal(uint32(2)))
		})
	})
	Describe("CommandChannelKey", func() {
		It("Should return zero when no channel service is configured", func(ctx SpecContext) {
			Expect(svc.CommandChannelKey()).To(Equal(channel.Key(0)))
		})
	})
	Describe("Key msgpack decoding", func() {
		var codec = msgpack.Codec
		DescribeTable("Should decode task.Key from various types",
			func(ctx SpecContext, value any, expected task.Key) {
				data := MustSucceed(codec.Encode(ctx, value))
				var k task.Key
				Expect(codec.Decode(ctx, data, &k)).To(Succeed())
				Expect(k).To(Equal(expected))
			},
			Entry("string", "281543696187399", task.Key(281543696187399)),
			Entry("uint64", uint64(281543696187399), task.Key(281543696187399)),
			Entry("uint32", uint32(123456), task.Key(123456)),
			Entry("int64", int64(123456789), task.Key(123456789)),
			Entry("int32", int32(123456), task.Key(123456)),
			Entry("float64", float64(123456), task.Key(123456)),
			Entry("float32", float32(1234), task.Key(1234)),
		)
		It("Should decode StatusDetails with task key as string", func(ctx SpecContext) {
			type statusDetailsWithString struct {
				Data    map[string]any `msgpack:"data"`
				Task    string         `msgpack:"task"`
				Running bool           `msgpack:"running"`
			}
			original := statusDetailsWithString{
				Task:    "281543696187399",
				Running: true,
				Data:    map[string]any{"test": true},
			}
			data := MustSucceed(codec.Encode(ctx, original))
			var decoded task.StatusDetails
			Expect(codec.Decode(ctx, data, &decoded)).To(Succeed())
			Expect(decoded.Task).To(Equal(task.Key(281543696187399)))
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should decode StatusDetails with task key as float64", func(ctx SpecContext) {
			type statusDetailsWithFloat struct {
				Data    map[string]any `msgpack:"data"`
				Task    float64        `msgpack:"task"`
				Running bool           `msgpack:"running"`
			}
			original := statusDetailsWithFloat{
				Task:    float64(65536),
				Running: true,
				Data:    map[string]any{"test": true},
			}
			data := MustSucceed(codec.Encode(ctx, original))
			var decoded task.StatusDetails
			Expect(codec.Decode(ctx, data, &decoded)).To(Succeed())
			Expect(decoded.Task).To(Equal(task.Key(65536)))
			Expect(decoded.Running).To(BeTrue())
		})
	})

	Describe("Create", func() {
		It("Should correctly create a task and assign it a unique key", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
		})
		It("Should correctly increment the task count", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 2)))
			Expect(m.Name).To(Equal("Test Task"))
			m = &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 3)))
			Expect(m.Name).To(Equal("Test Task"))
		})
	})

	Describe("Copy", func() {

		It("Should copy a task", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 4)))
			Expect(m.Name).To(Equal("Test Task"))
			t := MustSucceed(w.Copy(ctx, m.Key, "Copied Task", false))
			Expect(t.Key).To(Equal(task.NewKey(testRack.Key, 5)))
		})

		It("Should create a snapshot of an existing task", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 6)))
			Expect(m.Name).To(Equal("Test Task"))
			t := MustSucceed(w.Copy(ctx, m.Key, "Snapshotted Task", true))
			Expect(t.Key).To(Equal(task.NewKey(testRack.Key, 7)))
			Expect(t.Snapshot).To(BeTrue())
		})

	})

	Describe("Retrieve", func() {
		It("Should correctly retrieve a task", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 8)))
			Expect(m.Name).To(Equal("Test Task"))
			var res task.Task
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*m))
		})

		It("Should filter tasks by snapshot status", func(ctx SpecContext) {
			regular := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Regular Task",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			snapshot := &task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Name:     "Snapshot Task",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot)).To(Succeed())
			var snapshots []task.Task
			Expect(svc.NewRetrieve().WhereSnapshot(true).Entries(&snapshots).Exec(ctx, tx)).To(Succeed())
			Expect(snapshots).To(HaveLen(1))
			Expect(snapshots[0].Name).To(Equal("Snapshot Task"))
			Expect(snapshots[0].Snapshot).To(BeTrue())
			var regulars []task.Task
			Expect(svc.NewRetrieve().WhereSnapshot(false).Entries(&regulars).Exec(ctx, tx)).To(Succeed())
			Expect(len(regulars)).To(BeNumerically(">", 0))
			for _, t := range regulars {
				Expect(t.Snapshot).To(BeFalse())
			}
		})

		It("Should combine WhereSnapshot with other filters", func(ctx SpecContext) {
			snapshot1 := &task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Name:     "Snapshot Task 1",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot1)).To(Succeed())
			snapshot2 := &task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Name:     "Snapshot Task 2",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot2)).To(Succeed())
			var res task.Task
			Expect(svc.NewRetrieve().WhereSnapshot(true).WhereNames("Snapshot Task 1").Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("Snapshot Task 1"))
			Expect(res.Snapshot).To(BeTrue())
		})

		It("Should filter tasks by internal status", func(ctx SpecContext) {
			regular := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Regular Task 2",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			internal := &task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Name:     "Internal Task",
				Internal: true,
			}
			Expect(w.Create(ctx, internal)).To(Succeed())
			var internals []task.Task
			Expect(svc.NewRetrieve().WhereInternal(true).Entries(&internals).Exec(ctx, tx)).To(Succeed())
			Expect(internals).To(HaveLen(1))
			Expect(internals[0].Name).To(Equal("Internal Task"))
			Expect(internals[0].Internal).To(BeTrue())
			var regulars []task.Task
			Expect(svc.NewRetrieve().WhereInternal(false).Entries(&regulars).Exec(ctx, tx)).To(Succeed())
			Expect(len(regulars)).To(BeNumerically(">", 0))
			for _, t := range regulars {
				Expect(t.Internal).To(BeFalse())
			}
		})

		It("Should combine WhereInternal with other filters", func(ctx SpecContext) {
			internal1 := &task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Name:     "Internal Task 1",
				Internal: true,
			}
			Expect(w.Create(ctx, internal1)).To(Succeed())
			internal2 := &task.Task{
				Key:      task.NewKey(testRack.Key, 0),
				Name:     "Internal Task 2",
				Internal: true,
			}
			Expect(w.Create(ctx, internal2)).To(Succeed())
			var res task.Task
			Expect(svc.NewRetrieve().WhereInternal(true).WhereNames("Internal Task 1").Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("Internal Task 1"))
			Expect(res.Internal).To(BeTrue())
		})
	})

	Describe("Delete", func() {
		It("Should correctly delete a task and its associated status", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(w.Delete(ctx, m.Key, false)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			var deletedStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				WhereKeys(task.OntologyID(m.Key).String()).
				Entry(&deletedStatus).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Status", func() {
		It("Should create an unknown status when creating a task", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Status Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())

			var taskStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				WhereKeys(task.OntologyID(m.Key).String()).
				Entry(&taskStatus).
				Exec(ctx, tx)).To(Succeed())
			Expect(taskStatus.Variant).To(Equal(xstatus.VariantWarning))
			Expect(taskStatus.Message).To(Equal("Status Test Task status unknown"))
			Expect(taskStatus.Details.Task).To(Equal(m.Key))
		})

		It("Should use the provided status when creating a task", func(ctx SpecContext) {
			providedStatus := &task.Status{
				Variant:     xstatus.VariantSuccess,
				Message:     "Custom task status",
				Description: "Task is running",
				Time:        telem.Now(),
				Details: task.StatusDetails{
					Running: true,
				},
			}
			m := &task.Task{
				Key:    task.NewKey(testRack.Key, 0),
				Name:   "Task with custom status",
				Status: providedStatus,
			}
			Expect(w.Create(ctx, m)).To(Succeed())

			var taskStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				WhereKeys(task.OntologyID(m.Key).String()).
				Entry(&taskStatus).
				Exec(ctx, tx)).To(Succeed())
			Expect(taskStatus.Variant).To(Equal(xstatus.VariantSuccess))
			Expect(taskStatus.Message).To(Equal("Custom task status"))
			Expect(taskStatus.Description).To(Equal("Task is running"))
			// Key should be auto-assigned
			Expect(taskStatus.Key).To(Equal(task.OntologyID(m.Key).String()))
			// Name should be auto-filled
			Expect(taskStatus.Name).To(Equal(m.Name))
			// Details.Task should be auto-filled
			Expect(taskStatus.Details.Task).To(Equal(m.Key))
			// Provided details should be preserved
			Expect(taskStatus.Details.Running).To(BeTrue())
		})

		It("Should return a validation error if provided status has empty variant", func(ctx SpecContext) {
			providedStatus := &task.Status{
				Time:    telem.Now(),
				Message: "Status with no variant",
			}
			m := &task.Task{
				Key:    task.NewKey(testRack.Key, 0),
				Name:   "Task with invalid status",
				Status: providedStatus,
			}
			Expect(w.Create(ctx, m)).Error().To(MatchError(ContainSubstring("variant")))
		})
		It("Should create an unknown status when copying a task", func(ctx SpecContext) {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Original Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())

			copied := MustSucceed(w.Copy(ctx, m.Key, "Copied Task", false))

			var copiedStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				WhereKeys(task.OntologyID(copied.Key).String()).
				Entry(&copiedStatus).
				Exec(ctx, tx)).To(Succeed())
			Expect(copiedStatus.Variant).To(Equal(xstatus.VariantWarning))
			Expect(copiedStatus.Message).To(Equal("Copied Task status unknown"))
			Expect(copiedStatus.Details.Task).To(Equal(copied.Key))
		})
	})

	Describe("Suspect Rack", func() {
		It("Should propagate rack warning status to tasks on that rack", func(ctx SpecContext) {
			r := rack.Rack{Name: "suspect rack"}
			Expect(rackService.NewWriter(nil).Create(ctx, &r)).To(Succeed())

			t := &task.Task{
				Key:  task.NewKey(r.Key, 0),
				Name: "Test Task",
			}
			Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())

			Eventually(func(g Gomega) {
				var taskStatus task.Status
				g.Expect(status.NewRetrieve[task.StatusDetails](stat).
					WhereKeys(task.OntologyID(t.Key).String()).
					Entry(&taskStatus).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(taskStatus.Variant).To(Equal(xstatus.VariantWarning))
				g.Expect(taskStatus.Message).To(ContainSubstring("not running"))
				g.Expect(taskStatus.Details.Task).To(Equal(t.Key))
			}).Should(Succeed())
		})
	})

	Describe("Command", func() {
		Describe("String", func() {
			It("Should return a string representation of the command", func(ctx SpecContext) {
				c := &task.Command{
					Key:  "cmd",
					Task: task.Key(12345),
					Type: "doc",
				}
				Expect(c.String()).To(Equal("doc (key=cmd, task=12345)"))
			})
		})
	})

	Describe("migration", func() {
		It("Should create unknown statuses for tasks missing them", func(ctx SpecContext) {
			db := gorp.Wrap(memkv.New())
			otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustSucceed(search.Open())
			g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
			stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
				Ontology: otg,
				DB:       db,
				Group:    g,
				Label:    labelSvc,
				Search:   searchIdx,
			}))
			rackSvc := MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
				DB:           db,
				Ontology:     otg,
				Group:        g,
				HostProvider: mock.StaticHostKeyProvider(1),
				Status:       stat,
				Search:       searchIdx,
			}))

			testRack := &rack.Rack{Name: "Migration Test Rack"}
			Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

			t := task.Task{
				Key:  task.NewKey(testRack.Key, 99),
				Name: "Migration Test Task",
			}
			Expect(gorp.NewCreate[task.Key, task.Task]().
				Entry(&t).
				Exec(ctx, db)).To(Succeed())

			svc := MustSucceed(task.OpenService(ctx, task.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Rack:     rackSvc,
				Status:   stat,
				Search:   searchIdx,
			}))

			var restoredStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				WhereKeys(task.OntologyID(t.Key).String()).
				Entry(&restoredStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(restoredStatus.Variant).To(Equal(xstatus.VariantWarning))
			Expect(restoredStatus.Message).To(Equal("Migration Test Task status unknown"))
			Expect(restoredStatus.Details.Task).To(Equal(t.Key))

			Expect(svc.Close()).To(Succeed())
			Expect(rackSvc.Close()).To(Succeed())
			Expect(stat.Close()).To(Succeed())
			Expect(labelSvc.Close()).To(Succeed())
			Expect(g.Close()).To(Succeed())
			Expect(searchIdx.Close()).To(Succeed())
			Expect(otg.Close()).To(Succeed())
			Expect(db.Close()).To(Succeed())
		})

		It("Should not create statuses for tasks that already have them", func(ctx SpecContext) {
			db := gorp.Wrap(memkv.New())
			otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustSucceed(search.Open())
			g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
			stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
				Ontology: otg,
				DB:       db,
				Group:    g,
				Label:    labelSvc,
				Search:   searchIdx,
			}))
			rackSvc := MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
				DB:           db,
				Ontology:     otg,
				Group:        g,
				HostProvider: mock.StaticHostKeyProvider(1),
				Status:       stat,
				Search:       searchIdx,
			}))

			testRack := &rack.Rack{Name: "Migration Test Rack"}
			Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

			svc := MustSucceed(task.OpenService(ctx, task.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Rack:     rackSvc,
				Status:   stat,
				Search:   searchIdx,
			}))
			t := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Task With Status",
			}
			Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())

			var taskStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				WhereKeys(task.OntologyID(t.Key).String()).
				Entry(&taskStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(taskStatus.Variant).To(Equal(xstatus.VariantWarning))
			Expect(taskStatus.Message).To(Equal("Task With Status status unknown"))

			Expect(svc.Close()).To(Succeed())
			Expect(rackSvc.Close()).To(Succeed())
			Expect(stat.Close()).To(Succeed())
			Expect(labelSvc.Close()).To(Succeed())
			Expect(g.Close()).To(Succeed())
			Expect(searchIdx.Close()).To(Succeed())
			Expect(otg.Close()).To(Succeed())
			Expect(db.Close()).To(Succeed())
		})
	})

	Describe("Observe", func() {
		It("Should notify when a task is created", func(ctx SpecContext) {
			tx := db.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			w := svc.NewWriter(tx)
			t := &task.Task{
				Key:  task.NewKey(testRack.Key, 999),
				Name: "observe-test",
				Type: "test",
			}
			Expect(w.Create(ctx, t)).To(Succeed())
			called := false
			svc.Observe().OnChange(func(ctx context.Context, _ gorp.TxReader[task.Key, task.Task]) {
				called = true
			})
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(called).To(BeTrue())
		})
	})
})
