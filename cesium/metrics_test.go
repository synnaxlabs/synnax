// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Metrics", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		ShouldNotLeakRoutinesJustBeforeEach()
		Context("FS: "+fsName, Ordered, func() {
			var (
				db      *cesium.DB
				fs      fs.FS
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

			Describe("Metrics", func() {
				It("Should return zero metrics for an empty database", func() {
					sub := MustSucceed(fs.Sub("empty-metrics"))
					emptyDB := openDBOnFS(sub)
					defer func() { Expect(emptyDB.Close()).To(Succeed()) }()

					m := emptyDB.Metrics()
					Expect(m.DiskSize).To(Equal(telem.Size(0)))
					Expect(m.ChannelCount).To(Equal(0))
				})

				It("Should return correct channel count for unary channels", func() {
					sub := MustSucceed(fs.Sub("unary-metrics"))
					subDB := openDBOnFS(sub)
					defer func() { Expect(subDB.Close()).To(Succeed()) }()

					indexKey := GenerateChannelKey()
					dataKey := GenerateChannelKey()

					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      indexKey,
						Name:     "index",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					})).To(Succeed())

					m := subDB.Metrics()
					Expect(m.ChannelCount).To(Equal(1))

					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      dataKey,
						Name:     "data",
						Index:    indexKey,
						DataType: telem.Float64T,
					})).To(Succeed())

					m = subDB.Metrics()
					Expect(m.ChannelCount).To(Equal(2))
				})

				It("Should return correct channel count including virtual channels", func() {
					sub := MustSucceed(fs.Sub("virtual-metrics"))
					subDB := openDBOnFS(sub)
					defer func() { Expect(subDB.Close()).To(Succeed()) }()

					indexKey := GenerateChannelKey()
					virtualKey := GenerateChannelKey()

					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      indexKey,
						Name:     "index",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					})).To(Succeed())

					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      virtualKey,
						Name:     "virtual",
						Virtual:  true,
						DataType: telem.Float64T,
					})).To(Succeed())

					m := subDB.Metrics()
					Expect(m.ChannelCount).To(Equal(2))
				})

				It("Should return correct disk size after writing data", func() {
					sub := MustSucceed(fs.Sub("size-metrics"))
					subDB := openDBOnFS(sub)
					defer func() { Expect(subDB.Close()).To(Succeed()) }()

					indexKey := GenerateChannelKey()
					dataKey := GenerateChannelKey()

					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      indexKey,
						Name:     "index",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					})).To(Succeed())

					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      dataKey,
						Name:     "data",
						Index:    indexKey,
						DataType: telem.Int64T,
					})).To(Succeed())

					m := subDB.Metrics()
					Expect(m.DiskSize).To(Equal(telem.Size(0)))

					Expect(subDB.Write(ctx, 1*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{indexKey, dataKey},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5),
							telem.NewSeriesV[int64](10, 20, 30, 40, 50),
						},
					))).To(Succeed())

					m = subDB.Metrics()
					// 5 timestamps (8 bytes each) + 5 int64s (8 bytes each) = 80 bytes
					Expect(m.DiskSize).To(Equal(telem.Size(80)))
				})

				It("Should be consistent with Size() method", func() {
					sub := MustSucceed(fs.Sub("consistency-metrics"))
					subDB := openDBOnFS(sub)
					defer func() { Expect(subDB.Close()).To(Succeed()) }()

					indexKey := GenerateChannelKey()

					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      indexKey,
						Name:     "index",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					})).To(Succeed())

					Expect(subDB.Write(ctx, 1*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{indexKey},
						[]telem.Series{telem.NewSeriesSecondsTSV(1, 2, 3)},
					))).To(Succeed())

					m := subDB.Metrics()
					Expect(m.DiskSize).To(Equal(subDB.Size()))
				})
			})
		})
	}
})
