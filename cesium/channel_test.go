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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
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
						"channel <9992>: cannot create the channel because it already exists",
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
						"index: channel <9981>: not an index",
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
						Expect(subDB.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      key,
								DataType: telem.TimestampT,
								IsIndex:  true,
							},
						)).To(MatchError(cesium.ErrDBClosed))

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
					Expect(db.Drop([]cesium.ChannelKey{data})).To(Succeed())
					Expect(
						db.RetrieveChannel(ctx, data),
					).Error().
						To(MatchError(cesium.ErrChannelNotFound))
				})
			})
		})
	}
})
