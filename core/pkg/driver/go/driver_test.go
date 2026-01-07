// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package godriver_test

import (
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	godriver "github.com/synnaxlabs/synnax/pkg/driver/go"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Config", Ordered, func() {
	var (
		db           *gorp.DB
		rackService  *rack.Service
		taskService  *task.Service
		framerSvc    *framer.Service
		channelSvc   *channel.Service
		factory      godriver.Factory
		hostProvider = mock.StaticHostKeyProvider(1)
	)

	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		labelSvc := MustSucceed(label.OpenService(ctx, label.Config{DB: db, Ontology: otg, Group: g}))
		stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{Ontology: otg, DB: db, Group: g, Label: labelSvc}))
		rackService = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
		}))
		taskService = MustSucceed(task.OpenService(ctx, task.Config{DB: db, Ontology: otg, Group: g, Rack: rackService, Status: stat}))
		factory = &mockFactory{name: "test"}

		DeferCleanup(func() {
			Expect(db.Close()).To(Succeed())
		})
	})

	Describe("Validate", func() {
		It("should fail when DB is nil", func() {
			cfg := godriver.Config{
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Factory: factory,
				Host:    hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Rack is nil", func() {
			cfg := godriver.Config{
				DB:      db,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Factory: factory,
				Host:    hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Task is nil", func() {
			cfg := godriver.Config{
				DB:      db,
				Rack:    rackService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Factory: factory,
				Host:    hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Factory is nil", func() {
			cfg := godriver.Config{
				DB:      db,
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Host:    hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Host is zero", func() {
			cfg := godriver.Config{
				DB:      db,
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Factory: factory,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})
	})
})

var _ = Describe("Driver", Ordered, func() {
	var (
		dist         mock.Node
		rackService  *rack.Service
		taskService  *task.Service
		channelSvc   *channel.Service
		framerSvc    *framer.Service
		hostProvider = mock.StaticHostKeyProvider(1)
	)

	// openDriver creates a driver with the given factory and registers cleanup.
	openDriver := func(factory godriver.Factory) *godriver.Driver {
		driver := MustSucceed(godriver.Open(ctx, godriver.Config{
			DB:      dist.DB,
			Rack:    rackService,
			Task:    taskService,
			Framer:  framerSvc,
			Channel: channelSvc,
			Factory: factory,
			Host:    hostProvider,
		}))
		DeferCleanup(func() { Expect(driver.Close()).To(Succeed()) })
		return driver
	}

	// newTask creates a task on the given rack with an auto-incrementing local key.
	var taskCounter atomic.Uint32
	newTask := func(rackKey rack.Key) task.Task {
		return task.Task{
			Key:  task.NewKey(rackKey, taskCounter.Add(1)),
			Name: "Test Task",
			Type: "test",
		}
	}

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc := MustSucceed(label.OpenService(
			ctx,
			label.Config{
				DB:       dist.DB,
				Ontology: dist.Ontology,
				Group:    dist.Group,
			}),
		)
		stat := MustSucceed(status.OpenService(
			ctx,
			status.ServiceConfig{
				Ontology: dist.Ontology,
				DB:       dist.DB,
				Group:    dist.Group,
				Label:    labelSvc,
			}),
		)
		rackService = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           dist.DB,
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
		}))
		taskService = MustSucceed(task.OpenService(ctx, task.Config{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Rack:     rackService,
			Status:   stat,
		}))
		channelSvc = dist.Channel
		framerSvc = dist.Framer

		DeferCleanup(func() {
			Expect(dist.Close()).To(Succeed())
		})
	})

	Describe("Open", func() {
		It("should create driver with valid config", func() {
			driver := openDriver(&mockFactory{name: "test"})
			Expect(driver).ToNot(BeNil())
			Expect(driver.RackKey()).ToNot(BeZero())
		})

		It("should create rack in rack service", func() {
			driver := openDriver(&mockFactory{name: "test"})
			var racks []rack.Rack
			Expect(rackService.NewRetrieve().
				WhereKeys(driver.RackKey()).
				Entries(&racks).
				Exec(ctx, nil)).To(Succeed())
			Expect(racks).To(HaveLen(1))
			Expect(racks[0].Embedded).To(BeTrue())
		})

		It("should fail with invalid config", func() {
			_, err := godriver.Open(ctx, godriver.Config{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Task Management", func() {
		It("should configure task via factory when task is created", func() {
			var configuredTask atomic.Value
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					mt := &mockTask{key: t.Key}
					configuredTask.Store(mt)
					return mt, true, nil
				},
			}
			driver := openDriver(factory)

			t := newTask(driver.RackKey())
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configuredTask.Load() != nil }).Should(BeTrue())
			Expect(configuredTask.Load().(*mockTask).Key()).To(Equal(t.Key))
		})

		It("should stop existing task before reconfiguration", func() {
			var (
				stopCount    atomic.Int32
				configCount  atomic.Int32
				initialReady = make(chan struct{})
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					if configCount.Add(1) == 1 {
						close(initialReady)
					}
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopCount.Add(1); return nil },
					}, true, nil
				},
			}
			driver := openDriver(factory)

			t := newTask(driver.RackKey())
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(initialReady).Should(BeClosed())
			Expect(stopCount.Load()).To(BeZero())

			t.Name = "Updated Task"
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() int32 { return stopCount.Load() }).Should(Equal(int32(1)))
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(2)))
		})

		It("should only process tasks on its rack", func() {
			var configuredCount atomic.Int32
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					configuredCount.Add(1)
					return &mockTask{key: t.Key}, true, nil
				},
			}
			openDriver(factory)

			otherRack := rack.Rack{Name: "Other Rack"}
			Expect(rackService.NewWriter(nil).Create(ctx, &otherRack)).To(Succeed())

			t := newTask(otherRack.Key)
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Consistently(func() int32 { return configuredCount.Load() }).Should(BeZero())
		})

		It("should delete task and stop it", func() {
			var (
				stopped      atomic.Bool
				initialReady = make(chan struct{})
				readyOnce    sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					readyOnce.Do(func() { close(initialReady) })
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopped.Store(true); return nil },
					}, true, nil
				},
			}
			driver := openDriver(factory)

			t := newTask(driver.RackKey())
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(initialReady).Should(BeClosed())
			Expect(stopped.Load()).To(BeFalse())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Eventually(func() bool { return stopped.Load() }).Should(BeTrue())
		})

		It("should not configure task when factory returns not handled", func() {
			var (
				configureCalled atomic.Bool
				stopCalled      atomic.Bool
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					configureCalled.Store(true)
					return nil, false, nil
				},
			}
			driver := openDriver(factory)

			t := newTask(driver.RackKey())
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configureCalled.Load() }).Should(BeTrue())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Consistently(func() bool { return stopCalled.Load() }).Should(BeFalse())
		})

		It("should handle factory configuration error gracefully", func() {
			var (
				configCalled atomic.Bool
				stopCalled   atomic.Bool
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					configCalled.Store(true)
					return nil, true, errors.New("factory configuration failed")
				},
			}
			driver := openDriver(factory)

			t := newTask(driver.RackKey())
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configCalled.Load() }).Should(BeTrue())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Consistently(func() bool { return stopCalled.Load() }).Should(BeFalse())
		})

		It("should handle task stop error gracefully during reconfiguration", func() {
			var (
				stopCalled   atomic.Bool
				configCount  atomic.Int32
				initialReady = make(chan struct{})
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					if configCount.Add(1) == 1 {
						close(initialReady)
					}
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopCalled.Store(true); return errors.New("stop failed") },
					}, true, nil
				},
			}
			driver := openDriver(factory)

			t := newTask(driver.RackKey())
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(initialReady).Should(BeClosed())

			t.Name = "Updated"
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return stopCalled.Load() }).Should(BeTrue())
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(2)))
		})
	})

	Describe("Close", func() {
		It("should stop all tasks", func() {
			var (
				stopCount     atomic.Int32
				allConfigured = make(chan struct{})
				configCount   atomic.Int32
			)
			const expectedTasks = int32(3)

			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					if configCount.Add(1) == expectedTasks {
						close(allConfigured)
					}
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopCount.Add(1); return nil },
					}, true, nil
				},
			}

			driver := MustSucceed(godriver.Open(ctx, godriver.Config{
				DB:      dist.DB,
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Factory: factory,
				Host:    hostProvider,
			}))

			for i := 0; i < int(expectedTasks); i++ {
				t := newTask(driver.RackKey())
				Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			}

			Eventually(allConfigured).Should(BeClosed())
			Expect(stopCount.Load()).To(BeZero())

			Expect(driver.Close()).To(Succeed())
			Expect(stopCount.Load()).To(Equal(expectedTasks))
		})

		It("should handle stop errors during close gracefully", func() {
			var (
				stopCalled  atomic.Bool
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (godriver.Task, bool, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopCalled.Store(true); return errors.New("stop failed") },
					}, true, nil
				},
			}

			driver := MustSucceed(godriver.Open(ctx, godriver.Config{
				DB:      dist.DB,
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Factory: factory,
				Host:    hostProvider,
			}))

			t := newTask(driver.RackKey())
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Eventually(configReady).Should(BeClosed())

			Expect(driver.Close()).To(Succeed())
			Expect(stopCalled.Load()).To(BeTrue())
		})

		It("should be idempotent", func() {
			driver := openDriver(&mockFactory{name: "test"})
			Expect(driver.Close()).To(Succeed())
			Expect(driver.Close()).To(Succeed())
		})
	})
})
