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
	"context"
	"encoding/json"
	"io"
	"math"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/core"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Control", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS:"+fsName, Ordered, func() {
			var (
				fs      fs.FS
				cleanUp func() error
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
			})
			AfterAll(func() {
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Nominal", func() {
				var db *cesium.DB
				BeforeAll(func() {
					db = openDBOnFS(fs)
					Expect(db.ConfigureControlUpdateChannel(ctx, math.MaxUint32, "control")).To(Succeed())
				})
				AfterAll(func() {
					Expect(db.Close()).To(Succeed())
				})

				Describe("Single Channel, Two Writer Contention", func() {
					It("Should correctly manage control authority between two writers", func() {
						var (
							indexChKey = GenerateChannelKey()
							dataChKey  = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "John", Key: indexChKey, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Sandler", Key: dataChKey, DataType: telem.Int16T, Index: indexChKey},
						)).To(Succeed())
						start := telem.SecondTS * 10
						By("Opening the first writer")
						w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							ControlSubject:    xcontrol.Subject{Name: "Writer One"},
							Start:             start,
							Channels:          []cesium.ChannelKey{indexChKey, dataChKey},
							Authorities:       []xcontrol.Authority{xcontrol.AuthorityAbsolute - 2},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))
						By("Opening the second writer")
						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:             start,
							ControlSubject:    xcontrol.Subject{Name: "Writer Two"},
							Channels:          []cesium.ChannelKey{indexChKey, dataChKey},
							Authorities:       []xcontrol.Authority{xcontrol.AuthorityAbsolute - 2},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))
						streamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
							Channels:    []cesium.ChannelKey{math.MaxUint32},
							SendOpenAck: true,
						}))
						ctx, cancel := signal.Isolated()
						defer cancel()
						stIn, stOut := confluence.Attach(streamer, 2)
						streamer.Flow(ctx)
						Eventually(stOut.Outlet()).Should(Receive())

						By("Writing to the first writer")
						Expect(MustSucceed(w1.Write(telem.MultiFrame(
							[]cesium.ChannelKey{indexChKey, dataChKey},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(10, 11, 12),
								telem.NewSeriesV[int16](1, 2, 3),
							},
						)))).To(BeTrue())

						By("Failing to write to the second writer")
						w2Frame := telem.MultiFrame(
							[]cesium.ChannelKey{indexChKey, dataChKey},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(12, 13, 14),
								telem.NewSeriesV[int16](4, 5, 6),
							},
						)

						authorized := MustSucceed(w2.Write(w2Frame))
						Expect(authorized).To(BeFalse())

						Expect(w2.SetAuthority(cesium.WriterConfig{
							Authorities: []xcontrol.Authority{xcontrol.AuthorityAbsolute - 1},
						})).To(Succeed())

						By("Propagating the control transfer")
						Eventually(stOut.Outlet()).Should(Receive())

						By("Writing to the second writer")
						authorized = MustSucceed(w2.Write(w2Frame))
						Expect(authorized).To(BeTrue())

						end := MustSucceed(w2.Commit())
						Expect(end).To(Equal(telem.SecondTS*14 + 1))

						By("Shutting down the writers")
						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
						stIn.Close()
						Expect(ctx.Wait()).To(Succeed())

						By("Reading the data")
						f := MustSucceed(db.Read(
							ctx,
							start.SpanRange(10*telem.Second),
							dataChKey,
						))
						Expect(f.Count()).To(Equal(1))
						Expect(f.SeriesAt(0).Data).To(Equal(telem.NewSeriesV[int16](1, 2, 3, 4, 5, 6).Data))
					})

					It("Should correctly hand off control authority when a writer is force closed via context cancellation", func() {
						var (
							indexChKey = GenerateChannelKey()
							dataChKey  = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "Cat", Key: indexChKey, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Mouse", Key: dataChKey, DataType: telem.Int16T, Index: indexChKey},
						)).To(Succeed())
						start := telem.SecondTS * 10

						streamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
							Channels:    []cesium.ChannelKey{math.MaxUint32},
							SendOpenAck: true,
						}))

						stIn, stOut := confluence.Attach(streamer, 2)
						ctx2, cancel2 := signal.Isolated()
						defer cancel2()
						streamer.Flow(ctx2, confluence.CloseOutputInletsOnExit())
						Eventually(stOut.Outlet()).Should(Receive())

						By("Opening the first writer")
						w1 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
							ControlSubject:    xcontrol.Subject{Name: "Writer One"},
							Start:             start,
							Channels:          []cesium.ChannelKey{indexChKey, dataChKey},
							Authorities:       []xcontrol.Authority{xcontrol.AuthorityAbsolute - 2},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))

						By("Opening the second writer")
						w2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
							Start:             start,
							ControlSubject:    xcontrol.Subject{Name: "Writer Two"},
							Channels:          []cesium.ChannelKey{indexChKey, dataChKey},
							Authorities:       []xcontrol.Authority{xcontrol.AuthorityAbsolute - 3},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))

						ctx1, cancel1 := signal.Isolated()

						w1In, _ := confluence.Attach(w1, 2)
						w2In, w2Out := confluence.Attach(w2, 2)

						w1.Flow(ctx1, confluence.CloseOutputInletsOnExit())
						w2.Flow(ctx2, confluence.CloseOutputInletsOnExit())

						runtime.Gosched()
						time.Sleep(1 * time.Millisecond)

						By("Writing to the first writer")
						w1In.Inlet() <- cesium.WriterRequest{
							Command: cesium.WriterWrite,
							Frame: telem.MultiFrame(
								[]cesium.ChannelKey{indexChKey, dataChKey},
								[]telem.Series{
									telem.NewSeriesSecondsTSV(10, 11, 12),
									telem.NewSeriesV[int16](1, 2, 3),
								}),
						}

						var res cesium.StreamerResponse
						Eventually(stOut.Outlet()).Should(Receive(&res))
						var d cesium.ControlUpdate
						Expect(json.Unmarshal(res.Frame.SeriesAt(0).Data, &d)).To(Succeed())
						Expect(d.Transfers).To(HaveLen(2))
						Expect(d.Transfers[0].To).ToNot(BeNil())
						Expect(d.Transfers[0].To.Subject.Name).To(Equal("Writer One"))

						By("Force closing the first writer through context cancellation")
						cancel1()
						Expect(ctx1.Wait()).To(HaveOccurredAs(context.Canceled))

						By("Propagating the control transfer")
						Eventually(stOut.Outlet()).Should(Receive(&res))
						Expect(json.Unmarshal(res.Frame.SeriesAt(0).Data, &d)).To(Succeed())
						Expect(d.Transfers).To(HaveLen(2))
						Expect(d.Transfers[0].To).ToNot(BeNil())
						Expect(d.Transfers[0].To.Subject.Name).To(Equal("Writer Two"))

						By("Writing to the second writer")
						w2In.Inlet() <- cesium.WriterRequest{
							Command: cesium.WriterWrite,
							Frame: telem.MultiFrame(
								[]cesium.ChannelKey{indexChKey, dataChKey},
								[]telem.Series{
									telem.NewSeriesSecondsTSV(13, 14, 15),
									telem.NewSeriesV[int16](4, 5, 6),
								},
							),
						}

						By("Committing the second writer")
						w2In.Inlet() <- cesium.WriterRequest{
							Command: cesium.WriterCommit,
						}
						var r cesium.WriterResponse
						Eventually(w2Out.Outlet()).Should(Receive(&r))

						By("Shutting down the second writer")
						w2In.Close()
						stIn.Close()
						Expect(ctx2.Wait()).To(Succeed())

						By("Reading the data")
						f := MustSucceed(db.Read(
							ctx,
							start.SpanRange(10*telem.Second),
							dataChKey,
						))
						Expect(f.Count()).To(Equal(1))
						Expect(f.SeriesAt(0).Data).To(Equal(telem.NewSeriesV[int16](1, 2, 3, 4, 5, 6).Data))
					})
				})

				Describe("Mismatched Authorization", func() {
					var (
						indexChKey, dataChKey, virtualChKey cesium.ChannelKey
						w1                                  *cesium.Writer
						w2                                  *cesium.Writer
						dataStreamerIn, controlStreamerIn   confluence.Inlet[cesium.StreamerRequest]
						dataStreamerOut, controlStreamerOut confluence.Outlet[cesium.StreamerResponse]
						shutdown                            io.Closer
					)
					BeforeEach(func() {
						indexChKey = GenerateChannelKey()
						dataChKey = GenerateChannelKey()
						virtualChKey = GenerateChannelKey()
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "persisted", Key: indexChKey, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "persisted_idx", Key: dataChKey, DataType: telem.Int16T, Index: indexChKey},
							cesium.Channel{Name: "virtual_cmd", Key: virtualChKey, DataType: telem.Uint8T, Virtual: true},
						)).To(Succeed())
						start := telem.SecondTS * 10
						By("Opening the first writer")
						w1 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							ControlSubject:    xcontrol.Subject{Name: "Writer One"},
							Start:             start,
							Channels:          []cesium.ChannelKey{virtualChKey, indexChKey, dataChKey},
							Authorities:       []xcontrol.Authority{100},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))
						By("Opening the second writer")
						w2 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:             start,
							ControlSubject:    xcontrol.Subject{Name: "Writer Two"},
							Channels:          []cesium.ChannelKey{indexChKey, dataChKey},
							Authorities:       []xcontrol.Authority{0},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))
						dataStreamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
							Channels:    []cesium.ChannelKey{virtualChKey, indexChKey, dataChKey},
							SendOpenAck: true,
						}))
						sCtx, cancel := signal.Isolated()
						shutdown = signal.NewHardShutdown(sCtx, cancel)
						dataStreamerIn, dataStreamerOut = confluence.Attach(dataStreamer, 2)
						dataStreamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
						Eventually(dataStreamerOut.Outlet()).Should(Receive())

						controlStateStreamer := MustSucceed(db.NewStreamer(sCtx, cesium.StreamerConfig{
							Channels:    []cesium.ChannelKey{math.MaxUint32},
							SendOpenAck: true,
						}))
						controlStreamerIn, controlStreamerOut = confluence.Attach(controlStateStreamer, 2)
						controlStateStreamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
						Eventually(controlStreamerOut.Outlet()).Should(Receive())
					})
					AfterEach(func() {
						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
						dataStreamerIn.Close()
						controlStreamerIn.Close()
						Eventually(dataStreamerOut.Outlet()).Should(BeClosed())
						Eventually(controlStreamerOut.Outlet()).Should(BeClosed())
						Expect(shutdown.Close()).To(Succeed())
					})

					// Set up:
					// Auto-Commit Enabled On Both
					// Writer 1: Virtual Channel, Persisted Channel, Persisted Channel Index
					// Writer 2: Persisted Channel, Persisted Channel Index
					//
					// Writer 1 starts of at authority 100 on all channels
					// Writer 1 writes - confirm successful receive
					// Writer 2 starts off at authority 0 on all channels
					// Writer 2 changes to authority 200
					// Writer 1 writes to virtual channel (no persisted) - confirm successful write
					Specify("One Virtual, One Persisted, Different Authorities, Partial Contention", func() {
						By("Writing to the first writer")
						Expect(MustSucceed(w1.Write(telem.MultiFrame(
							[]cesium.ChannelKey{virtualChKey, indexChKey, dataChKey},
							[]telem.Series{
								telem.NewSeriesV[uint8](1),
								telem.NewSeriesSecondsTSV(11),
								telem.NewSeriesV[int16](2),
							},
						)))).To(BeTrue())

						var fr cesium.StreamerResponse
						Eventually(dataStreamerOut.Outlet()).Should(Receive(&fr))

						Expect(w2.SetAuthority(cesium.WriterConfig{
							Authorities: []xcontrol.Authority{200},
						})).To(Succeed())

						By("By propagating the control transfer")
						Eventually(controlStreamerOut.Outlet()).Should(Receive())

						By("Writing to the first writer")
						Expect(MustSucceed(w1.Write(telem.MultiFrame(
							[]cesium.ChannelKey{virtualChKey},
							[]telem.Series{telem.NewSeriesV[uint8](1)},
						))))

						var r cesium.StreamerResponse
						Eventually(dataStreamerOut.Outlet()).Should(Receive(&r))

					})

					It("Should stream partially successful writes", func() {
						Expect(w2.SetAuthority(cesium.WriterConfig{
							Authorities: []xcontrol.Authority{200},
						})).To(Succeed())
						Expect(MustSucceed(w1.Write(telem.MultiFrame(
							[]cesium.ChannelKey{virtualChKey, indexChKey, dataChKey},
							[]telem.Series{
								telem.NewSeriesV[uint8](1),
								telem.NewSeriesSecondsTSV(11),
								telem.NewSeriesV[int16](2),
							},
						)))).To(BeFalse())

						var r cesium.StreamerResponse
						Eventually(dataStreamerOut.Outlet()).Should(Receive(&r))
						Expect(r.Frame.Count()).To(Equal(1))
						Expect(r.Frame.Get(virtualChKey).Series[0]).To(telem.MatchSeriesDataV[uint8](1))
					})
				})

				// Specs testing the control digest system correctly propagates control
				// changes between contending writers.
				Describe("Control digests", func() {
					It("Should propagate the control states of channels", func() {
						var k1, k2, k3 = GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
						Expect(db.CreateChannel(ctx,
							cesium.Channel{Name: "Seattle", Key: k1, Virtual: true, DataType: telem.StringT},
							cesium.Channel{Name: "Austin", Key: k2, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Ann Arbor", Key: k3, DataType: telem.Int64T, Index: k2},
						)).To(Succeed())
						w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:          0,
							Channels:       []core.ChannelKey{k1, k2},
							ControlSubject: xcontrol.Subject{Key: "1111", Name: "writer1"},
							Authorities:    []xcontrol.Authority{xcontrol.AuthorityAbsolute - 1},
						}))
						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:          2,
							Channels:       []core.ChannelKey{k2, k3},
							ControlSubject: xcontrol.Subject{Key: "2222", Name: "writer2"},
							Authorities:    []xcontrol.Authority{xcontrol.AuthorityAbsolute},
						}))

						t := db.ControlStates().Transfers
						Expect(t).To(HaveLen(4))
						names := lo.Map(t, func(t control.Transfer, _ int) string {
							return t.To.Subject.Name
						})
						Expect(names).To(ConsistOf("writer1", "writer2", "writer2", "cesium_internal_control_digest"))

						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
					})

				})
			})

			Describe("Error paths", func() {
				It("Should not allow control channel with key 0", func() {
					db := openDBOnFS(fs)
					Expect(db.ConfigureControlUpdateChannel(ctx, 0, "cat")).To(MatchError(ContainSubstring("key: must be positive")))
					Expect(db.Close()).To(Succeed())
				})

				It("Should not allow configuring a control channel with datatype not string", func() {
					db := openDBOnFS(fs)
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Name:     "Deshon",
						Key:      key,
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(db.ConfigureControlUpdateChannel(ctx, key, "dog")).To(MatchError(ContainSubstring("must be a string virtual")))
					Expect(db.Close()).To(Succeed())
				})
			})

		})
	}
})
