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
	"strconv"
	"strings"

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

			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
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
				iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
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
					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
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
					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
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
					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
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
					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
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
					_, err := iteratorSvc.Open(ctx, iterator.Config{
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
					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
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

			Describe("Three Domain Calculations", func() {
				var (
					threeDomainIndexCh *channel.Channel
					threeDomainDataCh  *channel.Channel
					threeDomainIdxData telem.MultiSeries
				)
				BeforeAll(func() {
					threeDomainIndexCh = &channel.Channel{
						Name:     "three_domain_time",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					}
					Expect(dist.Channel.Create(ctx, threeDomainIndexCh)).To(Succeed())
					threeDomainDataCh = &channel.Channel{
						Name:       "three_domain_sensor",
						DataType:   telem.Float32T,
						LocalIndex: threeDomainIndexCh.LocalKey,
					}
					Expect(dist.Channel.Create(ctx, threeDomainDataCh)).To(Succeed())
					keys := []channel.Key{threeDomainIndexCh.Key(), threeDomainDataCh.Key()}

					threeDomainIdxData = telem.MultiSeries{Series: []telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesSecondsTSV(5, 6),
						telem.NewSeriesSecondsTSV(10, 11),
					}}

					// First domain
					w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
						Start:            telem.SecondTS,
						Keys:             keys,
						EnableAutoCommit: config.True(),
					}))
					fr := core.MultiFrame(
						keys,
						[]telem.Series{
							threeDomainIdxData.Series[0],
							telem.NewSeriesV[float32](1, 2),
						},
					)
					MustSucceed(w.Write(fr))
					Expect(w.Close()).To(Succeed())

					// Second domain
					w = MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
						Start:            telem.SecondTS * 5,
						Keys:             keys,
						EnableAutoCommit: config.True(),
					}))
					fr = core.MultiFrame(
						keys,
						[]telem.Series{
							threeDomainIdxData.Series[1],
							telem.NewSeriesV[float32](5, 6),
						},
					)
					MustSucceed(w.Write(fr))
					Expect(w.Close()).To(Succeed())

					// Third domain
					w = MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
						Start:            telem.SecondTS * 10,
						Keys:             keys,
						EnableAutoCommit: config.True(),
					}))
					fr = core.MultiFrame(
						keys,
						[]telem.Series{
							threeDomainIdxData.Series[2],
							telem.NewSeriesV[float32](10, 11),
						},
					)
					MustSucceed(w.Write(fr))
					Expect(w.Close()).To(Succeed())
				})

				It("Should correctly calculate values across three domains with proper alignment", func() {
					calc := &channel.Channel{
						Name:       "three_domain_calc",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor * 2",
					}
					Expect(dist.Channel.Create(ctx, calc)).To(Succeed())

					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
						Keys:   []channel.Key{calc.Key(), calc.Index()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify calculated values and alignments for all three domains
					v := iter.Value().Get(calc.Key())
					Expect(v.Series).To(HaveLen(3))

					// Domain 0: sensor = [1, 2], calc = [2, 4]
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](2, 4))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))

					// Domain 1: sensor = [5, 6], calc = [10, 12]
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](10, 12))
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))

					// Domain 2: sensor = [10, 11], calc = [20, 22]
					Expect(v.Series[2]).To(telem.MatchSeriesDataV[float32](20, 22))
					Expect(v.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					// Verify index alignments match
					idxV := iter.Value().Get(calc.Index())
					Expect(idxV.Series).To(HaveLen(3))
					Expect(idxV.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(idxV.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(idxV.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should correctly handle nested calculations across three domains", func() {
					// B depends on three_domain_sensor (concrete)
					calcB := &channel.Channel{
						Name:       "three_domain_calc_b",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor * 2",
					}
					Expect(dist.Channel.Create(ctx, calcB)).To(Succeed())

					// C depends on B (calculated)
					calcC := &channel.Channel{
						Name:       "three_domain_calc_c",
						DataType:   telem.Float32T,
						Expression: "return three_domain_calc_b + 10",
					}
					Expect(dist.Channel.Create(ctx, calcC)).To(Succeed())

					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
						Keys:   []channel.Key{calcC.Key(), calcC.Index()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify calculated values and alignments
					// sensor = [1,2], [5,6], [10,11]
					// calc_b = sensor * 2 = [2,4], [10,12], [20,22]
					// calc_c = calc_b + 10 = [12,14], [20,22], [30,32]
					v := iter.Value().Get(calcC.Key())
					Expect(v.Series).To(HaveLen(3))

					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](12, 14))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))

					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](20, 22))
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))

					Expect(v.Series[2]).To(telem.MatchSeriesDataV[float32](30, 32))
					Expect(v.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					// Verify index alignments match
					idxV := iter.Value().Get(calcC.Index())
					Expect(idxV.Series).To(HaveLen(3))
					Expect(idxV.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(idxV.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(idxV.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should correctly handle diamond dependency across three domains", func() {
					// C depends on three_domain_sensor
					calcC := &channel.Channel{
						Name:       "three_domain_diamond_c",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor + 10",
					}
					Expect(dist.Channel.Create(ctx, calcC)).To(Succeed())

					// D also depends on three_domain_sensor
					calcD := &channel.Channel{
						Name:       "three_domain_diamond_d",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor * 5",
					}
					Expect(dist.Channel.Create(ctx, calcD)).To(Succeed())

					// E depends on both C and D
					calcE := &channel.Channel{
						Name:       "three_domain_diamond_e",
						DataType:   telem.Float32T,
						Expression: "return three_domain_diamond_c + three_domain_diamond_d",
					}
					Expect(dist.Channel.Create(ctx, calcE)).To(Succeed())

					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
						Keys:   []channel.Key{calcE.Key(), calcE.Index()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify calculated values
					// sensor = [1,2], [5,6], [10,11]
					// calc_c = sensor + 10 = [11,12], [15,16], [20,21]
					// calc_d = sensor * 5 = [5,10], [25,30], [50,55]
					// calc_e = calc_c + calc_d = [16,22], [40,46], [70,76]
					v := iter.Value().Get(calcE.Key())
					Expect(v.Series).To(HaveLen(3))

					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](16, 22))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))

					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](40, 46))
					// Note: Diamond pattern may cause alignment increment
					Expect(v.Series[1].Alignment.SampleIndex()).To(Equal(uint32(0)))

					Expect(v.Series[2]).To(telem.MatchSeriesDataV[float32](70, 76))
					Expect(v.Series[2].Alignment.SampleIndex()).To(Equal(uint32(0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should correctly handle mixed calculated and concrete channels across three domains", func() {
					calcMixed := &channel.Channel{
						Name:       "three_domain_mixed_calc",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor * 3",
					}
					Expect(dist.Channel.Create(ctx, calcMixed)).To(Succeed())

					// Request both concrete and calculated channels
					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
						Keys: []channel.Key{
							threeDomainDataCh.Key(),
							calcMixed.Key(),
							calcMixed.Index(),
						},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify concrete channel has original values with correct alignments
					concreteV := iter.Value().Get(threeDomainDataCh.Key())
					Expect(concreteV.Series).To(HaveLen(3))
					Expect(concreteV.Series[0]).To(telem.MatchSeriesDataV[float32](1, 2))
					Expect(concreteV.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(concreteV.Series[1]).To(telem.MatchSeriesDataV[float32](5, 6))
					Expect(concreteV.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(concreteV.Series[2]).To(telem.MatchSeriesDataV[float32](10, 11))
					Expect(concreteV.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					// Verify calculated channel
					calcV := iter.Value().Get(calcMixed.Key())
					Expect(calcV.Series).To(HaveLen(3))
					Expect(calcV.Series[0]).To(telem.MatchSeriesDataV[float32](3, 6))
					Expect(calcV.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(calcV.Series[1]).To(telem.MatchSeriesDataV[float32](15, 18))
					Expect(calcV.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(calcV.Series[2]).To(telem.MatchSeriesDataV[float32](30, 33))
					Expect(calcV.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should correctly handle large gap between domains", func() {
					// Create channels specifically for this test with large time gap
					gapIndexCh := &channel.Channel{
						Name:     "gap_domain_time",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					}
					Expect(dist.Channel.Create(ctx, gapIndexCh)).To(Succeed())
					gapDataCh := &channel.Channel{
						Name:       "gap_domain_sensor",
						DataType:   telem.Float32T,
						LocalIndex: gapIndexCh.LocalKey,
					}
					Expect(dist.Channel.Create(ctx, gapDataCh)).To(Succeed())
					keys := []channel.Key{gapIndexCh.Key(), gapDataCh.Key()}

					// First domain at t=1s
					w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
						Start:            telem.SecondTS,
						Keys:             keys,
						EnableAutoCommit: config.True(),
					}))
					MustSucceed(w.Write(core.MultiFrame(
						keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(1, 2, 3),
							telem.NewSeriesV[float32](1, 2, 3),
						},
					)))
					Expect(w.Close()).To(Succeed())

					// Second domain at t=1000s (large gap)
					w = MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
						Start:            telem.SecondTS * 1000,
						Keys:             keys,
						EnableAutoCommit: config.True(),
					}))
					MustSucceed(w.Write(core.MultiFrame(
						keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(1000, 1001, 1002),
							telem.NewSeriesV[float32](1000, 1001, 1002),
						},
					)))
					Expect(w.Close()).To(Succeed())

					calc := &channel.Channel{
						Name:       "gap_domain_calc",
						DataType:   telem.Float32T,
						Expression: "return gap_domain_sensor + 100",
					}
					Expect(dist.Channel.Create(ctx, calc)).To(Succeed())

					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
						Keys:   []channel.Key{calc.Key(), calc.Index()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					v := iter.Value().Get(calc.Key())
					Expect(v.Series).To(HaveLen(2))

					// Domain 0
					Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](101, 102, 103))
					Expect(v.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))

					// Domain 1
					Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](1100, 1101, 1102))
					Expect(v.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

				It("Should correctly handle multiple calculations on same source across three domains", func() {
					calcDouble := &channel.Channel{
						Name:       "three_domain_double",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor * 2",
					}
					Expect(dist.Channel.Create(ctx, calcDouble)).To(Succeed())

					calcSquare := &channel.Channel{
						Name:       "three_domain_square",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor * three_domain_sensor",
					}
					Expect(dist.Channel.Create(ctx, calcSquare)).To(Succeed())

					calcPlusTen := &channel.Channel{
						Name:       "three_domain_plus_ten",
						DataType:   telem.Float32T,
						Expression: "return three_domain_sensor + 10",
					}
					Expect(dist.Channel.Create(ctx, calcPlusTen)).To(Succeed())

					iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
						Keys:   []channel.Key{calcDouble.Key(), calcSquare.Key(), calcPlusTen.Key()},
						Bounds: telem.TimeRangeMax,
					}))

					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

					// Verify calcDouble: sensor * 2
					doubleV := iter.Value().Get(calcDouble.Key())
					Expect(doubleV.Series).To(HaveLen(3))
					Expect(doubleV.Series[0]).To(telem.MatchSeriesDataV[float32](2, 4))
					Expect(doubleV.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(doubleV.Series[1]).To(telem.MatchSeriesDataV[float32](10, 12))
					Expect(doubleV.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(doubleV.Series[2]).To(telem.MatchSeriesDataV[float32](20, 22))
					Expect(doubleV.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					// Verify calcSquare: sensor * sensor
					squareV := iter.Value().Get(calcSquare.Key())
					Expect(squareV.Series).To(HaveLen(3))
					Expect(squareV.Series[0]).To(telem.MatchSeriesDataV[float32](1, 4))
					Expect(squareV.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(squareV.Series[1]).To(telem.MatchSeriesDataV[float32](25, 36))
					Expect(squareV.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(squareV.Series[2]).To(telem.MatchSeriesDataV[float32](100, 121))
					Expect(squareV.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					// Verify calcPlusTen: sensor + 10
					plusTenV := iter.Value().Get(calcPlusTen.Key())
					Expect(plusTenV.Series).To(HaveLen(3))
					Expect(plusTenV.Series[0]).To(telem.MatchSeriesDataV[float32](11, 12))
					Expect(plusTenV.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
					Expect(plusTenV.Series[1]).To(telem.MatchSeriesDataV[float32](15, 16))
					Expect(plusTenV.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
					Expect(plusTenV.Series[2]).To(telem.MatchSeriesDataV[float32](20, 21))
					Expect(plusTenV.Series[2].Alignment).To(Equal(telem.NewAlignment(2, 0)))

					Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
		})
	})

	Describe("Downsampling", func() {
		It("Should correctly downsample with a factor of 2", func() {
			indexCh := &channel.Channel{
				Name:     "downsample_time_2",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "downsample_sensor_2",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())
			keys := []channel.Key{indexCh.Key(), dataCh.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:            telem.SecondTS,
				Keys:             keys,
				EnableAutoCommit: config.True(),
			}))
			fr := core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8),
				},
			)
			Expect(w.Write(fr)).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:             keys,
				Bounds:           telem.TimeRangeMax,
				DownsampleFactor: 2,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

			iterValue := iter.Value()

			v := iterValue.Get(dataCh.Key())
			Expect(v.Series).To(HaveLen(1))
			Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](1, 3, 5, 7))

			idxV := iterValue.Get(indexCh.Key())
			Expect(idxV.Series).To(HaveLen(1))
			Expect(idxV.Series[0]).
				To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(1, 3, 5, 7)))

			Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		It("Should correctly downsample with a factor of 3", func() {
			indexCh := &channel.Channel{
				Name:     "downsample_time_3",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "downsample_sensor_3",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())
			keys := []channel.Key{indexCh.Key(), dataCh.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:            telem.SecondTS,
				Keys:             keys,
				EnableAutoCommit: config.True(),
			}))
			fr := core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8, 9),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8, 9),
				},
			)
			Expect(w.Write(fr)).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:             keys,
				Bounds:           telem.TimeRangeMax,
				DownsampleFactor: 3,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

			iterValue := iter.Value()

			idxValue := iterValue.Get(indexCh.Key())
			Expect(idxValue.Series).To(HaveLen(1))
			Expect(idxValue.Series[0]).
				To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(1, 4, 7)))

			v := iterValue.Get(dataCh.Key())
			Expect(v.Series).To(HaveLen(1))
			Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](1, 4, 7))

			Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})
		DescribeTable("Should not downsample when factor is 0 or 1 or negative", func(factor int) {
			suffix := strconv.Itoa(factor)
			if strings.HasPrefix(suffix, "-") {
				suffix = "neg_" + suffix[1:]
			}
			indexCh := &channel.Channel{
				Name:     "downsample_time" + suffix,
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "downsample_sensor" + suffix,
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())
			keys := []channel.Key{indexCh.Key(), dataCh.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:            telem.SecondTS,
				Keys:             keys,
				EnableAutoCommit: config.True(),
			}))
			fr := core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4),
					telem.NewSeriesV[float32](1, 2, 3, 4),
				},
			)
			MustSucceed(w.Write(fr))
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:             keys,
				Bounds:           telem.TimeRangeMax,
				DownsampleFactor: factor,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
			v := iter.Value().Get(dataCh.Key())
			Expect(v.Series).To(HaveLen(1))
			Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](1, 2, 3, 4))
			Expect(iter.Close()).To(Succeed())
		},
			Entry("factor is 0", 0),
			Entry("factor is 1", 1),
			Entry("factor is negative", -1),
		)

		It("Should correctly combine downsampling with calculations", func() {
			indexCh := &channel.Channel{
				Name:     "downsample_calc_time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh1 := &channel.Channel{
				Name:       "downsample_calc_sensor1",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh1)).To(Succeed())
			dataCh2 := &channel.Channel{
				Name:       "downsample_calc_sensor2",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh2)).To(Succeed())

			calculation := &channel.Channel{
				Name:       "downsample_calc_output",
				DataType:   telem.Float32T,
				Expression: "return downsample_calc_sensor1 + downsample_calc_sensor2",
			}
			Expect(dist.Channel.Create(ctx, calculation)).To(Succeed())

			keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:            telem.SecondTS,
				Keys:             keys,
				EnableAutoCommit: config.True(),
			}))
			fr := core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8),
				},
			)
			Expect(w.Write(fr)).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:             []channel.Key{calculation.Key(), calculation.Index()},
				Bounds:           telem.TimeRangeMax,
				DownsampleFactor: 2,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

			// sensor1 + sensor2 = [2, 4, 6, 8, 10, 12, 14, 16]
			// downsampled by 2 = [2, 6, 10, 14]
			v := iter.Value().Get(calculation.Key())
			Expect(v.Series).To(HaveLen(1))
			Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](2, 6, 10, 14))

			Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		It("Should correctly downsample across multiple domains", func() {
			indexCh := &channel.Channel{
				Name:     "downsample_multi_time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "downsample_multi_sensor",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())
			keys := []channel.Key{indexCh.Key(), dataCh.Key()}

			// First domain
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:            telem.SecondTS,
				Keys:             keys,
				EnableAutoCommit: config.True(),
			}))
			Expect(w.Write(core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4),
					telem.NewSeriesV[float32](1, 2, 3, 4),
				},
			))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			// Second domain
			w = MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start:            telem.SecondTS * 10,
				Keys:             keys,
				EnableAutoCommit: config.True(),
			}))
			Expect(w.Write(core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12, 13),
					telem.NewSeriesV[float32](10, 11, 12, 13),
				},
			))).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:             keys,
				Bounds:           telem.TimeRangeMax,
				DownsampleFactor: 2,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())

			v := iter.Value().Get(dataCh.Key())
			Expect(v.Series).To(HaveLen(2))
			// Domain 0: [1, 2, 3, 4] downsampled by 2 = [1, 3]
			Expect(v.Series[0]).To(telem.MatchSeriesDataV[float32](1, 3))
			// Domain 1: [10, 11, 12, 13] downsampled by 2 = [10, 12]
			Expect(v.Series[1]).To(telem.MatchSeriesDataV[float32](10, 12))

			Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})
	})
})
