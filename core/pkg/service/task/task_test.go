// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
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
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		stat = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
		}))
		rackService = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:                  db,
			Ontology:            otg,
			Group:               g,
			HostProvider:        mock.StaticHostKeyProvider(1),
			Status:              stat,
			HealthCheckInterval: 10 * telem.Millisecond,
		}))
		svc = MustSucceed(task.OpenService(ctx, task.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackService,
			Status:   stat,
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(rackService.NewWriter(db).Create(ctx, testRack)).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	AfterAll(func() {
		Expect(svc.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	Describe("Task", func() {
		It("Should construct and deconstruct a key from its components", func() {
			rk := rack.NewKey(cluster.NodeKey(1), 1)
			k := task.NewKey(rk, 2)
			Expect(k.Rack()).To(Equal(rk))
			Expect(k.LocalKey()).To(Equal(uint32(2)))
		})
	})

	Describe("Create", func() {
		It("Should correctly create a task and assign it a unique key", func() {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
		})
		It("Should correctly increment the task count", func() {
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

		It("Should copy a task", func() {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 4)))
			Expect(m.Name).To(Equal("Test Task"))
			t, err := w.Copy(ctx, m.Key, "Copied Task", false)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Key).To(Equal(task.NewKey(testRack.Key, 5)))
		})

		It("Should create a snapshot of an existing task", func() {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 6)))
			Expect(m.Name).To(Equal("Test Task"))
			t, err := w.Copy(ctx, m.Key, "Snapshotted Task", true)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Key).To(Equal(task.NewKey(testRack.Key, 7)))
			Expect(t.Snapshot).To(BeTrue())
		})

	})

	Describe("Retrieve", func() {
		It("Should correctly retrieve a task", func() {
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

		It("Should filter tasks by snapshot status", func() {
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

		It("Should combine WhereSnapshot with other filters", func() {
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

		It("Should filter tasks by internal status", func() {
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

		It("Should combine WhereInternal with other filters", func() {
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
		It("Should correctly delete a task", func() {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(testRack.Key, 17)))
			Expect(m.Name).To(Equal("Test Task"))
			Expect(w.Delete(ctx, m.Key, false)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})

	Describe("Status", func() {
		It("Should create an unknown status when creating a task", func() {
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
			Expect(taskStatus.Variant).To(Equal(xstatus.WarningVariant))
			Expect(taskStatus.Message).To(Equal("Status Test Task status unknown"))
			Expect(taskStatus.Details.Task).To(Equal(m.Key))
		})

		It("Should use the provided status when creating a task", func() {
			providedStatus := &task.Status{
				Variant:     xstatus.SuccessVariant,
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
			Expect(taskStatus.Variant).To(Equal(xstatus.SuccessVariant))
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

		It("Should return a validation error if provided status has empty variant", func() {
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
		It("Should create an unknown status when copying a task", func() {
			m := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Original Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())

			copied, err := w.Copy(ctx, m.Key, "Copied Task", false)
			Expect(err).ToNot(HaveOccurred())

			var copiedStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				WhereKeys(task.OntologyID(copied.Key).String()).
				Entry(&copiedStatus).
				Exec(ctx, tx)).To(Succeed())
			Expect(copiedStatus.Variant).To(Equal(xstatus.WarningVariant))
			Expect(copiedStatus.Message).To(Equal("Copied Task status unknown"))
			Expect(copiedStatus.Details.Task).To(Equal(copied.Key))
		})
	})

	Describe("Suspect Rack", func() {
		It("Should propagate rack warning status to tasks on that rack", func() {
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
				g.Expect(taskStatus.Variant).To(Equal(xstatus.WarningVariant))
				g.Expect(taskStatus.Message).To(ContainSubstring("not running"))
				g.Expect(taskStatus.Details.Task).To(Equal(t.Key))
			}).Should(Succeed())
		})
	})
})
