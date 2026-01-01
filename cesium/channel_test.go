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
	"math"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/resource"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel", Ordered, func() {
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

			Describe("Create", func() {
				DescribeTable("Validation", func(substring string, channels ...cesium.Channel) {
					Expect(db.CreateChannel(ctx, channels...)).To(MatchError(ContainSubstring(substring)))
				},
					Entry("ChannelKey has no datatype",
						"data_type: required",
						cesium.Channel{Name: "cat", Key: 9990, IsIndex: true},
						cesium.Channel{Name: "dog", Key: 9991, Index: 9990},
					),
					Entry("ChannelKey key already exists",
						"cannot create channel [Isaac]<9992> because it already exists",
						cesium.Channel{Name: "Bob", Key: 9992, DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Key: 9992, Name: "Isaac", DataType: telem.TimeStampT, IsIndex: true},
					),
					Entry("ChannelKey IsIndex - Non Int64 Series Variant",
						"data_type: index channel must be of type timestamp",
						cesium.Channel{Name: "Richard", Key: 9993, IsIndex: true, DataType: telem.Float32T},
					),
					Entry("ChannelKey IsIndex - LocalIndex non-zero",
						"index: index channel cannot be indexed by another channel",
						cesium.Channel{Name: "Feynman", Key: 9995, IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Name: "Cavendish", Key: 9996, IsIndex: true, Index: 9995, DataType: telem.TimeStampT},
					),
					Entry("ChannelKey has index - LocalIndex does not exist",
						"index: index channel with key 9994 does not exist",
						cesium.Channel{Name: "Laplatz", Key: 9997, Index: 9994, DataType: telem.Float64T},
					),
					Entry("ChannelKey has no index",
						"index: non-indexed channel must have an index",
						cesium.Channel{Name: "Steinbeck", Key: 9998, DataType: telem.Float32T},
					),
					Entry("ChannelKey has index - provided index key is not an indexed channel",
						"index: channel [Sarah]<9981> is not an index",
						cesium.Channel{Name: "Hemingway", Key: 9980, DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Name: "Sarah", Key: 9981, DataType: telem.Float64T, Index: 9980},
						cesium.Channel{Name: "Kathy", Key: 9982, Index: 9981, DataType: telem.Float32T},
					),
				)
				Describe("DB Closed", func() {
					It("Should not allow creating a channel", func() {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(sub)
						Expect(subDB.Close()).To(Succeed())
						err := subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.TimeStampT, IsIndex: true})
						Expect(err).To(HaveOccurredAs(resource.NewErrClosed("cesium.db")))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
					It("Should not allow retrieving channels", func() {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(sub)
						Expect(subDB.CreateChannel(ctx, cesium.Channel{
							Key:      key,
							Name:     "Lebron",
							IsIndex:  true,
							DataType: telem.TimeStampT,
						})).To(Succeed())
						Expect(subDB.Close()).To(Succeed())

						_, err := subDB.RetrieveChannel(ctx, key)
						Expect(err).To(HaveOccurredAs(resource.NewErrClosed("cesium.db")))
						_, err = subDB.RetrieveChannels(ctx, key)
						Expect(err).To(HaveOccurredAs(resource.NewErrClosed("cesium.db")))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
				})
			})

			Describe("Retrieve", func() {
				var k1, k2, k3 cesium.ChannelKey
				BeforeEach(func() {
					k1, k2, k3 = GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(ctx, []cesium.Channel{
						{Name: "Christian", Key: k1, DataType: telem.TimeStampT, IsIndex: true},
						{Name: "Ben", Key: k2, DataType: telem.Uint32T, Index: k1},
						{Name: "Bohmer", Key: k3, DataType: telem.Int8T, Index: k1},
					}...)).To(Succeed())
				})
				It("Should retrieve multiple channels", func() {
					chs := MustSucceed(db.RetrieveChannels(ctx, k1, k2, k3))
					Expect(chs).To(HaveLen(3))
					Expect(chs[0].Key).To(Equal(k1))
					Expect(chs[1].Key).To(Equal(k2))
					Expect(chs[2].Key).To(Equal(k3))
				})
				It("Should fail if one retrieval fails", func() {
					chs, err := db.RetrieveChannels(ctx, k1, k2, math.MaxUint32)
					Expect(chs).To(HaveLen(0))
					Expect(err).To(MatchError(cesium.ErrChannelNotFound))
				})
			})

			Describe("Rekey", func() {
				var (
					unaryKey      = GenerateChannelKey()
					unaryKeyNew   = GenerateChannelKey()
					virtualKey    = GenerateChannelKey()
					virtualKeyNew = GenerateChannelKey()
					indexKey      = GenerateChannelKey()
					indexKeyNew   = GenerateChannelKey()
					dataKey       = GenerateChannelKey()
					data2Key      = GenerateChannelKey()
					indexErrorKey = GenerateChannelKey()
					dataKey1      = GenerateChannelKey()
					errorKey1     = GenerateChannelKey()
					errorKey1New  = GenerateChannelKey()
					errorKey2     = GenerateChannelKey()
					errorKey2New  = GenerateChannelKey()
					errorKey3     = GenerateChannelKey()
					errorKey3New  = GenerateChannelKey()
					jsonDecoder   = &binary.JSONCodec{}

					channels = []cesium.Channel{
						{Name: "John", Key: unaryKey, DataType: telem.TimeStampT, IsIndex: true},
						{Name: "Woodcock", Key: virtualKey, Virtual: true, DataType: telem.Int64T},
						{Name: "Alex", Key: indexKey, DataType: telem.TimeStampT, IsIndex: true},
						{Name: "Van", Key: dataKey, DataType: telem.Int64T, Index: indexKey},
						{Name: "Humboldt", Key: data2Key, DataType: telem.Int64T, Index: indexKey},
						{Name: "Napoleon", Key: indexErrorKey, DataType: telem.TimeStampT, IsIndex: true},
						{Name: "Bonaparte", Key: dataKey1, DataType: telem.Int64T, Index: indexErrorKey},
						{Name: "Michael", Key: errorKey1, DataType: telem.TimeStampT, IsIndex: true},
						{Name: "Keaton", Key: errorKey2, Virtual: true, DataType: telem.Int64T},
					}
				)
				BeforeAll(func() {
					Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
				})
				It("Should rekey a unary channel into another", func() {
					By("Writing some data into the channel")
					series1 := telem.NewSeriesSecondsTSV(0, 1, 2, 3, 4)
					series2 := telem.NewSeriesSecondsTSV(5, 6, 7, 8, 9)
					Expect(db.WriteSeries(ctx, unaryKey, 0, series1)).To(Succeed())
					Expect(db.WriteSeries(ctx, unaryKey, 5*telem.SecondTS, series2)).To(Succeed())

					By("Re-keying the channel")
					Expect(db.RekeyChannel(ctx, unaryKey, unaryKeyNew)).To(Succeed())

					By("Asserting the old channel no longer exists")
					_, err := db.RetrieveChannel(ctx, unaryKey)
					Expect(err).To(MatchError(channel.ErrNotFound))
					Expect(MustSucceed(fs.Exists(channelKeyToPath(unaryKey)))).To(BeFalse())

					By("Asserting the channel can be found at the new key")
					ch := MustSucceed(db.RetrieveChannel(ctx, unaryKeyNew))
					Expect(ch.Key).To(Equal(unaryKeyNew))

					By("Asserting that reads and writes on the channel still work")
					series3 := telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14)
					Expect(db.WriteSeries(ctx, unaryKeyNew, 10*telem.SecondTS, series3)).To(Succeed())
					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, unaryKeyNew))
					Expect(f.SeriesAt(0)).To(telem.MatchWrittenSeries(series1))
					Expect(f.SeriesAt(1)).To(telem.MatchWrittenSeries(series2))
					Expect(f.SeriesAt(2)).To(telem.MatchWrittenSeries(series3))

					By("Asserting that the meta file got changed too", func() {
						f := MustSucceed(fs.Open(channelKeyToPath(unaryKeyNew)+"/meta.json", os.O_RDWR))
						s := MustSucceed(f.Stat()).Size()
						var (
							buf = make([]byte, s)
						)
						_, err := f.Read(buf)
						Expect(err).ToNot(HaveOccurred())
						err = jsonDecoder.Decode(ctx, buf, &ch)
						Expect(err).ToNot(HaveOccurred())

						Expect(ch.Key).To(Equal(unaryKeyNew))
						Expect(f.Close()).To(Succeed())
					})

				})

				It("Should rekey a virtual channel into another", func() {
					By("Re-keying the channel")
					Expect(db.RekeyChannel(ctx, virtualKey, virtualKeyNew)).To(Succeed())

					By("Asserting the old channel no longer exists")
					_, err := db.RetrieveChannel(ctx, virtualKey)
					Expect(err).To(MatchError(channel.ErrNotFound))
					Expect(MustSucceed(fs.Exists(channelKeyToPath(virtualKey)))).To(BeFalse())

					By("Asserting the channel and data can be found at the new key")
					ch := MustSucceed(db.RetrieveChannel(ctx, virtualKeyNew))
					Expect(ch.Key).To(Equal(virtualKeyNew))

					By("Asserting that the meta file got changed too", func() {
						f := MustSucceed(fs.Open(channelKeyToPath(virtualKeyNew)+"/meta.json", os.O_RDWR))
						s := MustSucceed(f.Stat()).Size()
						var (
							buf = make([]byte, s)
						)
						_, err := f.Read(buf)
						Expect(err).ToNot(HaveOccurred())
						err = jsonDecoder.Decode(ctx, buf, &ch)
						Expect(err).ToNot(HaveOccurred())

						Expect(ch.Key).To(Equal(virtualKeyNew))
						Expect(f.Close()).To(Succeed())
					})
				})

				It("Should rekey an index channel", func() {
					By("Writing some data into the channel")
					indexSeries1 := telem.NewSeriesSecondsTSV(2, 3, 5, 7, 11)
					dataSeries1 := telem.NewSeriesV[int64](2, 3, 5, 7, 11)
					data2Series1 := telem.NewSeriesV[int64](20, 30, 50, 70, 110)

					Expect(db.Write(ctx, 2*telem.SecondTS, telem.MultiFrame(
						[]channel.Key{indexKey, dataKey, data2Key},
						[]telem.Series{indexSeries1, dataSeries1, data2Series1},
					))).To(Succeed())

					By("Re-keying the channel")
					Expect(db.RekeyChannel(ctx, indexKey, indexKeyNew)).To(Succeed())

					By("Asserting the old channel no longer exists")
					_, err := db.RetrieveChannel(ctx, indexKey)
					Expect(err).To(MatchError(channel.ErrNotFound))
					Expect(MustSucceed(fs.Exists(channelKeyToPath(indexKey)))).To(BeFalse())

					By("Asserting the channel can be found at the new key")
					ch := MustSucceed(db.RetrieveChannel(ctx, indexKeyNew))
					Expect(ch.Key).To(Equal(indexKeyNew))

					By("Asserting that reads and writes on the channel still work")
					indexSeries2 := telem.NewSeriesSecondsTSV(13, 17, 19, 23, 29)
					dataSeries2 := telem.NewSeriesV[int64](13, 17, 19, 23, 29)
					data2Series2 := telem.NewSeriesV[int64](130, 170, 190, 230, 290)

					Expect(db.Write(ctx, 13*telem.SecondTS, telem.MultiFrame(
						[]channel.Key{indexKeyNew, dataKey, data2Key},
						[]telem.Series{indexSeries2, dataSeries2, data2Series2},
					))).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, indexKeyNew, dataKey, data2Key))
					Expect(f.SeriesAt(0)).To(telem.MatchWrittenSeries(indexSeries1))
					Expect(f.SeriesAt(1)).To(telem.MatchWrittenSeries(indexSeries2))

					Expect(f.SeriesAt(2)).To(telem.MatchWrittenSeries(dataSeries1))
					Expect(f.SeriesAt(3)).To(telem.MatchWrittenSeries(dataSeries2))

					Expect(f.SeriesAt(4)).To(telem.MatchWrittenSeries(data2Series1))
					Expect(f.SeriesAt(5)).To(telem.MatchWrittenSeries(data2Series2))

					By("Asserting that the meta file got changed too", func() {
						f := MustSucceed(fs.Open(channelKeyToPath(indexKeyNew)+"/meta.json", os.O_RDWR))
						s := MustSucceed(f.Stat()).Size()
						var (
							buf = make([]byte, s)
						)
						_, err := f.Read(buf)
						Expect(err).ToNot(HaveOccurred())
						err = jsonDecoder.Decode(ctx, buf, &ch)
						Expect(err).ToNot(HaveOccurred())

						Expect(ch.Key).To(Equal(indexKeyNew))
						Expect(f.Close()).To(Succeed())
					})
				})

				Describe("Rekey of channel with a writer", func() {
					Specify("Unary", func() {
						By("Opening a writer")
						w := MustSucceed(db.OpenWriter(
							ctx,
							cesium.WriterConfig{
								Start:          0,
								Channels:       []cesium.ChannelKey{errorKey1},
								ControlSubject: control.Subject{Key: "rekey writer"},
							}),
						)

						By("Trying to rekey")
						Expect(db.RekeyChannel(ctx, errorKey1, errorKey1New)).To(MatchError(ContainSubstring("1 unclosed writers/iterators")))

						By("Closing writer")
						Expect(w.Close()).To(Succeed())

						By("Asserting that rekey is successful now")
						Expect(db.RekeyChannel(ctx, errorKey1, errorKey1New)).To(Succeed())

						By("Asserting the old channel no longer exists")
						_, err := db.RetrieveChannel(ctx, errorKey1)
						Expect(err).To(MatchError(channel.ErrNotFound))
						Expect(MustSucceed(fs.Exists(channelKeyToPath(errorKey1)))).To(BeFalse())

						By("Asserting the channel can be found at the new key")
						ch := MustSucceed(db.RetrieveChannel(ctx, errorKey1New))
						Expect(ch.Key).To(Equal(errorKey1New))

						By("Asserting that reads and writes on the channel still work")
						series1 := telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14)
						Expect(db.WriteSeries(ctx, errorKey1New, 10*telem.SecondTS, series1)).To(Succeed())
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, errorKey1New))
						Expect(f.SeriesAt(0)).To(telem.MatchWrittenSeries(series1))

						By("Asserting that the meta file got changed too", func() {
							f := MustSucceed(fs.Open(channelKeyToPath(errorKey1New)+"/meta.json", os.O_RDWR))
							s := MustSucceed(f.Stat()).Size()
							var (
								buf = make([]byte, s)
							)
							_, err := f.Read(buf)
							Expect(err).ToNot(HaveOccurred())
							err = jsonDecoder.Decode(ctx, buf, &ch)
							Expect(err).ToNot(HaveOccurred())

							Expect(ch.Key).To(Equal(errorKey1New))
							Expect(f.Close()).To(Succeed())
						})
					})

					Specify("Virtual", func() {
						By("Opening writers")
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{errorKey2}, ControlSubject: control.Subject{Key: "rekey writer"}}))

						By("Trying to rekey")
						Expect(db.RekeyChannel(ctx, errorKey2, errorKey2New)).To(MatchError(ContainSubstring("1 unclosed writers")))

						By("Closing writer")
						Expect(w.Close()).To(Succeed())

						By("Asserting that rekey is successful now")
						Expect(db.RekeyChannel(ctx, errorKey2, errorKey2New)).To(Succeed())

						By("Asserting the old channel no longer exists")
						_, err := db.RetrieveChannel(ctx, errorKey2)
						Expect(err).To(MatchError(channel.ErrNotFound))
						Expect(MustSucceed(fs.Exists(channelKeyToPath(errorKey2)))).To(BeFalse())

						By("Asserting the channel can be found at the new key")
						ch := MustSucceed(db.RetrieveChannel(ctx, errorKey2New))
						Expect(ch.Key).To(Equal(errorKey2New))

						By("Asserting that the meta file got changed too", func() {
							f := MustSucceed(fs.Open(channelKeyToPath(errorKey2New)+"/meta.json", os.O_RDWR))
							s := MustSucceed(f.Stat()).Size()
							var (
								buf = make([]byte, s)
							)
							_, err := f.Read(buf)
							Expect(err).ToNot(HaveOccurred())
							err = jsonDecoder.Decode(ctx, buf, &ch)
							Expect(err).ToNot(HaveOccurred())

							Expect(ch.Key).To(Equal(errorKey2New))
							Expect(f.Close()).To(Succeed())
						})
					})
				})

				It("Should do nothing for a channel that does not exist", func() {
					By("Trying to rekey")
					Expect(db.RekeyChannel(ctx, errorKey3, errorKey3New)).To(Succeed())
				})
			})

			Describe("Rename", func() {
				It("Should rename a channel into a different name while channel is being used", func() {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Name: "fermat", DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{key}}))
					series1 := telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14)
					MustSucceed(w.Write(telem.MultiFrame([]cesium.ChannelKey{key}, []telem.Series{series1})))

					Expect(db.RenameChannel(ctx, key, "laplace")).To(Succeed())
					series2 := telem.NewSeriesSecondsTSV(20, 21, 22)
					MustSucceed(w.Write(telem.MultiFrame([]cesium.ChannelKey{key}, []telem.Series{series2})))
					Expect(w.Close()).To(Succeed())

					ch := MustSucceed(db.RetrieveChannel(ctx, key))
					Expect(ch.Name).To(Equal("laplace"))
					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
					Expect(f.Count()).To(Equal(1))
					Expect(f.SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 20, 21, 22)))

					var (
						subFS = MustSucceed(fs.Sub(strconv.Itoa(int(key))))
						meta  = MustSucceed(subFS.Open("meta.json", os.O_RDONLY))
						codec = &binary.JSONCodec{}
						buf   = make([]byte, MustSucceed(meta.Stat()).Size())
						newCh cesium.Channel
					)

					_, err := meta.Read(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(meta.Close()).To(Succeed())

					Expect(codec.Decode(ctx, buf, &newCh)).To(Succeed())
					Expect(newCh.Name).To(Equal("laplace"))
				})
				It("Should correctly rename multiple channels", func() {
					key1 := GenerateChannelKey()
					key2 := GenerateChannelKey()
					key3 := GenerateChannelKey()
					key4 := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key1, Name: "fermat", IsIndex: true, DataType: telem.TimeStampT})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key2, Name: "laplace", Index: key1, DataType: telem.Float32T})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key3, Name: "newton", IsIndex: true, DataType: telem.TimeStampT})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key4, Name: "descartes", Virtual: true, DataType: telem.StringT})).To(Succeed())

					Expect(db.RenameChannels(ctx,
						[]cesium.ChannelKey{key1, key2, key3, key4},
						[]string{"newton2", "fermat3", "laplace4", "descartes5"},
					)).To(Succeed())

					ch := MustSucceed(db.RetrieveChannel(ctx, key1))
					Expect(ch.Name).To(Equal("newton2"))
					ch = MustSucceed(db.RetrieveChannel(ctx, key2))
					Expect(ch.Name).To(Equal("fermat3"))
					ch = MustSucceed(db.RetrieveChannel(ctx, key3))
					Expect(ch.Name).To(Equal("laplace4"))
					ch = MustSucceed(db.RetrieveChannel(ctx, key4))
					Expect(ch.Name).To(Equal("descartes5"))
				})
				It("Should correctly rename if a channel is provided twice", func() {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Name: "1", IsIndex: true, DataType: telem.TimeStampT})).To(Succeed())
					Expect(db.RenameChannels(ctx,
						[]cesium.ChannelKey{key, key, key, key},
						[]string{"2", "3", "4", "5"},
					)).To(Succeed())

					ch := MustSucceed(db.RetrieveChannel(ctx, key))
					Expect(ch.Name).To(Equal("5"))
				})
				It("Should error if the channel is not found", func() {
					key := GenerateChannelKey()
					Expect(db.RenameChannel(ctx, key, "new_name")).To(HaveOccurredAs(cesium.ErrChannelNotFound))
				})
			})
		})
	}
})
