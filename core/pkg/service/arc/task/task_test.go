// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	svcarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// graphNodeSpec describes a graph node along with its function type and configuration
// parameter values, which live in the graph's Configs map keyed by node key.
type graphNodeSpec struct {
	key string
	typ string
	cfg map[string]any
}

// buildGraphNodes converts node specs into the graph's Nodes slice and Configs map,
// storing each node's function type under the "type" key in its config entry.
func buildGraphNodes(specs ...graphNodeSpec) (graph.Nodes, map[string]msgpack.EncodedJSON) {
	nodes := make(graph.Nodes, len(specs))
	configs := make(map[string]msgpack.EncodedJSON, len(specs))
	for i, s := range specs {
		nodes[i] = graph.Node{Key: s.key}
		cfg := msgpack.EncodedJSON{"type": s.typ}
		maps.Copy(cfg, s.cfg)
		configs[s.key] = cfg
	}
	return nodes, configs
}

func moduleNotFoundGetter(context.Context, uuid.UUID) (svcarc.Arc, error) {
	return svcarc.Arc{}, query.ErrNotFound
}

var _ = Describe("Task", Ordered, func() {
	var (
		node       mock.Node
		statusSvc  *status.Service
		channelSvc *channel.Service
		framerSvc  *framer.Service
		rangerSvc  *ranger.Service
	)

	BeforeAll(func(ctx SpecContext) {
		node = mock.NewNode(ctx)
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       node.DB,
			Ontology: node.Ontology,
			Group:    node.Group,
			Search:   node.Search,
		}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Group:    node.Group,
			Ontology: node.Ontology,
			Label:    labelSvc,
			Search:   node.Search,
		}))
		channelSvc = MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			Status:  statusSvc,
		}))
		framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
			Framer:  node.Framer,
			Channel: channelSvc,
			Status:  statusSvc,
		}))
		rangerSvc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       node.DB,
			Ontology: node.Ontology,
			Group:    node.Group,
			Label:    labelSvc,
			Search:   node.Search,
		}))
	})

	newFactoryWith := func(getModule func(context.Context, uuid.UUID) (svcarc.Arc, error)) driver.Factory {
		return MustSucceed(arctask.NewFactory(arctask.FactoryConfig{
			Channel:    channelSvc,
			Framer:     framerSvc,
			Status:     statusSvc,
			GetProgram: getModule,
			Ranger:     rangerSvc,
		}))
	}

	newGraphFactory := func(g graph.Graph) driver.Factory {
		return newFactoryWith(func(ctx context.Context, key uuid.UUID) (svcarc.Arc, error) {
			resolver := channelSvc.NewArcSymbolResolver(nil)
			root := arc.NewRoot(resolver, arcstatus.NewSymbols()...)
			module, err := arc.CompileGraph(ctx, g, root)
			if err != nil {
				return svcarc.Arc{}, err
			}
			return svcarc.Arc{Key: key, Name: "test-arc", Graph: g, Program: &module}, nil
		})
	}

	newTextFactory := func(ctx context.Context, prof arc.Text) driver.Factory {
		return newFactoryWith(func(_ context.Context, _ uuid.UUID) (svcarc.Arc, error) {
			resolver := channelSvc.NewArcSymbolResolver(nil)
			root := arc.NewRoot(resolver, arcstatus.NewSymbols()...)
			module, err := arc.CompileText(ctx, prof, root)
			if err != nil {
				return svcarc.Arc{}, err
			}
			return svcarc.Arc{Key: uuid.New(), Name: "test-arc", Text: prof, Program: &module}, nil
		})
	}

	configToMap := func(cfg arctask.Config) map[string]any {
		cfgJSON := MustSucceed(json.Marshal(cfg))
		var cfgMap map[string]any
		Expect(json.Unmarshal(cfgJSON, &cfgMap)).To(Succeed())
		return cfgMap
	}

	newTask := func(ctx context.Context, factory driver.Factory) driver.Task {
		svcTask := task.Task{
			Key:    task.NewKey(rack.NewKey(1, 1), 1),
			Name:   "test-task",
			Type:   arctask.Type,
			Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
		}
		return MustSucceed(factory.ConfigureTask(ctx, svcTask))
	}

	simpleGraph := func(chKey channel.Key) graph.Graph {
		nodes, configs := buildGraphNodes(
			graphNodeSpec{key: "on", typ: "on", cfg: map[string]any{"channel": chKey}},
		)
		return graph.Graph{Nodes: nodes, Configs: configs}
	}

	createVirtualCh := func(ctx context.Context, prefix string, dataType telem.DataType) *channel.Channel {
		ch := &channel.Channel{
			Name:     prefix + "_" + uuid.NewString()[:8],
			Virtual:  true,
			DataType: dataType,
		}
		Expect(channelSvc.Create(ctx, ch)).To(Succeed())
		return ch
	}

	openTestStreamer := func(ctx context.Context, keys channel.Keys, bufferSize int) (
		responses <-chan framer.StreamerResponse,
		close func(),
	) {
		streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        keys,
			SendOpenAck: true,
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

	bangBangProg := func(ch1, ch2, stopSignal, startSignal *channel.Channel) arc.Text {
		return arc.Text{
			Raw: fmt.Sprintf(`
				authority (%s 210 %s 210)

				func high_bang() {
					%s = 1
				}

				func low_bang() {
					%s = 1
				}

				sequence bb {
					stage start {
						set_authority{value=220, channel=%s},
						set_authority{value=220, channel=%s},
						interval{period=50ms} -> high_bang{},
						interval{period=50ms} -> low_bang{},
						%s => stop
					}
					stage stop {
						0 -> %s,
						0 -> %s,
						wait{duration=100ms} => yield
					}
					stage yield {
						set_authority{value=0, channel=%s},
						set_authority{value=0, channel=%s},
						%s => start
					}
				}

				%s => bb
			`,
				ch1.Name, ch2.Name,
				ch1.Name,
				ch2.Name,
				ch1.Name, ch2.Name,
				stopSignal.Name,
				ch1.Name, ch2.Name,
				ch1.Name, ch2.Name,
				startSignal.Name,
				startSignal.Name,
			),
		}
	}

	Describe("Factory.ConfigureTask", func() {
		It("Should return ErrTaskNotHandled for non-arc task types", func(ctx SpecContext) {
			factory := MustSucceed(arctask.NewFactory(arctask.FactoryConfig{
				Channel: channelSvc,
				Framer:  framerSvc,
				Status:  statusSvc,
				Ranger:  rangerSvc,
				GetProgram: func(context.Context, uuid.UUID) (svcarc.Arc, error) {
					return svcarc.Arc{}, nil
				},
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   "not-arc",
				Config: map[string]any{},
			}
			Expect(factory.ConfigureTask(ctx, svcTask)).Error().
				To(MatchError(driver.ErrTaskNotHandled))
		})

		It("Should create Task for arc type", func(ctx SpecContext) {
			ch := &channel.Channel{Name: "factory_test_ch", Virtual: true, DataType: telem.Float32T}
			Expect(channelSvc.Create(ctx, ch)).To(Succeed())
			t := newTask(ctx, newGraphFactory(simpleGraph(ch.Key())))
			Expect(t).ToNot(BeNil())
		})

		It("Should return error for invalid config", func(ctx SpecContext) {
			factory := MustSucceed(arctask.NewFactory(arctask.FactoryConfig{
				Channel:    channelSvc,
				Framer:     framerSvc,
				Status:     statusSvc,
				GetProgram: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
				Ranger:     rangerSvc,
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   arctask.Type,
				Config: map[string]any{"arc_key": "not-a-valid-uuid"},
			}
			Expect(factory.ConfigureTask(ctx, svcTask)).Error().
				To(HaveOccurred())
		})

		It("Should return error when CompileProgram fails", func(ctx SpecContext) {
			factory := MustSucceed(arctask.NewFactory(arctask.FactoryConfig{
				Channel:    channelSvc,
				Framer:     framerSvc,
				Status:     statusSvc,
				GetProgram: moduleNotFoundGetter,
				Ranger:     rangerSvc,
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			Expect(factory.ConfigureTask(ctx, svcTask)).Error().
				To(MatchError(query.ErrNotFound))
		})

		It("Should set error status when config is invalid", func(ctx SpecContext) {
			factory := MustSucceed(arctask.NewFactory(arctask.FactoryConfig{
				Channel:    channelSvc,
				Framer:     framerSvc,
				Status:     statusSvc,
				GetProgram: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
				Ranger:     rangerSvc,
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 2),
				Name:   "test-invalid-config",
				Type:   arctask.Type,
				Config: map[string]any{"arc_key": "not-a-valid-uuid"},
			}
			Expect(factory.ConfigureTask(ctx, svcTask)).Error().
				To(HaveOccurred())
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				Where(status.MatchKeys[task.StatusDetails](task.OntologyID(svcTask.Key).String())).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("error"))
			Expect(stat.Message).To(ContainSubstring("invalid UUID"))
			Expect(stat.Details.Running).To(BeFalse())
		})

		It("Should set error status when GetProgram fails", func(ctx SpecContext) {
			factory := MustSucceed(arctask.NewFactory(arctask.FactoryConfig{
				Channel:    channelSvc,
				Framer:     framerSvc,
				Status:     statusSvc,
				GetProgram: moduleNotFoundGetter,
				Ranger:     rangerSvc,
			}))
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 3),
				Name:   "test-module-not-found",
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			Expect(factory.ConfigureTask(ctx, svcTask)).Error().
				To(MatchError(query.ErrNotFound))
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				Where(status.MatchKeys[task.StatusDetails](task.OntologyID(svcTask.Key).String())).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("error"))
			Expect(stat.Message).To(ContainSubstring("not found"))
			Expect(stat.Details.Running).To(BeFalse())
		})

		It("Should set success status when task is configured", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     "config_status_test_ch_" + uuid.NewString()[:8],
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(channelSvc.Create(ctx, ch)).To(Succeed())
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 4),
				Name:   "test-config-success",
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			t := MustSucceed(
				newGraphFactory(simpleGraph(ch.Key())).
					ConfigureTask(ctx, svcTask),
			)
			Expect(t).ToNot(BeNil())
			defer func() { Expect(t.Stop()).To(Succeed()) }()
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				Where(status.MatchKeys[task.StatusDetails](task.OntologyID(svcTask.Key).String())).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("success"))
			Expect(stat.Message).To(Equal("Task configured successfully"))
			Expect(stat.Details.Running).To(BeFalse())
		})

		It("Should auto-start task and set running status when auto_start is true", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     "auto_start_test_ch_" + uuid.NewString()[:8],
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(channelSvc.Create(ctx, ch)).To(Succeed())
			svcTask := task.Task{
				Key:  task.NewKey(rack.NewKey(1, 1), 5),
				Name: "test-auto-start",
				Type: arctask.Type,
				Config: configToMap(arctask.Config{
					ArcKey:    uuid.New(),
					AutoStart: true,
				}),
			}
			t := MustSucceed(newGraphFactory(
				simpleGraph(ch.Key())).
				ConfigureTask(ctx, svcTask))
			Expect(t).ToNot(BeNil())
			defer func() { Expect(t.Stop()).To(Succeed()) }()
			var stat task.Status
			Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
				Where(status.MatchKeys[task.StatusDetails](task.OntologyID(svcTask.Key).String())).
				Entry(&stat).Exec(ctx, nil)).To(Succeed())
			Expect(stat.Variant).To(BeEquivalentTo("success"))
			Expect(stat.Message).To(Equal("Task started successfully"))
			Expect(stat.Details.Running).To(BeTrue())
		})

	})

	Describe("FactoryConfig", func() {
		full := func() arctask.FactoryConfig {
			return arctask.FactoryConfig{
				Channel:    channelSvc,
				Framer:     framerSvc,
				Status:     statusSvc,
				GetProgram: func(context.Context, uuid.UUID) (svcarc.Arc, error) { return svcarc.Arc{}, nil },
				Ranger:     rangerSvc,
			}
		}

		Describe("Validate", func() {
			It("Should succeed when all required fields are set", func() {
				Expect(full().Validate()).To(Succeed())
			})

			DescribeTable("Should fail when a required field is missing",
				func(clear func(*arctask.FactoryConfig), field string) {
					cfg := full()
					clear(&cfg)
					Expect(cfg.Validate()).To(MatchError(ContainSubstring(field)))
				},
				Entry("channel", func(c *arctask.FactoryConfig) { c.Channel = nil }, "channel"),
				Entry("framer", func(c *arctask.FactoryConfig) { c.Framer = nil }, "framer"),
				Entry("status", func(c *arctask.FactoryConfig) { c.Status = nil }, "status"),
				Entry("get_program", func(c *arctask.FactoryConfig) { c.GetProgram = nil }, "get_program"),
				Entry("ranger", func(c *arctask.FactoryConfig) { c.Ranger = nil }, "ranger"),
			)
		})

		Describe("Override", func() {
			It("Should prefer the other config's fields when set", func() {
				src := full()
				merged := arctask.FactoryConfig{}.Override(src)
				Expect(merged.Channel).To(BeIdenticalTo(src.Channel))
				Expect(merged.Framer).To(BeIdenticalTo(src.Framer))
				Expect(merged.Status).To(BeIdenticalTo(src.Status))
				Expect(merged.Ranger).To(BeIdenticalTo(src.Ranger))
				Expect(merged.GetProgram).ToNot(BeNil())
			})

			It("Should preserve the receiver's fields when other's are nil", func() {
				src := full()
				merged := src.Override(arctask.FactoryConfig{})
				Expect(merged.Channel).To(BeIdenticalTo(src.Channel))
				Expect(merged.Framer).To(BeIdenticalTo(src.Framer))
				Expect(merged.Status).To(BeIdenticalTo(src.Status))
				Expect(merged.Ranger).To(BeIdenticalTo(src.Ranger))
			})
		})
	})

	Describe("Task Lifecycle", func() {
		var arcTask driver.Task

		BeforeEach(func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     "lifecycle_test_ch_" + uuid.NewString()[:8],
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(channelSvc.Create(ctx, ch)).To(Succeed())
			arcTask = newTask(ctx, newGraphFactory(simpleGraph(ch.Key())))
		})

		AfterEach(func() {
			if arcTask != nil {
				Expect(arcTask.Stop()).To(Succeed())
			}
		})

		It("Should start task with start command", func(ctx SpecContext) {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		})

		It("Should be idempotent on start", func(ctx SpecContext) {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		})

		It("Should stop task with stop command", func(ctx SpecContext) {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "stop"})).To(Succeed())
		})

		It("Should be idempotent on stop", func(ctx SpecContext) {
			Expect(arcTask.Stop()).To(Succeed())
			Expect(arcTask.Stop()).To(Succeed())
		})

		It("Should support restart after stop", func(ctx SpecContext) {
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "stop"})).To(Succeed())
			Expect(arcTask.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		})

		It("Should return error for unknown command type", func(ctx SpecContext) {
			Expect(arcTask.Exec(ctx, task.Command{Type: "unknown"})).
				Error().To(MatchError(ContainSubstring("unsupported command")))
		})
	})

	Describe("ConfigureTask Error Paths", func() {
		It("Should return error when graph has unknown node type", func(ctx SpecContext) {
			badNodes, badConfigs := buildGraphNodes(
				graphNodeSpec{key: "bad", typ: "nonexistent_type"},
			)
			badNodeGraph := graph.Graph{Nodes: badNodes, Configs: badConfigs}
			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 1),
				Name:   "test-bad-node",
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			Expect(newGraphFactory(badNodeGraph).ConfigureTask(ctx, svcTask)).
				Error().To(MatchError(ContainSubstring("undefined symbol")))
		})
	})

	Describe("Alarm Flow", func() {
		It("Should update alarm statuses based on telemetry", func(ctx SpecContext) {
			ch := &channel.Channel{Name: "ox_pt_1", Virtual: true, DataType: telem.Float32T}
			Expect(channelSvc.Create(ctx, ch)).To(Succeed())

			alarmNodes, alarmConfigs := buildGraphNodes(
				graphNodeSpec{key: "on", typ: "on", cfg: map[string]any{"channel": ch.Key()}},
				graphNodeSpec{key: "constant", typ: "constant", cfg: map[string]any{"value": 10}},
				graphNodeSpec{key: "ge", typ: "ge"},
				graphNodeSpec{key: "stable_for", typ: "stable_for", cfg: map[string]any{"duration": 0}},
				graphNodeSpec{key: "select", typ: "select"},
				graphNodeSpec{key: "status_success", typ: "status.set", cfg: map[string]any{
					"key_or_name": "ox_alarm", "message": "OX Pressure Nominal", "variant": "success",
				}},
				graphNodeSpec{key: "status_error", typ: "status.set", cfg: map[string]any{
					"key_or_name": "ox_alarm", "message": "OX Pressure Exceed", "variant": "error",
				}},
			)
			alarmGraph := graph.Graph{
				Nodes:   alarmNodes,
				Configs: alarmConfigs,
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: graph.Handle{Node: "on", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "ge", Param: ir.LHSInputParam},
					}},
					{Edge: ir.Edge{
						Source: graph.Handle{Node: "constant", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "ge", Param: ir.RHSInputParam},
					}},
					{Edge: ir.Edge{
						Source: graph.Handle{Node: "ge", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "stable_for", Param: ir.DefaultInputParam},
					}},
					{Edge: ir.Edge{
						Source: graph.Handle{Node: "stable_for", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "select", Param: ir.DefaultOutputParam},
					}},
					// status_success/error fire on select outputs (edges below).
					{Edge: ir.Edge{
						Source: graph.Handle{Node: "select", Param: "false"},
						Target: graph.Handle{Node: "status_success", Param: ir.DefaultOutputParam},
					}},
					{Edge: ir.Edge{
						Source: graph.Handle{Node: "select", Param: "true"},
						Target: graph.Handle{Node: "status_error", Param: ir.DefaultOutputParam},
					}},
				},
			}

			t := newTask(ctx, newGraphFactory(alarmGraph))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
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
					Where(status.Match(func(_ gorp.Context, _ status.Retrieve[svcarc.StatusDetails], s *status.Status[svcarc.StatusDetails]) (bool, error) {
						return s.Name == "ox_alarm", nil
					})).Entry(&stat).Exec(ctx, nil)).To(Succeed())
				g.Expect(stat.Variant).To(BeEquivalentTo("error"))
			}).Should(Succeed())
		})

	})

	Describe("Sequence with consecutive status.set steps", func() {
		It("Should advance through every status.set step", func(ctx SpecContext) {
			trig := createVirtualCh(ctx, "seq_status_trig", telem.Uint8T)
			base := "seq_status_" + uuid.NewString()[:8]
			prog := arc.Text{Raw: fmt.Sprintf(`
				import status

				sequence main {
				    status.set{key_or_name="%[1]s_a", message="m", variant="info"},
				    status.set{key_or_name="%[1]s_b", message="m", variant="error"},
				    status.set{key_or_name="%[1]s_c", message="m", variant="warning"},
				    status.set{key_or_name="%[1]s_d", message="m", variant="loading"},
				}

				%[2]s => main
			`, base, trig.Name)}

			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 7),
				Name:   "test-status-sequence",
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			t := MustSucceed(newTextFactory(ctx, prog).ConfigureTask(ctx, svcTask))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)
			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{trig.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(trig.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			byName := func(name string) status.Status[svcarc.StatusDetails] {
				var stat status.Status[svcarc.StatusDetails]
				Expect(status.NewRetrieve[svcarc.StatusDetails](statusSvc).
					Where(status.Match(func(_ gorp.Context, _ status.Retrieve[svcarc.StatusDetails], s *status.Status[svcarc.StatusDetails]) (bool, error) {
						return s.Name == name, nil
					})).Entry(&stat).Exec(ctx, nil)).To(Succeed())
				return stat
			}

			Eventually(func(g Gomega) {
				g.Expect(status.NewRetrieve[svcarc.StatusDetails](statusSvc).
					Where(status.Match(func(_ gorp.Context, _ status.Retrieve[svcarc.StatusDetails], s *status.Status[svcarc.StatusDetails]) (bool, error) {
						return s.Name == base+"_d", nil
					})).Entry(&status.Status[svcarc.StatusDetails]{}).Exec(ctx, nil)).To(Succeed())
			}).Should(Succeed())

			Expect(byName(base + "_b").Variant).To(BeEquivalentTo("error"))
			Expect(byName(base + "_c").Variant).To(BeEquivalentTo("warning"))
			Expect(byName(base + "_d").Variant).To(BeEquivalentTo("loading"))
		})
	})

	Describe("Status Reporting", func() {
		It("Should set a task-level warning when status.set matches multiple statuses by name", func(ctx SpecContext) {
			ch := &channel.Channel{Name: "report_trigger", Virtual: true, DataType: telem.Float32T}
			Expect(channelSvc.Create(ctx, ch)).To(Succeed())

			dupName := "dup_alarm_" + uuid.NewString()[:8]
			w := status.NewWriter[any](statusSvc, nil)
			Expect(w.Set(ctx, &status.Status[any]{
				Key: uuid.NewString(), Name: dupName, Variant: status.VariantInfo,
				Message: "first", Time: telem.Now(),
			})).To(Succeed())
			Expect(w.Set(ctx, &status.Status[any]{
				Key: uuid.NewString(), Name: dupName, Variant: status.VariantInfo,
				Message: "second", Time: telem.Now(),
			})).To(Succeed())

			reportNodes, reportConfigs := buildGraphNodes(
				graphNodeSpec{key: "on", typ: "on", cfg: map[string]any{"channel": ch.Key()}},
				graphNodeSpec{key: "status_set", typ: "status.set", cfg: map[string]any{
					"key_or_name": dupName, "message": "ping", "variant": "success",
				}},
			)
			reportGraph := graph.Graph{
				Nodes:   reportNodes,
				Configs: reportConfigs,
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: graph.Handle{Node: "on", Param: ir.DefaultOutputParam},
						Target: graph.Handle{Node: "status_set", Param: ir.DefaultOutputParam},
					}},
				},
			}

			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 42),
				Name:   "test-status-report",
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			t := MustSucceed(newGraphFactory(reportGraph).ConfigureTask(ctx, svcTask))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)
			fw := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{ch.Key()},
				Start: telem.Now(),
			}))
			Expect(fw.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1)))).To(BeTrue())
			Expect(fw.Close()).To(Succeed())

			taskKey := task.OntologyID(svcTask.Key).String()
			Eventually(func(g Gomega) {
				var stat task.Status
				g.Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
					Where(status.MatchKeys[task.StatusDetails](taskKey)).
					Entry(&stat).Exec(ctx, nil)).To(Succeed())
				g.Expect(stat.Variant).To(BeEquivalentTo("warning"))
				g.Expect(stat.Message).To(ContainSubstring("multiple statuses named"))
				g.Expect(stat.Message).To(ContainSubstring("test-status-report"))
			}).Should(Succeed())
		})
	})

	Describe("Interval Timing", func() {
		It("Should fire intervals without any streaming data", func(ctx SpecContext) {
			indexCh := &channel.Channel{
				Name:     "interval_idx_" + uuid.NewString()[:8],
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(channelSvc.Create(ctx, indexCh)).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "interval_data_" + uuid.NewString()[:8],
				LocalIndex: indexCh.LocalKey,
				DataType:   telem.Uint8T,
			}
			Expect(channelSvc.Create(ctx, dataCh)).To(Succeed())

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func output() {
						%s = 42.0
					}
					interval{period=50ms} -> output{}
				`, dataCh.Name),
			}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()
			time.Sleep(10 * time.Millisecond)

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(dataCh.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))

			Eventually(responses).Should(Receive(&fr))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))

			Eventually(responses).Should(Receive(&fr))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))
		})

		It("Should process both intervals and streaming data", func(ctx SpecContext) {
			inputCh := createVirtualCh(ctx, "combined_input", telem.Float32T)
			outputCh := createVirtualCh(ctx, "combined_output", telem.Float32T)
			intervalCh := createVirtualCh(ctx, "combined_interval", telem.Uint8T)

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

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{
				outputCh.Key(),
				intervalCh.Key(),
			}, 10)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
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

		It("Should fire Wait node without streaming data", func(ctx SpecContext) {
			outputCh := createVirtualCh(ctx, "wait_output", telem.Uint8T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func output() {
						%s = 1
					}
					wait{duration=50ms} -> output{}
				`, outputCh.Name),
			}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{outputCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(outputCh.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(outputCh.Key()).Series[0], 0)).To(Equal(uint8(1)))
		})

		It("Should handle multiple intervals with different periods", func(ctx SpecContext) {
			output1Ch := createVirtualCh(ctx, "multi_interval_1", telem.Uint8T)
			output2Ch := createVirtualCh(ctx, "multi_interval_2", telem.Uint8T)

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

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{
				output1Ch.Key(),
				output2Ch.Key(),
			}, 10)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

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

		It("Should stop cleanly when only intervals exist", func(ctx SpecContext) {
			outputCh := createVirtualCh(ctx, "clean_stop", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func tick() {
						%s = 1
					}
					interval{period=10ms} -> tick{}
				`, outputCh.Name),
			}

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			time.Sleep(50 * time.Millisecond)
			Expect(t.Stop()).To(Succeed())
		})
	})

	Describe("Entry Node Startup", func() {
		It("Should fire a constant entry node at startup with no reads or intervals", func(ctx SpecContext) {
			outputCh := createVirtualCh(ctx, "startup_const", telem.Uint8T)
			prog := arc.Text{Raw: fmt.Sprintf(`
				42 -> %s
			`, outputCh.Name)}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{outputCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(outputCh.Key()).Series[0], 0)).
				To(Equal(uint8(42)))
			Consistently(responses, 100*time.Millisecond).ShouldNot(Receive())
		})

		It("Should fire entry nodes at startup while a channel-read path waits for input", func(ctx SpecContext) {
			triggerCh := createVirtualCh(ctx, "startup_trigger", telem.Uint8T)
			constOut := createVirtualCh(ctx, "startup_const_out", telem.Uint8T)
			exprOut := createVirtualCh(ctx, "startup_expr_out", telem.Uint8T)
			triggerOut := createVirtualCh(ctx, "startup_trigger_out", telem.Uint8T)

			prog := arc.Text{Raw: fmt.Sprintf(`
				func pass() {
					%s = %s
				}
				42 -> %s
				40 + 2 -> %s
				%s -> pass{}
			`, triggerOut.Name, triggerCh.Name, constOut.Name, exprOut.Name, triggerCh.Name)}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{
				constOut.Key(), exprOut.Key(), triggerOut.Key(),
			}, 10)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var gotConst, gotExpr bool
			for !gotConst || !gotExpr {
				var fr framer.StreamerResponse
				Eventually(responses).Should(Receive(&fr))
				if s := fr.Frame.Get(constOut.Key()); s.Len() > 0 {
					Expect(telem.ValueAt[uint8](s.Series[0], 0)).To(Equal(uint8(42)))
					gotConst = true
				}
				if s := fr.Frame.Get(exprOut.Key()); s.Len() > 0 {
					Expect(telem.ValueAt[uint8](s.Series[0], 0)).To(Equal(uint8(42)))
					gotExpr = true
				}
				Expect(fr.Frame.Get(triggerOut.Key()).Len()).To(BeZero())
			}

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{triggerCh.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(triggerCh.Key(), telem.NewSeriesV[uint8](7)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			var gotTrigger bool
			for !gotTrigger {
				var fr framer.StreamerResponse
				Eventually(responses).Should(Receive(&fr))
				if s := fr.Frame.Get(triggerOut.Key()); s.Len() > 0 {
					Expect(telem.ValueAt[uint8](s.Series[0], 0)).To(Equal(uint8(7)))
					gotTrigger = true
				}
			}
		})

		It("Should fire constant and expression entry nodes once when paired with an interval", func(ctx SpecContext) {
			constOut := createVirtualCh(ctx, "once_const_out", telem.Uint8T)
			exprOut := createVirtualCh(ctx, "once_expr_out", telem.Uint8T)
			tickOut := createVirtualCh(ctx, "once_tick", telem.Uint8T)

			prog := arc.Text{Raw: fmt.Sprintf(`
				func tick() {
					%s = 1
				}
				42 -> %s
				40 + 2 -> %s
				interval{period=20ms} -> tick{}
			`, tickOut.Name, constOut.Name, exprOut.Name)}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{
				constOut.Key(), exprOut.Key(), tickOut.Key(),
			}, 20)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var (
				fr         framer.StreamerResponse
				constCount int
				exprCount  int
				tickCount  int
			)
			for tickCount < 3 {
				Eventually(responses).Should(Receive(&fr))
				if s := fr.Frame.Get(constOut.Key()); s.Len() > 0 {
					Expect(telem.ValueAt[uint8](s.Series[0], 0)).To(Equal(uint8(42)))
					constCount++
				}
				if s := fr.Frame.Get(exprOut.Key()); s.Len() > 0 {
					Expect(telem.ValueAt[uint8](s.Series[0], 0)).To(Equal(uint8(42)))
					exprCount++
				}
				if fr.Frame.Get(tickOut.Key()).Len() > 0 {
					tickCount++
				}
			}
			Expect(constCount).To(Equal(1))
			Expect(exprCount).To(Equal(1))
		})
	})

	Describe("Control Authority", func() {
		It("Should apply static authority from authority block", func(ctx SpecContext) {
			ch := createVirtualCh(ctx, "auth_static", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 100
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(200)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})

		It("Should block lower-authority competing writers", func(ctx SpecContext) {
			ch := createVirtualCh(ctx, "auth_block", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 200
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
		})

		It("Should default to absolute authority without authority block", func(ctx SpecContext) {
			ch := createVirtualCh(ctx, "auth_default", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(254)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
		})

		It("Should apply per-channel authority overrides", func(ctx SpecContext) {
			ch1 := createVirtualCh(ctx, "auth_perchan_1", telem.Uint8T)
			ch2 := createVirtualCh(ctx, "auth_perchan_2", telem.Uint8T)
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

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{ch1.Key(), ch2.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			wA := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch1.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()
			Expect(wA.Write(frame.NewUnary(ch1.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())

			wB := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch2.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(wB.Close()).To(Succeed()) }()
			Expect(wB.Write(frame.NewUnary(ch2.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})

		It("Should write data with non-default authority", func(ctx SpecContext) {
			ch := createVirtualCh(ctx, "auth_write_data", telem.Uint8T)
			prog := arc.Text{
				Raw: fmt.Sprintf(`
					authority 100
					func output() {
						%s = 42
					}
					interval{period=50ms} -> output{}
				`, ch.Name),
			}

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{ch.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(ch.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(ch.Key()).Series[0], 0)).To(Equal(uint8(42)))
		})

		It("Should dynamically escalate authority via set_authority", func(ctx SpecContext) {
			dataCh := createVirtualCh(ctx, "dyn_esc_data", telem.Uint8T)
			triggerCh := createVirtualCh(ctx, "dyn_esc_trigger", telem.Uint8T)
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

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			trigW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{triggerCh.Key()},
				Start: telem.Now(),
			}))
			Expect(trigW.Write(frame.NewUnary(triggerCh.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(trigW.Close()).To(Succeed())

			// Receive data frames to ensure the runtime has processed the trigger
			Eventually(responses).Should(Receive(&fr))
			Eventually(responses).Should(Receive(&fr))

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{dataCh.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
			Expect(w.Write(frame.NewUnary(dataCh.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
		})

		It("Should dynamically de-escalate authority via set_authority", func(ctx SpecContext) {
			dataCh := createVirtualCh(ctx, "dyn_deesc_data", telem.Uint8T)
			triggerCh := createVirtualCh(ctx, "dyn_deesc_trigger", telem.Uint8T)
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

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			wBefore := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{dataCh.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			Expect(wBefore.Write(frame.NewUnary(dataCh.Key(), telem.NewSeriesV[uint8](99)))).To(BeFalse())
			Expect(wBefore.Close()).To(Succeed())

			trigW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{triggerCh.Key()},
				Start: telem.Now(),
			}))
			Expect(trigW.Write(frame.NewUnary(triggerCh.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(trigW.Close()).To(Succeed())

			// Receive data frames to ensure the runtime has processed the trigger
			Eventually(responses).Should(Receive(&fr))
			Eventually(responses).Should(Receive(&fr))

			wAfter := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{dataCh.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wAfter.Close()).To(Succeed()) }()
			Expect(wAfter.Write(frame.NewUnary(dataCh.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})

		It("Should continue writing data after dynamic authority change", func(ctx SpecContext) {
			dataCh := createVirtualCh(ctx, "dyn_cont_data", telem.Uint8T)
			triggerCh := createVirtualCh(ctx, "dyn_cont_trigger", telem.Uint8T)
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

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{dataCh.Key()}, 2)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			var fr framer.StreamerResponse
			Eventually(responses).Should(Receive(&fr))

			trigW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{triggerCh.Key()},
				Start: telem.Now(),
			}))
			Expect(trigW.Write(frame.NewUnary(triggerCh.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(trigW.Close()).To(Succeed())

			Eventually(responses).Should(Receive(&fr))
			Expect(fr.Frame.Get(dataCh.Key()).Len()).To(BeEquivalentTo(1))
			Expect(telem.ValueAt[uint8](fr.Frame.Get(dataCh.Key()).Series[0], 0)).To(Equal(uint8(42)))
		})

		It("Should release per-channel authority on both channels after bang-bang start → stop → yield", func(ctx SpecContext) {
			ch1 := createVirtualCh(ctx, "bb_ch1", telem.Uint8T)
			ch2 := createVirtualCh(ctx, "bb_ch2", telem.Uint8T)
			stopSignal := createVirtualCh(ctx, "bb_stop", telem.Uint8T)
			startSignal := createVirtualCh(ctx, "bb_start", telem.Uint8T)
			prog := bangBangProg(ch1, ch2, stopSignal, startSignal)

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{ch1.Key(), ch2.Key()}, 20)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			startW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{startSignal.Key()},
				Start: telem.Now(),
			}))
			Expect(startW.Write(frame.NewUnary(startSignal.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(startW.Close()).To(Succeed())

			var fr framer.StreamerResponse
			Eventually(responses, 500*time.Millisecond).Should(Receive(&fr))

			clearW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{startSignal.Key()},
				Start: telem.Now(),
			}))
			Expect(clearW.Write(frame.NewUnary(startSignal.Key(), telem.NewSeriesV[uint8](0)))).To(BeTrue())
			Expect(clearW.Close()).To(Succeed())
			time.Sleep(100 * time.Millisecond)

			stopW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{stopSignal.Key()},
				Start: telem.Now(),
			}))
			Expect(stopW.Write(frame.NewUnary(stopSignal.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(stopW.Close()).To(Succeed())

			time.Sleep(300 * time.Millisecond)

			w1 := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch1.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(1)},
				Sync:        new(true),
			}))
			defer func() { Expect(w1.Close()).To(Succeed()) }()
			Expect(w1.Write(frame.NewUnary(ch1.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())

			w2 := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch2.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(1)},
				Sync:        new(true),
			}))
			defer func() { Expect(w2.Close()).To(Succeed()) }()
			Expect(w2.Write(frame.NewUnary(ch2.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})

		It("Should release authority on both channels when entering yield, ignoring stale virtual start signal", func(ctx SpecContext) {
			ch1 := createVirtualCh(ctx, "bb2_ch1", telem.Uint8T)
			ch2 := createVirtualCh(ctx, "bb2_ch2", telem.Uint8T)
			stopSignal := createVirtualCh(ctx, "bb2_stop", telem.Uint8T)
			startSignal := createVirtualCh(ctx, "bb2_start", telem.Uint8T)
			prog := bangBangProg(ch1, ch2, stopSignal, startSignal)

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{ch1.Key(), ch2.Key()}, 20)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			startW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{startSignal.Key()},
				Start: telem.Now(),
			}))
			Expect(startW.Write(frame.NewUnary(startSignal.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(startW.Close()).To(Succeed())

			var fr framer.StreamerResponse
			Eventually(responses, 500*time.Millisecond).Should(Receive(&fr))

			stopW := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{stopSignal.Key()},
				Start: telem.Now(),
			}))
			Expect(stopW.Write(frame.NewUnary(stopSignal.Key(), telem.NewSeriesV[uint8](1)))).To(BeTrue())
			Expect(stopW.Close()).To(Succeed())

			time.Sleep(300 * time.Millisecond)

			w1 := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch1.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(w1.Close()).To(Succeed()) }()
			Expect(w1.Write(frame.NewUnary(ch1.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())

			w2 := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        channel.Keys{ch2.Key()},
				Start:       telem.Now(),
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(w2.Close()).To(Succeed()) }()
			Expect(w2.Write(frame.NewUnary(ch2.Key(), telem.NewSeriesV[uint8](99)))).To(BeTrue())
		})
	})

	Describe("Runtime Error Handling", func() {
		It("Should report WASM division by zero via status service", func(ctx SpecContext) {
			inputCh := createVirtualCh(ctx, "div_zero_input", telem.Int32T)
			outputCh := createVirtualCh(ctx, "div_zero_output", telem.Int32T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func divide_test() {
						%s = 10 / %s
					}
					%s -> divide_test{}
				`, outputCh.Name, inputCh.Name, inputCh.Name),
			}

			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 100),
				Name:   "test-div-zero",
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			t := MustSucceed(newTextFactory(ctx, prog).ConfigureTask(ctx, svcTask))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() {
				Expect(t.Stop()).To(Succeed())
			}()

			time.Sleep(20 * time.Millisecond)

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{inputCh.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[int32](0)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			Eventually(func(g Gomega) {
				var stat task.Status
				g.Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
					Where(status.MatchKeys[task.StatusDetails](task.OntologyID(svcTask.Key).String())).
					Entry(&stat).Exec(ctx, nil)).To(Succeed())
				g.Expect(stat.Variant).To(BeEquivalentTo("warning"))
				g.Expect(stat.Message).To(ContainSubstring("Runtime error in"))
				g.Expect(stat.Message).To(ContainSubstring("divide_test"))
				g.Expect(stat.Description).To(ContainSubstring("integer divide by zero"))
				g.Expect(stat.Details.Running).To(BeTrue())
			}).Should(Succeed())
		})

		It("Should read config param channel value correctly", func(ctx SpecContext) {
			inputCh := createVirtualCh(ctx, "cfg_read_input", telem.Uint8T)
			maxCh := createVirtualCh(ctx, "cfg_read_max", telem.Float32T)
			counterCh := createVirtualCh(ctx, "cfg_read_counter", telem.Float32T)

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

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{counterCh.Key()}, 10)
			defer closeStreamer()

			t := newTask(ctx, newTextFactory(ctx, prog))
			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)

			// Write max value of 5.0 to the max channel
			wMax := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{maxCh.Key()},
				Start: telem.Now(),
			}))
			Expect(wMax.Write(frame.NewUnary(maxCh.Key(), telem.NewSeriesV[float32](5.0)))).To(BeTrue())
			Expect(wMax.Close()).To(Succeed())

			time.Sleep(20 * time.Millisecond)

			// Write a rising edge (0 -> 1) to the input channel
			wInput := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
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

		It("Should continue execution after runtime error", func(ctx SpecContext) {
			inputCh := createVirtualCh(ctx, "recover_input", telem.Int32T)
			outputCh := createVirtualCh(ctx, "recover_output", telem.Int32T)

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func divide_recover() {
						%s = 10 / %s
					}
					%s -> divide_recover{}
				`, outputCh.Name, inputCh.Name, inputCh.Name),
			}

			svcTask := task.Task{
				Key:    task.NewKey(rack.NewKey(1, 1), 101),
				Name:   "test-div-recover",
				Type:   arctask.Type,
				Config: configToMap(arctask.Config{ArcKey: uuid.New()}),
			}
			t := MustSucceed(newTextFactory(ctx, prog).ConfigureTask(ctx, svcTask))

			responses, closeStreamer := openTestStreamer(ctx, channel.Keys{outputCh.Key()}, 5)
			defer closeStreamer()

			Expect(t.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
			defer func() { Expect(t.Stop()).To(Succeed()) }()

			time.Sleep(20 * time.Millisecond)

			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{inputCh.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(inputCh.Key(), telem.NewSeriesV[int32](0)))).To(BeTrue())

			Eventually(func(g Gomega) {
				var stat task.Status
				g.Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
					Where(status.MatchKeys[task.StatusDetails](task.OntologyID(svcTask.Key).String())).
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
