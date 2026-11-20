// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer control moved from Cesium", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var db *testDB
			BeforeEach(func(ctx SpecContext) { db = mustOpenDBOnFS(ctx, openFS()) })

			It(
				"Should leave the samples of another writer uncommitted when a commit skips them",
				func(ctx SpecContext) {
					ix, x, iy, y := GenerateChannelKey(), GenerateChannelKey(),
						GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      ix,
							Name:     "ix",
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Key:      x,
							Name:     "x",
							Index:    ix,
							DataType: telem.Int64T,
						},
						cesium.Channel{
							Key:      iy,
							Name:     "iy",
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Key:      y,
							Name:     "y",
							Index:    iy,
							DataType: telem.Int64T,
						},
					)).To(Succeed())
					keys := []cesium.ChannelKey{ix, x, iy, y}
					fr := func(values ...int64) cesium.Frame {
						stamps := make([]telem.TimeStamp, len(values))
						for i, v := range values {
							stamps[i] = telem.TimeStamp(v) * telem.SecondTS
						}
						return telem.MultiFrame(keys, []telem.Series{
							telem.NewSeries(stamps),
							telem.NewSeries(values),
							telem.NewSeries(stamps),
							telem.NewSeries(values),
						})
					}
					a := MustSucceed(db.OpenWriter(ctx, storageConfig{
						ControlSubject: control.Subject{Key: "a"},
						Channels:       keys,
						Start:          10 * telem.SecondTS,
						Authorities:    []control.Authority{100},
					}))
					MustSucceed(a.Write(ctx, fr(10, 11)))
					b := MustSucceed(db.OpenWriter(ctx, storageConfig{
						ControlSubject: control.Subject{Key: "b"},
						Channels:       keys,
						Start:          10 * telem.SecondTS,
						Authorities:    []control.Authority{50, 50, 200, 200},
					}))
					expectUnauthorized(b.Write(ctx, fr(12)))
					MustSucceed(b.Commit(ctx))
					Expect(
						MustSucceed(db.Read(ctx, telem.TimeRangeMax, x)).Get(x).Len(),
					).
						To(BeZero())
					Expect(
						MustSucceed(db.Read(ctx, telem.TimeRangeMax, y)).Get(y).Len(),
					).
						To(Equal(int64(3)))
					Expect(
						a.Commit(ctx),
					).Error().
						To(MatchError(control.ErrUnauthorized))
					series := MustSucceed(
						db.Read(ctx, telem.TimeRangeMax, x),
					).Get(x).
						Series
					Expect(series).To(HaveLen(1))
					Expect(series[0]).To(telem.MatchSeriesDataV[int64](10, 11))
					Expect(b.Close()).To(Succeed())
					Expect(a.Close()).To(Succeed())
				},
			)
			It(
				"Should write nothing of a group whose index the writer does not control",
				func(ctx SpecContext) {
					idx, data := GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      idx,
							Name:     "idx",
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Key:      data,
							Name:     "data",
							Index:    idx,
							DataType: telem.Int64T,
						},
					)).To(Succeed())
					keys := []cesium.ChannelKey{idx, data}
					holder := MustSucceed(db.OpenWriter(ctx, storageConfig{
						ControlSubject: control.Subject{Key: "holder"},
						Channels:       keys,
						Start:          10 * telem.SecondTS,
						Authorities:    []control.Authority{100},
					}))
					w := MustSucceed(db.OpenWriter(ctx, storageConfig{
						ControlSubject: control.Subject{Key: "data_only"},
						Channels:       keys,
						Start:          10 * telem.SecondTS,
						Authorities:    []control.Authority{50, 200},
					}))
					expectUnauthorized(
						w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
							telem.NewSeriesSecondsTSV(10, 11),
							telem.NewSeriesV[int64](1, 2),
						})),
					)
					MustSucceed(w.Commit(ctx))
					Expect(
						MustSucceed(
							db.Read(ctx, telem.TimeRangeMax, data),
						).Get(data).
							Len(),
					).
						To(BeZero())
					Expect(w.Close()).To(Succeed())
					Expect(holder.Close()).To(Succeed())
				},
			)

			Describe("Write Authority", func() {
				DescribeTable(
					"Should reject an authority count that does not match the channels",
					func(
						ctx SpecContext,
						channels func(idx, d1, d2 cesium.ChannelKey) []cesium.ChannelKey,
						authorities []control.Authority,
					) {
						idx := GenerateChannelKey()
						d1, d2 := GenerateChannelKey(), GenerateChannelKey()
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "Byrd",
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
							cesium.Channel{
								Key:      d1,
								Name:     "Balchen",
								Index:    idx,
								DataType: telem.Int64T,
							},
							cesium.Channel{
								Key:      d2,
								Name:     "Siple",
								Index:    idx,
								DataType: telem.Int64T,
							},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, storageConfig{
							Channels: []cesium.ChannelKey{idx, d1, d2},
							Start:    10 * telem.SecondTS,
						}))
						Expect(w.SetAuthority(storageConfig{
							Channels:    channels(idx, d1, d2),
							Authorities: authorities,
						})).To(MatchError(validate.ErrValidation))
						Expect(w.Close()).To(MatchError(validate.ErrValidation))
					},
					Entry(
						"fewer authorities than channels",
						func(idx, d1, d2 cesium.ChannelKey) []cesium.ChannelKey {
							return []cesium.ChannelKey{idx, d1, d2}
						},
						[]control.Authority{10, 200},
					),
					Entry(
						"more authorities than channels",
						func(idx, d1, d2 cesium.ChannelKey) []cesium.ChannelKey {
							return []cesium.ChannelKey{idx, d1}
						},
						[]control.Authority{10, 200, 30},
					),
					Entry(
						"several authorities with no channels",
						func(idx, d1, d2 cesium.ChannelKey) []cesium.ChannelKey {
							return nil
						},
						[]control.Authority{10, 200},
					),
				)

				It("Should set the authority of writers", func(ctx SpecContext) {
					var (
						key  = GenerateChannelKey()
						key2 = GenerateChannelKey()
					)
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      key,
							Name:     "John",
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Key:      key2,
							Name:     "Krakauer",
							Virtual:  true,
							DataType: telem.StringT,
						},
					)).To(Succeed())
					w1 := MustSucceed(db.OpenWriter(ctx, storageConfig{
						Channels: []cesium.ChannelKey{key, key2},
						Start:    10 * telem.SecondTS,
						Authorities: []control.Authority{
							control.Authority(100),
							control.Authority(110),
						},
					}))

					w2 := MustSucceed(db.OpenWriter(ctx, storageConfig{
						Channels: []cesium.ChannelKey{key, key2},
						Start:    10 * telem.SecondTS,
						Authorities: []control.Authority{
							control.Authority(110),
							control.Authority(100),
						},
					}))

					expectUnauthorized(w1.Write(ctx, telem.MultiFrame(
						[]cesium.ChannelKey{key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12, 13),
						},
					)))

					Expect(w1.SetAuthority(storageConfig{
						Channels: []cesium.ChannelKey{key, key2},
						Authorities: []control.Authority{
							control.AuthorityAbsolute,
							control.Authority(0),
						},
					})).To(Succeed())

					expectUnauthorized(w2.Write(ctx, telem.MultiFrame(
						[]cesium.ChannelKey{key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12, 13),
						},
					)))

					MustSucceed(w1.Write(ctx, telem.MultiFrame(
						[]cesium.ChannelKey{key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12, 13),
						},
					)))

					expectUnauthorized(w1.Write(ctx, telem.MultiFrame(
						[]cesium.ChannelKey{key2},
						[]telem.Series{
							telem.NewSeriesV("hehe"),
						},
					)))
					Expect(w1.Close()).To(Succeed())
					Expect(w2.Close()).To(Succeed())
				})

				It(
					"Should serialize index writes and commits across a control transfer",
					func(ctx SpecContext) {
						key := GenerateChannelKey()
						Expect(db.CreateChannel(ctx, cesium.Channel{
							Key:      key,
							Name:     "Shackleton",
							IsIndex:  true,
							DataType: telem.TimestampT,
						})).To(Succeed())
						w1 := MustSucceed(db.OpenWriter(ctx, storageConfig{
							Channels:    []cesium.ChannelKey{key},
							Start:       10 * telem.SecondTS,
							Authorities: []control.Authority{100},
						}))
						started := make(chan struct{})
						stopped := make(chan struct{})
						go func() {
							defer GinkgoRecover()
							defer close(stopped)
							for i := range telem.TimeStamp(1_000_000) {
								if i == 10 {
									close(started)
								}
								if _, err := w1.Write(
									ctx,
									telem.UnaryFrame(
										key,
										telem.NewSeriesV(10*telem.SecondTS+i),
									),
								); err != nil {
									Expect(
										err,
									).To(MatchError(control.ErrUnauthorized))
									return
								}
								if _, err := w1.Commit(ctx); err != nil {
									Expect(
										err,
									).To(MatchError(control.ErrUnauthorized))
									return
								}
							}
						}()
						Eventually(started).Should(BeClosed())
						w2 := MustSucceed(db.OpenWriter(ctx, storageConfig{
							Channels:    []cesium.ChannelKey{key},
							Start:       20 * telem.SecondTS,
							Authorities: []control.Authority{200},
						}))
						for i := range telem.TimeStamp(10) {
							MustSucceed(w2.Write(ctx, telem.UnaryFrame(
								key,
								telem.NewSeriesV(20*telem.SecondTS+i),
							)))
							MustSucceed(w2.Commit(ctx))
						}
						Eventually(stopped).Should(BeClosed())
						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
					},
				)

				It(
					"Should write below its start in a claim that another writer has written",
					func(ctx SpecContext) {
						key := GenerateChannelKey()
						Expect(db.CreateChannel(ctx, cesium.Channel{
							Key:      key,
							Name:     "Amundsen",
							IsIndex:  true,
							DataType: telem.TimestampT,
						})).To(Succeed())
						w1 := MustSucceed(db.OpenWriter(ctx, storageConfig{
							Channels: []cesium.ChannelKey{key},
							Start:    10 * telem.SecondTS,
							Authorities: []control.Authority{
								control.AuthorityAbsolute - 1,
							},
						}))
						MustSucceed(w1.Write(ctx, telem.UnaryFrame(
							key,
							telem.NewSeriesSecondsTSV(10, 11),
						)))
						w2 := MustSucceed(db.OpenWriter(ctx, storageConfig{
							Channels:    []cesium.ChannelKey{key},
							Start:       20 * telem.SecondTS,
							Authorities: []control.Authority{control.AuthorityAbsolute},
						}))
						MustSucceed(w2.Write(ctx, telem.UnaryFrame(
							key,
							telem.NewSeriesSecondsTSV(12, 13),
						)))
						Expect(
							MustSucceed(w2.Commit(ctx)),
						).To(Equal(13*telem.SecondTS + 1))
						Expect(w2.Close()).To(Succeed())
						Expect(w1.Close()).To(Succeed())
						series := MustSucceed(
							db.Read(ctx, telem.TimeRangeMax, key),
						).Get(key).
							Series
						Expect(series).To(HaveLen(1))
						Expect(series[0]).To(telem.MatchSeriesDataV[telem.TimeStamp](
							10*telem.SecondTS,
							11*telem.SecondTS,
							12*telem.SecondTS,
							13*telem.SecondTS,
						))
					},
				)

				It("Should finish writes while control moves between writers", func(
					ctx SpecContext,
				) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Scott",
						IsIndex:  true,
						DataType: telem.TimestampT,
					})).To(Succeed())
					low := MustSucceed(db.OpenWriter(ctx, storageConfig{
						Channels:    []cesium.ChannelKey{key},
						Start:       10 * telem.SecondTS,
						Authorities: []control.Authority{100},
					}))
					high := MustSucceed(db.OpenWriter(ctx, storageConfig{
						Channels:    []cesium.ChannelKey{key},
						Start:       10 * telem.SecondTS,
						Authorities: []control.Authority{50},
					}))
					done := make(chan struct{})
					go func() {
						defer GinkgoRecover()
						defer close(done)
						for i := range telem.TimeStamp(200) {
							if _, err := low.Write(ctx, telem.UnaryFrame(
								key,
								telem.NewSeriesV(10*telem.SecondTS+i),
							)); err != nil {
								Expect(err).To(MatchError(control.ErrUnauthorized))
							}
						}
					}()
					for i := range 200 {
						auth := control.Authority(50)
						if i%2 == 0 {
							auth = 200
						}
						Expect(high.SetAuthority(storageConfig{
							Authorities: []control.Authority{auth},
						})).To(Succeed())
					}
					Eventually(done).Should(BeClosed())
					Expect(high.Close()).To(Succeed())
					Expect(low.Close()).To(Succeed())
				})
			})

			Describe("Error On ErrUnauthorized Open", func() {
				var (
					key cesium.ChannelKey
					w2  *storageWriter
				)
				BeforeEach(func(ctx SpecContext) {
					key = GenerateChannelKey()
					Expect(
						db.CreateChannel(
							ctx,
							cesium.Channel{
								Name:     "We",
								Key:      key,
								DataType: telem.TimestampT,
								IsIndex:  true,
							},
						),
					).To(Succeed())
					MustOpen(
						db.OpenWriter(
							ctx,
							storageConfig{
								Channels: []cesium.ChannelKey{key},
								Start:    1 * telem.SecondTS,
							},
						),
					)
				})
				Context("False", func() {
					It(
						"Should refuse writes and stay open if writer is not authorized to write",
						func(ctx SpecContext) {
							w2 = MustSucceed(
								db.OpenWriter(
									ctx,
									storageConfig{
										Channels: []cesium.ChannelKey{key},
										Start:    1 * telem.SecondTS,
									},
								),
							)
							Consistently(func() error {
								_, err := w2.Write(ctx, telem.MultiFrame(
									[]cesium.ChannelKey{key},
									[]telem.Series{telem.NewSeriesV[int64](1, 2, 3, 4)},
								))
								return err
							}).Should(MatchError(control.ErrUnauthorized))
							Expect(w2.Close()).To(Succeed())
						},
					)
				})
				Context("True", func() {
					It(
						"Should return an error if writer is not authorized to write",
						func(ctx SpecContext) {
							updates, disconnect := observeControl(db)
							defer disconnect()
							w2, err := db.OpenWriter(
								ctx,
								storageConfig{
									Channels:          []cesium.ChannelKey{key},
									Start:             1 * telem.SecondTS,
									ErrOnUnauthorized: new(true),
								},
							)
							Expect(err).To(MatchError(control.ErrUnauthorized))
							Expect(w2).To(BeNil())
							Consistently(updates).ShouldNot(Receive())
						},
					)
				})
			})

			Describe("End domain", func() {
				var (
					idx, data cesium.ChannelKey
					keys      []cesium.ChannelKey
				)
				createPair := func(
					ctx SpecContext,
					dataType telem.DataType,
				) (cesium.ChannelKey, cesium.ChannelKey) {
					GinkgoHelper()
					i, d := GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      i,
							Name:     "End domain index",
							IsIndex:  true,
							DataType: telem.TimestampT,
						},
						cesium.Channel{
							Key:      d,
							Name:     "End domain data",
							Index:    i,
							DataType: dataType,
						},
					)).To(Succeed())
					return i, d
				}
				BeforeEach(func(ctx SpecContext) {
					idx, data = createPair(ctx, telem.Int64T)
					keys = []cesium.ChannelKey{idx, data}
				})
				frameOf := func(
					i, d cesium.ChannelKey,
					seconds ...telem.TimeStamp,
				) telem.Frame[cesium.ChannelKey] {
					stamps := make([]telem.TimeStamp, len(seconds))
					values := make([]int64, len(seconds))
					for j, s := range seconds {
						stamps[j] = s * telem.SecondTS
						values[j] = int64(s)
					}
					return telem.MultiFrame(
						[]cesium.ChannelKey{i, d},
						[]telem.Series{
							telem.NewSeries(stamps),
							telem.NewSeries(values),
						},
					)
				}
				frame := func(seconds ...telem.TimeStamp) telem.Frame[cesium.ChannelKey] {
					return frameOf(idx, data, seconds...)
				}
				config := func(
					key string,
					channels []cesium.ChannelKey,
					start telem.TimeStamp,
					authority control.Authority,
				) storageConfig {
					return storageConfig{
						ControlSubject: control.Subject{Key: key, Name: key},
						Channels:       channels,
						Start:          start,
						Authorities:    []control.Authority{authority},
					}
				}
				openLive := func(ctx SpecContext) *storageWriter {
					GinkgoHelper()
					return MustSucceed(db.OpenWriter(
						ctx,
						config("live", keys, 10*telem.SecondTS, 1),
					))
				}
				read := func(ctx SpecContext, key cesium.ChannelKey) telem.MultiSeries {
					GinkgoHelper()
					return MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key)
				}

				It(
					"Should keep an index while another writer holds its data channel",
					func(ctx SpecContext) {
						timestamps := MustSucceed(db.OpenWriter(ctx, config(
							"timestamps",
							[]cesium.ChannelKey{idx},
							1*telem.SecondTS,
							1,
						)))
						values := MustSucceed(db.OpenWriter(ctx, config(
							"values",
							[]cesium.ChannelKey{data},
							1*telem.SecondTS,
							1,
						)))
						MustSucceed(timestamps.Write(ctx, telem.UnaryFrame(
							idx,
							telem.NewSeriesSecondsTSV(1, 2, 3),
						)))
						MustSucceed(timestamps.Commit(ctx))
						MustSucceed(values.Write(ctx, telem.UnaryFrame(
							data,
							telem.NewSeriesV[int64](1, 2, 3),
						)))
						MustSucceed(values.Commit(ctx))
						Expect(
							timestamps.EndDomain(ctx, idx),
						).To(MatchError(control.ErrUnauthorized))
						MustSucceed(timestamps.Write(ctx, telem.UnaryFrame(
							idx,
							telem.NewSeriesSecondsTSV(10, 11),
						)))
						MustSucceed(timestamps.Commit(ctx))
						MustSucceed(values.Write(ctx, telem.UnaryFrame(
							data,
							telem.NewSeriesV[int64](4, 5),
						)))
						MustSucceed(values.Commit(ctx))
						Expect(values.Close()).To(Succeed())
						Expect(timestamps.Close()).To(Succeed())
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
						Expect(f.Get(data).Series).To(HaveLen(1))
						Expect(f.Get(data).Series[0]).
							To(telem.MatchSeriesDataV[int64](1, 2, 3, 4, 5))
					},
				)
				It(
					"Should end no domain while another writer holds a sibling channel",
					func(ctx SpecContext) {
						sibling := GenerateChannelKey()
						Expect(db.CreateChannel(ctx, cesium.Channel{
							Key:      sibling,
							Name:     "End domain sibling",
							Index:    idx,
							DataType: telem.Int64T,
						})).To(Succeed())
						live := openLive(ctx)
						other := MustSucceed(db.OpenWriter(ctx, config(
							"other",
							[]cesium.ChannelKey{sibling},
							1*telem.SecondTS,
							1,
						)))
						MustSucceed(live.Write(ctx, frame(1, 2)))
						MustSucceed(live.Commit(ctx))
						Expect(
							live.EndDomain(ctx, idx),
						).To(MatchError(control.ErrUnauthorized))
						MustSucceed(live.Write(ctx, frame(10)))
						MustSucceed(live.Commit(ctx))
						Expect(other.Close()).To(Succeed())
						Expect(live.Close()).To(Succeed())
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
						Expect(f.Get(data).Series).To(HaveLen(1))
					},
				)
				It(
					"Should leave the domain of another writer alone",
					func(ctx SpecContext) {
						low := openLive(ctx)
						high := MustSucceed(db.OpenWriter(
							ctx,
							config("hi", keys, 10*telem.SecondTS, 2),
						))
						MustSucceed(high.Write(ctx, frame(10, 11)))
						MustSucceed(high.Commit(ctx))
						Expect(
							low.EndDomain(ctx, idx),
						).To(MatchError(control.ErrUnauthorized))
						MustSucceed(high.Write(ctx, frame(12)))
						MustSucceed(high.Commit(ctx))
						Expect(high.Close()).To(Succeed())
						Expect(low.Close()).To(Succeed())
						Expect(read(ctx, data).Series).To(HaveLen(1))
					},
				)
				It(
					"Should keep an index domain when it lost a data channel",
					func(ctx SpecContext) {
						first, second := GenerateChannelKey(), GenerateChannelKey()
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      first,
								Name:     "End domain first",
								Index:    idx,
								DataType: telem.Int64T,
							},
							cesium.Channel{
								Key:      second,
								Name:     "End domain second",
								Index:    idx,
								DataType: telem.Int64T,
							},
						)).To(Succeed())
						all := []cesium.ChannelKey{idx, first, data, second}
						values := telem.NewSeriesV[int64](1)
						frameAt := func(at telem.TimeStamp) telem.Frame[cesium.ChannelKey] {
							return telem.MultiFrame(all, []telem.Series{
								telem.NewSeriesSecondsTSV(at),
								values,
								values,
								values,
							})
						}
						live := MustSucceed(db.OpenWriter(
							ctx,
							config("live", all, 10*telem.SecondTS, 1),
						))
						MustSucceed(live.Write(ctx, frameAt(10)))
						MustSucceed(live.Commit(ctx))
						hi := MustSucceed(db.OpenWriter(ctx, config(
							"hi",
							[]cesium.ChannelKey{data},
							10*telem.SecondTS,
							2,
						)))
						Expect(
							live.EndDomain(ctx, idx),
						).To(MatchError(control.ErrUnauthorized))
						Expect(hi.Close()).To(Succeed())
						MustSucceed(live.Write(ctx, frameAt(20)))
						MustSucceed(live.Commit(ctx))
						Expect(live.Close()).To(Succeed())
						Expect(read(ctx, idx).Series).To(HaveLen(1))
						Expect(read(ctx, data).Series).To(HaveLen(1))
						after := (20 * telem.SecondTS).SpanRange(telem.Second)
						for _, key := range all {
							Expect(read(ctx, key).Len()).To(Equal(int64(2)))
							Expect(
								MustSucceed(db.Read(ctx, after, key)).Get(key).Len(),
							).
								To(Equal(int64(1)))
						}
					},
				)
				It(
					"Should end the indexes it controls when it lost another",
					func(ctx SpecContext) {
						idx2, data2 := createPair(ctx, telem.Int64T)
						idx3, data3 := createPair(ctx, telem.Int64T)
						all := []cesium.ChannelKey{idx, data, idx2, data2, idx3, data3}
						frames := func(seconds telem.TimeStamp) telem.Frame[cesium.ChannelKey] {
							return frame(seconds).Extend(
								frameOf(idx2, data2, seconds),
								frameOf(idx3, data3, seconds),
							)
						}
						live := MustSucceed(db.OpenWriter(
							ctx,
							config("live", all, 10*telem.SecondTS, 1),
						))
						MustSucceed(live.Write(ctx, frames(10)))
						MustSucceed(live.Commit(ctx))
						hi := MustSucceed(db.OpenWriter(
							ctx,
							config("hi", keys, 10*telem.SecondTS, 2),
						))
						Expect(
							live.EndDomain(ctx, idx, idx2, idx3),
						).To(MatchError(control.ErrUnauthorized))
						Expect(hi.Close()).To(Succeed())
						MustSucceed(live.Write(ctx, frames(20)))
						MustSucceed(live.Commit(ctx))
						Expect(live.Close()).To(Succeed())
						Expect(read(ctx, idx).Series).To(HaveLen(1))
						Expect(read(ctx, idx2).Series).To(HaveLen(2))
						Expect(read(ctx, idx3).Series).To(HaveLen(2))
					},
				)
				It(
					"Should start a new domain for a writer that takes control back",
					func(ctx SpecContext) {
						low := openLive(ctx)
						MustSucceed(low.Write(ctx, frame(10, 11)))
						MustSucceed(low.Commit(ctx))
						high := MustSucceed(db.OpenWriter(
							ctx,
							config("hi", keys, 10*telem.SecondTS, 2),
						))
						MustSucceed(high.Write(ctx, frame(12)))
						Expect(high.EndDomain(ctx, idx)).To(Succeed())
						Expect(high.Close()).To(Succeed())
						MustSucceed(low.Write(ctx, frame(20, 21)))
						MustSucceed(low.Commit(ctx))
						Expect(low.Close()).To(Succeed())
						series := read(ctx, data).Series
						Expect(series).To(HaveLen(2))
						Expect(series[0]).To(telem.MatchSeriesDataV[int64](10, 11, 12))
						Expect(series[1]).To(telem.MatchSeriesDataV[int64](20, 21))
					},
				)
			})
		})
	}
})
