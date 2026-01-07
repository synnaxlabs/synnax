// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	godriver "github.com/synnaxlabs/synnax/pkg/driver/go"
	svcarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Task", Ordered, func() {
	var (
		dist      mock.Node
		statusSvc *status.Service
		labelSvc  *label.Service
	)

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc = MustSucceed(label.OpenService(ctx, label.Config{
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

	newContext := func() godriver.Context {
		return godriver.NewContext(ctx, statusSvc)
	}

	newFactory := func(g graph.Graph) *runtime.Factory {
		return runtime.NewFactory(runtime.FactoryConfig{
			Channel: dist.Channel,
			Framer:  dist.Framer,
			Status:  statusSvc,
			GetModule: func(ctx context.Context, key uuid.UUID) (svcarc.Arc, error) {
				resolver := symbol.CreateResolver(dist.Channel)
				module := MustSucceed(arc.CompileGraph(ctx, g, arc.WithResolver(resolver)))
				return svcarc.Arc{Key: key, Name: "test-arc", Graph: g, Module: module}, nil
			},
		})
	}

	newTask := func(factory *runtime.Factory) godriver.Task {
		cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
		svcTask := task.Task{
			Key:    task.NewKey(rack.NewKey(1, 1), 1),
			Name:   "test-task",
			Type:   runtime.TaskType,
			Config: string(cfgJSON),
		}
		t, handled, err := factory.ConfigureTask(newContext(), svcTask)
		Expect(err).ToNot(HaveOccurred())
		Expect(handled).To(BeTrue())
		return t
	}

	simpleGraph := func(chKey channel.Key) graph.Graph {
		return graph.Graph{
			Nodes: []graph.Node{{Key: "on", Type: "on", Config: map[string]any{"channel": chKey}}},
		}
	}

	Describe("Factory.ConfigureTask", func() {
		It("Should return false for non-arc task types", func() {
			factory := runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
			})
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   "not-arc",
				Config: "{}",
			}
			t, handled, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).ToNot(HaveOccurred())
			Expect(handled).To(BeFalse())
			Expect(t).To(BeNil())
		})

		It("Should create Task for arc type", func() {
			ch := &channel.Channel{Name: "factory_test_ch", Virtual: true, DataType: telem.Float32T}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

			t := newTask(newFactory(simpleGraph(ch.Key())))
			Expect(t).ToNot(BeNil())
		})

		It("Should return error for invalid config JSON", func() {
			factory := runtime.NewFactory(runtime.FactoryConfig{
				Channel:   dist.Channel,
				Framer:    dist.Framer,
				Status:    statusSvc,
				GetModule: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
			})
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   runtime.TaskType,
				Config: "invalid json",
			}
			t, handled, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(HaveOccurred())
			Expect(handled).To(BeTrue())
			Expect(t).To(BeNil())
		})

		It("Should return error when GetModule fails", func() {
			factory := runtime.NewFactory(runtime.FactoryConfig{
				Channel: dist.Channel,
				Framer:  dist.Framer,
				Status:  statusSvc,
				GetModule: func(context.Context, uuid.UUID) (svcarc.Arc, error) {
					return svcarc.Arc{}, errors.New("module not found")
				},
			})
			cfgJSON := MustSucceed(json.Marshal(runtime.TaskConfig{ArcKey: uuid.New()}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   runtime.TaskType,
				Config: string(cfgJSON),
			}
			t, handled, err := factory.ConfigureTask(newContext(), svcTask)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("module not found"))
			Expect(handled).To(BeTrue())
			Expect(t).To(BeNil())
		})
	})

	Describe("Task Lifecycle", func() {
		var arcTask godriver.Task

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
				_ = arcTask.Stop(ctx, false)
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
			Expect(arcTask.Stop(ctx, false)).To(Succeed())
			Expect(arcTask.Stop(ctx, false)).To(Succeed())
		})

		It("Should support restart after stop", func() {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "stop"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		})

		It("Should handle unknown command types gracefully", func() {
			Expect(arcTask.Exec(ctx, task.Command{Type: "unknown"})).To(Succeed())
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
					{Source: graph.Handle{Node: "on", Param: ir.DefaultOutputParam}, Target: graph.Handle{Node: "ge", Param: ir.LHSInputParam}},
					{Source: graph.Handle{Node: "constant", Param: ir.DefaultOutputParam}, Target: graph.Handle{Node: "ge", Param: ir.RHSInputParam}},
					{Source: graph.Handle{Node: "ge", Param: ir.DefaultOutputParam}, Target: graph.Handle{Node: "stable_for", Param: ir.DefaultInputParam}},
					{Source: graph.Handle{Node: "stable_for", Param: ir.DefaultOutputParam}, Target: graph.Handle{Node: "select", Param: ir.DefaultOutputParam}},
					{Source: graph.Handle{Node: "select", Param: "false"}, Target: graph.Handle{Node: "status_success", Param: ir.DefaultOutputParam}},
					{Source: graph.Handle{Node: "select", Param: "true"}, Target: graph.Handle{Node: "status_error", Param: ir.DefaultOutputParam}},
				},
			}

			t := newTask(newFactory(alarmGraph))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop(ctx, false)).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{ch.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(core.UnaryFrame(ch.Key(), telem.NewSeriesV[float32](20)))).To(BeTrue())
			time.Sleep(20 * time.Millisecond)
			Expect(w.Write(core.UnaryFrame(ch.Key(), telem.NewSeriesV[float32](25)))).To(BeTrue())
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
