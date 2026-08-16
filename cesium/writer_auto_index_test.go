// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer AutoIndex", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db *cesium.DB
				fs fs.FS
			)
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				fs = openFS()
				db = DeferClose(openDBOnFS(ctx, fs))
			})

			Describe("Data-only writers", func() {
				It(
					"Should stamp the missing index with monotonic timestamps",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auto_idx_1",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "auto_data_1",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						before := telem.Now()
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(
							w.Write(
								telem.UnaryFrame(
									data,
									telem.NewSeriesV[float64](1, 2, 3),
								),
							),
						)
						MustSucceed(w.Commit())
						after := telem.Now()

						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
						ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
						Expect(ts).To(HaveLen(3))
						Expect(ts[0]).To(BeNumerically(">=", before))
						Expect(ts[2]).To(BeNumerically("<=", after+telem.TimeStamp(3)))
						Expect(ts[1]).To(Equal(ts[0] + 1))
						Expect(ts[2]).To(Equal(ts[0] + 2))
					},
				)

				It(
					"Should remain strictly monotonic across multiple writes",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auto_idx_2",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "auto_data_2",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(
							w.Write(
								telem.UnaryFrame(
									data,
									telem.NewSeriesV[float64](1, 2, 3),
								),
							),
						)
						MustSucceed(
							w.Write(
								telem.UnaryFrame(
									data,
									telem.NewSeriesV[float64](4, 5, 6),
								),
							),
						)
						MustSucceed(w.Commit())

						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
						ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
						Expect(ts).To(HaveLen(6))
						for i := 1; i < len(ts); i++ {
							Expect(ts[i]).To(BeNumerically(">", ts[i-1]))
						}
					},
				)
			})

			Describe("Mixed user-provided and auto-generated timestamps", func() {
				It(
					"Should pass through caller-supplied index series untouched",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "mix_idx_1",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "mix_data_1",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data},
							AutoIndex: new(true),
							Sync:      new(true),
							Start:     100 * telem.SecondTS,
						}))
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{idx, data},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(100, 101, 102),
								telem.NewSeriesV[float64](1, 2, 3),
							},
						)))
						MustSucceed(w.Commit())

						f := MustSucceed(
							db.Read(
								ctx,
								(100 * telem.SecondTS).Range(103*telem.SecondTS),
								idx,
							),
						)
						ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
						Expect(ts).To(Equal([]telem.TimeStamp{
							100 * telem.SecondTS,
							101 * telem.SecondTS,
							102 * telem.SecondTS,
						}))
					},
				)

				It(
					"Should reject an auto-stamp that would regress the index past a caller-provided future timestamp",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "mix_idx_2",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "mix_data_2",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						future := telem.Now() + telem.HourTS
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{idx, data},
							[]telem.Series{
								telem.NewSeriesV(future),
								telem.NewSeriesV[float64](1),
							},
						)))
						Expect(
							w.Write(
								telem.UnaryFrame(data, telem.NewSeriesV[float64](2)),
							),
						).
							Error().
							To(MatchError(validate.ErrValidation))
						Expect(w.Close()).To(MatchError(validate.ErrValidation))
					},
				)

				It(
					"Does not advance the auto-stamp clock when the caller writes explicit timestamps",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "mix_idx_3",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "mix_data_3",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						before := telem.Now()
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						explicit := before + 10*telem.MillisecondTS
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{idx, data},
							[]telem.Series{
								telem.NewSeriesV(explicit),
								telem.NewSeriesV[float64](1),
							},
						)))
						Eventually(telem.Now).Should(BeNumerically(">", explicit))
						MustSucceed(
							w.Write(
								telem.UnaryFrame(data, telem.NewSeriesV[float64](2)),
							),
						)
						MustSucceed(w.Commit())

						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
						ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
						Expect(ts).To(HaveLen(2))
						Expect(ts[0]).To(Equal(explicit))
						Expect(ts[1]).To(BeNumerically(">", explicit))
					},
				)
			})

			Describe("Virtual channels", func() {
				It(
					"Should auto-stamp the indexed channel while leaving a virtual channel in the same frame untouched",
					func(ctx SpecContext) {
						var (
							idx     = GenerateChannelKey()
							data    = GenerateChannelKey()
							virtual = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "virt_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "virt_data",
								Index:    idx,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      virtual,
								Name:     "virt_chan",
								Virtual:  true,
								DataType: telem.Int64T,
							},
						)).To(Succeed())
						before := telem.Now()
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data, virtual},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{data, virtual},
							[]telem.Series{
								telem.NewSeriesV[float64](1, 2, 3),
								telem.NewSeriesV[int64](10, 20, 30),
							},
						)))
						MustSucceed(w.Commit())

						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
						ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
						Expect(ts).To(HaveLen(3))
						Expect(ts[0]).To(BeNumerically(">=", before))
					},
				)
			})

			Describe("Multiple data channels per index", func() {
				It(
					"Should stamp once for an index shared across data channels",
					func(ctx SpecContext) {
						var (
							idx   = GenerateChannelKey()
							data1 = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "shared_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "shared_data_1",
								Index:    idx,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "shared_data_2",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data1, data2},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{data1, data2},
							[]telem.Series{
								telem.NewSeriesV[float64](1, 2, 3),
								telem.NewSeriesV[float64](4, 5, 6),
							},
						)))
						MustSucceed(w.Commit())

						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
						ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
						Expect(ts).To(HaveLen(3))
					},
				)
			})

			Describe("Multiple indexes in one writer", func() {
				It(
					"Should size each implicit index to its own referencing data channel",
					func(ctx SpecContext) {
						var (
							idx1  = GenerateChannelKey()
							data1 = GenerateChannelKey()
							idx2  = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx1,
								Name:     "multi_idx_a",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "multi_data_a",
								Index:    idx1,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      idx2,
								Name:     "multi_idx_b",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "multi_data_b",
								Index:    idx2,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						before := telem.Now()
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data1, data2},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{data1, data2},
							[]telem.Series{
								telem.NewSeriesV[float64](1, 2, 3),
								telem.NewSeriesV[float64](10, 20, 30, 40),
							},
						)))
						MustSucceed(w.Commit())

						ts1 := telem.UnmarshalSeries[telem.TimeStamp](
							MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, idx1),
							).SeriesAt(0),
						)
						Expect(ts1).To(HaveLen(3))
						Expect(ts1[0]).To(BeNumerically(">=", before))
						Expect(ts1[1]).To(Equal(ts1[0] + 1))
						Expect(ts1[2]).To(Equal(ts1[0] + 2))

						ts2 := telem.UnmarshalSeries[telem.TimeStamp](
							MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, idx2),
							).SeriesAt(0),
						)
						Expect(ts2).To(HaveLen(4))
					},
				)

				It(
					"Should generate co-aligned timestamps across distinct indexes in the same write",
					func(ctx SpecContext) {
						var (
							idx1  = GenerateChannelKey()
							data1 = GenerateChannelKey()
							idx2  = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx1,
								Name:     "coalign_idx_a",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "coalign_data_a",
								Index:    idx1,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      idx2,
								Name:     "coalign_idx_b",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "coalign_data_b",
								Index:    idx2,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data1, data2},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{data1, data2},
							[]telem.Series{
								telem.NewSeriesV[float64](1, 2, 3),
								telem.NewSeriesV[float64](4, 5, 6),
							},
						)))
						MustSucceed(w.Commit())

						ts1 := telem.UnmarshalSeries[telem.TimeStamp](
							MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, idx1),
							).SeriesAt(0),
						)
						ts2 := telem.UnmarshalSeries[telem.TimeStamp](
							MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, idx2),
							).SeriesAt(0),
						)
						Expect(ts1).To(Equal(ts2))
					},
				)

				It(
					"Should auto-stamp one index while passing through a user-provided value for another",
					func(ctx SpecContext) {
						var (
							idx1  = GenerateChannelKey()
							data1 = GenerateChannelKey()
							idx2  = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx1,
								Name:     "mixed_multi_idx_a",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "mixed_multi_data_a",
								Index:    idx1,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      idx2,
								Name:     "mixed_multi_idx_b",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "mixed_multi_data_b",
								Index:    idx2,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						before := telem.Now()
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{idx1, data1, data2},
							Start:     100 * telem.SecondTS,
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{idx1, data1, data2},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(100, 101),
								telem.NewSeriesV[float64](1, 2),
								telem.NewSeriesV[float64](3, 4),
							},
						)))
						MustSucceed(w.Commit())

						ts1 := telem.UnmarshalSeries[telem.TimeStamp](
							MustSucceed(
								db.Read(
									ctx,
									(100 * telem.SecondTS).Range(102*telem.SecondTS),
									idx1,
								),
							).SeriesAt(0),
						)
						Expect(
							ts1,
						).To(Equal([]telem.TimeStamp{100 * telem.SecondTS, 101 * telem.SecondTS}))

						ts2 := telem.UnmarshalSeries[telem.TimeStamp](
							MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, idx2),
							).SeriesAt(0),
						)
						Expect(ts2).To(HaveLen(2))
						Expect(ts2[0]).To(BeNumerically(">=", before))
						Expect(ts2[1]).To(Equal(ts2[0] + 1))
					},
				)
			})

			Describe("Default Start", func() {
				It(
					"Defaults Start to telem.Now() when AutoIndex is true and Start is zero",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "default_start_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "default_start_data",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						before := telem.Now()
						w := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						MustSucceed(
							w.Write(
								telem.UnaryFrame(
									data,
									telem.NewSeriesV[float64](1, 2, 3),
								),
							),
						)
						MustSucceed(w.Commit())

						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
						ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
						Expect(ts).To(HaveLen(3))
						Expect(ts[0]).To(BeNumerically(">=", before))
					},
				)

				It(
					"Rejects an explicit index series with timestamps before the defaulted Start",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "default_start_reject_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "default_start_reject_data",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:  []cesium.ChannelKey{data},
							AutoIndex: new(true),
							Sync:      new(true),
						}))
						Expect(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{idx, data},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(1),
								telem.NewSeriesV[float64](1),
							},
						))).Error().To(MatchError(validate.ErrValidation))
						Expect(w.Close()).To(MatchError(validate.ErrValidation))
					},
				)

				It("Preserves an explicitly provided Start", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      idx,
							Name:     "default_start_preserve_idx",
							IsIndex:  true,
							DataType: telem.TimeStampT,
						},
						cesium.Channel{
							Key:      data,
							Name:     "default_start_preserve_data",
							Index:    idx,
							DataType: telem.Float64T,
						},
					)).To(Succeed())

					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:  []cesium.ChannelKey{data},
						AutoIndex: new(true),
						Sync:      new(true),
						Start:     50 * telem.SecondTS,
					}))
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{idx, data},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(100, 101),
							telem.NewSeriesV[float64](1, 2),
						},
					)))
					MustSucceed(w.Commit())
					Expect(w.Close()).To(Succeed())

					f := MustSucceed(
						db.Read(
							ctx,
							(50 * telem.SecondTS).Range(102*telem.SecondTS),
							idx,
						),
					)
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(Equal([]telem.TimeStamp{
						100 * telem.SecondTS,
						101 * telem.SecondTS,
					}))
				})
			})

			Describe("Disabled", func() {
				It(
					"Should reject a frame missing the index when AutoIndex is false",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "off_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "off_data",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{idx, data},
							Start:    1 * telem.SecondTS,
							Sync:     new(true),
						}))
						Expect(
							w.Write(
								telem.UnaryFrame(data, telem.NewSeriesV[float64](1, 2)),
							),
						).
							Error().
							To(MatchError(validate.ErrValidation))
						Expect(w.Close()).To(MatchError(validate.ErrValidation))
					},
				)
			})

			Describe("SetAuthority propagation", func() {
				It(
					"Raises an implicit index when its referencing data channel is raised",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auth_idx_1",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "auth_data_1",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{data},
							Authorities:    []control.Authority{control.Authority(50)},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "wA"},
						}))

						wB := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(100)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "wB"},
						}))

						authorized := MustSucceed(
							wB.Write(
								telem.UnaryFrame(idx, telem.NewSeriesSecondsTSV(10)),
							),
						)
						Expect(authorized).To(BeTrue())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{data},
							Authorities: []control.Authority{control.Authority(200)},
						})).To(Succeed())

						authorized = MustSucceed(
							wB.Write(
								telem.UnaryFrame(idx, telem.NewSeriesSecondsTSV(11)),
							),
						)
						Expect(authorized).To(BeFalse())
					},
				)

				It(
					"Tracks the max across data channels sharing an implicit index",
					func(ctx SpecContext) {
						var (
							idx   = GenerateChannelKey()
							data1 = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auth_idx_2",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "auth_data_2a",
								Index:    idx,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "auth_data_2b",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{data1, data2},
							Authorities: []control.Authority{
								control.Authority(50),
								control.Authority(50),
							},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "wA_max"},
						}))

						wB := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(150)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "wB_max"},
						}))

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(10),
						)))).To(BeTrue())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{data1},
							Authorities: []control.Authority{control.Authority(100)},
						})).To(Succeed())
						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(11),
						)))).To(BeTrue())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{data2},
							Authorities: []control.Authority{control.Authority(200)},
						})).To(Succeed())
						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(12),
						)))).To(BeFalse())
					},
				)

				It(
					"Broadcasts a single authority across multiple data channels and propagates to the implicit index",
					func(ctx SpecContext) {
						var (
							idx   = GenerateChannelKey()
							data1 = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auth_bcast_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "auth_bcast_data1",
								Index:    idx,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "auth_bcast_data2",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{data1, data2},
							Authorities: []control.Authority{
								control.Authority(50),
								control.Authority(50),
							},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "wA_bcast"},
						}))

						wB := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(100)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "wB_bcast"},
						}))

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(10),
						)))).To(BeTrue())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{data1, data2},
							Authorities: []control.Authority{control.Authority(200)},
						})).To(Succeed())

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(11),
						)))).To(BeFalse())
					},
				)

				It(
					"Lowers the implicit index when both referencing data channels are lowered",
					func(ctx SpecContext) {
						var (
							idx   = GenerateChannelKey()
							data1 = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auth_lower_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "auth_lower_data1",
								Index:    idx,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "auth_lower_data2",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{data1, data2},
							Authorities: []control.Authority{
								control.Authority(200),
								control.Authority(200),
							},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "wA_lower"},
						}))

						wB := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(150)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "wB_lower"},
						}))

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(10),
						)))).To(BeFalse())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Channels: []cesium.ChannelKey{data1, data2},
							Authorities: []control.Authority{
								control.Authority(50),
								control.Authority(50),
							},
						})).To(Succeed())

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(11),
						)))).To(BeTrue())
					},
				)

				It(
					"Propagates from a broadcast SetAuthority through subsequent per-channel calls",
					func(ctx SpecContext) {
						var (
							idx   = GenerateChannelKey()
							data1 = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auth_chain_idx",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "auth_chain_data1",
								Index:    idx,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "auth_chain_data2",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{data1, data2},
							Authorities:    []control.Authority{control.Authority(50)},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "wA_chain"},
						}))

						wB := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(150)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "wB_chain"},
						}))

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(10),
						)))).To(BeTrue())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Authorities: []control.Authority{control.Authority(100)},
						})).To(Succeed())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{data1},
							Authorities: []control.Authority{control.Authority(200)},
						})).To(Succeed())

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(11),
						)))).To(BeFalse())
					},
				)

				It(
					"Leaves the index alone when explicitly named in SetAuthority",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "auth_idx_3",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "auth_data_3",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{data},
							Authorities:    []control.Authority{control.Authority(50)},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "wA_explicit"},
						}))

						wB := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(100)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "wB_explicit"},
						}))

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(10),
						)))).To(BeTrue())

						Expect(wA.SetAuthority(cesium.WriterConfig{
							Channels: []cesium.ChannelKey{data, idx},
							Authorities: []control.Authority{
								control.Authority(250),
								control.Authority(30),
							},
						})).To(Succeed())

						Expect(MustSucceed(wB.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(11),
						)))).To(BeTrue())
					},
				)
			})

			Describe("Open-time authority inheritance", func() {
				It(
					"Inherits the broadcast authority on an implicit index",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "open_idx_1",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "open_data_1",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{data},
							Authorities:    []control.Authority{control.Authority(150)},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "open_wA"},
						}))

						wLow := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(100)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "open_wLow"},
						}))
						Expect(MustSucceed(wLow.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(10),
						)))).To(BeFalse())

						wHigh := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(200)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "open_wHigh"},
						}))
						Expect(MustSucceed(wHigh.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(11),
						)))).To(BeTrue())
					},
				)

				It(
					"Takes the max across data channels sharing an index",
					func(ctx SpecContext) {
						var (
							idx   = GenerateChannelKey()
							data1 = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "open_idx_2",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "open_data_2a",
								Index:    idx,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "open_data_2b",
								Index:    idx,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{data1, data2},
							Authorities: []control.Authority{
								control.Authority(50),
								control.Authority(200),
							},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "open_max_wA"},
						}))

						wBetween := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(100)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "open_max_wBetween"},
						}))
						Expect(MustSucceed(wBetween.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(10),
						)))).To(BeFalse())

						wAbove := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx},
							Authorities:    []control.Authority{control.Authority(220)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "open_max_wAbove"},
						}))
						Expect(MustSucceed(wAbove.Write(telem.UnaryFrame(
							idx, telem.NewSeriesSecondsTSV(11),
						)))).To(BeTrue())
					},
				)

				It(
					"Resolves authority per-index when distinct indexes are implicit",
					func(ctx SpecContext) {
						var (
							idx1  = GenerateChannelKey()
							data1 = GenerateChannelKey()
							idx2  = GenerateChannelKey()
							data2 = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx1,
								Name:     "open_perindex_idx_a",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data1,
								Name:     "open_perindex_data_a",
								Index:    idx1,
								DataType: telem.Float64T,
							},
							cesium.Channel{
								Key:      idx2,
								Name:     "open_perindex_idx_b",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      data2,
								Name:     "open_perindex_data_b",
								Index:    idx2,
								DataType: telem.Float64T,
							},
						)).To(Succeed())

						MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{data1, data2},
							Authorities: []control.Authority{
								control.Authority(80),
								control.Authority(180),
							},
							AutoIndex:      new(true),
							Sync:           new(true),
							Start:          1 * telem.SecondTS,
							ControlSubject: control.Subject{Key: "open_perindex_wA"},
						}))

						wIdx1 := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx1},
							Authorities:    []control.Authority{control.Authority(100)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "open_perindex_wIdx1"},
						}))
						Expect(MustSucceed(wIdx1.Write(telem.UnaryFrame(
							idx1, telem.NewSeriesSecondsTSV(10),
						)))).To(BeTrue())

						wIdx2 := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{idx2},
							Authorities:    []control.Authority{control.Authority(100)},
							Sync:           new(true),
							ControlSubject: control.Subject{Key: "open_perindex_wIdx2"},
						}))
						Expect(MustSucceed(wIdx2.Write(telem.UnaryFrame(
							idx2, telem.NewSeriesSecondsTSV(10),
						)))).To(BeFalse())
					},
				)
			})
		})
	}
})
