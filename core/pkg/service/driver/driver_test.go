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
	"context"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Driver", func() {
	embeddedRackKey := func(ctx context.Context) rack.Key {
		var r rack.Rack
		Expect(rackService.NewRetrieve().
			Where(rack.MatchEmbedded(true), rack.MatchName("Node 1")).
			Entry(&r).
			Exec(ctx, nil)).To(Succeed())
		return r.Key
	}

	openDriver := func(ctx context.Context, factory driver.Factory) *driver.Driver {
		return MustOpen(driver.Open(ctx, driver.Config{
			DB:        dist.DB,
			Rack:      rackService,
			Task:      taskService,
			Framer:    framerSvc,
			Channel:   channelSvc,
			Status:    statusSvc,
			Factories: []driver.Factory{factory},
			Host:      hostProvider,
		}))
	}

	var taskCounter atomic.Uint32
	newTask := func(rackKey rack.Key) task.Task {
		return task.Task{
			Key:  task.NewKey(rackKey, taskCounter.Add(1)),
			Name: "Test Task",
			Type: "test",
		}
	}

	writeCommand := func(ctx context.Context, cmd task.Command) {
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

	Describe("Open", func() {
		It("should create driver with valid config", func(ctx SpecContext) {
			driver := openDriver(ctx, &mockFactory{name: "test"})
			Expect(driver).ToNot(BeNil())
			Expect(embeddedRackKey(ctx)).ToNot(BeZero())
		})

		It("should create rack in rack service", func(ctx SpecContext) {
			openDriver(ctx, &mockFactory{name: "test"})
			var racks []rack.Rack
			Expect(rackService.NewRetrieve().
				WhereKeys(embeddedRackKey(ctx)).
				Entries(&racks).
				Exec(ctx, nil)).To(Succeed())
			Expect(racks).To(HaveLen(1))
			Expect(racks[0].Embedded).To(BeTrue())
		})

		It("should set integrations on the rack from factory names", func(ctx SpecContext) {
			MustOpen(driver.Open(ctx, driver.Config{
				DB:      dist.DB,
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Status:  statusSvc,
				Factories: []driver.Factory{
					&mockFactory{name: "arc"},
					&mockFactory{name: "opc"},
				},
				Host: hostProvider,
			}))
			var r rack.Rack
			Expect(rackService.NewRetrieve().
				Where(rack.MatchEmbedded(true), rack.MatchName("Node 1")).
				Entry(&r).
				Exec(ctx, nil)).To(Succeed())
			Expect(r.Integrations).To(Equal([]string{"arc", "opc"}))
		})

		It("should update integrations on existing rack when reopened with different factories", func(ctx SpecContext) {
			d1 := MustSucceed(driver.Open(ctx, driver.Config{
				DB:        dist.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{&mockFactory{name: "arc"}},
				Host:      hostProvider,
			}))
			Expect(d1.Close()).To(Succeed())

			MustOpen(driver.Open(ctx, driver.Config{
				DB:      dist.DB,
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Status:  statusSvc,
				Factories: []driver.Factory{
					&mockFactory{name: "arc"},
					&mockFactory{name: "ni"},
					&mockFactory{name: "opc"},
				},
				Host: hostProvider,
			}))

			var r rack.Rack
			Expect(rackService.NewRetrieve().
				Where(rack.MatchEmbedded(true), rack.MatchName("Node 1")).
				Entry(&r).
				Exec(ctx, nil)).To(Succeed())
			Expect(r.Integrations).To(Equal([]string{"arc", "ni", "opc"}))
		})

		It("should fail with invalid config", func(ctx SpecContext) {
			_, err := driver.Open(ctx, driver.Config{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Task Management", func() {
		It("should configure task via factory when task is created", func(ctx SpecContext) {
			var configuredTask atomic.Value
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					mt := &mockTask{key: t.Key}
					configuredTask.Store(mt)
					return mt, nil
				},
			}
			openDriver(ctx, factory)

			t := newTask(embeddedRackKey(ctx))
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configuredTask.Load() != nil }).Should(BeTrue())
			Expect(configuredTask.Load().(*mockTask).key).To(Equal(t.Key))
		})

		It("should stop existing task before reconfiguration", func(ctx SpecContext) {
			var (
				stopCount   atomic.Int32
				configCount atomic.Int32
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
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
			openDriver(ctx, factory)
			t := newTask(embeddedRackKey(ctx))
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

		It("should only process tasks on its rack", func(ctx SpecContext) {
			var configuredCount atomic.Int32
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					configuredCount.Add(1)
					return &mockTask{key: t.Key}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)
			countAfterOpen := configuredCount.Load()

			otherRack := rack.Rack{Name: "Other Rack"}
			Expect(rackService.NewWriter(nil).Create(ctx, &otherRack)).To(Succeed())

			t := newTask(otherRack.Key)
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Consistently(func() int32 { return configuredCount.Load() }).Should(Equal(countAfterOpen))
		})

		It("should delete task and stop it", func(ctx SpecContext) {
			var (
				stopped      atomic.Bool
				initialReady = make(chan struct{})
				readyOnce    sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					readyOnce.Do(func() { close(initialReady) })
					return &mockTask{
						key:      t.Key,
						stopFunc: func() error { stopped.Store(true); return nil },
					}, nil
				},
			}
			openDriver(ctx, factory)

			t := newTask(embeddedRackKey(ctx))
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(initialReady).Should(BeClosed())
			Expect(stopped.Load()).To(BeFalse())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Eventually(func() bool { return stopped.Load() }).Should(BeTrue())
		})

		It("should handle stop error gracefully during deletion", func(ctx SpecContext) {
			var (
				stopCalled  atomic.Bool
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key: t.Key,
						stopFunc: func() error {
							stopCalled.Store(true)
							return errors.New("stop failed")
						},
					}, nil
				},
			}
			openDriver(ctx, factory)

			t := newTask(embeddedRackKey(ctx))
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())
			Eventually(configReady).Should(BeClosed())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Eventually(func() bool { return stopCalled.Load() }).Should(BeTrue())
		})

		It("should not configure task when factory returns not handled", func(ctx SpecContext) {
			var (
				configureCalled atomic.Bool
				stopCalled      atomic.Bool
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					configureCalled.Store(true)
					return nil, driver.ErrTaskNotHandled
				},
			}
			openDriver(ctx, factory)

			t := newTask(embeddedRackKey(ctx))
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configureCalled.Load() }).Should(BeTrue())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Consistently(func() bool { return stopCalled.Load() }).Should(BeFalse())
		})

		It("should handle factory configuration error gracefully", func(ctx SpecContext) {
			var (
				configCalled atomic.Bool
				stopCalled   atomic.Bool
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(context.Context, task.Task) (driver.Task, error) {
					configCalled.Store(true)
					return nil, errors.New("factory configuration failed")
				},
			}
			openDriver(ctx, factory)

			t := newTask(embeddedRackKey(ctx))
			w := taskService.NewWriter(nil)
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return configCalled.Load() }).Should(BeTrue())

			Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
			Consistently(func() bool { return stopCalled.Load() }).Should(BeFalse())
		})

		It("should continue processing new tasks after a configuration error", func(ctx SpecContext) {
			var (
				knownKeys   sync.Map
				configCount atomic.Int32
				execCalled  atomic.Bool
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if _, ok := knownKeys.Load(t.Key); !ok {
						return nil, driver.ErrTaskNotHandled
					}
					n := configCount.Add(1)
					if n == 1 {
						return nil, errors.New("first task fails")
					}
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCalled.Store(true)
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)

			// First task: configuration fails.
			t1 := newTask(embeddedRackKey(ctx))
			knownKeys.Store(t1.Key, true)
			Expect(taskService.NewWriter(nil).Create(ctx, &t1)).To(Succeed())
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(1)))

			// Second task: configuration succeeds, proving the driver is still
			// functional after the first error.
			t2 := newTask(embeddedRackKey(ctx))
			knownKeys.Store(t2.Key, true)
			Expect(taskService.NewWriter(nil).Create(ctx, &t2)).To(Succeed())
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(2)))

			writeCommand(ctx, task.Command{Task: t2.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() bool { return execCalled.Load() }).Should(BeTrue())
		})

		It("should handle task stop error gracefully during reconfiguration", func(ctx SpecContext) {
			var (
				stopCalled  atomic.Bool
				configCount atomic.Int32
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
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
			openDriver(ctx, factory)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			w := taskService.NewWriter(nil)
			countBeforeCreate := configCount.Load()
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() int32 { return configCount.Load() }).Should(BeNumerically(">", countBeforeCreate))

			t.Name = "Updated"
			Expect(w.Create(ctx, &t)).To(Succeed())

			Eventually(func() bool { return stopCalled.Load() }).Should(BeTrue())
		})

		It("should configure existing tasks on startup", func(ctx SpecContext) {
			var configuredTasks sync.Map

			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
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
			rackKey := embeddedRackKey(ctx)

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

			MustOpen(driver.Open(ctx, driver.Config{
				DB:        dist.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}))

			Expect(embeddedRackKey(ctx)).To(Equal(rackKey))
			Eventually(func() bool {
				_, ok1 := configuredTasks.Load(t1.Key)
				_, ok2 := configuredTasks.Load(t2.Key)
				return ok1 && ok2
			}).Should(BeTrue())
		})
	})

	Describe("Close", func() {
		It("should stop all tasks", func(ctx SpecContext) {
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
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
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
				t := newTask(embeddedRackKey(ctx))
				testTaskKeys.Store(t.Key, true)
				Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			}

			Eventually(allConfigured).Should(BeClosed())
			Expect(stopCount.Load()).To(BeZero())

			Expect(driver.Close()).To(Succeed())
			Expect(stopCount.Load()).To(Equal(expectedTasks))
		})

		It("should handle stop errors during close gracefully", func(ctx SpecContext) {
			var (
				stopCalled  atomic.Bool
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
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

			t := newTask(embeddedRackKey(ctx))
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Eventually(configReady).Should(BeClosed())

			Expect(driver.Close()).To(Succeed())
			Expect(stopCalled.Load()).To(BeTrue())
		})

		It("should be idempotent", func(ctx SpecContext) {
			driver := openDriver(ctx, &mockFactory{name: "test"})
			Expect(driver.Close()).To(Succeed())
			Expect(driver.Close()).To(Succeed())
		})
	})

	Describe("Heartbeat", func() {
		It("should send periodic status updates", func(ctx SpecContext) {
			MustOpen(driver.Open(ctx, driver.Config{
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

			statusKey := rack.OntologyID(embeddedRackKey(ctx)).String()
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

		It("should use the configured heartbeat interval", func(ctx SpecContext) {
			MustOpen(driver.Open(ctx, driver.Config{
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

			statusKey := rack.OntologyID(embeddedRackKey(ctx)).String()
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

		It("should stop heartbeat when driver is closed", func(ctx SpecContext) {
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

			statusKey := rack.OntologyID(embeddedRackKey(ctx)).String()
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
		It("should handle malformed command JSON without crashing", func(ctx SpecContext) {
			var execCalled atomic.Bool
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCalled.Store(true)
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			// Write valid JSON that won't unmarshal into task.Command
			// (task field expects a number, not a string).
			w := MustSucceed(framerSvc.OpenWriter(ctx, writer.Config{
				Keys:  channel.Keys{taskService.CommandChannelKey()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(
				taskService.CommandChannelKey(),
				MustSucceed(telem.NewJSONSeriesV(
					map[string]any{"task": "not-a-number", "type": "start"},
				)),
			))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			Consistently(func() bool { return execCalled.Load() }, "200ms", "50ms").
				Should(BeFalse())
		})

		It("should execute command on configured task", func(ctx SpecContext) {
			var (
				execCalled  atomic.Bool
				receivedCmd atomic.Value
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key: t.Key,
						execFunc: func(_ context.Context, cmd task.Command) error {
							execCalled.Store(true)
							receivedCmd.Store(cmd)
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			// Allow streamer to boot up
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Eventually(configReady).Should(BeClosed())

			cmd := task.Command{
				Task: t.Key,
				Type: "start",
				Key:  "cmd-1",
			}
			writeCommand(ctx, cmd)

			Eventually(func() bool { return execCalled.Load() }, "2s").Should(BeTrue())
			stored := receivedCmd.Load().(task.Command)
			Expect(stored.Type).To(Equal("start"))
			Expect(stored.Key).To(Equal("cmd-1"))
		})

		It("should ignore commands for unknown tasks", func(ctx SpecContext) {
			var execCalled atomic.Bool
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCalled.Store(true)
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			unknownTaskKey := task.NewKey(embeddedRackKey(ctx), 99999)
			cmd := task.Command{
				Task: unknownTaskKey,
				Type: "start",
				Key:  "cmd-unknown",
			}
			writeCommand(ctx, cmd)

			Consistently(func() bool { return execCalled.Load() }, "200ms", "50ms").Should(BeFalse())
		})

		It("should ignore commands for tasks on other racks", func(ctx SpecContext) {
			var execCalled atomic.Bool
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCalled.Store(true)
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(5 * time.Millisecond)

			otherRack := rack.Rack{Name: "Other Rack for Commands"}
			Expect(rackService.NewWriter(nil).Create(ctx, &otherRack)).To(Succeed())

			otherTaskKey := task.NewKey(otherRack.Key, 1)
			cmd := task.Command{
				Task: otherTaskKey,
				Type: "start",
				Key:  "cmd-other-rack",
			}
			writeCommand(ctx, cmd)

			Consistently(func() bool { return execCalled.Load() }, "200ms", "50ms").Should(BeFalse())
		})

		It("should handle command execution errors gracefully", func(ctx SpecContext) {
			var (
				execCalled  atomic.Bool
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCalled.Store(true)
							return errors.New("execution failed")
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(5 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Eventually(configReady).Should(BeClosed())

			cmd := task.Command{
				Task: t.Key,
				Type: "failing-command",
				Key:  "cmd-fail",
			}
			writeCommand(ctx, cmd)

			Eventually(func() bool { return execCalled.Load() }, "2s").Should(BeTrue())
		})

		It("should log warning for unsupported command without crashing", func(ctx SpecContext) {
			var (
				execCalled  atomic.Bool
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCalled.Store(true)
							return driver.ErrUnsupportedCommand
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Eventually(configReady).Should(BeClosed())

			cmd := task.Command{
				Task: t.Key,
				Type: "unsupported",
				Key:  "cmd-unsupported",
			}
			writeCommand(ctx, cmd)

			Eventually(func() bool { return execCalled.Load() }, "2s").Should(BeTrue())
		})
	})

	Describe("Timeouts", func() {
		It("should pass timeouts to ConfigureTask", func(ctx SpecContext) {
			var (
				configureStarted = make(chan struct{})
				startOnce        sync.Once
				timedOut         atomic.Bool
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					cfgCtx context.Context,
					t task.Task,
				) (driver.Task, error) {
					startOnce.Do(func() { close(configureStarted) })
					// Block until context is canceled (simulates a well-behaved but
					// slow implementation that respects cancellation).
					<-cfgCtx.Done()
					timedOut.Store(true)
					return nil, cfgCtx.Err()
				},
			}
			MustOpen(driver.Open(ctx, driver.Config{
				DB:          dist.DB,
				Rack:        rackService,
				Task:        taskService,
				Framer:      framerSvc,
				Channel:     channelSvc,
				Status:      statusSvc,
				Factories:   []driver.Factory{factory},
				Host:        hostProvider,
				TaskTimeout: 50 * time.Millisecond,
			}))

			t := newTask(embeddedRackKey(ctx))
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())

			Eventually(configureStarted, "1s").Should(BeClosed())
			// The goroutine should receive context cancellation after the timeout.
			Eventually(func() bool { return timedOut.Load() }).Should(BeTrue())
		})

		It("should timeout a hanging Exec", func(ctx SpecContext) {
			var (
				execStarted = make(chan struct{}, 1)
				configReady = make(chan struct{})
				readyOnce   sync.Once
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					readyOnce.Do(func() { close(configReady) })
					return &mockTask{
						key: t.Key,
						execFunc: func(eCtx context.Context, _ task.Command) error {
							select {
							case execStarted <- struct{}{}:
							default:
							}
							// Block until context is canceled by the timeout.
							<-eCtx.Done()
							return eCtx.Err()
						},
					}, nil
				},
			}
			MustOpen(driver.Open(ctx, driver.Config{
				DB:          dist.DB,
				Rack:        rackService,
				Task:        taskService,
				Framer:      framerSvc,
				Channel:     channelSvc,
				Status:      statusSvc,
				Factories:   []driver.Factory{factory},
				Host:        hostProvider,
				TaskTimeout: 50 * time.Millisecond,
			}))
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Eventually(configReady).Should(BeClosed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(execStarted).Should(Receive())
		})
	})

	Describe("Parallelism", func() {
		It("should configure existing tasks in parallel on startup", func(ctx SpecContext) {
			var (
				configCount  atomic.Int32
				allConfiging = make(chan struct{})
				configGate   = make(chan struct{})
			)
			const numTasks = 3

			// Pre-create tasks before opening the driver.
			d1 := MustSucceed(driver.Open(ctx, driver.Config{
				DB:        dist.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{&mockFactory{name: "noop"}},
				Host:      hostProvider,
			}))
			rackKey := embeddedRackKey(ctx)
			for range numTasks {
				t := task.Task{
					Key:  task.NewKey(rackKey, taskCounter.Add(1)),
					Name: "Parallel Task",
					Type: "test",
				}
				Expect(taskService.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			}
			Expect(d1.Close()).To(Succeed())

			// Open a new driver with a factory that blocks until all tasks are being
			// configured concurrently.
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if configCount.Add(1) == numTasks {
						close(allConfiging)
					}
					<-configGate
					return &mockTask{key: t.Key}, nil
				},
			}

			openDone := make(chan *driver.Driver, 1)
			go func() {
				defer GinkgoRecover()
				d := MustSucceed(driver.Open(ctx, driver.Config{
					DB:        dist.DB,
					Rack:      rackService,
					Task:      taskService,
					Framer:    framerSvc,
					Channel:   channelSvc,
					Status:    statusSvc,
					Factories: []driver.Factory{factory},
					Host:      hostProvider,
				}))
				openDone <- d
			}()

			// If sequential, only 1 would be configuring at a time — never reaching
			// numTasks.
			Eventually(allConfiging).Should(BeClosed())
			close(configGate)

			d2 := <-openDone
			Expect(d2.Close()).To(Succeed())
		})
	})
})
