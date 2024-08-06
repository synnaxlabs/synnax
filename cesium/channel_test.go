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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"math"
	"os"
	"strconv"
)

var _ = Describe("Channel", Ordered, func() {
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
			Describe("Create", func() {
				DescribeTable("Validation", func(expected error, channels ...cesium.Channel) {
					Expect(db.CreateChannel(ctx, channels...)).To(HaveOccurredAs(expected))
				},
					Entry("ChannelKey has no datatype",
						validate.FieldError{Field: "data_type", Message: "field must be set"},
						cesium.Channel{Key: 9990, Rate: 10 * telem.Hz},
					),
					Entry("ChannelKey key already exists",
						errors.Wrap(validate.Error, "cannot create channel [Isaac]<9991> because it already exists"),
						cesium.Channel{Key: 9991, DataType: telem.Float32T, Rate: 10 * telem.Hz},
						cesium.Channel{Key: 9991, Name: "Isaac", Rate: 10 * telem.Hz, DataType: telem.Float64T},
					),
					Entry("ChannelKey IsIndex - Non Int64 Series Variant",
						validate.FieldError{Field: "data_type", Message: "index channel must be of type timestamp"},
						cesium.Channel{Key: 9992, IsIndex: true, DataType: telem.Float32T},
					),
					Entry("ChannelKey IsIndex - LocalIndex non-zero",
						validate.FieldError{Field: "index", Message: "index channel cannot be indexed by another channel"},
						cesium.Channel{Key: 9995, IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: 9996, IsIndex: true, Index: 9995, DataType: telem.TimeStampT},
					),
					Entry("ChannelKey has index - LocalIndex does not exist",
						errors.Wrapf(validate.Error, "[cesium] - index channel <%s> does not exist", "9994"),
						cesium.Channel{Key: 9997, Index: 9994, DataType: telem.Float64T},
					),
					Entry("ChannelKey has no index - fixed rate not provided",
						validate.FieldError{Field: "rate", Message: "must be positive"},
						cesium.Channel{Key: 9998, DataType: telem.Float32T},
					),
					Entry("ChannelKey has index - provided index key is not an indexed channel",
						validate.FieldError{
							Field:   "index",
							Message: "channel <9980> is not an index",
						},
						cesium.Channel{Key: 9980, DataType: telem.Float32T, Rate: 1 * telem.Hz},
						cesium.Channel{Key: 9981, Index: 9980, DataType: telem.Float32T},
					),
				)
				Describe("DB Closed", func() {
					It("Should not allow creating a channel", func() {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(sub)
						Expect(subDB.Close()).To(Succeed())
						err := subDB.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.BytesT})
						Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
					It("Should not allow retrieving channels", func() {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(sub)
						Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.BytesT})).To(Succeed())
						Expect(subDB.Close()).To(Succeed())

						_, err := subDB.RetrieveChannel(ctx, key)
						Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))
						_, err = subDB.RetrieveChannels(ctx, key)
						Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
				})
			})
			Describe("Retrieve", func() {
				var k1, k2, k3 cesium.ChannelKey
				BeforeEach(func() {
					k1, k2, k3 = GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(ctx, []cesium.Channel{
						{Key: k1, DataType: telem.TimeStampT, IsIndex: true},
						{Key: k2, DataType: telem.Uint32T, Index: k1},
						{Key: k3, DataType: telem.Int8T, Rate: 1 * telem.KHz},
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
						{Key: unaryKey, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						{Key: virtualKey, Virtual: true, DataType: telem.Int64T},
						{Key: indexKey, DataType: telem.TimeStampT, IsIndex: true},
						{Key: dataKey, DataType: telem.Int64T, Index: indexKey},
						{Key: data2Key, DataType: telem.Int64T, Index: indexKey},
						{Key: indexErrorKey, DataType: telem.TimeStampT, IsIndex: true},
						{Key: dataKey1, DataType: telem.Int64T, Index: indexErrorKey},
						{Key: errorKey1, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						{Key: errorKey2, Virtual: true, DataType: telem.Int64T},
					}
				)
				BeforeAll(func() {
					Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
				})
				It("Should rekey a unary channel into another", func() {
					By("Writing some data into the channel")
					Expect(db.WriteArray(ctx, unaryKey, 0, telem.NewSeriesV[int64](2, 3, 5, 7, 11))).To(Succeed())
					Expect(db.WriteArray(ctx, unaryKey, 5*telem.SecondTS, telem.NewSeriesV[int64](13, 17, 19, 23, 29))).To(Succeed())

					By("Re-keying the channel")
					Expect(db.RekeyChannel(unaryKey, unaryKeyNew)).To(Succeed())

					By("Asserting the old channel no longer exists")
					_, err := db.RetrieveChannel(ctx, unaryKey)
					Expect(err).To(MatchError(core.ErrChannelNotFound))
					Expect(MustSucceed(fs.Exists(channelKeyToPath(unaryKey)))).To(BeFalse())

					By("Asserting the channel can be found at the new key")
					ch := MustSucceed(db.RetrieveChannel(ctx, unaryKeyNew))
					Expect(ch.Key).To(Equal(unaryKeyNew))

					By("Asserting that reads and writes on the channel still work")
					Expect(db.WriteArray(ctx, unaryKeyNew, 10*telem.SecondTS, telem.NewSeriesV[int64](31, 37, 41, 43, 47))).To(Succeed())
					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, unaryKeyNew))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](2, 3, 5, 7, 11).Data))
					Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](13, 17, 19, 23, 29).Data))
					Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](31, 37, 41, 43, 47).Data))

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
					})

				})

				It("Should rekey a virtual channel into another", func() {
					By("Re-keying the channel")
					Expect(db.RekeyChannel(virtualKey, virtualKeyNew)).To(Succeed())

					By("Asserting the old channel no longer exists")
					_, err := db.RetrieveChannel(ctx, virtualKey)
					Expect(err).To(MatchError(core.ErrChannelNotFound))
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
					})
				})

				It("Should rekey an index channel", func() {
					By("Writing some data into the channel")
					Expect(db.Write(ctx, 2*telem.SecondTS, cesium.NewFrame(
						[]core.ChannelKey{indexKey, dataKey, data2Key},
						[]telem.Series{telem.NewSecondsTSV(2, 3, 5, 7, 11), telem.NewSeriesV[int64](2, 3, 5, 7, 11), telem.NewSeriesV[int64](20, 30, 50, 70, 110)},
					))).To(Succeed())

					By("Re-keying the channel")
					Expect(db.RekeyChannel(indexKey, indexKeyNew)).To(Succeed())

					By("Asserting the old channel no longer exists")
					_, err := db.RetrieveChannel(ctx, indexKey)
					Expect(err).To(MatchError(core.ErrChannelNotFound))
					Expect(MustSucceed(fs.Exists(channelKeyToPath(indexKey)))).To(BeFalse())

					By("Asserting the channel can be found at the new key")
					ch := MustSucceed(db.RetrieveChannel(ctx, indexKeyNew))
					Expect(ch.Key).To(Equal(indexKeyNew))

					By("Asserting that reads and writes on the channel still work")
					Expect(db.Write(ctx, 13*telem.SecondTS, cesium.NewFrame(
						[]core.ChannelKey{indexKeyNew, dataKey, data2Key},
						[]telem.Series{telem.NewSecondsTSV(13, 17, 19, 23, 29), telem.NewSeriesV[int64](13, 17, 19, 23, 29), telem.NewSeriesV[int64](130, 170, 190, 230, 290)},
					))).To(Succeed())
					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, indexKeyNew, dataKey, data2Key))
					Expect(f.Series[0].Data).To(Equal(telem.NewSecondsTSV(2, 3, 5, 7, 11).Data))
					Expect(f.Series[1].Data).To(Equal(telem.NewSecondsTSV(13, 17, 19, 23, 29).Data))

					Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](2, 3, 5, 7, 11).Data))
					Expect(f.Series[3].Data).To(Equal(telem.NewSeriesV[int64](13, 17, 19, 23, 29).Data))

					Expect(f.Series[4].Data).To(Equal(telem.NewSeriesV[int64](20, 30, 50, 70, 110).Data))
					Expect(f.Series[5].Data).To(Equal(telem.NewSeriesV[int64](130, 170, 190, 230, 290).Data))

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
					})
				})

				Describe("Rekey of channel with a writer", func() {
					Specify("Unary", func() {
						By("Opening a writer")
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{errorKey1}, ControlSubject: control.Subject{Key: "rekey writer"}}))

						By("Trying to rekey")
						Expect(db.RekeyChannel(errorKey1, errorKey1New)).To(MatchError(ContainSubstring("1 unclosed writers/iterators")))

						By("Closing writer")
						Expect(w.Close()).To(Succeed())

						By("Asserting that rekey is successful now")
						Expect(db.RekeyChannel(errorKey1, errorKey1New)).To(Succeed())

						By("Asserting the old channel no longer exists")
						_, err := db.RetrieveChannel(ctx, errorKey1)
						Expect(err).To(MatchError(core.ErrChannelNotFound))
						Expect(MustSucceed(fs.Exists(channelKeyToPath(errorKey1)))).To(BeFalse())

						By("Asserting the channel can be found at the new key")
						ch := MustSucceed(db.RetrieveChannel(ctx, errorKey1New))
						Expect(ch.Key).To(Equal(errorKey1New))

						By("Asserting that reads and writes on the channel still work")
						Expect(db.WriteArray(ctx, errorKey1New, 10*telem.SecondTS, telem.NewSeriesV[int64](31, 37, 41, 43, 47))).To(Succeed())
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, errorKey1New))
						Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](31, 37, 41, 43, 47).Data))

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
						})
					})

					Specify("Virtual", func() {
						By("Opening writers")
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{errorKey2}, ControlSubject: control.Subject{Key: "rekey writer"}}))

						By("Trying to rekey")
						Expect(db.RekeyChannel(errorKey2, errorKey2New)).To(MatchError(ContainSubstring("1 unclosed writers")))

						By("Closing writer")
						Expect(w.Close()).To(Succeed())

						By("Asserting that rekey is successful now")
						Expect(db.RekeyChannel(errorKey2, errorKey2New)).To(Succeed())

						By("Asserting the old channel no longer exists")
						_, err := db.RetrieveChannel(ctx, errorKey2)
						Expect(err).To(MatchError(core.ErrChannelNotFound))
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
						})
					})
				})

				It("Should do nothing for a channel that does not exist", func() {
					By("Trying to rekey")
					Expect(db.RekeyChannel(errorKey3, errorKey3New)).To(Succeed())
				})
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

					_, err = cesium.Open("", cesium.WithFS(s), cesium.WithInstrumentation(PanicLogger()))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("field must be set"))
				})

				It("Should not error when db gets created with proper numeric folders", func() {
					s := MustSucceed(fs.Sub("sub0"))
					db := openDBOnFS(s)
					key := GenerateChannelKey()

					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
					Expect(db.Close()).To(Succeed())

					db = openDBOnFS(s)
					ch, err := db.RetrieveChannel(ctx, key)
					Expect(err).ToNot(HaveOccurred())

					Expect(ch.Key).To(Equal(key))
					Expect(ch.Rate).To(Equal(1 * telem.Hz))

					Expect(db.Write(ctx, 1*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{key},
						[]telem.Series{telem.NewSeriesV[int64](1, 2, 3, 4, 5)},
					))).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](1, 2, 3, 4, 5).Data))
				})

				It("Should not error when db is opened on existing directory", func() {
					s := MustSucceed(fs.Sub("sub3"))
					db := openDBOnFS(s)
					indexKey := GenerateChannelKey()
					key := GenerateChannelKey()

					By("Opening two channels")
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: indexKey, IsIndex: true, DataType: telem.TimeStampT})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Index: indexKey, DataType: telem.Int64T})).To(Succeed())
					Expect(db.Write(ctx, 1*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{indexKey, key},
						[]telem.Series{telem.NewSecondsTSV(1, 2, 3, 4, 5), telem.NewSeriesV[int64](1, 2, 3, 4, 5)},
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
					Expect(db.Write(ctx, 11*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{key, indexKey},
						[]telem.Series{telem.NewSeriesV[int64](11, 12, 13, 14, 15), telem.NewSecondsTSV(11, 12, 13, 14, 15)},
					))).To(Succeed())

					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
					Expect(f.Series[0].TimeRange).To(Equal((1 * telem.SecondTS).Range(5*telem.SecondTS + 1)))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](1, 2, 3, 4, 5).Data))

					Expect(f.Series[1].TimeRange).To(Equal((11 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
					Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](11, 12, 13, 14, 15).Data))
				})
			})

			Describe("Rename", func() {
				It("Should rename a channel into a different name while channel is being used", func() {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Name: "fermat", Rate: 2 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{key}, EnableAutoCommit: config.True()}))
					Expect(w.Write(cesium.NewFrame([]cesium.ChannelKey{key}, []telem.Series{telem.NewSeriesV[int64](10, 11, 12, 13)}))).To(BeTrue())

					Expect(db.RenameChannel(ctx, key, "laplace")).To(Succeed())
					Expect(w.Write(cesium.NewFrame([]cesium.ChannelKey{key}, []telem.Series{telem.NewSeriesV[int64](20, 21, 22)}))).To(BeTrue())
					Expect(w.Close()).To(Succeed())

					ch := MustSucceed(db.RetrieveChannel(ctx, key))
					Expect(ch.Name).To(Equal("laplace"))
					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
					Expect(f.Series).To(HaveLen(1))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](10, 11, 12, 13, 20, 21, 22).Data))

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
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key1, Name: "fermat", Rate: 2 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key2, Name: "laplace", Rate: 2 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key3, Name: "newton", Rate: 2 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
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
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Name: "1", Rate: 2 * telem.KHz, DataType: telem.Uint32T})).To(Succeed())
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
