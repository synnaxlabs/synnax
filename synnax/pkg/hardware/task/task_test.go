// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/hardware/task"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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
		g := MustSucceed(group.OpenService(group.Config{DB: db, Ontology: otg}))
		rackSvc := MustSucceed(rack.OpenService(ctx, rack.Config{DB: db, Ontology: otg, Group: g, HostProvider: mock.StaticHostKeyProvider(1)}))
		svc = MustSucceed(task.OpenService(ctx, task.Config{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			Rack:         rackSvc,
			HostProvider: mock.StaticHostKeyProvider(1),
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
			rk := rack.NewKey(core.NodeKey(1), 1)
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
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
			m = &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 2)))
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
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
			t, err := w.Copy(ctx, m.Key, "New Task", false)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Key).To(Equal(task.NewKey(rack_.Key, 2)))
		})

		It("Should create a snapshot of an existing task", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
			t, err := w.Copy(ctx, m.Key, "New Task", true)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Key).To(Equal(task.NewKey(rack_.Key, 2)))
			Expect(t.Snapshot).To(BeTrue())
		})

		It("Should not allow the re-creation of a snapshot", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
			t, err := w.Copy(ctx, m.Key, "New Task", true)
			Expect(err).ToNot(HaveOccurred())
			t.Config = "New Config"
			Expect(w.Create(ctx, &t)).To(HaveOccurredAs(validate.Error))
		})

	})

	Describe("Retrieve", func() {
		It("Should correctly retrieve a task", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
			var res task.Task
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(*m))
		})
	})

	Describe("Delete", func() {
		It("Should correctly delete a task", func() {
			m := &task.Task{
				Key:  task.NewKey(rack_.Key, 0),
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
			Expect(w.Delete(ctx, m.Key, false)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(m.Key).Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
		It("Should prevent the deletion of internal tasks", func() {
			m := &task.Task{
				Key:      task.NewKey(rack_.Key, 0),
				Name:     "Test Task",
				Internal: true,
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(task.NewKey(rack_.Key, 1)))
			Expect(m.Name).To(Equal("Test Task"))
			Expect(w.Delete(ctx, m.Key, false)).To(HaveOccurredAs(validate.Error))
		})
	})

})
