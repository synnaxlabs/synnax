// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
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
	svcarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("StreamIterator", Ordered, func() {
	var (
		builder     = mock.NewCluster()
		dist        mock.Node
		iteratorSvc *iterator.Service
		arcSvc      *svcarc.Service
	)
	BeforeAll(func() {
		dist = builder.Provision(ctx)
		labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
			Label:    labelSvc,
		}))
		arcSvc = MustSucceed(svcarc.OpenService(ctx, svcarc.ServiceConfig{
			DB:       dist.DB,
			Channel:  dist.Channel,
			Framer:   dist.Framer,
			Status:   statusSvc,
			Ontology: dist.Ontology,
		}))
		iteratorSvc = MustSucceed(iterator.NewService(iterator.ServiceConfig{
			DistFramer: dist.Framer,
			Channel:    dist.Channel,
			Arc:        arcSvc,
		}))
	})

	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
	})
	Describe("Basic Iteration", func() {
		It("Should read written frames correctly", func() {
			ch := &channel.Channel{
				Name:     "Matt",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:            telem.SecondTS,
				Keys:             []channel.Key{ch.Key()},
				EnableAutoCommit: config.True(),
			}))
			fr := core.UnaryFrame(ch.Key(), telem.NewSeriesSecondsTSV(1, 2, 3))
			MustSucceed(w.Write(fr))
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
				Keys:   []channel.Key{ch.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
			Expect(iter.Value().Frame).To(telem.MatchWrittenFrame[channel.Key](fr.Frame))
			Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		Describe("Calculations", func() {
			var (
				indexCh *channel.Channel
				dataCh1 *channel.Channel
				dataCh2 *channel.Channel
			)
			BeforeAll(func() {
				indexCh = &channel.Channel{
					Name:     "time",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				}
				Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
				dataCh1 = &channel.Channel{
					Name:       "sensor_1",
					DataType:   telem.Float32T,
					LocalIndex: indexCh.LocalKey,
				}
				Expect(dist.Channel.Create(ctx, dataCh1)).To(Succeed())
				dataCh2 = &channel.Channel{
					Name:       "sensor_2",
					DataType:   telem.Float32T,
					LocalIndex: indexCh.LocalKey,
				}
				Expect(dist.Channel.Create(ctx, dataCh2)).To(Succeed())
				keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
				w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start:            telem.SecondTS,
					Keys:             keys,
					EnableAutoCommit: config.True(),
				}))
				fr := core.MultiFrame(
					keys,
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5),
						telem.NewSeriesV[float32](1, 2, 3, 4, 5),
						telem.NewSeriesV[float32](-1, -2, -3, -4, -5),
					},
				)
				MustSucceed(w.Write(fr))
				Expect(w.Close()).To(Succeed())
			})

			It("Should correctly calculate output values", func() {
				key := uuid.New()
				prog := arc.Graph{
					Stages: []arc.Stage{
						{
							Key: "calculation",
							Params: ir.NamedTypes{
								Keys:   []string{"sensor_1", "sensor_2"},
								Values: []ir.Type{ir.F32{}, ir.F32{}},
							},
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{
							Key:  "sensor_1_on",
							Type: "on",
							Config: map[string]any{
								"channel": dataCh1.Key(),
							},
						}},
						{Node: ir.Node{
							Key:  "sensor_2_on",
							Type: "on",
							Config: map[string]any{
								"channel": dataCh2.Key(),
							},
						}},
						{Node: ir.Node{
							Key:  "calculation",
							Type: "calculation",
						}},
					},
				}
				calculation := &channel.Channel{
					Name:        "Output",
					DataType:    telem.Float32T,
					Calculation: key,
				}
				Expect(dist.Channel.Create(ctx, calculation)).To(Succeed())
				prog.Nodes = append(prog.Nodes, graph.Node{Node: ir.Node{
					Key:    "writer",
					Config: map[string]any{"channel": calculation.Key()},
				}})
				Expect(arcSvc.NewWriter(nil).Create(ctx, &svcarc.Arc{
					Key:   key,
					Graph: prog,
				})).To(Succeed())

				iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
					Keys:   []channel.Key{calculation.Key()},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
				Expect(iter.Value().Get(calculation.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](0, 0, 0, 0, 0))
				Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
				Expect(iter.Close()).To(Succeed())
			})
		})
	})
})
