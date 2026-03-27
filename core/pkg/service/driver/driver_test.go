// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Driver", func() {
	embeddedRackKey := func() rack.Key {
		var r rack.Rack
		Expect(rackService.NewRetrieve().
			WhereEmbedded(true).
			WhereName("Node 1").
			Entry(&r).
			Exec(ctx, nil)).To(Succeed())
		return r.Key
	}

	openDriver := func(factory driver.Factory) *driver.Driver {
		driver := MustSucceed(driver.Open(ctx, driver.Config{
			DB:        dist.DB,
			Rack:      rackService,
			Task:      taskService,
			Framer:    framerSvc,
			Channel:   channelSvc,
			Status:    statusSvc,
			Factories: []driver.Factory{factory},
			Host:      hostProvider,
		}))
		DeferCleanup(func() { Expect(driver.Close()).To(Succeed()) })
		return driver
	}

	var taskCounter atomic.Uint32
	newTask := func(rackKey rack.Key) task.Task {
		return task.Task{
			Key:  task.NewKey(rackKey, taskCounter.Add(1)),
			Name: "Test Task",
			Type: "test",
		}
	}

	Describe("Open", func() {
		It("should create driver with valid config", func() {
			driver := openDriver(&mockFactory{name: "test"})
			Expect(driver).ToNot(BeNil())
			Expect(embeddedRackKey()).ToNot(BeZero())
		})

		It("should create rack in rack service", func() {
			openDriver(&mockFactory{name: "test"})
			var racks []rack.Rack
			Expect(rackService.NewRetrieve().
				WhereKeys(embeddedRackKey()).
				Entries(&racks).
				Exec(ctx, nil)).To(Succeed())
			Expect(racks).To(HaveLen(1))
			Expect(racks[0].Embedded).To(BeTrue())
		})

		It("should fail with invalid config", func() {
			_, err := driver.Open(ctx, driver.Config{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Task Management", func() {
		It("should configure task via factory when task is created", func() {
			var configuredTask atomic.Value
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					mt := &mockTask{key: t.Key}
					configuredTask.Store(mt)
					return mt, nil
				},
			}
			openDriver(factory)

			t := newTask(embeddedRackKey())
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configuredTask.Load() != nil }).Should(BeTrue())
			Expect(configuredTask.Load().(*mockTask).key).To(Equal(t.Key))
		})

		It("should stop existing task before reconfiguration", func() {
			var (
				stopCount   atomic.Int32
				configCount atomic.Int32
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					configCount.Add(1)
					return &mockTask{
						key: t.Key,
						stopFunc: func() error {
							if t.Key == taskKey.Load() {
								stopCount.Add(1)
							}
							return nil
						},
					}, nil
				},
			}
			openDriver(factory)
			t := newTask(embeddedRackKey())
			taskKey.Store(t.Key)
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())
			countAfterCreate := configCount.Load()
			Eventually(func() int32 { return configCount.Load() }).Should(
				BeNumerically(">=", countAfterCreate),
			)
			Expect(stopCount.Load()).To(BeZero())

			t.Name = "Updated Task"
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() int32 { return stopCount.Load() }).Should(Equal(int32(1)))
		})

		It("should only process tasks on its rack", func() {
			var configuredCount atomic.Int32
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					configuredCount.Add(1)
					return &mockTask{key: t.Key}, nil
				},
			}
			openDriver(factory)
			time.Sleep(50 * time.Millisecond)
			countAfterOpen := configuredCount.Load()

			otherRack := rack.Rack{Name: "Other Rack"}
			Expect(rackService.NewWriter(nil).Create(ctx, &otherRack)).To(Succeed())

			t := newTask(otherRack.Key)
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Consistently(func() int32 { return configuredCount.Load() }).Should(Equal(countAfterOpen))
		})

		It("should delete task and stop it", func() {
			var (
				stopped      atomic.Bool
				initialReady = make(chan struct{})
				readyOnce    sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					readyOnce.Do(func() { close(initialReady) })
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopped.Store(true); return nil },
					}, nil
				},
			}
			openDriver(factory)

			t := newTask(embeddedRackKey())
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
				configureFunc: func(t task.Task) (driver.Task, error) {
					configureCalled.Store(true)
					return nil, driver.ErrTaskNotHandled
				},
			}
			openDriver(factory)

			t := newTask(embeddedRackKey())
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
				configureFunc: func(t task.Task) (driver.Task, error) {
					configCalled.Store(true)
					return nil, errors.New("factory configuration failed")
				},
			}
			openDriver(factory)

			t := newTask(embeddedRackKey())
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configCalled.Load() }).Should(BeTrue())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Consistently(func() bool { return stopCalled.Load() }).Should(BeFalse())
		})

		It("should handle task stop error gracefully during reconfiguration", func() {
			var (
				stopCalled  atomic.Bool
				configCount atomic.Int32
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					configCount.Add(1)
					return &mockTask{
						key: t.Key,
						stopFunc: func() error {
							if t.Key == taskKey.Load() {
								stopCalled.Store(true)
							}
							return errors.New("stop failed")
						},
					}, nil
				},
			}
			openDriver(factory)

			t := newTask(embeddedRackKey())
			taskKey.Store(t.Key)
			w := taskService.NewWriter(nil)
			countBeforeCreate := configCount.Load()
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() int32 { return configCount.Load() }).Should(BeNumerically(">", countBeforeCreate))

			t.Name = "Updated"
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return stopCalled.Load() }).Should(BeTrue())
		})

		It("should configure existing tasks on startup", func() {
			var configuredTasks sync.Map

			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					configuredTasks.Store(t.Key, true)
					return &mockTask{key: t.Key}, nil
				},
			}

			d1 := MustSucceed(driver.Open(ctx, driver.Config{
				DB:        dist.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}))
			rackKey := embeddedRackKey()

			t1 := task.Task{
				Key:  task.NewKey(rackKey, taskCounter.Add(1)),
				Name: "Pre-existing Task 1",
				Type: "test",
			}
			t2 := task.Task{
				Key:  task.NewKey(rackKey, taskCounter.Add(1)),
				Name: "Pre-existing Task 2",
				Type: "test",
			}
			Expect(taskService.NewWriter(nil).Create(ctx, &t1)).To(Succeed())
			Expect(taskService.NewWriter(nil).Create(ctx, &t2)).To(Succeed())

			Eventually(func() bool {
				_, ok1 := configuredTasks.Load(t1.Key)
				_, ok2 := configuredTasks.Load(t2.Key)
				return ok1 && ok2
			}).Should(BeTrue())

			Expect(d1.Close()).To(Succeed())
			configuredTasks = sync.Map{}

			d2 := MustSucceed(driver.Open(ctx, driver.Config{
				DB:        dist.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}))
			DeferCleanup(func() { Expect(d2.Close()).To(Succeed()) })

			Expect(embeddedRackKey()).To(Equal(rackKey))
			Eventually(func() bool {
				_, ok1 := configuredTasks.Load(t1.Key)
				_, ok2 := configuredTasks.Load(t2.Key)
				return ok1 && ok2
			}).Should(BeTrue())
		})
	})

	Describe("Close", func() {
		It("should stop all tasks", func() {
			var (
				stopCount     atomic.Int32
				configCount   atomic.Int32
				testTaskKeys  sync.Map
				allConfigured = make(chan struct{})
				closeOnce     sync.Once
			)
			const expectedTasks = int32(3)

			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					if _, isTestTask := testTaskKeys.Load(t.Key); isTestTask {
						if configCount.Add(1) == expectedTasks {
							closeOnce.Do(func() { close(allConfigured) })
						}
					}
					return &mockTask{
						key: t.Key,
						stopFunc: func() error {
							if _, isTestTask := testTaskKeys.Load(t.Key); isTestTask {
								stopCount.Add(1)
							}
							return nil
						},
					}, nil
				},
			}

			driver := MustSucceed(driver.Open(ctx, driver.Config{
				DB:        dist.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}))

			for range expectedTasks {
				t := newTask(embeddedRackKey())
				testTaskKeys.Store(t.Key, true)
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
				configureFunc: func(t task.Task) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopCalled.Store(true); return errors.New("stop failed") },
					}, nil
				},
			}

			driver := MustSucceed(driver.Open(ctx, driver.Config{
				DB:        dist.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}))

			t := newTask(embeddedRackKey())
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

	Describe("Heartbeat", func() {
		It("should send periodic status updates", func() {
			driver := MustSucceed(driver.Open(ctx, driver.Config{
				DB:                dist.DB,
				Rack:              rackService,
				Task:              taskService,
				Framer:            framerSvc,
				Channel:           channelSvc,
				Status:            statusSvc,
				Factories:         []driver.Factory{&mockFactory{name: "test"}},
				Host:              hostProvider,
				HeartbeatInterval: 50 * time.Millisecond,
			}))
			DeferCleanup(func() { Expect(driver.Close()).To(Succeed()) })

			statusKey := rack.OntologyID(embeddedRackKey()).String()
			Eventually(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(gorp.NewRetrieve[string, status.Status[any]]().
					WhereKeys(statusKey).
					Entries(&statuses).
					Exec(ctx, dist.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
				g.Expect(statuses[0].Variant).To(Equal(xstatus.VariantSuccess))
			}).Should(Succeed())
		})

		It("should use the configured heartbeat interval", func() {
			driver := MustSucceed(driver.Open(ctx, driver.Config{
				DB:                dist.DB,
				Rack:              rackService,
				Task:              taskService,
				Framer:            framerSvc,
				Channel:           channelSvc,
				Status:            statusSvc,
				Factories:         []driver.Factory{&mockFactory{name: "test"}},
				Host:              hostProvider,
				HeartbeatInterval: 25 * time.Millisecond,
			}))
			DeferCleanup(func() { Expect(driver.Close()).To(Succeed()) })

			statusKey := rack.OntologyID(embeddedRackKey()).String()
			var firstTime telem.TimeStamp
			Eventually(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(gorp.NewRetrieve[string, status.Status[any]]().
					WhereKeys(statusKey).
					Entries(&statuses).
					Exec(ctx, dist.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
				firstTime = statuses[0].Time
			}).Should(Succeed())

			Eventually(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(gorp.NewRetrieve[string, status.Status[any]]().
					WhereKeys(statusKey).
					Entries(&statuses).
					Exec(ctx, dist.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
				g.Expect(statuses[0].Time).To(BeNumerically(">", firstTime))
			}).Should(Succeed())
		})

		It("should stop heartbeat when driver is closed", func() {
			driver := MustSucceed(driver.Open(ctx, driver.Config{
				DB:                dist.DB,
				Rack:              rackService,
				Task:              taskService,
				Framer:            framerSvc,
				Channel:           channelSvc,
				Status:            statusSvc,
				Factories:         []driver.Factory{&mockFactory{name: "test"}},
				Host:              hostProvider,
				HeartbeatInterval: 25 * time.Millisecond,
			}))

			statusKey := rack.OntologyID(embeddedRackKey()).String()
			Eventually(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(gorp.NewRetrieve[string, status.Status[any]]().
					WhereKeys(statusKey).
					Entries(&statuses).
					Exec(ctx, dist.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
			}).Should(Succeed())

			Expect(driver.Close()).To(Succeed())

			var lastTime telem.TimeStamp
			var statuses []status.Status[any]
			Expect(gorp.NewRetrieve[string, status.Status[any]]().
				WhereKeys(statusKey).
				Entries(&statuses).
				Exec(ctx, dist.DB)).To(Succeed())
			lastTime = statuses[0].Time

			Consistently(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(gorp.NewRetrieve[string, status.Status[any]]().
					WhereKeys(statusKey).
					Entries(&statuses).
					Exec(ctx, dist.DB)).To(Succeed())
				g.Expect(statuses[0].Time).To(Equal(lastTime))
			}, "100ms", "25ms").Should(Succeed())
		})
	})

	Describe("Command Processing", func() {
		writeCommand := func(cmd task.Command) {
			w := MustSucceed(framerSvc.OpenWriter(ctx, writer.Config{
				Keys:  channel.Keys{taskService.CommandChannelKey()},
				Start: telem.Now(),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(
				taskService.CommandChannelKey(),
				MustSucceed(telem.NewJSONSeriesV(cmd)),
			))).To(BeTrue())
		}

		It("should execute command on configured task", func() {
			var (
				execCalled  atomic.Bool
				receivedCmd atomic.Value
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key: t.Key,
						execFunc: func(cmd task.Command) error {
							execCalled.Store(true)
							receivedCmd.Store(cmd)
							return nil
						},
					}, nil
				},
			}
			openDriver(factory)
			// Allow streamer to boot up
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey())
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Eventually(configReady).Should(BeClosed())

			cmd := task.Command{
				Task: t.Key,
				Type: "start",
				Key:  "cmd-1",
			}
			writeCommand(cmd)

			Eventually(func() bool { return execCalled.Load() }, "2s").Should(BeTrue())
			stored := receivedCmd.Load().(task.Command)
			Expect(stored.Type).To(Equal("start"))
			Expect(stored.Key).To(Equal("cmd-1"))
		})

		It("should ignore commands for unknown tasks", func() {
			var execCalled atomic.Bool
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					return &mockTask{
						key:      t.Key,
						execFunc: func(cmd task.Command) error { execCalled.Store(true); return nil },
					}, nil
				},
			}
			openDriver(factory)
			time.Sleep(50 * time.Millisecond)

			unknownTaskKey := task.NewKey(embeddedRackKey(), 99999)
			cmd := task.Command{
				Task: unknownTaskKey,
				Type: "start",
				Key:  "cmd-unknown",
			}
			writeCommand(cmd)

			Consistently(func() bool { return execCalled.Load() }, "200ms", "50ms").Should(BeFalse())
		})

		It("should ignore commands for tasks on other racks", func() {
			var execCalled atomic.Bool
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					return &mockTask{
						key:      t.Key,
						execFunc: func(cmd task.Command) error { execCalled.Store(true); return nil },
					}, nil
				},
			}
			openDriver(factory)
			time.Sleep(5 * time.Millisecond)

			otherRack := rack.Rack{Name: "Other Rack for Commands"}
			Expect(rackService.NewWriter(nil).Create(ctx, &otherRack)).To(Succeed())

			otherTaskKey := task.NewKey(otherRack.Key, 1)
			cmd := task.Command{
				Task: otherTaskKey,
				Type: "start",
				Key:  "cmd-other-rack",
			}
			writeCommand(cmd)

			Consistently(func() bool { return execCalled.Load() }, "200ms", "50ms").Should(BeFalse())
		})

		It("should handle command execution errors gracefully", func() {
			var (
				execCalled  atomic.Bool
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(t task.Task) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key: t.Key,
						execFunc: func(cmd task.Command) error {
							execCalled.Store(true)
							return errors.New("execution failed")
						},
					}, nil
				},
			}
			openDriver(factory)
			time.Sleep(5 * time.Millisecond)

			t := newTask(embeddedRackKey())
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Eventually(configReady).Should(BeClosed())

			cmd := task.Command{
				Task: t.Key,
				Type: "failing-command",
				Key:  "cmd-fail",
			}
			writeCommand(cmd)

			Eventually(func() bool { return execCalled.Load() }, "2s").Should(BeTrue())
		})
	})
})
