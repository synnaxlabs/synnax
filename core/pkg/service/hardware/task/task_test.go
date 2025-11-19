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
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Task", Ordered, func() {
	var (
		db    *gorp.DB
		svc   *task.Service
		otg   *ontology.Ontology
		w     task.Writer
		tx    gorp.Tx
		rack_ *rack.Rack
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
		stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
		}))
		rackSvc := MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
		}))
		svc = MustSucceed(task.OpenService(ctx, task.Config{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			Rack:         rackSvc,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
		}))
		rack_ = &rack.Rack{Name: "Test Rack"}
		Expect(rackSvc.NewWriter(db).Create(ctx, rack_)).To(Succeed())
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
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
		})
		It("Should correctly increment the task count", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 2)))
			Expect(m.Name).To(Equal("Test Task"))
			m = &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 3)))
			Expect(m.Name).To(Equal("Test Task"))
		})
	})

	Describe("Copy", func() {

		It("Should copy a task", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 4)))
			Expect(m.Name).To(Equal("Test Task"))
			t, err := w.Copy(ctx, m.Key, "Copied Task", false)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Key).To(Equal(task.NewKey(rack_.Key, 5)))
		})

		It("Should create a snapshot of an existing task", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 6)))
			Expect(m.Name).To(Equal("Test Task"))
			t, err := w.Copy(ctx, m.Key, "Snapshotted Task", true)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Key).To(Equal(task.NewKey(rack_.Key, 7)))
			Expect(t.Snapshot).To(BeTrue())
		})

	})

	Describe("Retrieve", func() {
		It("Should correctly retrieve a task", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 8)))
			Expect(m.Name).To(Equal("Test Task"))
			var res task.Task
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*m))
		})

		It("Should filter tasks by snapshot status", func() {
			regular := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Regular Task",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			snapshot := &task.Task{
				Key:      task.NewKey(rack_.Key, 0),
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
				Key:      task.NewKey(rack_.Key, 0),
				Name:     "Snapshot Task 1",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot1)).To(Succeed())
			snapshot2 := &task.Task{
				Key:      task.NewKey(rack_.Key, 0),
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
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Regular Task 2",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			internal := &task.Task{
				Key:      task.NewKey(rack_.Key, 0),
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
				Key:      task.NewKey(rack_.Key, 0),
				Name:     "Internal Task 1",
				Internal: true,
			}
			Expect(w.Create(ctx, internal1)).To(Succeed())
			internal2 := &task.Task{
				Key:      task.NewKey(rack_.Key, 0),
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
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 17)))
			Expect(m.Name).To(Equal("Test Task"))
			Expect(w.Delete(ctx, m.Key, false)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})
})
