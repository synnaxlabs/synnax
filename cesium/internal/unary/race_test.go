// Copyright 2025 Synnax Labs, Inc.
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
				fs                xfs.FS
				cleanUp           func() error
				indexKey, dataKey core.ChannelKey
				indexDB           *unary.DB
				dataDB            *unary.DB
			)
			BeforeEach(func() {
				indexKey = GenerateChannelKey()
				dataKey = GenerateChannelKey()
				fs, cleanUp = makeFS()
				indexFS, dataFS := MustSucceed(fs.Sub("index")), MustSucceed(fs.Sub("data"))
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
				Expect(indexDB.Close()).To(Succeed())
				Expect(dataDB.Close()).To(Succeed())
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
			})
		})
	}
})
