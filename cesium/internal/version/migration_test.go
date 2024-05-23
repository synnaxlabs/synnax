// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"os"
	"strconv"
)

var _ = Describe("Migration Test", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db          *cesium.DB
				fs          xfs.FS
				cleanUp     func() error
				jsonEncoder = binary.JSONEncoderDecoder{}
				index       = testutil.GenerateChannelKey()
				basic       = testutil.GenerateChannelKey()
				rate        = testutil.GenerateChannelKey()
				//virtual  = testutil.GenerateChannelKey()
				channels = []cesium.Channel{
					{Key: index, DataType: telem.TimeStampT, IsIndex: true},
					{Key: basic, DataType: telem.Int64T, Index: index},
					{Key: rate, DataType: telem.Int64T, Rate: 1 * telem.Hz},
				}
			)
			BeforeEach(func() { fs, cleanUp = makeFS() })
			AfterEach(func() { Expect(cleanUp()).To(Succeed()) })
			Specify("V1 to V2", func() {
				By("Creating a database in V2")
				db = MustSucceed(cesium.Open("", cesium.WithFS(fs)))
				Expect(db.CreateChannel(ctx, channels...)).To(Succeed())

				By("Writing data")
				Expect(db.Write(ctx, 0, cesium.NewFrame(
					[]cesium.ChannelKey{index, basic, rate},
					[]telem.Series{
						telem.NewSecondsTSV(0, 1, 2, 3, 10, 11, 12),
						telem.NewSeriesV[int64](100, 101, 102, 103, 1000, 1001, 1002),
						telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5),
					},
				))).To(Succeed())

				Expect(db.Write(ctx, 20*telem.SecondTS, cesium.NewFrame(
					[]cesium.ChannelKey{index, basic, rate},
					[]telem.Series{
						telem.NewSecondsTSV(20, 22, 24),
						telem.NewSeriesV[int64](2000, 2002, 2004),
						telem.NewSeriesV[int64](20, 21, 22, 23, 24),
					},
				))).To(Succeed())

				Expect(db.Write(ctx, 30*telem.SecondTS, cesium.NewFrame(
					[]cesium.ChannelKey{index, basic, rate},
					[]telem.Series{
						telem.NewSecondsTSV(30, 31, 32),
						telem.NewSeriesV[int64](3000, 3001, 3002),
						telem.NewSeriesV[int64](30, 31, 32, 33, 34),
					},
				))).To(Succeed())

				By("Closing the DB")
				Expect(db.Close()).To(Succeed())

				By("Changing the meta file to force the channel to load as V1")
				for _, ch := range channels {
					channelFS := MustSucceed(fs.Sub(strconv.Itoa(int(ch.Key))))
					ch.Version = 1
					encoded := MustSucceed(jsonEncoder.Encode(ctx, ch))
					w := MustSucceed(channelFS.Open("meta.json", os.O_WRONLY))
					_, err := w.Write(encoded)
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Close()).To(Succeed())
				}

				By("Re-opening the database as V1 to trigger a migration")
				db, err := cesium.Open("", cesium.WithFS(fs))
				Expect(err).ToNot(HaveOccurred())

				By("Asserting that the version got migrated, the meta file got changed, and the format is correct")
				for _, ch := range channels {
					chInDB := MustSucceed(db.RetrieveChannel(ctx, ch.Key))
					Expect(chInDB.Version).To(Equal(uint8(2)))

					var (
						channelFS = MustSucceed(fs.Sub(strconv.Itoa(int(ch.Key))))
						r         = MustSucceed(channelFS.Open("meta.json", os.O_RDONLY))
						s         = MustSucceed(r.Stat()).Size()
						buf       = make([]byte, s)
						chInMeta  cesium.Channel
					)

					_, err := r.Read(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(r.Close()).To(Succeed())

					err = jsonEncoder.Decode(ctx, buf, &chInMeta)
					Expect(err).ToNot(HaveOccurred())
					Expect(chInMeta).To(Equal(chInDB))

					Expect(MustSucceed(channelFS.Exists("tombstone.domain"))).To(BeTrue())
					r = MustSucceed(channelFS.Open("index.domain", os.O_RDONLY))
					buf = make([]byte, 4)
					_, err = r.Read(buf)
					Expect(r.Close()).To(Succeed())
					Expect(MustSucceed(channelFS.Stat("index.domain")).Size()).To(Equal(int64(telem.ByteOrder.Uint32(buf)*26 + 4)))
				}

				Expect(db.Close()).To(Succeed())
			})
		})
	}
})
