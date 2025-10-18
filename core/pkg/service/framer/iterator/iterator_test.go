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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

		FDescribe("Calculations", func() {
			var (
				indexCh *channel.Channel
				idxData telem.Series
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
				idxData = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
				fr := core.MultiFrame(
					keys,
					[]telem.Series{
						idxData,
						telem.NewSeriesV[float32](1, 2, 3, 4, 5),
						telem.NewSeriesV[float32](-2, -3, -4, -5, -6),
					},
				)
				MustSucceed(w.Write(fr))
				Expect(w.Close()).To(Succeed())
			})

			It("Should correctly calculate output values", func() {
				calculation := &channel.Channel{
					Name:       "Output",
					DataType:   telem.Float32T,
					Expression: "{return sensor_1 + sensor_2}",
				}
				Expect(dist.Channel.Create(ctx, calculation)).To(Succeed())
				iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
					Keys:   []channel.Key{calculation.Key()},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				for {
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					v := iter.Value().Get(calculation.Key())
					if v.Len() > 0 {
						Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](-1, -1, -1, -1, -1))
						idx := iter.Value().Get(calculation.Index())
						Expect(idx.Series).To(HaveLen(1))
						Expect(idx.Series[0]).To(telem.MatchSeriesData(idxData))
						break
					}
				}
				Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
				Expect(iter.Close()).To(Succeed())
			})
		})
	})
})
