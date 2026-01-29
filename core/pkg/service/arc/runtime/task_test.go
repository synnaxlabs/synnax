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
	"github.com/synnaxlabs/x/query"
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

	newFactory := func(g graph.Graph) *runtime.Factory {
		return MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
			Channel: dist.Channel,
			Framer:  dist.Framer,
			Status:  statusSvc,
			GetModule: func(ctx context.Context, key uuid.UUID) (svcarc.Arc, error) {
				resolver := symbol.CreateResolver(dist.Channel)
				module, err := arc.CompileGraph(ctx, g, arc.WithResolver(resolver))
				if err != nil {
					return svcarc.Arc{}, err
				}
				return svcarc.Arc{Key: key, Name: "test-arc", Graph: g, Module: &module}, nil
			},
		}))
	}

	configToMap := func(cfg runtime.TaskConfig) map[string]any {
		cfgJSON := MustSucceed(json.Marshal(cfg))
		var cfgMap map[string]any
		Expect(json.Unmarshal(cfgJSON, &cfgMap)).To(Succeed())
		return cfgMap
	}

	newTask := func(factory *runtime.Factory) driver.Task {
		svcTask := task.Task{
			Key:    task.NewKey(rack.NewKey(1, 1), 1),
			Name:   "test-task",
			Type:   runtime.TaskType,
			Config: configToMap(runtime.TaskConfig{ArcKey: uuid.New()}),
		}
		t := MustBeOk(MustSucceed2(factory.ConfigureTask(newContext(), svcTask)))
		return t
	}

	simpleGraph := func(chKey channel.Key) graph.Graph {
		return graph.Graph{
			Nodes: []graph.Node{{Key: "on", Type: "on", Config: map[string]any{"channel": chKey}}},
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
				Config: map[string]any{},
			}
			t, handled := MustSucceed2(factory.ConfigureTask(newContext(), svcTask))
			Expect(handled).To(BeFalse())
			Expect(t).To(BeNil())
		})

		It("Should create Task for arc type", func() {
			ch := &channel.Channel{Name: "factory_test_ch", Virtual: true, DataType: telem.Float32T}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			t := newTask(newFactory(simpleGraph(ch.Key())))
			Expect(t).ToNot(BeNil())
		})

		It("Should return error for invalid config", func() {
			factory := MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   runtime.TaskType,
				Config: map[string]any{"arc_key": "not-a-valid-uuid"},
			}
			task, ok, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(HaveOccurred())
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
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   runtime.TaskType,
				Config: configToMap(runtime.TaskConfig{ArcKey: uuid.New()}),
			}
			t, handled, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(MatchError(query.ErrNotFound))
			Expect(handled).To(BeTrue())
			Expect(t).To(BeNil())
		})

		It("Should set error status when config is invalid", func() {
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
				Config: map[string]any{"arc_key": "not-a-valid-uuid"},
			}
			_, _, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(HaveOccurred())
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				WhereKeys(task.OntologyID(svcTask.Key).String()).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("error"))
			Expect(stat.Details.Running).To(BeFalse())
		})

		It("Should set error status when GetModule fails", func() {
			factory := MustSucceed(runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: moduleNotFoundGetter,
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 3),
				Name:   "test-module-not-found",
				Type:   runtime.TaskType,
				Config: configToMap(runtime.TaskConfig{ArcKey: uuid.New()}),
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
				Config: configToMap(runtime.TaskConfig{ArcKey: uuid.New()}),
			}
			t, handled := MustSucceed2(
				newFactory(simpleGraph(ch.Key())).
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
				Key:    task.NewKey(rack.NewKey(1, 1), 5),
				Name:   "test-auto-start",
				Type:   runtime.TaskType,
				Config: configToMap(runtime.TaskConfig{ArcKey: uuid.New(), AutoStart: true}),
			}
			t, handled := MustSucceed2(newFactory(
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
			arcTask = newTask(newFactory(simpleGraph(ch.Key())))
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

	Describe("Pipeline Creation", func() {
		It("Should create stream pipeline for read channels", func() {
			ch := &channel.Channel{
				Name:     "stream_test_ch_" + uuid.NewString()[:8],
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			t := newTask(newFactory(simpleGraph(ch.Key())))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(t.Stop(false)).To(Succeed())
		})

		It("Should create write pipeline for write channels", func() {
			indexCh := &channel.Channel{
				Name:     "write_idx_" + uuid.NewString()[:8],
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "write_data_" + uuid.NewString()[:8],
				LocalIndex: indexCh.LocalKey,
				DataType:   telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())
			writeGraph := graph.Graph{
				Nodes: []graph.Node{
					{Key: "const", Type: "constant", Config: map[string]any{"value": 42.0}},
					{Key: "sink", Type: "write", Config: map[string]any{"channel": dataCh.Key()}},
				},
				Edges: []graph.Edge{
					{
						Source: graph.Handle{Node: "const", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "sink", Param: ir.DefaultInputParam},
					},
				},
			}
			t := newTask(newFactory(writeGraph))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(t.Stop(false)).To(Succeed())
		})
	})

	Describe("ConfigureTask Error Paths", func() {
		It("Should return error when graph has unknown node type", func() {
			badNodeGraph := graph.Graph{
				Nodes: []graph.Node{{Key: "bad", Type: "nonexistent_type", Config: map[string]any{}}},
			}
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Name:   "test-bad-node",
				Type:   runtime.TaskType,
				Config: configToMap(runtime.TaskConfig{ArcKey: uuid.New()}),
			}
			_, ok, err := newFactory(badNodeGraph).ConfigureTask(newContext(), svcTask)
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

			t := newTask(newFactory(alarmGraph))
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
})
