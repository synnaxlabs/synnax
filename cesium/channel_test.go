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
	"math"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel", Ordered, func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db *cesium.DB
				fs fs.FS
			)
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				fs = openFS()
				db = mustOpenDBOnFS(ctx, fs)
			})

			Describe("Create", func() {
				DescribeTable(
					"Validation",
					func(ctx SpecContext, substring string, channels ...cesium.Channel) {
						Expect(
							db.CreateChannel(ctx, channels...),
						).To(MatchError(ContainSubstring(substring)))
					},
					Entry("ChannelKey has no datatype",
						"data_type: required",
						cesium.Channel{Name: "cat", Key: 9990, IsIndex: true},
						cesium.Channel{Name: "dog", Key: 9991, Index: 9990},
					),
					Entry(
						"ChannelKey key already exists",
						"cannot create channel [Isaac]<9992> because it already exists",
						cesium.Channel{
							Name:     "Bob",
							Key:      9992,
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						cesium.Channel{
							Key:      9992,
							Name:     "Isaac",
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
					),
					Entry(
						"ChannelKey IsIndex - Non Int64 Series Variant",
						"data_type: index channel must be of type timestamp",
						cesium.Channel{
							Name:     "Richard",
							Key:      9993,
							IsIndex:  true,
							DataType: telem.Float32T,
						},
					),
					Entry(
						"ChannelKey IsIndex - LocalIndex non-zero",
						"index: index channel cannot be indexed by another channel",
						cesium.Channel{
							Name:     "Feynman",
							Key:      9995,
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Name:     "Cavendish",
							Key:      9996,
							IsIndex:  true,
							Index:    9995,
							DataType: telem.TimestampT,
						},
					),
					Entry(
						"ChannelKey has index - LocalIndex does not exist",
						"index: index channel with key 9994 does not exist",
						cesium.Channel{
							Name:     "Laplatz",
							Key:      9997,
							Index:    9994,
							DataType: telem.Float64T,
						},
					),
					Entry(
						"ChannelKey has no index",
						"index: non-indexed channel must have an index",
						cesium.Channel{
							Name:     "Steinbeck",
							Key:      9998,
							DataType: telem.Float32T,
						},
					),
					Entry(
						"ChannelKey has index - provided index key is not an indexed channel",
						"index: channel [Sarah]<9981> is not an index",
						cesium.Channel{
							Name:     "Hemingway",
							Key:      9980,
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						cesium.Channel{
							Name:     "Sarah",
							Key:      9981,
							DataType: telem.Float64T,
							Index:    9980,
						},
						cesium.Channel{
							Name:     "Kathy",
							Key:      9982,
							Index:    9981,
							DataType: telem.Float32T,
						},
					),
				)
				Describe("DB Closed", func() {
					It("Should not allow creating a channel", func(ctx SpecContext) {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(ctx, sub)
						Expect(subDB.Close()).To(Succeed())
						err := subDB.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      key,
								DataType: telem.TimestampT,
								IsIndex:  true,
							},
						)
						Expect(err).To(MatchError(cesium.ErrDBClosed))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
					It("Should not allow retrieving channels", func(ctx SpecContext) {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(ctx, sub)
						Expect(subDB.CreateChannel(ctx, cesium.Channel{
							Key:      key,
							Name:     "Lebron",
							IsIndex:  true,
							DataType: telem.TimestampT,
						})).To(Succeed())
						Expect(subDB.Close()).To(Succeed())

						Expect(subDB.RetrieveChannel(ctx, key)).Error().
							To(MatchError(cesium.ErrDBClosed))
						Expect(subDB.RetrieveChannels(ctx, key)).
							Error().To(MatchError(cesium.ErrDBClosed))
						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
				})
			})

			Describe("Retrieve", func() {
				var k1, k2, k3 cesium.ChannelKey
				BeforeEach(func(ctx SpecContext) {
					k1, k2, k3 = GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(ctx, []cesium.Channel{
						{
							Name:     "Christian",
							Key:      k1,
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						{Name: "Ben", Key: k2, DataType: telem.Uint32T, Index: k1},
						{Name: "Bohmer", Key: k3, DataType: telem.Int8T, Index: k1},
					}...)).To(Succeed())
				})
				It("Should retrieve multiple channels", func(ctx SpecContext) {
					chs := MustSucceed(db.RetrieveChannels(ctx, k1, k2, k3))
					Expect(chs).To(HaveLen(3))
					Expect(chs[0].Key).To(Equal(k1))
					Expect(chs[1].Key).To(Equal(k2))
					Expect(chs[2].Key).To(Equal(k3))
				})
				It("Should fail if one retrieval fails", func(ctx SpecContext) {
					chs, err := db.RetrieveChannels(ctx, k1, k2, math.MaxUint32)
					Expect(chs).To(BeEmpty())
					Expect(err).To(MatchError(cesium.ErrChannelNotFound))
				})
			})

			Describe("Variable Channel Lifecycle", func() {
				It(
					"Should create and retrieve a variable channel",
					func(ctx SpecContext) {
						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "var-lc-idx",
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "var-lc-str",
								Index:    idx,
								DataType: telem.StringT,
							},
						)).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, data))
						Expect(ch.DataType).To(Equal(telem.StringT))
						Expect(ch.Name).To(Equal("var-lc-str"))
					},
				)
				It("Should rename a variable channel", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      idx,
							Name:     "var-rn-idx",
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Key:      data,
							Name:     "var-rn-str",
							Index:    idx,
							DataType: telem.StringT,
						},
					)).To(Succeed())
					Expect(db.RenameChannel(ctx, data, "renamed")).To(Succeed())
					ch := MustSucceed(db.RetrieveChannel(ctx, data))
					Expect(ch.Name).To(Equal("renamed"))
				})
				It("Should delete a variable channel", func(ctx SpecContext) {
					var (
						idx  = GenerateChannelKey()
						data = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      idx,
							Name:     "var-del-idx",
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Key:      data,
							Name:     "var-del-str",
							Index:    idx,
							DataType: telem.StringT,
						},
					)).To(Succeed())
					Expect(db.DeleteChannel(data)).To(Succeed())
					Expect(
						db.RetrieveChannel(ctx, data),
					).Error().
						To(MatchError(cesium.ErrChannelNotFound))
				})
			})

			Describe("Rename", func() {
				It(
					"Should rename a channel into a different name while channel is being used",
					func(ctx SpecContext) {
						key := GenerateChannelKey()
						Expect(
							db.CreateChannel(
								ctx,
								cesium.Channel{
									Key:      key,
									Name:     "fermat",
									DataType: telem.TimestampT,
									IsIndex:  true,
								},
							),
						).To(Succeed())
						w := MustSucceed(
							db.OpenWriter(
								ctx,
								cesium.WriterConfig{
									Start:    0,
									Channels: []cesium.ChannelKey{key},
								},
							),
						)
						series1 := telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14)
						MustSucceed(
							w.Write(
								telem.MultiFrame(
									[]cesium.ChannelKey{key},
									[]telem.Series{series1},
								),
							),
						)

						Expect(db.RenameChannel(ctx, key, "laplace")).To(Succeed())
						series2 := telem.NewSeriesSecondsTSV(20, 21, 22)
						MustSucceed(
							w.Write(
								telem.MultiFrame(
									[]cesium.ChannelKey{key},
									[]telem.Series{series2},
								),
							),
						)
						Expect(w.Close()).To(Succeed())

						ch := MustSucceed(db.RetrieveChannel(ctx, key))
						Expect(ch.Name).To(Equal("laplace"))
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
						Expect(f.Count()).To(Equal(1))
						Expect(
							f.SeriesAt(0),
						).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 20, 21, 22)))

						var (
							subFS = MustSucceed(fs.Sub(strconv.Itoa(int(key))))
							meta  = MustSucceed(subFS.Open("meta.json", os.O_RDONLY))
							buf   = make([]byte, MustSucceed(meta.Stat()).Size())
							newCh cesium.Channel
						)

						MustSucceed(meta.Read(buf))
						Expect(meta.Close()).To(Succeed())

						Expect(json.Codec.Decode(ctx, buf, &newCh)).To(Succeed())
						Expect(newCh.Name).To(Equal("laplace"))
					},
				)
				It("Should correctly rename multiple channels", func(ctx SpecContext) {
					key1 := GenerateChannelKey()
					key2 := GenerateChannelKey()
					key3 := GenerateChannelKey()
					key4 := GenerateChannelKey()
					Expect(
						db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      key1,
								Name:     "fermat",
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
						),
					).To(Succeed())
					Expect(
						db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      key2,
								Name:     "laplace",
								Index:    key1,
								DataType: telem.Float32T,
							},
						),
					).To(Succeed())
					Expect(
						db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      key3,
								Name:     "newton",
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
						),
					).To(Succeed())
					Expect(
						db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      key4,
								Name:     "descartes",
								Virtual:  true,
								DataType: telem.StringT,
							},
						),
					).To(Succeed())

					Expect(db.RenameChannels(ctx, map[cesium.ChannelKey]string{
						key1: "newton2",
						key2: "fermat3",
						key3: "laplace4",
						key4: "descartes5",
					})).To(Succeed())

					ch := MustSucceed(db.RetrieveChannel(ctx, key1))
					Expect(ch.Name).To(Equal("newton2"))
					ch = MustSucceed(db.RetrieveChannel(ctx, key2))
					Expect(ch.Name).To(Equal("fermat3"))
					ch = MustSucceed(db.RetrieveChannel(ctx, key3))
					Expect(ch.Name).To(Equal("laplace4"))
					ch = MustSucceed(db.RetrieveChannel(ctx, key4))
					Expect(ch.Name).To(Equal("descartes5"))
				})
				It("Should error if the channel is not found", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(
						db.RenameChannel(ctx, key, "new_name"),
					).To(MatchError(cesium.ErrChannelNotFound))
				})
			})
		})
	}
})
