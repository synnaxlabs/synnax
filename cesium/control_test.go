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
	"context"
	"encoding/json"
	"math"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	xcontrol "github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/core"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Control", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS:"+fsName, Ordered, func() {
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

			Describe("Nominal", func() {
				var (
					db *cesium.DB
				)
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
							indexCHKey = GenerateChannelKey()
							dataChKey  = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "John", Key: indexCHKey, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Sandler", Key: dataChKey, DataType: telem.Int16T, Index: indexCHKey},
						)).To(Succeed())
						start := telem.SecondTS * 10
						By("Opening the first writer")
						w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							ControlSubject:    control.Subject{Name: "Writer One"},
							Start:             start,
							Channels:          []cesium.ChannelKey{indexCHKey, dataChKey},
							Authorities:       []control.Authority{control.AuthorityAbsolute - 2},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))
						By("Opening the second writer")
						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:             start,
							ControlSubject:    control.Subject{Name: "Writer Two"},
							Channels:          []cesium.ChannelKey{indexCHKey, dataChKey},
							Authorities:       []control.Authority{control.AuthorityAbsolute - 2},
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
						Expect(MustSucceed(w1.Write(telem.MultiFrame[cesium.ChannelKey](
							[]cesium.ChannelKey{indexCHKey, dataChKey},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(10, 11, 12),
								telem.NewSeriesV[int16](1, 2, 3),
							},
						)))).To(BeTrue())

						By("Failing to write to the second writer")
						w2Frame := telem.MultiFrame[cesium.ChannelKey](
							[]cesium.ChannelKey{indexCHKey, dataChKey},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(12, 13, 14),
								telem.NewSeriesV[int16](4, 5, 6),
							},
						)

						authorized := MustSucceed(w2.Write(w2Frame))
						Expect(authorized).To(BeFalse())

						Expect(w2.SetAuthority(cesium.WriterConfig{
							Authorities: []control.Authority{control.AuthorityAbsolute - 1},
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
							indexCHKey = GenerateChannelKey()
							dataChKey  = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "Cat", Key: indexCHKey, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Mouse", Key: dataChKey, DataType: telem.Int16T, Index: indexCHKey},
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
							ControlSubject:    control.Subject{Name: "Writer One"},
							Start:             start,
							Channels:          []cesium.ChannelKey{indexCHKey, dataChKey},
							Authorities:       []control.Authority{control.AuthorityAbsolute - 2},
							ErrOnUnauthorized: config.False(),
							Sync:              config.True(),
						}))

						By("Opening the second writer")
						w2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
							Start:             start,
							ControlSubject:    control.Subject{Name: "Writer Two"},
							Channels:          []cesium.ChannelKey{indexCHKey, dataChKey},
							Authorities:       []control.Authority{control.AuthorityAbsolute - 3},
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
							Frame: telem.MultiFrame[cesium.ChannelKey](
								[]cesium.ChannelKey{indexCHKey, dataChKey},
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
							Frame: telem.MultiFrame[cesium.ChannelKey](
								[]cesium.ChannelKey{indexCHKey, dataChKey},
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
							ControlSubject: control.Subject{Key: "1111", Name: "writer1"},
							Authorities:    []control.Authority{control.AuthorityAbsolute - 1},
						}))
						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Start:          2,
							Channels:       []core.ChannelKey{k2, k3},
							ControlSubject: control.Subject{Key: "2222", Name: "writer2"},
							Authorities:    []control.Authority{control.AuthorityAbsolute},
						}))

						t := db.ControlStates().Transfers
						Expect(t).To(HaveLen(4))
						names := lo.Map(t, func(t xcontrol.Transfer, _ int) string {
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
