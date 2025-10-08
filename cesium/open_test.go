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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Open", func() {
	for fsName, makeFS := range fileSystems {
		ShouldNotLeakRoutinesJustBeforeEach()
		Context("FS: "+fsName, Ordered, func() {
			var (
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
			})
			AfterAll(func() {
				Expect(cleanUp()).To(Succeed())
			})
			Describe("Opening db on existing folder", func() {
				It("Should not panic when opening a db in a directory with already existing files", func() {
					s := MustSucceed(fs.Sub("sub"))
					MustSucceed(s.Sub("1234notnumeric"))
					f := MustSucceed(s.Open("123.txt", os.O_CREATE))
					Expect(f.Close()).To(Succeed())

					db := openDBOnFS(s)
					Expect(db.Close()).To(Succeed())
				})

				It("Should error when numeric folders do not have meta.json file", func() {
					s := MustSucceed(fs.Sub("sub"))
					_, err := s.Sub("1")
					Expect(err).ToNot(HaveOccurred())

					db, err := cesium.Open(ctx, "", cesium.WithFS(s), cesium.WithInstrumentation(PanicLogger()))
					Expect(db).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("required"))
				})

				It("Should not error when db gets created with proper numeric folders", func() {
					s := MustSucceed(fs.Sub("sub0"))
					db := openDBOnFS(s)
					key := GenerateChannelKey()

					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Edison",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					})).To(Succeed())
					Expect(db.Close()).To(Succeed())

					db = openDBOnFS(s)
					ch, err := db.RetrieveChannel(ctx, key)
					Expect(err).ToNot(HaveOccurred())

					Expect(ch.Key).To(Equal(key))
					Expect(ch.IsIndex).To(BeTrue())

					Expect(db.Write(ctx, 1*telem.SecondTS, telem.MultiFrame[cesium.ChannelKey](
						[]cesium.ChannelKey{key},
						[]telem.Series{telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)},
					))).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
					Expect(f.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)))
					Expect(db.Close()).To(Succeed())
				})

				It("Should not error when db is opened on existing directory", func() {
					s := MustSucceed(fs.Sub("sub3"))
					db := openDBOnFS(s)
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
					Expect(db.Write(ctx, 1*telem.SecondTS, telem.MultiFrame[cesium.ChannelKey](
						[]cesium.ChannelKey{indexKey, key},
						[]telem.Series{telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5), telem.NewSeriesV[int64](1, 2, 3, 4, 5)},
					))).To(Succeed())

					By("Closing the db")
					Expect(db.Close()).To(Succeed())

					By("Reopening the db on the file system with existing data")
					db = openDBOnFS(s)
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
					Expect(db.Write(ctx, 11*telem.SecondTS, telem.MultiFrame[cesium.ChannelKey](
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
