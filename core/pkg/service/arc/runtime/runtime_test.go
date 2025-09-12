package runtime_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Runtime", Ordered, func() {
	var (
		dist      mock.Node
		statusSvc *status.Service
	)

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		statusSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Group:    dist.Group,
			Signals:  dist.Signals,
			Ontology: dist.Ontology,
		}))
	})

	AfterAll(func() {
		Expect(dist.Close()).To(Succeed())
		Expect(statusSvc.Close()).To(Succeed())
	})

	It("Should run a basic value printer", func() {
		ch := &channel.Channel{
			Name:     "ox_pt_1",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

		cfg := runtime.Config{
			Channel: dist.Channel,
			Framer:  dist.Framer,
			Status:  statusSvc,
		}

		resolver := MustSucceed(runtime.CreateResolver(cfg))

		graph := arc.Graph{
			Nodes: []graph.Node{
				{Node: arc.Node{
					Key:    "first",
					Type:   "on",
					Config: map[string]any{"channel": ch.Key()},
				}},
				{Node: arc.Node{Key: "printer", Type: "printer"}},
			},
			Edges: []arc.Edge{
				{
					Source: arc.Handle{Node: "first"},
					Target: arc.Handle{Node: "printer"},
				},
			},
		}
		cfg.Module = MustSucceed(arc.CompileGraph(ctx, graph, arc.WithResolver(resolver)))
		Expect(cfg.Module.Nodes).To(HaveLen(2))
		Expect(cfg.Module.Edges).To(HaveLen(1))

		r := MustSucceed(runtime.Open(ctx, cfg))
		Expect(r.Close()).To(Succeed())
	})

	Describe("Alarm", func() {
		FIt("Should update alarm statuses", func() {
			ch := &channel.Channel{
				Name:     "ox_pt_1",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

			cfg := runtime.Config{
				Channel: dist.Channel,
				Framer:  dist.Framer,
				Status:  statusSvc,
			}

			resolver := MustSucceed(runtime.CreateResolver(cfg))

			graph := arc.Graph{
				Nodes: []graph.Node{
					{Node: arc.Node{
						Key:    "on",
						Type:   "on",
						Config: map[string]any{"channel": ch.Key()},
					}},
					{Node: arc.Node{
						Key:    "constant",
						Type:   "constant",
						Config: map[string]any{"value": 10},
					}},
					{Node: arc.Node{
						Key:    "ge",
						Type:   "ge",
						Config: map[string]any{},
					}},
					{Node: arc.Node{
						Key:  "stable_for",
						Type: "stable_for",
						Config: map[string]any{
							"duration": int(telem.Millisecond * 1),
						},
					}},
					{Node: arc.Node{
						Key:  "select",
						Type: "select",
					}},
					{Node: arc.Node{
						Key:  "status_success",
						Type: "set_status",
						Config: map[string]any{
							"key":     "ox_alarm",
							"variant": "success",
							"message": "OX Pressure Nominal",
						},
					}},
					{Node: arc.Node{
						Key:  "status_error",
						Type: "set_status",
						Config: map[string]any{
							"key":     "ox_alarm",
							"variant": "error",
							"message": "OX Pressure Alarm",
						},
					}},
				},
				Edges: []arc.Edge{
					{
						Source: arc.Handle{Node: "on"},
						Target: arc.Handle{Node: "ge", Param: "a"},
					},
					{
						Source: arc.Handle{Node: "constant"},
						Target: arc.Handle{Node: "ge", Param: "b"},
					},
					{
						Source: arc.Handle{Node: "ge"},
						Target: arc.Handle{Node: "stable_for"},
					},
					{
						Source: arc.Handle{Node: "stable_for"},
						Target: arc.Handle{Node: "select"},
					},
					{
						Source: arc.Handle{Node: "select", Param: "false"},
						Target: arc.Handle{Node: "status_success"},
					},
					{
						Source: arc.Handle{Node: "select", Param: "true"},
						Target: arc.Handle{Node: "status_error"},
					},
				},
			}
			cfg.Module = MustSucceed(arc.CompileGraph(ctx, graph, arc.WithResolver(resolver)))
			Expect(cfg.Module.Nodes).To(HaveLen(7))
			Expect(cfg.Module.Edges).To(HaveLen(6))

			r := MustSucceed(runtime.Open(ctx, cfg))

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{ch.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(core.UnaryFrame(
				ch.Key(),
				telem.NewSeriesV[float32](20),
			))).To(BeTrue())
			time.Sleep(time.Millisecond * 2)
			Expect(w.Write(core.UnaryFrame(
				ch.Key(),
				telem.NewSeriesV[float32](25),
			))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			Expect(r.Close()).To(Succeed())
		})
	})
})
