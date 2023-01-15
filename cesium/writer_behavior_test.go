// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer Behavior", Ordered, func() {
	var db cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Happy Path", func() {
		Context("Indexed", func() {
			Specify("Basic Write", func() {
				By("Creating a channel")
				Expect(db.CreateChannel(
					cesium.Channel{Key: "basic1Index", IsIndex: true, DataType: telem.TimeStampT},
					cesium.Channel{Key: "basic1", Index: "basic1Index", DataType: telem.Int64T},
				)).To(Succeed())
				w := MustSucceed(db.NewWriter(cesium.WriterConfig{
					Channels: []string{"basic1", "basic1Index"},
					Start:    10 * telem.SecondTS,
				}))

				By("Writing data to the channel")
				Expect(w.Write(cesium.NewFrame(
					[]string{"basic1Index", "basic1"},
					[]telem.Array{
						telem.NewSecondsTSV(10, 11, 12, 13),
						telem.NewArrayV[int64](1, 2, 3, 4),
					}),
				)).To(BeTrue())
				Expect(w.Commit()).To(BeTrue())
				Expect(w.Close()).To(Succeed())

				By("Reading the data back")
				frame := MustSucceed(db.Read(telem.TimeRangeMax, "basic1"))
				Expect(frame.Arrays[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
				tsFrame := MustSucceed(db.Read(telem.TimeRangeMax, "basic1Index"))
				Expect(tsFrame.Arrays[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
			})
		})
	})
	Describe("Open Errors", func() {
		Specify("Channels with different indexes", func() {
			Expect(db.CreateChannel(
				cesium.Channel{Key: "diffIndex1", IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: "diffIndex2", IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: "diffIndex3", Index: "diffIndex1", DataType: telem.Int64T},
				cesium.Channel{Key: "diffIndex4", Index: "diffIndex2", DataType: telem.Int64T},
			)).To(Succeed())
			_, err := db.NewWriter(cesium.WriterConfig{
				Channels: []string{"diffIndex3", "diffIndex4"},
				Start:    10 * telem.SecondTS,
			})
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("channels must have the same index"))
		})
		Specify("Channels with different data rates", func() {
			Expect(db.CreateChannel(
				cesium.Channel{Key: "diffRate2", DataType: telem.Int64T, Rate: 1},
				cesium.Channel{Key: "diffRate3", DataType: telem.Int64T, Rate: 2},
			)).To(Succeed())
			_, err := db.NewWriter(cesium.WriterConfig{
				Channels: []string{"diffRate2", "diffRate3"},
				Start:    10 * telem.SecondTS,
			})
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("channels must have the same rate"))
		})
		Specify("Channel with index and channel with rate", func() {
			Expect(db.CreateChannel(
				cesium.Channel{Key: "indexRate1", IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: "indexRate2", DataType: telem.Int64T, Rate: 1},
			)).To(Succeed())
			_, err := db.NewWriter(cesium.WriterConfig{
				Channels: []string{"indexRate1", "indexRate2"},
				Start:    10 * telem.SecondTS,
			})
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("channels must have the same index"))
		})
		Specify("Channel that does not exist", func() {
			_, err := db.NewWriter(cesium.WriterConfig{
				Channels: []string{"doesNotExist"},
				Start:    10 * telem.SecondTS,
			})
			Expect(err).To(MatchError(cesium.ChannelNotFound))
		})
	})
	Describe("Frame Errors", Ordered, func() {
		BeforeAll(func() {
			Expect(db.CreateChannel(
				cesium.Channel{Key: "frameErr1", DataType: telem.Int64T, Rate: 1 * telem.Hz},
				cesium.Channel{Key: "frameErr2", DataType: telem.Int64T, Rate: 1 * telem.Hz},
			))
		})
		Specify("Uneven Frame", func() {
			w := MustSucceed(db.NewWriter(cesium.WriterConfig{
				Channels: []string{"frameErr1", "frameErr2"},
				Start:    10 * telem.SecondTS,
			}))
			Expect(w.Write(cesium.NewFrame(
				[]string{"frameErr1", "frameErr2"},
				[]telem.Array{
					telem.NewArrayV[int64](1, 2, 3, 4),
					telem.NewArrayV[int64](1, 2, 3),
				}),
			)).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			err := w.Close()
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("uneven frame"))
		})
		Specify("Frame Without All Channels", func() {
			w := MustSucceed(db.NewWriter(cesium.WriterConfig{
				Channels: []string{"frameErr1", "frameErr2"},
				Start:    10 * telem.SecondTS,
			}))
			Expect(w.Write(cesium.NewFrame(
				[]string{"frameErr1"},
				[]telem.Array{
					telem.NewArrayV[int64](1, 2, 3, 4),
				},
			))).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			err := w.Close()
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("frame without data for all channels"))
		})
		Specify("Frame with Duplicate Channels", func() {
			w := MustSucceed(db.NewWriter(cesium.WriterConfig{
				Channels: []string{"frameErr1", "frameErr2"},
				Start:    10 * telem.SecondTS,
			}))
			Expect(w.Write(cesium.NewFrame(
				[]string{"frameErr1", "frameErr1"},
				[]telem.Array{
					telem.NewArrayV[int64](1, 2, 3, 4),
					telem.NewArrayV[int64](1, 2, 3, 4),
				},
			))).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			err := w.Close()
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("duplicate channel"))
		})
		Specify("Frame with Unknown Channel", func() {
			w := MustSucceed(db.NewWriter(cesium.WriterConfig{
				Channels: []string{"frameErr1", "frameErr2"},
				Start:    10 * telem.SecondTS,
			}))
			Expect(w.Write(cesium.NewFrame(
				[]string{"frameErr1", "frameErr3"},
				[]telem.Array{
					telem.NewArrayV[int64](1, 2, 3, 4),
					telem.NewArrayV[int64](1, 2, 3, 4),
				},
			))).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			err := w.Close()
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("frameErr3"))
			Expect(err.Error()).To(ContainSubstring("not specified"))
		})
	})
	Describe("Index Errors", func() {
		Context("Discontinuous Index", func() {
			Specify("Last sample is not the index", func() {
				Expect(db.CreateChannel(
					cesium.Channel{Key: "disc1Index", IsIndex: true, DataType: telem.TimeStampT},
					cesium.Channel{Key: "disc1", Index: "disc1Index", DataType: telem.Int64T},
				)).To(Succeed())
				w := MustSucceed(db.NewWriter(cesium.WriterConfig{
					Channels: []string{"disc1Index"},
					Start:    10 * telem.SecondTS,
				}))

				By("Writing data to the index correctly")
				Expect(w.Write(cesium.NewFrame(
					[]string{"disc1Index"},
					[]telem.Array{
						telem.NewSecondsTSV(10, 11, 12, 13),
					}),
				)).To(BeTrue())
				Expect(w.Commit()).To(BeTrue())
				Expect(w.Close()).To(Succeed())

				By("Writing data to channel where the last sample is not the index")
				w = MustSucceed(db.NewWriter(cesium.WriterConfig{
					Channels: []string{"disc1"},
					Start:    10 * telem.SecondTS,
				}))
				Expect(w.Write(cesium.NewFrame(
					[]string{"disc1"},
					[]telem.Array{
						telem.NewArrayV[int64](1, 2, 3, 4, 5),
					},
				))).To(BeTrue())
				Expect(w.Commit()).To(BeFalse())
				err := w.Close()
				Expect(err).To(MatchError(cesium.ErrDiscontinuous))
			})
			Specify("Index not defined at all", func() {
				Expect(db.CreateChannel(
					cesium.Channel{Key: "disc2Index", IsIndex: true, DataType: telem.TimeStampT},
					cesium.Channel{Key: "disc2", Index: "disc2Index", DataType: telem.Int64T},
				)).To(Succeed())
				w := MustSucceed(db.NewWriter(cesium.WriterConfig{
					Channels: []string{"disc2"},
					Start:    10 * telem.SecondTS,
				}))
				Expect(w.Write(cesium.NewFrame(
					[]string{"disc2"},
					[]telem.Array{
						telem.NewArrayV[int64](1, 2, 3, 4, 5),
					},
				))).To(BeTrue())
				Expect(w.Commit()).To(BeFalse())
				Expect(w.Close()).To(MatchError(cesium.ErrDiscontinuous))
			})
		})
	})
	Describe("Data Type Errors", func() {
		Specify("Invalid Data Type for Array", func() {
			Expect(db.CreateChannel(cesium.Channel{
				Key:      "dtErr",
				DataType: telem.Int64T,
				Rate:     1,
			})).To(Succeed())
			w := MustSucceed(db.NewWriter(cesium.WriterConfig{
				Channels: []string{"dtErr"},
				Start:    10 * telem.SecondTS,
			}))
			w = MustSucceed(db.NewWriter(cesium.WriterConfig{
				Channels: []string{"dtErr"},
				Start:    15 * telem.SecondTS,
			}))
			Expect(w.Write(cesium.NewFrame(
				[]string{"dtErr"},
				[]telem.Array{
					telem.NewArrayV[float64](1, 2, 3, 4, 5),
				},
			))).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			err := w.Close()
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("expected int64, got float64"))
		})
		Specify("Invalid Data Type for Index", func() {
			Expect(db.CreateChannel(cesium.Channel{
				Key:      "dtErrIndex",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			})).To(Succeed())
			w := MustSucceed(db.NewWriter(cesium.WriterConfig{
				Channels: []string{"dtErrIndex"},
				Start:    10 * telem.SecondTS,
			}))
			Expect(w.Write(cesium.NewFrame(
				[]string{"dtErrIndex"},
				[]telem.Array{
					telem.NewArrayV[int64](1, 2, 3, 4, 5),
				},
			))).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			err := w.Close()
			Expect(err).To(MatchError(validate.Error))
			Expect(err.Error()).To(ContainSubstring("expected timestamp, got int64"))
		})
	})
})
