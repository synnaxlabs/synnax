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

var _ = Describe("Writer AutoIndexing", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			ShouldNotLeakGoroutinesPerSpec()
			var (
				db *cesium.DB
				fs fs.FS
			)
			BeforeAll(func(ctx SpecContext) {
				fs = openFS()
				db = DeferClose(openDBOnFS(ctx, fs))
			})

			Describe("Data-only writers", func() {
				It("Should stamp the missing index with monotonic timestamps", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "auto_idx_1", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "auto_data_1", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					before := telem.Now()
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data},
						AutoIndexing: new(true),
						Sync:         new(true),
					}))
					MustSucceed(w.Write(telem.UnaryFrame(data, telem.NewSeriesV[float64](1, 2, 3))))
					MustSucceed(w.Commit())
					Expect(w.Close()).To(Succeed())
					after := telem.Now()

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(HaveLen(3))
					Expect(ts[0]).To(BeNumerically(">=", before))
					Expect(ts[2]).To(BeNumerically("<=", after+telem.TimeStamp(3)))
					Expect(ts[1]).To(Equal(ts[0] + 1))
					Expect(ts[2]).To(Equal(ts[0] + 2))
				})

				It("Should remain strictly monotonic across multiple writes", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "auto_idx_2", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "auto_data_2", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data},
						AutoIndexing: new(true),
						Sync:         new(true),
					}))
					MustSucceed(w.Write(telem.UnaryFrame(data, telem.NewSeriesV[float64](1, 2, 3))))
					MustSucceed(w.Write(telem.UnaryFrame(data, telem.NewSeriesV[float64](4, 5, 6))))
					MustSucceed(w.Commit())
					Expect(w.Close()).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(HaveLen(6))
					for i := 1; i < len(ts); i++ {
						Expect(ts[i]).To(BeNumerically(">", ts[i-1]))
					}
				})
			})

			Describe("Mixed user-provided and auto-generated timestamps", func() {
				It("Should pass through caller-supplied index series untouched", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "mix_idx_1", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "mix_data_1", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data},
						AutoIndexing: new(true),
						Sync:         new(true),
						Start:        100 * telem.SecondTS,
					}))
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{idx, data},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(100, 101, 102),
							telem.NewSeriesV[float64](1, 2, 3),
						},
					)))
					MustSucceed(w.Commit())
					Expect(w.Close()).To(Succeed())

					f := MustSucceed(db.Read(ctx, (100 * telem.SecondTS).Range(103*telem.SecondTS), idx))
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(Equal([]telem.TimeStamp{
						100 * telem.SecondTS,
						101 * telem.SecondTS,
						102 * telem.SecondTS,
					}))
				})

				It("Should resume auto-stamping from the high-water mark after a future user-provided value", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "mix_idx_2", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "mix_data_2", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())
					future := telem.Now() + telem.HourTS
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data},
						AutoIndexing: new(true),
						Sync:         new(true),
					}))
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{idx, data},
						[]telem.Series{
							telem.NewSeriesV(future),
							telem.NewSeriesV[float64](1),
						},
					)))
					MustSucceed(w.Write(telem.UnaryFrame(data, telem.NewSeriesV[float64](2))))
					MustSucceed(w.Commit())
					Expect(w.Close()).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(HaveLen(2))
					Expect(ts[0]).To(Equal(future))
					Expect(ts[1]).To(BeNumerically(">", future))
				})
			})

			Describe("Multiple data channels per index", func() {
				It("Should stamp once for an index shared across data channels", func(ctx SpecContext) {
					var (
						idx   = GenerateChannelKey()
						data1 = GenerateChannelKey()
						data2 = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "shared_idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data1, Name: "shared_data_1", Index: idx, DataType: telem.Float64T},
						cesium.Channel{Key: data2, Name: "shared_data_2", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data1, data2},
						AutoIndexing: new(true),
						Sync:         new(true),
					}))
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{data1, data2},
						[]telem.Series{
							telem.NewSeriesV[float64](1, 2, 3),
							telem.NewSeriesV[float64](4, 5, 6),
						},
					)))
					MustSucceed(w.Commit())
					Expect(w.Close()).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(HaveLen(3))
				})
			})

			Describe("Default Start", func() {
				It("Defaults Start to telem.Now() when AutoIndexing is true and Start is zero", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "default_start_idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "default_start_data", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					before := telem.Now()
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data},
						AutoIndexing: new(true),
						Sync:         new(true),
					}))
					MustSucceed(w.Write(telem.UnaryFrame(data, telem.NewSeriesV[float64](1, 2, 3))))
					MustSucceed(w.Commit())
					Expect(w.Close()).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(HaveLen(3))
					Expect(ts[0]).To(BeNumerically(">=", before))
				})

				It("Rejects an explicit index series with timestamps before the defaulted Start", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "default_start_reject_idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "default_start_reject_data", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data},
						AutoIndexing: new(true),
						Sync:         new(true),
					}))
					Expect(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{idx, data},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(1),
							telem.NewSeriesV[float64](1),
						},
					))).Error().To(MatchError(validate.ErrValidation))
					Expect(w.Close()).To(MatchError(validate.ErrValidation))
				})

				It("Preserves an explicitly provided Start", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "default_start_preserve_idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "default_start_preserve_data", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:     []cesium.ChannelKey{data},
						AutoIndexing: new(true),
						Sync:         new(true),
						Start:        50 * telem.SecondTS,
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

					f := MustSucceed(db.Read(ctx, (50*telem.SecondTS).Range(102*telem.SecondTS), idx))
					ts := telem.UnmarshalSeries[telem.TimeStamp](f.SeriesAt(0))
					Expect(ts).To(Equal([]telem.TimeStamp{
						100 * telem.SecondTS,
						101 * telem.SecondTS,
					}))
				})
			})

			Describe("Disabled", func() {
				It("Should reject a frame missing the index when AutoIndexing is false", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "off_idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "off_data", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{idx, data},
						Start:    1 * telem.SecondTS,
						Sync:     new(true),
					}))
					Expect(w.Write(telem.UnaryFrame(data, telem.NewSeriesV[float64](1, 2)))).
						Error().
						To(MatchError(validate.ErrValidation))
					Expect(w.Close()).To(MatchError(validate.ErrValidation))
				})
			})

			Describe("SetAuthority propagation", func() {
				It("Raises an implicit index when its referencing data channel is raised", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "auth_idx_1", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "auth_data_1", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:       []cesium.ChannelKey{data},
						Authorities:    []control.Authority{control.Authority(50)},
						AutoIndexing:   new(true),
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

					authorized := MustSucceed(wB.Write(telem.UnaryFrame(idx, telem.NewSeriesSecondsTSV(10))))
					Expect(authorized).To(BeTrue())

					Expect(wA.SetAuthority(cesium.WriterConfig{
						Channels:    []cesium.ChannelKey{data},
						Authorities: []control.Authority{control.Authority(200)},
					})).To(Succeed())

					authorized = MustSucceed(wB.Write(telem.UnaryFrame(idx, telem.NewSeriesSecondsTSV(11))))
					Expect(authorized).To(BeFalse())
				})

				It("Tracks the max across data channels sharing an implicit index", func(ctx SpecContext) {
					var (
						idx   = GenerateChannelKey()
						data1 = GenerateChannelKey()
						data2 = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "auth_idx_2", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data1, Name: "auth_data_2a", Index: idx, DataType: telem.Float64T},
						cesium.Channel{Key: data2, Name: "auth_data_2b", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{data1, data2},
						Authorities: []control.Authority{
							control.Authority(50),
							control.Authority(50),
						},
						AutoIndexing:   new(true),
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
				})

				It("Broadcasts a single authority across multiple data channels and propagates to the implicit index", func(ctx SpecContext) {
					var (
						idx   = GenerateChannelKey()
						data1 = GenerateChannelKey()
						data2 = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "auth_bcast_idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data1, Name: "auth_bcast_data1", Index: idx, DataType: telem.Float64T},
						cesium.Channel{Key: data2, Name: "auth_bcast_data2", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{data1, data2},
						Authorities: []control.Authority{
							control.Authority(50),
							control.Authority(50),
						},
						AutoIndexing:   new(true),
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
				})

				It("Leaves the index alone when explicitly named in SetAuthority", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "auth_idx_3", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "auth_data_3", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					wA := MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:       []cesium.ChannelKey{data},
						Authorities:    []control.Authority{control.Authority(50)},
						AutoIndexing:   new(true),
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
				})
			})

			Describe("Open-time authority inheritance", func() {
				It("Inherits the broadcast authority on an implicit index", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "open_idx_1", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data, Name: "open_data_1", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:       []cesium.ChannelKey{data},
						Authorities:    []control.Authority{control.Authority(150)},
						AutoIndexing:   new(true),
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
				})

				It("Takes the max across data channels sharing an index", func(ctx SpecContext) {
					var (
						idx   = GenerateChannelKey()
						data1 = GenerateChannelKey()
						data2 = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "open_idx_2", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: data1, Name: "open_data_2a", Index: idx, DataType: telem.Float64T},
						cesium.Channel{Key: data2, Name: "open_data_2b", Index: idx, DataType: telem.Float64T},
					)).To(Succeed())

					MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{data1, data2},
						Authorities: []control.Authority{
							control.Authority(50),
							control.Authority(200),
						},
						AutoIndexing:   new(true),
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
				})
			})
		})
	}
})
