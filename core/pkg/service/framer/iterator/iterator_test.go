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
	"github.com/synnaxlabs/synnax/pkg/service/arc"
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
		arcSvc      *arc.Service
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
		arcSvc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
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
				Start: telem.SecondTS,
				Keys:  []channel.Key{ch.Key()},
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
			Expect(iter.Value().Frame).To(telem.MatchWrittenFrame(fr.Frame))
			Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		Describe("Calculations", func() {
			var (
				indexCh *channel.Channel
				idxData telem.MultiSeries
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
				idxData = telem.MultiSeries{Series: []telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5),
					telem.NewSeriesSecondsTSV(6, 7, 8, 9, 10),
				}}
				fr := core.MultiFrame(
					keys,
					[]telem.Series{
						idxData.Series[0],
						telem.NewSeriesV[float32](1, 2, 3, 4, 5),
						telem.NewSeriesV[float32](-2, -3, -4, -5, -6),
					},
				)
				MustSucceed(w.Write(fr))
				Expect(w.Close()).To(Succeed())
				w = MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start:            telem.SecondTS * 6,
					Keys:             keys,
					EnableAutoCommit: config.True(),
				}))
				fr = core.MultiFrame(
					keys,
					[]telem.Series{
						idxData.Series[1],
						telem.NewSeriesV[float32](6, 7, 8, 9, 10),
						telem.NewSeriesV[float32](-3, -4, -5, -6, -7),
					},
				)
				MustSucceed(w.Write(fr))
				Expect(w.Close()).To(Succeed())
			})

			It("Should correctly calculate output values", func() {
				calculation := &channel.Channel{
					Name:       "output",
					DataType:   telem.Float32T,
					Expression: "return sensor_1",
				}
				Expect(dist.Channel.Create(ctx, calculation)).To(Succeed())
				iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
					Keys:   []channel.Key{calculation.Key(), calculation.Index()},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
				v := iter.Value().Get(calculation.Key())
				Expect(v.Series).To(HaveLen(2))
				Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](1, 2, 3, 4, 5))
				Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
				Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](6, 7, 8, 9, 10))
				Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
				v = iter.Value().Get(calculation.Index())
				Expect(v.Series).To(HaveLen(2))
				Expect(v.Series[0]).To(telem.MatchSeriesData(idxData.Series[0]))
				Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
				Expect(v.Series[1]).To(telem.MatchSeriesData(idxData.Series[1]))
				Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
				Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
				Expect(iter.Close()).To(Succeed())
			})

			Describe("Legacy Calculation", func() {
				It("Should correctly calculate output values", func() {
					legacyCalculation := &channel.Channel{
						Name:       "legacy_calculation",
						DataType:   telem.Float32T,
						Expression: "return sensor_1",
						Requires:   []channel.Key{dataCh1.Key()},
					}
					Expect(dist.Channel.Create(ctx, legacyCalculation)).To(Succeed())
					iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
						Keys:   []channel.Key{legacyCalculation.Key()},
						Bounds: telem.TimeRangeMax,
					}))
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					v := iter.Value().Get(legacyCalculation.Key())
					Expect(v.Series).To(HaveLen(1))
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](6, 7, 8, 9, 10))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(iter.Close()).To(Succeed())
				})
			})

			Describe("Nested Calculations", func() {
				It("Should correctly handle 2-level nesting (C → B → A)", func() {
					// Create B: calculated channel that depends on concrete channel A (sensor_1)
					calcB := &channel.Channel{
						Name:       "calc_b",
						DataType:   telem.Float32T,
						Expression: "return sensor_1 * 2",
					}
					Expect(dist.Channel.Create(ctx, calcB)).To(Succeed())

					// Create C: calculated channel that depends on calculated channel B
					calcC := &channel.Channel{
						Name:       "calc_c",
						DataType:   telem.Float32T,
						Expression: "return calc_b + 10",
					}
					Expect(dist.Channel.Create(ctx, calcC)).To(Succeed())

					// Open iterator requesting only the top-level calculated channel C
					iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
						Keys:   []channel.Key{calcC.Key(), calcC.Index()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify the calculated result
					// sensor_1 has values [1, 2, 3, 4, 5] and [6, 7, 8, 9, 10]
					// calc_b = sensor_1 * 2 = [2, 4, 6, 8, 10] and [12, 14, 16, 18, 20]
					// calc_c = calc_b + 10 = [12, 14, 16, 18, 20] and [22, 24, 26, 28, 30]
					v := iter.Value().Get(calcC.Key())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](12, 14, 16, 18, 20))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](22, 24, 26, 28, 30))
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))

					// Verify the index is correct
					v = iter.Value().Get(calcC.Index())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesData(idxData.Series[0]))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(v.Series[1]).To(telem.MatchSeriesData(idxData.Series[1]))
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should correctly handle 3-level nesting (D → C → B → A)", func() {
					// Create B: depends on sensor_1 (concrete)
					calcB := &channel.Channel{
						Name:       "calc_b_3level",
						DataType:   telem.Float32T,
						Expression: "return sensor_1 * 2",
					}
					Expect(dist.Channel.Create(ctx, calcB)).To(Succeed())

					// Create C: depends on B (calculated)
					calcC := &channel.Channel{
						Name:       "calc_c_3level",
						DataType:   telem.Float32T,
						Expression: "return calc_b_3level + 5",
					}
					Expect(dist.Channel.Create(ctx, calcC)).To(Succeed())

					// Create D: depends on C (calculated)
					calcD := &channel.Channel{
						Name:       "calc_d_3level",
						DataType:   telem.Float32T,
						Expression: "return calc_c_3level * 3",
					}
					Expect(dist.Channel.Create(ctx, calcD)).To(Succeed())

					// Open iterator requesting only the top-level calculated channel D
					iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
						Keys:   []channel.Key{calcD.Key(), calcD.Index()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify the calculated result
					// sensor_1 has values [1, 2, 3, 4, 5] and [6, 7, 8, 9, 10]
					// calc_b = sensor_1 * 2 = [2, 4, 6, 8, 10] and [12, 14, 16, 18, 20]
					// calc_c = calc_b + 5 = [7, 9, 11, 13, 15] and [17, 19, 21, 23, 25]
					// calc_d = calc_c * 3 = [21, 27, 33, 39, 45] and [51, 57, 63, 69, 75]
					v := iter.Value().Get(calcD.Key())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](21, 27, 33, 39, 45))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](51, 57, 63, 69, 75))
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))

					// Verify the index is correct
					v = iter.Value().Get(calcD.Index())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesData(idxData.Series[0]))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(v.Series[1]).To(telem.MatchSeriesData(idxData.Series[1]))
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should correctly handle multiple branches (diamond dependency)", func() {
					// Create a diamond pattern:
					// E depends on C and D
					// C depends on sensor_1 (A)
					// D depends on sensor_1 (A)
					// Tests that shared dependencies work correctly

					// Create C: depends on sensor_1 (concrete)
					calcC := &channel.Channel{
						Name:       "calc_c_diamond",
						DataType:   telem.Float32T,
						Expression: "return sensor_1 + 10",
					}
					Expect(dist.Channel.Create(ctx, calcC)).To(Succeed())

					// Create D: also depends on sensor_1 (concrete)
					calcD := &channel.Channel{
						Name:       "calc_d_diamond",
						DataType:   telem.Float32T,
						Expression: "return sensor_1 * 5",
					}
					Expect(dist.Channel.Create(ctx, calcD)).To(Succeed())

					// Create E: depends on both C and D (calculated)
					calcE := &channel.Channel{
						Name:       "calc_e_diamond",
						DataType:   telem.Float32T,
						Expression: "return calc_c_diamond + calc_d_diamond",
					}
					Expect(dist.Channel.Create(ctx, calcE)).To(Succeed())

					// Open iterator requesting only the top-level calculated channel E
					iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
						Keys:   []channel.Key{calcE.Key(), calcE.Index()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify the calculated result
					// sensor_1 has values [1, 2, 3, 4, 5] and [6, 7, 8, 9, 10]
					// calc_c = sensor_1 + 10 = [11, 12, 13, 14, 15] and [16, 17, 18, 19, 20]
					// calc_d = sensor_1 * 5 = [5, 10, 15, 20, 25] and [30, 35, 40, 45, 50]
					// calc_e = calc_c + calc_d = [16, 22, 28, 34, 40] and [46, 52, 58, 64, 70]
					v := iter.Value().Get(calcE.Key())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](16, 22, 28, 34, 40))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](46, 52, 58, 64, 70))
					// Note: Diamond pattern causes alignment increment, this is expected behavior
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					// Verify the index is correct
					v = iter.Value().Get(calcE.Index())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesData(idxData.Series[0]))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(v.Series[1]).To(telem.MatchSeriesData(idxData.Series[1]))
					// Note: Diamond pattern causes alignment increment, this is expected behavior
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should detect circular dependencies", func() {
					// This test verifies that circular dependencies are caught by the topological sort
					// Create a simple 2-node cycle: A → B → A

					// Create calc_circ_b that depends on calc_circ_a (doesn't exist yet, but Arc allows it)
					calcB := &channel.Channel{
						Name:       "calc_circ_b",
						DataType:   telem.Float32T,
						Expression: "return calc_circ_a + 1",
					}
					Expect(dist.Channel.Create(ctx, calcB)).To(Succeed())

					// Create calc_circ_a that depends on calc_circ_b (creating the cycle)
					calcA := &channel.Channel{
						Name:       "calc_circ_a",
						DataType:   telem.Float32T,
						Expression: "return calc_circ_b + 1",
					}
					Expect(dist.Channel.Create(ctx, calcA)).To(Succeed())

					// Now try to open an iterator - this should fail with circular dependency error
					_, err := iteratorSvc.Open(ctx, framer.IteratorConfig{
						Keys:   []channel.Key{calcA.Key()},
						Bounds: telem.TimeRangeMax,
					})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("circular dependency"))
				})

				It("Should handle mixed calculated and concrete channels", func() {
					// This test verifies that requesting both calculated and concrete channels
					// in the same iterator works correctly

					// Create a nested calculated channel
					calcMixed := &channel.Channel{
						Name:       "calc_mixed",
						DataType:   telem.Float32T,
						Expression: "return sensor_1 + sensor_2",
					}
					Expect(dist.Channel.Create(ctx, calcMixed)).To(Succeed())

					// Create another calculated that depends on the first
					calcMixedNested := &channel.Channel{
						Name:       "calc_mixed_nested",
						DataType:   telem.Float32T,
						Expression: "return calc_mixed * 2",
					}
					Expect(dist.Channel.Create(ctx, calcMixedNested)).To(Succeed())

					// Request both concrete channels (sensor_1, sensor_2) and calculated channels
					iter := MustSucceed(iteratorSvc.Open(ctx, framer.IteratorConfig{
						Keys: []channel.Key{
							dataCh1.Key(),           // concrete: sensor_1
							dataCh2.Key(),           // concrete: sensor_2
							calcMixedNested.Key(),   // calculated (nested)
							calcMixedNested.Index(), // index
						},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify concrete channel sensor_1 has original values
					v := iter.Value().Get(dataCh1.Key())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](1, 2, 3, 4, 5))
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](6, 7, 8, 9, 10))

					// Verify concrete channel sensor_2 has original values
					v = iter.Value().Get(dataCh2.Key())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](-2, -3, -4, -5, -6))
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](-3, -4, -5, -6, -7))

					// Verify calculated channel has correct values
					// sensor_1 = [1, 2, 3, 4, 5] and [6, 7, 8, 9, 10]
					// sensor_2 = [-2, -3, -4, -5, -6] and [-3, -4, -5, -6, -7]
					// calc_mixed = sensor_1 + sensor_2 = [-1, -1, -1, -1, -1] and [3, 3, 3, 3, 3]
					// calc_mixed_nested = calc_mixed * 2 = [-2, -2, -2, -2, -2] and [6, 6, 6, 6, 6]
					v = iter.Value().Get(calcMixedNested.Key())
					Expect(v.Series).To(HaveLen(2))
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](-2, -2, -2, -2, -2))
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](6, 6, 6, 6, 6))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
		})
	})
})
