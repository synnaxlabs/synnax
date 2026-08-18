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
	"io"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// observeControl subscribes to db's control updates and returns a channel receiving
// them. Sends are non-blocking, since handlers run under the DB's lock.
func observeControl(
	db *cesium.DB,
) (<-chan cesium.ControlUpdate, observe.Disconnect) {
	updates := make(chan cesium.ControlUpdate, 100)
	return updates, db.OnControlUpdate(
		func(_ context.Context, u cesium.ControlUpdate) {
			select {
			case updates <- u:
			default:
			}
		},
	)
}

var _ = Describe("Control", func() {
	for fsName, openFS := range FileSystems {
		Context("FS:"+fsName, Ordered, func() {
			var fs fs.FS

			BeforeAll(func() {
				ShouldNotLeakGoroutines()
				fs = openFS()
			})
			Describe("Nominal", func() {
				var db *cesium.DB
				BeforeAll(func(ctx SpecContext) {
					ShouldNotLeakGoroutines()
					db = mustOpenDBOnFS(ctx, fs)
				})

				Describe("Single Channel, Two Writer Contention", func() {
					It(
						"Should correctly manage control authority between two writers",
						func(ctx SpecContext) {
							var (
								indexChKey = GenerateChannelKey()
								dataChKey  = GenerateChannelKey()
							)
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "John",
									Key:      indexChKey,
									DataType: telem.TimestampT,
									IsIndex:  true,
								},
								cesium.Channel{
									Name:     "Sandler",
									Key:      dataChKey,
									DataType: telem.Int16T,
									Index:    indexChKey,
								},
							)).To(Succeed())
							start := telem.SecondTS * 10
							By("Opening the first writer")
							w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								ControlSubject: xcontrol.Subject{Name: "Writer One"},
								Start:          start,
								Channels: []cesium.ChannelKey{
									indexChKey,
									dataChKey,
								},
								Authorities: []xcontrol.Authority{
									xcontrol.AuthorityAbsolute - 2,
								},
								ErrOnUnauthorized: new(false),
								Sync:              new(true),
							}))
							By("Opening the second writer")
							w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Start:          start,
								ControlSubject: xcontrol.Subject{Name: "Writer Two"},
								Channels: []cesium.ChannelKey{
									indexChKey,
									dataChKey,
								},
								Authorities: []xcontrol.Authority{
									xcontrol.AuthorityAbsolute - 2,
								},
								ErrOnUnauthorized: new(false),
								Sync:              new(true),
							}))
							updates, disconnect := observeControl(db)
							defer disconnect()

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
								Authorities: []xcontrol.Authority{
									xcontrol.AuthorityAbsolute - 1,
								},
							})).To(Succeed())

							By("Propagating the control transfer")
							Eventually(updates).Should(Receive())

							By("Writing to the second writer")
							authorized = MustSucceed(w2.Write(w2Frame))
							Expect(authorized).To(BeTrue())

							end := MustSucceed(w2.Commit())
							Expect(end).To(Equal(telem.SecondTS*14 + 1))

							By("Shutting down the writers")
							Expect(w1.Close()).To(Succeed())
							Expect(w2.Close()).To(Succeed())

							By("Reading the data")
							f := MustSucceed(db.Read(
								ctx,
								start.SpanRange(10*telem.Second),
								dataChKey,
							))
							Expect(f.Count()).To(Equal(1))
							Expect(
								f.SeriesAt(0).Data,
							).To(Equal(telem.NewSeriesV[int16](1, 2, 3, 4, 5, 6).Data))
						},
					)

					It(
						"Should correctly hand off control authority when a writer is force closed via context cancellation",
						func(ctx SpecContext) {
							var (
								indexChKey = GenerateChannelKey()
								dataChKey  = GenerateChannelKey()
							)
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "Cat",
									Key:      indexChKey,
									DataType: telem.TimestampT,
									IsIndex:  true,
								},
								cesium.Channel{
									Name:     "Mouse",
									Key:      dataChKey,
									DataType: telem.Int16T,
									Index:    indexChKey,
								},
							)).To(Succeed())
							start := telem.SecondTS * 10

							updates, disconnect := observeControl(db)
							defer disconnect()
							ctx2, cancel2 := signal.Isolated()
							defer cancel2()

							By("Opening the first writer")
							w1 := MustSucceed(
								db.NewStreamWriter(ctx, cesium.WriterConfig{
									ControlSubject: xcontrol.Subject{
										Name: "Writer One",
									},
									Start: start,
									Channels: []cesium.ChannelKey{
										indexChKey,
										dataChKey,
									},
									Authorities: []xcontrol.Authority{
										xcontrol.AuthorityAbsolute - 2,
									},
									ErrOnUnauthorized: new(false),
									Sync:              new(true),
								}),
							)

							By("Opening the second writer")
							w2 := MustSucceed(
								db.NewStreamWriter(ctx, cesium.WriterConfig{
									Start: start,
									ControlSubject: xcontrol.Subject{
										Name: "Writer Two",
									},
									Channels: []cesium.ChannelKey{
										indexChKey,
										dataChKey,
									},
									Authorities: []xcontrol.Authority{
										xcontrol.AuthorityAbsolute - 3,
									},
									ErrOnUnauthorized: new(false),
									Sync:              new(true),
								}),
							)

							ctx1, cancel1 := signal.Isolated()

							w1In, _ := confluence.Attach(w1, 2)
							w2In, w2Out := confluence.Attach(w2, 2)

							w1.Flow(ctx1, confluence.CloseOutputInletsOnExit())
							w2.Flow(ctx2, confluence.CloseOutputInletsOnExit())

							runtime.Gosched()
							time.Sleep(1 * time.Millisecond)

							By("Writing to the first writer")
							w1In.Inlet() <- cesium.WriterRequest{
								Command: cesium.WriterCommandWrite,
								Frame: telem.MultiFrame(
									[]cesium.ChannelKey{indexChKey, dataChKey},
									[]telem.Series{
										telem.NewSeriesSecondsTSV(10, 11, 12),
										telem.NewSeriesV[int16](1, 2, 3),
									}),
							}

							var d cesium.ControlUpdate
							Eventually(updates).Should(Receive(&d))
							Expect(d.Transfers).To(HaveLen(2))
							Expect(d.Transfers[0].To).ToNot(BeNil())
							Expect(
								d.Transfers[0].To.Subject.Name,
							).To(Equal("Writer One"))

							By(
								"Force closing the first writer through context cancellation",
							)
							cancel1()
							Expect(ctx1.Wait()).To(MatchError(context.Canceled))

							By("Propagating the control transfer")
							Eventually(updates).Should(Receive(&d))
							Expect(d.Transfers).To(HaveLen(2))
							Expect(d.Transfers[0].To).ToNot(BeNil())
							Expect(
								d.Transfers[0].To.Subject.Name,
							).To(Equal("Writer Two"))

							By("Writing to the second writer")
							w2In.Inlet() <- cesium.WriterRequest{
								Command: cesium.WriterCommandWrite,
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
								Command: cesium.WriterCommandCommit,
							}
							var r cesium.WriterResponse
							Eventually(w2Out.Outlet()).Should(Receive(&r))

							By("Shutting down the second writer")
							w2In.Close()
							Expect(ctx2.Wait()).To(Succeed())

							By("Reading the data")
							f := MustSucceed(db.Read(
								ctx,
								start.SpanRange(10*telem.Second),
								dataChKey,
							))
							Expect(f.Count()).To(Equal(1))
							Expect(
								f.SeriesAt(0).Data,
							).To(Equal(telem.NewSeriesV[int16](1, 2, 3, 4, 5, 6).Data))
						},
					)
				})

				Describe("Mismatched Authorization", func() {
					var (
						indexChKey, dataChKey, virtualChKey cesium.ChannelKey
						w1                                  *cesium.Writer
						w2                                  *cesium.Writer
						dataStreamerIn                      confluence.Inlet[cesium.StreamerRequest]
						dataStreamerOut                     confluence.Outlet[cesium.StreamerResponse]
						controlUpdates                      <-chan cesium.ControlUpdate
						disconnectControl                   observe.Disconnect
						shutdown                            io.Closer
					)
					BeforeEach(func(ctx SpecContext) {
						indexChKey = GenerateChannelKey()
						dataChKey = GenerateChannelKey()
						virtualChKey = GenerateChannelKey()
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Name:     "persisted",
								Key:      indexChKey,
								DataType: telem.TimestampT,
								IsIndex:  true,
							},
							cesium.Channel{
								Name:     "persisted_idx",
								Key:      dataChKey,
								DataType: telem.Int16T,
								Index:    indexChKey,
							},
							cesium.Channel{
								Name:     "virtual_cmd",
								Key:      virtualChKey,
								DataType: telem.Uint8T,
								Virtual:  true,
							},
						)).To(Succeed())
						start := telem.SecondTS * 10
						By("Opening the first writer")
						w1 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							ControlSubject: xcontrol.Subject{Name: "Writer One"},
							Start:          start,
							Channels: []cesium.ChannelKey{
								virtualChKey,
								indexChKey,
								dataChKey,
							},
							Authorities:       []xcontrol.Authority{100},
							ErrOnUnauthorized: new(false),
							Sync:              new(true),
						}))
						By("Opening the second writer")
						w2 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:          start,
							ControlSubject: xcontrol.Subject{Name: "Writer Two"},
							Channels: []cesium.ChannelKey{
								indexChKey,
								dataChKey,
							},
							Authorities:       []xcontrol.Authority{0},
							ErrOnUnauthorized: new(false),
							Sync:              new(true),
						}))
						dataStreamer := MustSucceed(
							db.NewStreamer(ctx, cesium.StreamerConfig{
								Channels: []cesium.ChannelKey{
									virtualChKey,
									indexChKey,
									dataChKey,
								},
								SendOpenAck: true,
							}),
						)
						sCtx, cancel := signal.Isolated()
						shutdown = signal.NewHardShutdown(sCtx, cancel)
						dataStreamerIn, dataStreamerOut = confluence.Attach(
							dataStreamer,
							2,
						)
						dataStreamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
						Eventually(dataStreamerOut.Outlet()).Should(Receive())

						controlUpdates, disconnectControl = observeControl(db)
					})
					AfterEach(func() {
						disconnectControl()
						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
						dataStreamerIn.Close()
						Eventually(dataStreamerOut.Outlet()).Should(BeClosed())
						Expect(shutdown.Close()).To(Succeed())
					})

					// Set up:
					// Auto-Commit Enabled On Both
					// Writer 1: Virtual Channel, Persisted Channel, Persisted Channel
					// Index
					// Writer 2: Persisted Channel, Persisted Channel Index
					//
					// Writer 1 starts of at authority 100 on all channels
					// Writer 1 writes - confirm successful receive
					// Writer 2 starts off at authority 0 on all channels
					// Writer 2 changes to authority 200
					// Writer 1 writes to virtual channel (no persisted) - confirm
					// successful write
					Specify(
						"One Virtual, One Persisted, Different Authorities, Partial Contention",
						func() {
							By("Writing to the first writer")
							Expect(MustSucceed(w1.Write(telem.MultiFrame(
								[]cesium.ChannelKey{
									virtualChKey,
									indexChKey,
									dataChKey,
								},
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
							Eventually(controlUpdates).Should(Receive())

							By("Writing to the first writer")
							Expect(MustSucceed(w1.Write(telem.MultiFrame(
								[]cesium.ChannelKey{virtualChKey},
								[]telem.Series{telem.NewSeriesV[uint8](1)},
							)))).To(BeTrue())

							var r cesium.StreamerResponse
							Eventually(dataStreamerOut.Outlet()).Should(Receive(&r))
						},
					)

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
						Expect(
							r.Frame.Get(virtualChKey).Series[0],
						).To(telem.MatchSeriesDataV[uint8](1))
					})
				})

				Describe("ControlStates", func() {
					It(
						"Should report the leading control state of every channel",
						func(ctx SpecContext) {
							k1, k2, k3 := GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "Seattle",
									Key:      k1,
									Virtual:  true,
									DataType: telem.StringT,
								},
								cesium.Channel{
									Name:     "Austin",
									Key:      k2,
									DataType: telem.TimestampT,
									IsIndex:  true,
								},
								cesium.Channel{
									Name:     "Ann Arbor",
									Key:      k3,
									DataType: telem.Int64T,
									Index:    k2,
								},
							)).To(Succeed())
							w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Start:    0,
								Channels: []channel.Key{k1, k2},
								ControlSubject: xcontrol.Subject{
									Key:  "1111",
									Name: "writer1",
								},
								Authorities: []xcontrol.Authority{
									xcontrol.AuthorityAbsolute - 1,
								},
							}))
							w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Start:    2,
								Channels: []channel.Key{k2, k3},
								ControlSubject: xcontrol.Subject{
									Key:  "2222",
									Name: "writer2",
								},
								Authorities: []xcontrol.Authority{
									xcontrol.AuthorityAbsolute,
								},
							}))

							t := db.ControlStates().Transfers
							Expect(t).To(HaveLen(3))
							names := lo.Map(t, func(t control.Transfer, _ int) string {
								return t.To.Subject.Name
							})
							Expect(names).To(ConsistOf("writer1", "writer2", "writer2"))

							Expect(w1.Close()).To(Succeed())
							Expect(w2.Close()).To(Succeed())
						},
					)
				})

				Describe("OnControlUpdate", func() {
					It("Should notify handlers of acquires and releases", func(
						ctx SpecContext,
					) {
						k := GenerateChannelKey()
						Expect(db.CreateChannel(ctx, cesium.Channel{
							Name:     "Boulder",
							Key:      k,
							Virtual:  true,
							DataType: telem.StringT,
						})).To(Succeed())
						updates, disconnect := observeControl(db)
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:    0,
							Channels: []channel.Key{k},
							ControlSubject: xcontrol.Subject{
								Key:  "3333",
								Name: "boulder",
							},
						}))
						var acquire cesium.ControlUpdate
						Eventually(updates).Should(Receive(&acquire))
						Expect(acquire.Transfers).To(HaveLen(1))
						Expect(acquire.Transfers[0].From).To(BeNil())
						Expect(
							acquire.Transfers[0].To.Subject.Name,
						).To(Equal("boulder"))

						Expect(w.Close()).To(Succeed())
						var release cesium.ControlUpdate
						Eventually(updates).Should(Receive(&release))
						Expect(release.Transfers).To(HaveLen(1))
						Expect(release.Transfers[0].To).To(BeNil())
						Expect(
							release.Transfers[0].From.Subject.Name,
						).To(Equal("boulder"))

						disconnect()
						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:    0,
							Channels: []channel.Key{k},
							ControlSubject: xcontrol.Subject{
								Key:  "4444",
								Name: "denver",
							},
						}))
						Expect(w2.Close()).To(Succeed())
						Consistently(updates).ShouldNot(Receive())
					})
				})
			})
		})
	}
})
