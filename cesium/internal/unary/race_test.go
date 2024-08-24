// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"sync"
)

var _ = Describe("Unary racing", func() {
	for fsName, makeFS := range FileSystems {
		Context("FS:"+fsName, func() {
			var (
				fs                         xfs.FS
				cleanUp                    func() error
				rateKey, indexKey, dataKey core.ChannelKey
				rateDB                     *unary.DB
				indexDB                    *unary.DB
				dataDB                     *unary.DB
			)
			BeforeEach(func() {
				rateKey = GenerateChannelKey()
				indexKey = GenerateChannelKey()
				dataKey = GenerateChannelKey()
				fs, cleanUp = makeFS()
				rateFS, indexFS, dataFS := MustSucceed(fs.Sub("rate")), MustSucceed(fs.Sub("index")), MustSucceed(fs.Sub("data"))
				rateDB = MustSucceed(unary.Open(unary.Config{
					FS:        rateFS,
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      rateKey,
						DataType: telem.Int16T,
						Rate:     2 * telem.Hz,
					},
					FileSize:        1 * telem.ByteSize,
					Instrumentation: PanicLogger(),
				}))
				indexDB = MustSucceed(unary.Open(unary.Config{
					FS:        indexFS,
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      indexKey,
						IsIndex:  true,
						DataType: telem.TimeStampT,
					},
					FileSize:        1 * telem.ByteSize,
					Instrumentation: PanicLogger(),
				}))
				dataDB = MustSucceed(unary.Open(unary.Config{
					FS:        dataFS,
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      dataKey,
						DataType: telem.Int64T,
						Index:    indexKey,
					},
					FileSize:        1 * telem.ByteSize,
					Instrumentation: PanicLogger(),
				},
				))
				dataDB.SetIndex(indexDB.Index())
			})
			AfterEach(func() {
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Multiple deletes", func() {
				Specify("Overlapping regions – index", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 15, 16, 18, 19, 20, 21, 22, 24))).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13, 15, 16, 18, 19, 20, 21, 22, 24))).To(Succeed())

					var wg sync.WaitGroup
					wg.Add(4)

					for i := 0; i < 4; i++ {
						i := i
						go func() {
							defer GinkgoRecover()
							defer wg.Done()
							Expect(dataDB.Delete(ctx, (telem.TimeStamp(11+i) * telem.SecondTS).Range(telem.TimeStamp(12+i)*telem.SecondTS))).To(Succeed())
							Expect(dataDB.Delete(ctx, (telem.TimeStamp(16+i) * telem.SecondTS).Range(telem.TimeStamp(17+i)*telem.SecondTS))).To(Succeed())
						}()
					}

					wg.Wait()
					Expect(dataDB.GarbageCollect(ctx)).To(Succeed())
					// remaining: 10, 15, 20, 21, 22, 24

					f := MustSucceed(dataDB.Read(ctx, telem.TimeRangeMax))
					Expect(f.Series).To(HaveLen(3))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](10).Data))
					Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))
					Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](15).Data))
					Expect(f.Series[1].TimeRange).To(Equal((15 * telem.SecondTS).Range(16 * telem.SecondTS)))
					Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](20, 21, 22, 24).Data))
					Expect(f.Series[2].TimeRange).To(Equal((20 * telem.SecondTS).Range(24*telem.SecondTS + 1)))
				})
				Specify("Overlapping regions – rate", func() {
					Expect(unary.Write(ctx, rateDB, 10*telem.SecondTS, telem.NewSeriesV[int16](100, 105, 110, 115, 120, 125, 130, 135, 140, 145, 150, 155, 160, 165))).To(Succeed())

					var wg sync.WaitGroup
					wg.Add(5)

					for i := 0; i < 5; i++ {
						i := i
						go func() {
							defer GinkgoRecover()
							defer wg.Done()
							Expect(rateDB.Delete(ctx, (telem.TimeStamp(11+i) * telem.SecondTS).Range(telem.TimeStamp(12+i)*telem.SecondTS))).To(Succeed())
						}()
					}

					wg.Wait()
					Expect(dataDB.GarbageCollect(ctx)).To(Succeed())
					// remaining: 10, 16, 16.5

					f := MustSucceed(rateDB.Read(ctx, telem.TimeRangeMax))
					Expect(f.Series).To(HaveLen(2))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int16](100, 105).Data))
					Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))
					Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int16](160, 165).Data))
					Expect(f.Series[1].TimeRange).To(Equal((16 * telem.SecondTS).Range(16*telem.SecondTS + 500*telem.MillisecondTS + 1)))
				})
			})
		})
	}
})
