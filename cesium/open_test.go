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
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Open", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var fs fs.FS
			BeforeAll(func() {
				ShouldNotLeakGoroutines()
				fs = openFS()
			})
			Describe("Opening db on existing folder", func() {
				It("Should not panic when opening a db in a directory with already existing files", func(ctx SpecContext) {
					s := MustSucceed(fs.Sub("sub"))
					MustSucceed(s.Sub("1234notnumeric"))
					f := MustSucceed(s.Open("123.txt", os.O_CREATE))
					Expect(f.Close()).To(Succeed())

					db := openDBOnFS(ctx, s)
					Expect(db.Close()).To(Succeed())
				})

				It("Should error when numeric folders do not have meta.json file", func(ctx SpecContext) {
					s := MustSucceed(fs.Sub("sub"))
					MustSucceed(s.Sub("1"))

					Expect(cesium.Open(
						ctx,
						"",
						cesium.WithFS(s),
						cesium.WithInstrumentation(PanicLogger()),
					)).Error().To(MatchError(ContainSubstring("required")))
				})

				It("Should not error when db gets created with proper numeric folders", func(ctx SpecContext) {
					s := MustSucceed(fs.Sub("sub0"))
					db := openDBOnFS(ctx, s)
					key := GenerateChannelKey()

					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Edison",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					})).To(Succeed())
					Expect(db.Close()).To(Succeed())

					db = openDBOnFS(ctx, s)
					ch := MustSucceed(db.RetrieveChannel(ctx, key))

					Expect(ch.Key).To(Equal(key))
					Expect(ch.IsIndex).To(BeTrue())

					Expect(db.Write(ctx, 1*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{key},
						[]telem.Series{telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)},
					))).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
					Expect(f.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)))
					Expect(db.Close()).To(Succeed())
				})

				It("Should not error when db is opened on existing directory", func(ctx SpecContext) {
					s := MustSucceed(fs.Sub("sub3"))
					db := openDBOnFS(ctx, s)
					indexKey := GenerateChannelKey()
					key := GenerateChannelKey()

					By("Opening two channels")
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      indexKey,
						Name:     "Tesla",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Faraday",
						Index:    indexKey,
						DataType: telem.Int64T,
					})).To(Succeed())
					Expect(db.Write(ctx, 1*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{indexKey, key},
						[]telem.Series{telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5), telem.NewSeriesV[int64](1, 2, 3, 4, 5)},
					))).To(Succeed())

					By("Closing the db")
					Expect(db.Close()).To(Succeed())

					By("Reopening the db on the file system with existing data")
					db = openDBOnFS(ctx, s)
					ch := MustSucceed(db.RetrieveChannel(ctx, key))
					Expect(ch).ToNot(BeNil())
					Expect(ch.Key).To(Equal(key))
					Expect(ch.Index).To(Equal(indexKey))
					Expect(ch.DataType).To(Equal(telem.Int64T))

					ch = MustSucceed(db.RetrieveChannel(ctx, indexKey))
					Expect(ch).ToNot(BeNil())
					Expect(ch.Key).To(Equal(indexKey))
					Expect(ch.IsIndex).To(BeTrue())
					Expect(ch.DataType).To(Equal(telem.TimeStampT))

					By("Asserting that writes to the db still occurs normally")
					Expect(db.Write(ctx, 11*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{key, indexKey},
						[]telem.Series{telem.NewSeriesV[int64](11, 12, 13, 14, 15), telem.NewSeriesSecondsTSV(11, 12, 13, 14, 15)},
					))).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
					Expect(f.SeriesAt(0).TimeRange).To(Equal((1 * telem.SecondTS).Range(5*telem.SecondTS + 1)))
					Expect(f.SeriesAt(0).Data).To(Equal(telem.NewSeriesV[int64](1, 2, 3, 4, 5).Data))

					Expect(f.SeriesAt(1).TimeRange).To(Equal((11 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
					Expect(f.SeriesAt(1).Data).To(Equal(telem.NewSeriesV[int64](11, 12, 13, 14, 15).Data))

					Expect(db.Close()).To(Succeed())
				})
			})
		})
	}
})

var _ = Describe("Virtual Channels On Reopen", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var fs fs.FS
			BeforeAll(func() {
				ShouldNotLeakGoroutines()
				fs = openFS()
			})

			It("Should not survive a database reopen, while stored channels do", func(ctx SpecContext) {
				subFS := MustSucceed(fs.Sub("restart"))
				restartDB := openDBOnFS(ctx, subFS)
				virtualKey := GenerateChannelKey()
				storedKey := GenerateChannelKey()
				Expect(restartDB.CreateChannel(ctx,
					virtualChannel(virtualKey, "gone_on_restart"),
					cesium.Channel{
						Key:      storedKey,
						Name:     "kept_on_restart",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				)).To(Succeed())
				Expect(restartDB.Close()).To(Succeed())

				restartDB = openDBOnFS(ctx, subFS)
				Expect(restartDB.RetrieveChannel(ctx, virtualKey)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
				ch := MustSucceed(restartDB.RetrieveChannel(ctx, storedKey))
				Expect(ch.Name).To(Equal("kept_on_restart"))
				Expect(restartDB.Close()).To(Succeed())
			})

			It("Should remove directories persisted for virtual channels by previous versions", func(ctx SpecContext) {
				subFS := MustSucceed(fs.Sub("legacy"))
				key := GenerateChannelKey()
				chFS := MustSucceed(subFS.Sub(channelKeyToPath(key)))
				f := MustSucceed(chFS.Open("meta.json", os.O_CREATE|os.O_WRONLY))
				MustSucceed(f.Write(fmt.Appendf(nil,
					`{"key":%d,"name":"legacy_virtual","data_type":"int64","virtual":true,"version":2}`,
					key,
				)))
				Expect(f.Close()).To(Succeed())

				db := openDBOnFS(ctx, subFS)
				Expect(db.RetrieveChannel(ctx, key)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
				Expect(subFS.Exists(channelKeyToPath(key))).To(BeFalse())
				Expect(db.Close()).To(Succeed())
			})
		})
	}
})
