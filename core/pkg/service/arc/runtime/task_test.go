// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	svcarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func moduleNotFoundGetter(context.Context, uuid.UUID) (svcarc.Arc, error) {
	return svcarc.Arc{}, query.ErrNotFound
}

var _ = Describe("Task", Ordered, func() {
	var (
		dist      mock.Node
		statusSvc *status.Service
		labelSvc  *label.Service
	)

	ShouldNotLeakGoroutinesBeforeEach()

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc = MustSucceed(label.OpenService(ctx, label.ServiceConfig{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		statusSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Group:    dist.Group,
			Signals:  dist.Signals,
			Ontology: dist.Ontology,
			Label:    labelSvc,
		}))
	})

	AfterAll(func() {
		Expect(labelSvc.Close()).To(Succeed())
		Expect(statusSvc.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
	})

	newContext := func() driver.Context {
		return driver.NewContext(ctx, statusSvc)
	}

	newFactoryWith := func(getModule func(context.Context, uuid.UUID) (svcarc.Arc, error)) *runtime.Factory {
		return MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
			Channel:   dist.Channel,
			Framer:    dist.Framer,
			Status:    statusSvc,
			GetModule: getModule,
		}))
	}

	newGraphFactory := func(g graph.Graph) *runtime.Factory {
		return newFactoryWith(func(ctx context.Context, key uuid.UUID) (svcarc.Arc, error) {
			resolver := symbol.CreateResolver(dist.Channel)
			module, err := arc.CompileGraph(ctx, g, arc.WithResolver(resolver))
			if err != nil {
				return svcarc.Arc{}, err
			}
			return svcarc.Arc{Key: key, Name: "test-arc", Graph: g, Module: module}, nil
		})
	}

	newTextFactory := func(prof arc.Text) *runtime.Factory {
		return newFactoryWith(func(_ context.Context, _ uuid.UUID) (svcarc.Arc, error) {
			resolver := symbol.CreateResolver(dist.Channel)
			module, err := arc.CompileText(ctx, prof, arc.WithResolver(resolver))
			if err != nil {
				return svcarc.Arc{}, err
			}
			return svcarc.Arc{Key: uuid.New(), Name: "test-arc", Text: prof, Module: module}, nil
		})
	}

	newTask := func(factory *runtime.Factory) driver.Task {
		cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
		svcTask := task.Task{
			Key:    task.NewKey(rack.NewKey(1, 1), 1),
			Name:   "test-task",
			Type:   runtime.TaskType,
			Config: string(cfgJSON),
		}
		t := MustBeOk(MustSucceed2(factory.ConfigureTask(newContext(), svcTask)))
		return t
	}

	simpleGraph := func(chKey channel.Key) graph.Graph {
		return graph.Graph{
			Nodes: []graph.Node{{Key: "on", Type: "on", Config: map[string]any{"channel": chKey}}},
		}
	}

	createVirtualCh := func(prefix string, dataType telem.DataType) *channel.Channel {
		ch := &channel.Channel{
			Name:     prefix + "_" + uuid.NewString()[:8],
			Virtual:  true,
			DataType: dataType,
		}
		Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
		return ch
	}

	openTestStreamer := func(keys channel.Keys, bufferSize int) (
		responses <-chan framer.StreamerResponse,
		close func(),
	) {
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        keys,
			SendOpenAck: new(true),
		}))
		requests, res := confluence.Attach(streamer, bufferSize)
		sCtx, cancel := signal.Isolated()
		closer := signal.NewHardShutdown(sCtx, cancel)
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		Eventually(res.Outlet()).Should(Receive())
		return res.Outlet(), func() {
			requests.Close()
			confluence.Drain(res)
			Expect(closer.Close()).To(Succeed())
		}
	}

	Describe("Factory.ConfigureTask", func() {
		It("Should return false for non-arc task types", func() {
			factory := MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
				Channel: dist.Channel,
				Framer:  dist.Framer,
				Status:  statusSvc,
				GetModule: func(context.Context, uuid.UUID) (svcarc.Arc, error) {
					return svcarc.Arc{}, nil
				},
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   "not-arc",
				Config: "{}",
			}
			t, handled := MustSucceed2(factory.ConfigureTask(newContext(), svcTask))
			Expect(handled).To(BeFalse())
			Expect(t).To(BeNil())
		})

		It("Should create Task for arc type", func() {
			ch := &channel.Channel{Name: "factory_test_ch", Virtual: true, DataType: telem.Float32T}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			t := newTask(newGraphFactory(simpleGraph(ch.Key())))
			Expect(t).ToNot(BeNil())
		})

		It("Should return error for invalid config JSON", func() {
			factory := MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   runtime.TaskType,
				Config: "invalid json",
			}
			task, ok, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(MatchError(ContainSubstring("invalid character 'i'")))
			Expect(ok).To(BeTrue())
			Expect(task).To(BeNil())
		})

		It("Should return error when CompileModule fails", func() {
			factory := MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: moduleNotFoundGetter,
			}))
			cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   runtime.TaskType,
				Config: string(cfgJSON),
			}
			t, handled, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(MatchError(query.ErrNotFound))
			Expect(handled).To(BeTrue())
			Expect(t).To(BeNil())
		})

		It("Should set error status when config JSON is invalid", func() {
			factory := MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 2),
				Name:   "test-invalid-config",
				Type:   runtime.TaskType,
				Config: "invalid json",
			}
			_, _, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(HaveOccurred())
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				WhereKeys(task.OntologyID(svcTask.Key).String()).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("error"))
			Expect(stat.Message).To(ContainSubstring("invalid character"))
			Expect(stat.Details.Running).To(BeFalse())
		})

		It("Should set error status when GetModule fails", func() {
			factory := MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: moduleNotFoundGetter,
			}))
			cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 3),
				Name:   "test-module-not-found",
				Type:   runtime.TaskType,
				Config: string(cfgJSON),
			}
			_, _, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(MatchError(query.ErrNotFound))
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				WhereKeys(task.OntologyID(svcTask.Key).String()).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("error"))
			Expect(stat.Message).To(ContainSubstring("not found"))
			Expect(stat.Details.Running).To(BeFalse())
		})

		It("Should set success status when task is configured", func() {
			ch := &channel.Channel{
				Name:     "config_status_test_ch_" + uuid.NewString()[:8],
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 4),
				Name:   "test-config-success",
				Type:   runtime.TaskType,
				Config: string(MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))),
			}
			t, handled := MustSucceed2(
				newGraphFactory(simpleGraph(ch.Key())).
					ConfigureTask(newContext(), svcTask),
			)
			Expect(handled).To(BeTrue())
			Expect(t).ToNot(BeNil())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				WhereKeys(task.OntologyID(svcTask.Key).String()).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("success"))
			Expect(stat.Message).To(Equal("Task configured successfully"))
			Expect(stat.Details.Running).To(BeFalse())
		})

		It("Should auto-start task and set running status when auto_start is true", func() {
			ch := &channel.Channel{
				Name:     "auto_start_test_ch_" + uuid.NewString()[:8],
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			svcTask := task.Task{
				Key:  task.NewKey(rack.NewKey(1, 1), 5),
				Name: "test-auto-start",
				Type: runtime.TaskType,
				Config: string(MustSucceed(json.Marshal(runtime.TaskConfig{
					ArcKey:    uuid.New(),
					AutoStart: true,
				}))),
			}
			t, handled := MustSucceed2(newGraphFactory(
				simpleGraph(ch.Key())).
				ConfigureTask(newContext(), svcTask),
			)
			Expect(handled).To(BeTrue())
			Expect(t).ToNot(BeNil())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				WhereKeys(task.OntologyID(svcTask.Key).String()).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("success"))
			Expect(stat.Message).To(Equal("Task started successfully"))
			Expect(stat.Details.Running).To(BeTrue())
		})
	})

	Describe("Task Lifecycle", func() {
		var arcTask driver.Task

		BeforeEach(func() {
			ch := &channel.Channel{
				Name:     "lifecycle_test_ch_" + uuid.NewString()[:8],
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			arcTask = newTask(newGraphFactory(simpleGraph(ch.Key())))
		})

		AfterEach(func() {
			if arcTask != nil {
				Expect(arcTask.Stop(false)).To(Succeed())
			}
		})

		It("Should start task with start command", func() {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		})

		It("Should be idempotent on start", func() {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		})

		It("Should stop task with stop command", func() {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "stop"})).To(Succeed())
		})

		It("Should be idempotent on stop", func() {
			Expect(arcTask.Stop(false)).To(Succeed())
			Expect(arcTask.Stop(false)).To(Succeed())
		})

		It("Should support restart after stop", func() {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "stop"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		})

		It("Should return error for unknown command type", func() {
			Expect(arcTask.Exec(ctx, task.Command{Type: "unknown"})).
				Error().To(MatchError(ContainSubstring("invalid command")))
		})

		It("Should return correct task key", func() {
			Expect(arcTask.Key()).ToNot(Equal(task.Key(0)))
		})
	})

	Describe("ConfigureTask Error Paths", func() {
		It("Should return error when graph has unknown node type", func() {
			badNodeGraph := graph.Graph{
				Nodes: []graph.Node{{Key: "bad", Type: "nonexistent_type", Config: map[string]any{}}},
			}
			cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Name:   "test-bad-node",
				Type:   runtime.TaskType,
				Config: string(cfgJSON),
			}
			_, ok, err := newGraphFactory(badNodeGraph).ConfigureTask(newContext(), svcTask)
			Expect(ok).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("undefined symbol")))
		})
	})

	Describe("Alarm Flow", func() {
		It("Should update alarm statuses based on telemetry", func() {
			ch := &channel.Channel{Name: "ox_pt_1", Virtual: true, DataType: telem.Float32T}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

			alarmGraph := graph.Graph{
				Nodes: []graph.Node{
					{Key: "on", Type: "on", Config: map[string]any{"channel": ch.Key()}},
					{Key: "constant", Type: "constant", Config: map[string]any{"value": 10}},
					{Key: "ge", Type: "ge"},
					{Key: "stable_for", Type: "stable_for", Config: map[string]any{"duration": 0}},
					{Key: "select", Type: "select"},
					{Key: "status_success", Type: "set_status", Config: map[string]any{
						"status_key": "ox_alarm", "variant": "success", "name": "OX Alarm", "message": "OX Pressure Nominal",
					}},
					{Key: "status_error", Type: "set_status", Config: map[string]any{
						"status_key": "ox_alarm", "variant": "error", "name": "OX Alarm", "message": "OX Pressure Exceed",
					}},
				},
				Edges: []graph.Edge{
					{
						Source: graph.Handle{Node: "on", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "ge", Param: ir.LHSInputParam},
					},
					{
						Source: graph.Handle{Node: "constant", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "ge", Param: ir.RHSInputParam},
					},
					{
						Source: graph.Handle{Node: "ge", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "stable_for", Param: ir.DefaultInputParam},
					},
					{
						Source: graph.Handle{Node: "stable_for", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "select", Param: ir.DefaultOutputParam},
					},
					{
						Source: graph.Handle{Node: "select", Param: "false"},
						Target: graph.Handle{Node: "status_success", Param: ir.DefaultOutputParam},
					},
					{
						Source: graph.Handle{Node: "select", Param: "true"},
						Target: graph.Handle{Node: "status_error", Param: ir.DefaultOutputParam},
					},
				},
			}

			t := newTask(newGraphFactory(alarmGraph))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{ch.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](20)))).To(BeTrue())
			time.Sleep(20 * time.Millisecond)
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](25)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				var stat status.Status[svcarc.StatusDetails]
				g.Expect(status.NewRetrieve[svcarc.StatusDetails](statusSvc).
					WhereKeys("ox_alarm").Entry(&stat).Exec(ctx, nil)).To(Succeed())
				g.Expect(stat.Variant).To(BeEquivalentTo("error"))
			}).Should(Succeed())
		})
	})

	Describe("Interval Timing", func() {
		It("Should fire intervals without any streaming data", func() {
			indexCh := &channel.Channel{
				Name:     "interval_idx_" + uuid.NewString()[:8],
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "interval_data_" + uuid.NewString()[:8],
				LocalIndex: indexCh.LocalKey,
				DataType:   telem.Uint8T,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func output() {
						%s = 42.0
					}
					interval{period=50ms} -> output{}
				`, dataCh.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()
			time.Sleep(10 * time.Millisecond)

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(dataCh.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))

			Eventually(responses).Should(Receive(&fr))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))

			Eventually(responses).Should(Receive(&fr))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))
		})

		It("Should process both intervals and streaming data", func() {
			inputCh := createVirtualCh("combined_input", telem.Float32T)
			outputCh := createVirtualCh("combined_output", telem.Float32T)
			intervalCh := createVirtualCh("combined_interval", telem.Uint8T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func passthrough() {
						%s = %s
					}
					func tick() {
						%s = 1
					}
					%s -> passthrough{}
					interval{period=50ms} -> tick{}
				`, outputCh.Name, inputCh.Name, intervalCh.Name, inputCh.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{
				outputCh.Key(),
				intervalCh.Key(),
			}, 10)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  channel.Keys{inputCh.Key()},
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[float32](99.5)))).To(BeTrue())

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Eventually(responses).Should(Receive(&fr))
			Eventually(responses).Should(Receive(&fr))
		})

		It("Should fire Wait node without streaming data", func() {
			outputCh := createVirtualCh("wait_output", telem.Uint8T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func output() {
						%s = 1
					}
					wait{duration=50ms} -> output{}
				`, outputCh.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{outputCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(outputCh.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(outputCh.Key()).Series[0], 0)).To(Equal(uint8(1)))
		})

		It("Should handle multiple intervals with different periods", func() {
			output1Ch := createVirtualCh("multi_interval_1", telem.Uint8T)
			output2Ch := createVirtualCh("multi_interval_2", telem.Uint8T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func tick1() {
						%s = 1
					}
					func tick2() {
						%s = 2
					}
					interval{period=60ms} -> tick1{}
					interval{period=90ms} -> tick2{}
				`, output1Ch.Name, output2Ch.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{
				output1Ch.Key(),
				output2Ch.Key(),
			}, 10)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var (
				fr     framer.StreamerResponse
				count1 int
				count2 int
			)
			for count1 < 3 || count2 < 2 {
				Eventually(responses).Should(Receive(&fr))
				if fr.Frame.Get(output1Ch.Key()).Len() > 0 {
					Expect(telem.ValueAt[uint8](fr.Frame.Get(output1Ch.Key()).Series[0], 0)).To(Equal(uint8(1)))
					count1++
				}
				if fr.Frame.Get(output2Ch.Key()).Len() > 0 {
					Expect(telem.ValueAt[uint8](fr.Frame.Get(output2Ch.Key()).Series[0], 0)).To(Equal(uint8(2)))
					count2++
				}
			}
		})

		It("Should stop cleanly when only intervals exist", func() {
			outputCh := createVirtualCh("clean_stop", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func tick() {
						%s = 1
					}
					interval{period=10ms} -> tick{}
				`, outputCh.Name),
			}

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			time.Sleep(50 * time.Millisecond)
			Expect(t.Stop(false)).To(Succeed())
		})
	})

	Describe("Control Authority", func() {
		It("Should apply static authority from authority block", func() {
			ch := createVirtualCh("auth_static", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 100
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(200)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})

		It("Should block lower-authority competing writers", func() {
			ch := createVirtualCh("auth_block", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 200
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
		})

		It("Should default to absolute authority without authority block", func() {
			ch := createVirtualCh("auth_default", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(254)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
		})

		It("Should apply per-channel authority overrides", func() {
			ch1 := createVirtualCh("auth_perchan_1", telem.Uint8T)
			ch2 := createVirtualCh("auth_perchan_2", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority (100 %s 200)
					func output() {
						%s = 1
						%s = 2
					}
					interval{period=50ms} -> output{}
				`, ch1.Name, ch1.Name, ch2.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{ch1.Key(), ch2.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			wA := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch1.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()
			Expect(wA.Write(frame.NewUnary(ch1.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())

			wB := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch2.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(wB.Close()).To(Succeed()) }()
			Expect(wB.Write(frame.NewUnary(ch2.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})

		It("Should write data with non-default authority", func() {
			ch := createVirtualCh("auth_write_data", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 100
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(ch.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(ch.Key()).Series[0], 0)).To(Equal(uint8(42)))
		})

		It("Should dynamically escalate authority via set_authority", func() {
			dataCh := createVirtualCh("dyn_esc_data", telem.Uint8T)
			triggerCh := createVirtualCh("dyn_esc_trigger", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 100

					func output() {
						%s = 42
					}

					%s => seq

					sequence seq {
						stage escalated {
							1 -> set_authority{value=200}
						}
					}

					interval{period=50ms} -> output{}
				`, dataCh.Name, triggerCh.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			trigW := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{triggerCh.Key()},
				Start: telem.Now(),
			}))
			Expect(trigW.Write(frame.NewUnary(triggerCh.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(trigW.Close()).To(Succeed())

			// Receive data frames to ensure the runtime has processed the trigger
			Eventually(responses).Should(Receive(&fr))
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{dataCh.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(dataCh.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
		})

		It("Should dynamically de-escalate authority via set_authority", func() {
			dataCh := createVirtualCh("dyn_deesc_data", telem.Uint8T)
			triggerCh := createVirtualCh("dyn_deesc_trigger", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 200

					func output() {
						%s = 42
					}

					%s => seq

					sequence seq {
						stage deescalated {
							1 -> set_authority{value=50}
						}
					}

					interval{period=50ms} -> output{}
				`, dataCh.Name, triggerCh.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			wBefore := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{dataCh.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			Expect(wBefore.Write(frame.NewUnary(dataCh.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
			Expect(wBefore.Close()).To(Succeed())

			trigW := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{triggerCh.Key()},
				Start: telem.Now(),
			}))
			Expect(trigW.Write(frame.NewUnary(triggerCh.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(trigW.Close()).To(Succeed())

			// Receive data frames to ensure the runtime has processed the trigger
			Eventually(responses).Should(Receive(&fr))
			Eventually(responses).Should(Receive(&fr))

			wAfter := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{dataCh.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wAfter.Close()).To(Succeed()) }()
			Expect(wAfter.Write(frame.NewUnary(dataCh.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})

		It("Should continue writing data after dynamic authority change", func() {
			dataCh := createVirtualCh("dyn_cont_data", telem.Uint8T)
			triggerCh := createVirtualCh("dyn_cont_trigger", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 100

					func output() {
						%s = 42
					}

					%s => seq

					sequence seq {
						stage escalated {
							1 -> set_authority{value=200}
						}
					}

					interval{period=50ms} -> output{}
				`, dataCh.Name, triggerCh.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			trigW := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{triggerCh.Key()},
				Start: telem.Now(),
			}))
			Expect(trigW.Write(frame.NewUnary(triggerCh.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(trigW.Close()).To(Succeed())

			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(dataCh.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))
		})
	})

	Describe("Runtime Error Handling", func() {
		It("Should report WASM division by zero via status service", func() {
			inputCh := createVirtualCh("div_zero_input", telem.Int32T)
			outputCh := createVirtualCh("div_zero_output", telem.Int32T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func divide_test() {
						%s = 10 / %s
					}
					%s -> divide_test{}
				`, outputCh.Name, inputCh.Name, inputCh.Name),
			}

			cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 100),
				Name:   "test-div-zero",
				Type:   runtime.TaskType,
				Config: string(cfgJSON),
			}
			t := MustBeOk(MustSucceed2(newTextFactory(prog).ConfigureTask(newContext(), svcTask)))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() {
				Expect(t.Stop(false)).To(Succeed())
			}()

			time.Sleep(20 * time.Millisecond)

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{inputCh.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[int32](0)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			Eventually(func(g Gomega) {
				var stat task.Status
				g.Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
					WhereKeys(task.OntologyID(svcTask.Key).String()).
					Entry(&stat).Exec(ctx, nil)).To(Succeed())
				g.Expect(stat.Variant).To(BeEquivalentTo("warning"))
				g.Expect(stat.Message).To(ContainSubstring("Runtime error in"))
				g.Expect(stat.Message).To(ContainSubstring("divide_test"))
				g.Expect(stat.Description).To(ContainSubstring("integer divide by zero"))
				g.Expect(stat.Details.Running).To(BeTrue())
			}).Should(Succeed())
		})

		It("Should read config param channel value correctly", func() {
			inputCh := createVirtualCh("cfg_read_input", telem.Uint8T)
			maxCh := createVirtualCh("cfg_read_max", telem.Float32T)
			counterCh := createVirtualCh("cfg_read_counter", telem.Float32T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func count_rising_test{counter_ch chan f32, max_ch chan f32}(input u8) {
						prev $= input
						counter f32 $= 0
						read_val := max_ch + f32(0.0)

						if counter < read_val {
							counter = read_val
						}

						if input and not prev {
							counter = counter + 1.0
						}

						counter_ch = counter
						prev = input
					}

					%s -> count_rising_test{counter_ch=%s, max_ch=%s}
				`, inputCh.Name, counterCh.Name, maxCh.Name),
			}

			responses, closeStreamer := openTestStreamer(channel.Keys{counterCh.Key()}, 10)
			defer closeStreamer()

			t := newTask(newTextFactory(prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)

			// Write max value of 5.0 to the max channel
			wMax := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{maxCh.Key()},
				Start: telem.Now(),
			}))
			Expect(wMax.Write(frame.NewUnary(maxCh.Key(), telem.NewSeriesV[float32](5.0)))).To(BeTrue())
			Expect(wMax.Close()).To(Succeed())

			time.Sleep(20 * time.Millisecond)

			// Write a rising edge (0 -> 1) to the input channel
			wInput := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{inputCh.Key()},
				Start: telem.Now(),
			}))
			Expect(wInput.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[uint8](0)))).To(BeTrue())
			time.Sleep(20 * time.Millisecond)
			Expect(wInput.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(wInput.Close()).To(Succeed())

			// The counter should have picked up the max value (5.0) and then
			// incremented to 6.0 on the rising edge
			var foundExpected bool
			for i := 0; i < 10 && !foundExpected; i++ {
				var fr framer.StreamerResponse
				Eventually(responses).Should(Receive(&fr))
				series := fr.Frame.Get(counterCh.Key())
				if series.Len() > 0 {
					val := telem.ValueAt[float32](series.Series[0], -1)
					if val >= 5.0 {
						foundExpected = true
					}
				}
			}
			Expect(foundExpected).To(BeTrue(), "Expected counter to reflect max_ch value (>= 5.0)")
		})

		It("Should continue execution after runtime error", func() {
			inputCh := createVirtualCh("recover_input", telem.Int32T)
			outputCh := createVirtualCh("recover_output", telem.Int32T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func divide_recover() {
						%s = 10 / %s
					}
					%s -> divide_recover{}
				`, outputCh.Name, inputCh.Name, inputCh.Name),
			}

			cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 101),
				Name:   "test-div-recover",
				Type:   runtime.TaskType,
				Config: string(cfgJSON),
			}
			t := MustBeOk(MustSucceed2(newTextFactory(prog).ConfigureTask(newContext(), svcTask)))

			responses, closeStreamer := openTestStreamer(channel.Keys{outputCh.Key()}, 5)
			defer closeStreamer()

			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(false)).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{inputCh.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[int32](0)))).To(BeTrue())

			Eventually(func(g Gomega) {
				var stat task.Status
				g.Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
					WhereKeys(task.OntologyID(svcTask.Key).String()).
					Entry(&stat).Exec(ctx, nil)).To(Succeed())
				g.Expect(stat.Variant).To(BeEquivalentTo("warning"))
				g.Expect(stat.Description).To(ContainSubstring("integer divide by zero"))
			}).Should(Succeed())

			Expect(w.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[int32](2)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			var foundValid bool
			for i := 0; i < 5 && !foundValid; i++ {
				var fr framer.StreamerResponse
				Eventually(responses).Should(Receive(&fr))
				if fr.Frame.Get(outputCh.Key()).Len() > 0 {
					val := telem.ValueAt[int32](fr.Frame.Get(outputCh.Key()).Series[0], 0)
					if val == 5 {
						foundValid = true
					}
				}
			}
			Expect(foundValid).To(BeTrue(), "Expected to receive valid output (5) after error")
		})
	})
})
