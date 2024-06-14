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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db      *cesium.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
				db = openDBOnFS(fs)
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Accuracy", func() {
				var (
					dataKey, indexKey cesium.ChannelKey
					data, index       cesium.Channel
					i                 *cesium.Iterator
				)
				AfterAll(func() { Expect(i.Close()).To(Succeed()) })
				BeforeAll(func() {
					dataKey, indexKey = GenerateChannelKey(), GenerateChannelKey()
					index = cesium.Channel{Key: indexKey, IsIndex: true, DataType: telem.TimeStampT}
					data = cesium.Channel{Key: dataKey, Index: indexKey, DataType: telem.Uint16T}

					Expect(db.CreateChannel(ctx, index, data)).To(Succeed())
					Expect(db.Write(ctx, 0, cesium.NewFrame(
						[]cesium.ChannelKey{indexKey, dataKey},
						[]telem.Series{
							telem.NewSecondsTSV(0, 1, 2),
							telem.NewSeriesV[uint16](10, 11, 12),
						},
					))).To(Succeed())
					Expect(db.Write(ctx, 10*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{indexKey, dataKey},
						[]telem.Series{
							telem.NewSecondsTSV(10, 12, 15),
							telem.NewSeriesV[uint16](20, 22, 25),
						},
					))).To(Succeed())
					Expect(db.Write(ctx, 4*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{indexKey, dataKey},
						[]telem.Series{
							telem.NewSecondsTSV(4, 7, 9),
							telem.NewSeriesV[uint16](14, 17, 19),
						},
					))).To(Succeed())

					// Index: 0  1  2  4  7  9 10 12 15
					// Data: 10 11 12 14 17 19 20 22 25

					i = MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Channels: []cesium.ChannelKey{dataKey},
						Bounds:   telem.TimeRangeMax,
					}))
				})
				Specify("Forward - No bounds", func() {
					By("Seek First")
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Valid()).To(BeFalse())

					By("Next")
					Expect(i.Next(3 * telem.Second)).To(BeTrue())
					Expect(i.Value().Series).To(HaveLen(1))
					Expect(i.Value().Series[0].Data).To(EqualUnmarshal([]uint16{10, 11, 12}))
					Expect(i.Value().Series[0].TimeRange).To(Equal((0 * telem.SecondTS).Range(2*telem.SecondTS + 1)))

					Expect(i.Next(1 * telem.Second)).To(BeFalse())

					// Current view: 4 - 11
					Expect(i.Next(7 * telem.Second)).To(BeTrue())
					Expect(i.Value().Len()).To(Equal(int64(2)))
					Expect(i.Value().Series[0].Data).To(EqualUnmarshal([]uint16{17, 19}))
					Expect(i.Value().Series[0].TimeRange).To(Equal((4 * telem.SecondTS).Range(9*telem.SecondTS + 1)))
					Expect(i.Value().Series[1].Data).To(EqualUnmarshal([]uint16{20}))
					Expect(i.Value().Series[1].TimeRange).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))

					// Current view: 11 - 20
					Expect(i.Next(9 * telem.Second)).To(BeTrue())
					Expect(i.Value().Len()).To(Equal(int64(1)))
					Expect(i.Value().Series[0].Data).To(EqualUnmarshal([]uint16{22, 25}))
					Expect(i.Value().Series[0].TimeRange).To(Equal((11 * telem.SecondTS).Range(15*telem.SecondTS + 1)))

					Expect(i.Next(10 * telem.Second)).To(BeFalse())
				})
				Specify("With bounds", func() {})
			})

			Describe("Close", func() {
				It("Should not allow operations on a closed iterator", func() {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						i = MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{key}}))
						e = core.EntityClosed("cesium.iterator")
					)
					Expect(i.Close()).To(Succeed())
					Expect(i.Valid()).To(BeFalse())
					Expect(i.SeekFirst()).To(BeFalse())
					Expect(i.Valid()).To(BeFalse())
					Expect(i.Error()).To(HaveOccurredAs(e))
					Expect(i.Close()).To(Succeed())
				})

				It("Should not allow opening an iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow opening a stream iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.NewStreamIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow reading from a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.Read(ctx, telem.TimeRangeMax, key)
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})
			})
		})
	}
})
