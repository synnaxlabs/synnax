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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Driver", func() {
	embeddedRackKey := func(ctx context.Context) rack.Key {
		var r rack.Rack
		Expect(rackService.NewRetrieve().
			Where(rack.And(rack.MatchEmbedded(true), rack.MatchNames("Node 1"))).
			Entry(&r).
			Exec(ctx, nil)).To(Succeed())
		return r.Key
	}

	openDriver := func(ctx context.Context, factory driver.Factory) *driver.Driver {
		return MustOpen(driver.Open(ctx, driver.Config{
			DB:        node.DB,
			Rack:      rackService,
			Task:      taskService,
			Framer:    framerSvc,
			Channel:   channelSvc,
			Status:    statusSvc,
			Factories: []driver.Factory{factory},
			Host:      hostProvider,
		}))
	}

	newTask := func(rackKey rack.Key) task.Task {
		return task.Task{
			Key:  uuid.New(),
			Rack: rackKey,
			Name: "Test Task",
			Type: "test",
		}
	}

	writeCommand := func(ctx context.Context, cmd task.Command) {
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
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
				Where(rack.MatchKeys(embeddedRackKey(ctx))).
				Entries(&racks).
				Exec(ctx, nil)).To(Succeed())
			Expect(racks).To(HaveLen(1))
			Expect(racks[0].Embedded).To(BeTrue())
		})

		It("should set integrations on the rack from factory names", func(ctx SpecContext) {
			MustOpen(driver.Open(ctx, driver.Config{
				DB:      node.DB,
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
				Where(rack.And(rack.MatchEmbedded(true), rack.MatchNames("Node 1"))).
				Entry(&r).
				Exec(ctx, nil)).To(Succeed())
			Expect(r.Integrations).To(Equal([]string{"arc", "opc"}))
		})

		It("should update integrations on existing rack when reopened with different factories", func(ctx SpecContext) {
			d1 := DeferClose(MustSucceed(driver.Open(ctx, driver.Config{
				DB:        node.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{&mockFactory{name: "arc"}},
				Host:      hostProvider,
			})))
			Expect(d1.Close()).To(Succeed())

			MustOpen(driver.Open(ctx, driver.Config{
				DB:      node.DB,
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
				Where(rack.And(rack.MatchEmbedded(true), rack.MatchNames("Node 1"))).
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
		It("should configure on start command, not on create", func(ctx SpecContext) {
			var (
				configuredTask atomic.Value
				taskKey        atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					mt := &mockTask{key: t.Key}
					if t.Key == taskKey.Load() {
						configuredTask.Store(mt)
					}
					return mt, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			Consistently(func() bool { return configuredTask.Load() != nil }).Should(BeFalse())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() bool { return configuredTask.Load() != nil }).Should(BeTrue())
			Expect(configuredTask.Load().(*mockTask).key).To(Equal(t.Key))
		})

		It("should rebuild the task when the stored config changes between starts", func(ctx SpecContext) {
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
					if t.Key == taskKey.Load() {
						configCount.Add(1)
					}
					return &mockTask{
						key: t.Key,
						stopFunc: func(bool) error {
							if t.Key == taskKey.Load() {
								stopCount.Add(1)
							}
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			t.Config = map[string]any{"rate": 50}
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(1)))

			// Same stored config: a second start runs the live instance as-is.
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})
			Consistently(func() int32 { return configCount.Load() }).Should(Equal(int32(1)))
			Expect(stopCount.Load()).To(BeZero())

			t.Config = map[string]any{"rate": 100}
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-3"})
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(2)))
			Expect(stopCount.Load()).To(Equal(int32(1)))
		})

		It("should not rebuild when only metadata changes between starts", func(ctx SpecContext) {
			var (
				configCount atomic.Int32
				execCount   atomic.Int32
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if t.Key == taskKey.Load() {
						configCount.Add(1)
					}
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCount.Add(1)
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			t.Config = map[string]any{"rate": 50}
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() int32 { return execCount.Load() }).Should(Equal(int32(1)))

			t.Name = "Renamed Task"
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})
			Eventually(func() int32 { return execCount.Load() }).Should(Equal(int32(2)))
			Expect(configCount.Load()).To(Equal(int32(1)))
		})

		It("should ignore start commands for tasks on other racks", func(ctx SpecContext) {
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
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Consistently(func() int32 { return configuredCount.Load() }).Should(Equal(countAfterOpen))
		})

		It("should stop the live instance when a started task moves to another rack", func(ctx SpecContext) {
			var (
				stopped         atomic.Bool
				stoppedSilently atomic.Bool
				execCount       atomic.Int32
				taskKey         atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							execCount.Add(1)
							return nil
						},
						stopFunc: func(sendStatus bool) error {
							if t.Key == taskKey.Load() {
								stoppedSilently.Store(!sendStatus)
								stopped.Store(true)
							}
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() int32 { return execCount.Load() }).Should(Equal(int32(1)))

			otherRack := rack.Rack{Name: "Move Target Rack"}
			Expect(rackService.NewWriter(nil).Create(ctx, &otherRack)).To(Succeed())
			t.Rack = otherRack.Key
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			// The redeploy start lands on the new rack; for this driver it is the
			// teardown signal for the instance it still holds.
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})
			Eventually(func() bool { return stopped.Load() }).Should(BeTrue())
			Expect(stoppedSilently.Load()).To(BeTrue())
		})

		It("should ignore start commands for snapshot tasks", func(ctx SpecContext) {
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

			t := newTask(embeddedRackKey(ctx))
			t.Snapshot = true
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Consistently(func() int32 { return configuredCount.Load() }).Should(Equal(countAfterOpen))
		})

		It("should delete task and stop it", func(ctx SpecContext) {
			var (
				stopped    atomic.Bool
				configured atomic.Bool
				taskKey    atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if t.Key == taskKey.Load() {
						configured.Store(true)
					}
					return &mockTask{
						key: t.Key,
						stopFunc: func(bool) error {
							if t.Key == taskKey.Load() {
								stopped.Store(true)
							}
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})

			Eventually(func() bool { return configured.Load() }).Should(BeTrue())
			Expect(stopped.Load()).To(BeFalse())

			Expect(taskWriter.Delete(ctx, t.Key, false)).To(Succeed())
			Eventually(func() bool { return stopped.Load() }).Should(BeTrue())
		})

		It("should handle stop error gracefully during deletion", func(ctx SpecContext) {
			var (
				stopCalled atomic.Bool
				configured atomic.Bool
				taskKey    atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if t.Key == taskKey.Load() {
						configured.Store(true)
					}
					return &mockTask{
						key: t.Key,
						stopFunc: func(bool) error {
							if t.Key == taskKey.Load() {
								stopCalled.Store(true)
							}
							return errors.New("stop failed")
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() bool { return configured.Load() }).Should(BeTrue())

			Expect(taskWriter.Delete(ctx, t.Key, false)).To(Succeed())
			Eventually(func() bool { return stopCalled.Load() }).Should(BeTrue())
		})

		It("should not run the task when factory returns not handled", func(ctx SpecContext) {
			var (
				configureCalled atomic.Bool
				execCalled      atomic.Bool
				taskKey         atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if t.Key == taskKey.Load() {
						configureCalled.Store(true)
						return nil, driver.ErrTaskNotHandled
					}
					return &mockTask{key: t.Key, execFunc: func(context.Context, task.Command) error {
						execCalled.Store(true)
						return nil
					}}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})

			Eventually(func() bool { return configureCalled.Load() }).Should(BeTrue())
			Consistently(func() bool { return execCalled.Load() }).Should(BeFalse())
		})

		It("should handle factory configuration error gracefully", func(ctx SpecContext) {
			var (
				configCalled atomic.Bool
				execCalled   atomic.Bool
				taskKey      atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if t.Key == taskKey.Load() {
						configCalled.Store(true)
						return nil, errors.New("factory configuration failed")
					}
					return &mockTask{key: t.Key, execFunc: func(context.Context, task.Command) error {
						execCalled.Store(true)
						return nil
					}}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})

			Eventually(func() bool { return configCalled.Load() }).Should(BeTrue())
			Consistently(func() bool { return execCalled.Load() }).Should(BeFalse())
		})

		It("should acknowledge a start command whose deploy fails", func(ctx SpecContext) {
			var taskKey atomic.Value
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if t.Key == taskKey.Load() {
						return nil, errors.New("bad config")
					}
					return &mockTask{key: t.Key}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-ack"})

			Eventually(func(g Gomega) {
				var statuses []status.Status[task.StatusDetails]
				g.Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
					Where(status.MatchKeys[task.StatusDetails](
						task.OntologyID(t.Key).String(),
					)).
					Entries(&statuses).
					Exec(ctx, node.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
				g.Expect(statuses[0].Variant).To(Equal(status.VariantError))
				g.Expect(statuses[0].Message).To(ContainSubstring("bad config"))
				g.Expect(statuses[0].Details.Cmd).To(Equal("cmd-ack"))
				g.Expect(statuses[0].Details.Running).To(BeFalse())
			}).Should(Succeed())
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
			time.Sleep(50 * time.Millisecond)

			// First task: configuration fails.
			t1 := newTask(embeddedRackKey(ctx))
			knownKeys.Store(t1.Key, true)
			Expect(taskWriter.Create(ctx, &t1)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t1.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(1)))

			// Second task: configuration succeeds, proving the driver is still
			// functional after the first error.
			t2 := newTask(embeddedRackKey(ctx))
			knownKeys.Store(t2.Key, true)
			Expect(taskWriter.Create(ctx, &t2)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t2.Key, Type: "start", Key: "cmd-2"})
			Eventually(func() int32 { return configCount.Load() }).Should(Equal(int32(2)))
			Eventually(func() bool { return execCalled.Load() }).Should(BeTrue())
		})

		It("should continue configuring new tasks after a configuration panic", func(ctx SpecContext) {
			var (
				knownKeys         sync.Map
				configAttempts    atomic.Int32
				healthyConfigured atomic.Bool
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
					if configAttempts.Add(1) == 1 {
						panic("boom during configure")
					}
					healthyConfigured.Store(true)
					return &mockTask{key: t.Key}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t1 := newTask(embeddedRackKey(ctx))
			knownKeys.Store(t1.Key, true)
			Expect(taskWriter.Create(ctx, &t1)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t1.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() int32 { return configAttempts.Load() }).Should(Equal(int32(1)))

			t2 := newTask(embeddedRackKey(ctx))
			knownKeys.Store(t2.Key, true)
			Expect(taskWriter.Create(ctx, &t2)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t2.Key, Type: "start", Key: "cmd-2"})
			Eventually(func() bool { return healthyConfigured.Load() }).Should(BeTrue())
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
						stopFunc: func(bool) error {
							if t.Key == taskKey.Load() {
								stopCalled.Store(true)
							}
							return errors.New("stop failed")
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			t.Config = map[string]any{"rate": 50}
			taskKey.Store(t.Key)
			countBeforeCreate := configCount.Load()
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})

			Eventually(func() int32 { return configCount.Load() }).Should(BeNumerically(">", countBeforeCreate))

			t.Config = map[string]any{"rate": 100}
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})

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

			d1 := DeferClose(MustSucceed(driver.Open(ctx, driver.Config{
				DB:        node.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			})))
			rackKey := embeddedRackKey(ctx)

			t1 := task.Task{
				Rack: rackKey,
				Name: "Pre-existing Task 1",
				Type: "test",
			}
			t2 := task.Task{
				Rack: rackKey,
				Name: "Pre-existing Task 2",
				Type: "test",
			}
			Expect(taskWriter.Create(ctx, &t1)).To(Succeed())
			Expect(taskWriter.Create(ctx, &t2)).To(Succeed())

			// Sets don't deploy: nothing configures until the next boot.
			Consistently(func() bool {
				_, ok1 := configuredTasks.Load(t1.Key)
				_, ok2 := configuredTasks.Load(t2.Key)
				return ok1 || ok2
			}).Should(BeFalse())

			Expect(d1.Close()).To(Succeed())

			MustOpen(driver.Open(ctx, driver.Config{
				DB:        node.DB,
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

		It("should configure with startPending true on a start command", func(ctx SpecContext) {
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{key: t.Key}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() bool {
				pending, ok := factory.startPending.Load(t.Key)
				return ok && pending == true
			}).Should(BeTrue())
		})

		It("should configure with startPending false at boot", func(ctx SpecContext) {
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{key: t.Key}, nil
				},
			}
			d1 := MustSucceed(driver.Open(ctx, driver.Config{
				DB:        node.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}))
			rackKey := embeddedRackKey(ctx)
			t := newTask(rackKey)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			Expect(d1.Close()).To(Succeed())

			MustOpen(driver.Open(ctx, driver.Config{
				DB:        node.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}))
			Eventually(func() bool {
				pending, ok := factory.startPending.Load(t.Key)
				return ok && pending == false
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
						stopFunc: func(bool) error {
							if _, isTestTask := testTaskKeys.Load(t.Key); isTestTask {
								stopCount.Add(1)
							}
							return nil
						},
					}, nil
				},
			}

			driver := DeferClose(MustSucceed(driver.Open(ctx, driver.Config{
				DB:        node.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			})))

			time.Sleep(50 * time.Millisecond)
			for i := range expectedTasks {
				t := newTask(embeddedRackKey(ctx))
				testTaskKeys.Store(t.Key, true)
				Expect(taskWriter.Create(ctx, &t)).To(Succeed())
				writeCommand(ctx, task.Command{
					Task: t.Key,
					Type: "start",
					Key:  fmt.Sprintf("cmd-%d", i),
				})
			}

			Eventually(allConfigured).Should(BeClosed())
			Expect(stopCount.Load()).To(BeZero())

			Expect(driver.Close()).To(Succeed())
			Expect(stopCount.Load()).To(Equal(expectedTasks))
		})

		It("should handle stop errors during close gracefully", func(ctx SpecContext) {
			var (
				stopCalled atomic.Bool
				configured atomic.Bool
				taskKey    atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					if t.Key == taskKey.Load() {
						configured.Store(true)
					}
					return &mockTask{
						key: t.Key,
						stopFunc: func(bool) error {
							if t.Key == taskKey.Load() {
								stopCalled.Store(true)
							}
							return errors.New("stop failed")
						},
					}, nil
				},
			}

			driver := DeferClose(MustSucceed(driver.Open(ctx, driver.Config{
				DB:        node.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			})))
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})

			Eventually(func() bool { return configured.Load() }).Should(BeTrue())

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
				DB:                node.DB,
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
				g.Expect(statusSvc.NewRetrieve().
					Where(status.MatchKeys[any](statusKey)).
					Entries(&statuses).
					Exec(ctx, node.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
				g.Expect(statuses[0].Variant).To(Equal(status.VariantSuccess))
			}).Should(Succeed())
		})

		It("should use the configured heartbeat interval", func(ctx SpecContext) {
			MustOpen(driver.Open(ctx, driver.Config{
				DB:                node.DB,
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
				g.Expect(statusSvc.NewRetrieve().
					Where(status.MatchKeys[any](statusKey)).
					Entries(&statuses).
					Exec(ctx, node.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
				firstTime = statuses[0].Time
			}).Should(Succeed())

			Eventually(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(statusSvc.NewRetrieve().
					Where(status.MatchKeys[any](statusKey)).
					Entries(&statuses).
					Exec(ctx, node.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
				g.Expect(statuses[0].Time).To(BeNumerically(">", firstTime))
			}).Should(Succeed())
		})

		It("should stop heartbeat when driver is closed", func(ctx SpecContext) {
			driver := DeferClose(MustSucceed(driver.Open(ctx, driver.Config{
				DB:                node.DB,
				Rack:              rackService,
				Task:              taskService,
				Framer:            framerSvc,
				Channel:           channelSvc,
				Status:            statusSvc,
				Factories:         []driver.Factory{&mockFactory{name: "test"}},
				Host:              hostProvider,
				HeartbeatInterval: 25 * time.Millisecond,
			})))

			statusKey := rack.OntologyID(embeddedRackKey(ctx)).String()
			Eventually(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(statusSvc.NewRetrieve().
					Where(status.MatchKeys[any](statusKey)).
					Entries(&statuses).
					Exec(ctx, node.DB)).To(Succeed())
				g.Expect(statuses).To(HaveLen(1))
			}).Should(Succeed())

			Expect(driver.Close()).To(Succeed())

			var lastTime telem.TimeStamp
			var statuses []status.Status[any]
			Expect(statusSvc.NewRetrieve().
				Where(status.MatchKeys[any](statusKey)).
				Entries(&statuses).
				Exec(ctx, node.DB)).To(Succeed())
			lastTime = statuses[0].Time

			Consistently(func(g Gomega) {
				var statuses []status.Status[any]
				g.Expect(statusSvc.NewRetrieve().
					Where(status.MatchKeys[any](statusKey)).
					Entries(&statuses).
					Exec(ctx, node.DB)).To(Succeed())
				g.Expect(statuses[0].Time).To(Equal(lastTime))
			}, time.Millisecond*100, time.Millisecond*25).Should(Succeed())
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
			// (task field expects a UUID string).
			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{taskService.CommandChannelKey()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(
				taskService.CommandChannelKey(),
				MustSucceed(telem.NewJSONSeriesV(
					map[string]any{"task": "not-a-uuid", "type": "start"},
				)),
			))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			Consistently(func() bool { return execCalled.Load() }, time.Millisecond*200, time.Millisecond*50).
				Should(BeFalse())
		})

		It("should execute the start command on the freshly deployed task", func(ctx SpecContext) {
			var (
				execCalled  atomic.Bool
				receivedCmd atomic.Value
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(_ context.Context, cmd task.Command) error {
							if t.Key == taskKey.Load() {
								receivedCmd.Store(cmd)
								execCalled.Store(true)
							}
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			// Allow streamer to boot up
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			cmd := task.Command{
				Task: t.Key,
				Type: "start",
				Key:  "cmd-1",
			}
			writeCommand(ctx, cmd)

			Eventually(func() bool { return execCalled.Load() }, time.Second*2).Should(BeTrue())
			stored := receivedCmd.Load().(task.Command)
			Expect(stored.Type).To(Equal("start"))
			Expect(stored.Key).To(Equal("cmd-1"))
		})

		It("should route non-start commands to the live instance", func(ctx SpecContext) {
			var (
				receivedCmd atomic.Value
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(_ context.Context, cmd task.Command) error {
							if t.Key == taskKey.Load() && cmd.Type == "stop" {
								receivedCmd.Store(cmd)
							}
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			writeCommand(ctx, task.Command{Task: t.Key, Type: "stop", Key: "cmd-2"})

			Eventually(func() bool { return receivedCmd.Load() != nil }, time.Second*2).Should(BeTrue())
			Expect(receivedCmd.Load().(task.Command).Key).To(Equal("cmd-2"))
		})

		It("should ignore non-start commands with no live instance", func(ctx SpecContext) {
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

			t := newTask(embeddedRackKey(ctx))
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "stop", Key: "cmd-1"})

			Consistently(func() bool { return execCalled.Load() }, time.Millisecond*200, time.Millisecond*50).
				Should(BeFalse())
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

			unknownTaskKey := uuid.New()
			cmd := task.Command{
				Task: unknownTaskKey,
				Type: "start",
				Key:  "cmd-unknown",
			}
			writeCommand(ctx, cmd)

			Consistently(func() bool { return execCalled.Load() }, time.Millisecond*200, time.Millisecond*50).Should(BeFalse())
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
			time.Sleep(50 * time.Millisecond)

			otherRack := rack.Rack{Name: "Other Rack for Commands"}
			Expect(rackService.NewWriter(nil).Create(ctx, &otherRack)).To(Succeed())

			t := newTask(otherRack.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			cmd := task.Command{
				Task: t.Key,
				Type: "start",
				Key:  "cmd-other-rack",
			}
			writeCommand(ctx, cmd)

			Consistently(func() bool { return execCalled.Load() }, time.Millisecond*200, time.Millisecond*50).Should(BeFalse())
		})

		It("should handle command execution errors gracefully", func(ctx SpecContext) {
			var (
				execCalled atomic.Bool
				taskKey    atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							if t.Key == taskKey.Load() {
								execCalled.Store(true)
							}
							return errors.New("execution failed")
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-fail"})

			Eventually(func() bool { return execCalled.Load() }, time.Second*2).Should(BeTrue())
		})

		It("should not crash the process when a task's Exec panics", func(ctx SpecContext) {
			var (
				panicExecCalled   atomic.Bool
				healthyExecCalled atomic.Bool
				taskKey           atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(_ context.Context, cmd task.Command) error {
							if t.Key != taskKey.Load() {
								return nil
							}
							if cmd.Type == "panic" {
								panicExecCalled.Store(true)
								panic("boom during exec")
							}
							healthyExecCalled.Store(true)
							return nil
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			// Deploy the task, then trigger the panic via a non-start command.
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})
			Eventually(func() bool { return healthyExecCalled.Load() }, time.Second*2).Should(BeTrue())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "panic", Key: "cmd-panic"})
			Eventually(func() bool { return panicExecCalled.Load() }, time.Second*2).Should(BeTrue())

			healthyExecCalled.Store(false)
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-healthy"})
			Eventually(func() bool { return healthyExecCalled.Load() }, time.Second*2).Should(BeTrue())
		})

		It("should log warning for unsupported command without crashing", func(ctx SpecContext) {
			var (
				execCalled atomic.Bool
				taskKey    atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(context.Context, task.Command) error {
							if t.Key == taskKey.Load() {
								execCalled.Store(true)
							}
							return driver.ErrUnsupportedCommand
						},
					}, nil
				},
			}
			openDriver(ctx, factory)
			time.Sleep(50 * time.Millisecond)

			t := newTask(embeddedRackKey(ctx))
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-unsupported"})

			Eventually(func() bool { return execCalled.Load() }, time.Second*2).Should(BeTrue())
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
				DB:          node.DB,
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
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())
			writeCommand(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-1"})

			Eventually(configureStarted, time.Second).Should(BeClosed())
			// The goroutine should receive context cancellation after the timeout.
			Eventually(func() bool { return timedOut.Load() }).Should(BeTrue())
		})

		It("should timeout a hanging Exec", func(ctx SpecContext) {
			var (
				execStarted = make(chan struct{}, 1)
				taskKey     atomic.Value
			)
			factory := &mockFactory{
				name: "test",
				configureFunc: func(
					_ context.Context,
					t task.Task,
				) (driver.Task, error) {
					return &mockTask{
						key: t.Key,
						execFunc: func(eCtx context.Context, _ task.Command) error {
							if t.Key != taskKey.Load() {
								return nil
							}
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
				DB:          node.DB,
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
			taskKey.Store(t.Key)
			Expect(taskWriter.Create(ctx, &t)).To(Succeed())

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
			d1 := DeferClose(MustSucceed(driver.Open(ctx, driver.Config{
				DB:        node.DB,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Status:    statusSvc,
				Factories: []driver.Factory{&mockFactory{name: "noop"}},
				Host:      hostProvider,
			})))
			rackKey := embeddedRackKey(ctx)
			for range numTasks {
				t := task.Task{
					Rack: rackKey,
					Name: "Parallel Task",
					Type: "test",
				}
				Expect(taskWriter.Create(ctx, &t)).To(Succeed())
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

			var (
				gateOnce  sync.Once
				closeGate = func() { gateOnce.Do(func() { close(configGate) }) }
				d2        *driver.Driver
			)
			openDone := make(chan *driver.Driver, 1)
			// On failure the blocked Open must still complete and close, or
			// its goroutines hang suite-level cleanup.
			DeferCleanup(func() {
				closeGate()
				if d2 == nil {
					select {
					case d2 = <-openDone:
					case <-time.After(10 * time.Second):
						return
					}
				}
				Expect(d2.Close()).To(Succeed())
			})
			go func() {
				defer GinkgoRecover()
				d := MustSucceed(driver.Open(ctx, driver.Config{
					DB:        node.DB,
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
			closeGate()

			Eventually(openDone).Should(Receive(&d2))
		})
	})
})
