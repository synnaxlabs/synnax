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
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Runtime", Ordered, func() {
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

	Describe("Alarm", func() {
		It("Should update alarm statuses", func() {
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
							"status_key": "ox_alarm",
							"variant":    "success",
							"name":       "OX Alarm",
							"message":    "OX Pressure Nominal",
						},
					}},
					{Node: arc.Node{
						Key:  "status_error",
						Type: "set_status",
						Config: map[string]any{
							"status_key": "ox_alarm",
							"variant":    "error",
							"name":       "OX Alarm",
							"message":    "OX Pressure Exceed",
						},
					}},
				},
				Edges: []arc.Edge{
					{
						Source: arc.Handle{Node: "on", Param: "output"},
						Target: arc.Handle{Node: "ge", Param: "a"},
					},
					{
						Source: arc.Handle{Node: "constant", Param: "output"},
						Target: arc.Handle{Node: "ge", Param: "b"},
					},
					{
						Source: arc.Handle{Node: "ge", Param: "output"},
						Target: arc.Handle{Node: "stable_for", Param: "input"},
					},
					{
						Source: arc.Handle{Node: "stable_for", Param: "output"},
						Target: arc.Handle{Node: "select", Param: "input"},
					},
					{
						Source: arc.Handle{Node: "select", Param: "false"},
						Target: arc.Handle{Node: "status_success", Param: "input"},
					},
					{
						Source: arc.Handle{Node: "select", Param: "true"},
						Target: arc.Handle{Node: "status_error", Param: "input"},
					},
				},
			}
			cfg.Module = MustSucceed(arc.CompileGraph(ctx, graph, arc.WithResolver(resolver)))
			Expect(cfg.Module.Nodes).To(HaveLen(7))
			Expect(cfg.Module.Edges).To(HaveLen(6))

			r := MustSucceed(runtime.Open(ctx, cfg))
			time.Sleep(time.Millisecond * 20)

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{ch.Key()},
				Start: telem.Now(),
			}))
			Expect(w.Write(core.UnaryFrame(
				ch.Key(),
				telem.NewSeriesV[float32](20),
			))).To(BeTrue())
			time.Sleep(time.Millisecond * 20)
			Expect(w.Write(core.UnaryFrame(
				ch.Key(),
				telem.NewSeriesV[float32](25),
			))).To(BeTrue())

			Eventually(func(g Gomega) {
				var stat status.Status
				g.Expect(statusSvc.NewRetrieve().WhereKeys("ox_alarm").Entry(&stat).Exec(ctx, nil)).To(Succeed())
				g.Expect(stat.Variant).To(BeEquivalentTo("error"))
			}).To(Succeed())

			Expect(r.Close()).To(Succeed())
		})
	})
})
