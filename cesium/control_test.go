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
	"context"
	"encoding/json"
	"math"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/controller"
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
		if fsName != "memFS" {
			continue
		}
		Context("FS:"+fsName, Ordered, func() {
			var (
				db      *cesium.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
				db = openDBOnFS(fs)
				Expect(db.ConfigureControlUpdateChannel(ctx, math.MaxUint32)).To(Succeed())
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Single Channel, Two Writer Contention", func() {
				It("Should correctly manage control authority between two writers", func() {
					var ch1 cesium.ChannelKey = 1
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: ch1, DataType: telem.Int16T, Rate: 1 * telem.Hz})).To(Succeed())
					start := telem.SecondTS * 10
					By("Opening the first writer")
					w1 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
						ControlSubject:    control.Subject{Name: "Writer One"},
						Start:             start,
						Channels:          []cesium.ChannelKey{ch1},
						Authorities:       []control.Authority{control.Absolute - 2},
						ErrOnUnauthorized: config.False(),
						SendAuthErrors:    config.True(),
					}))
					By("Opening the second writer")
					w2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
						Start:             start,
						ControlSubject:    control.Subject{Name: "Writer Two"},
						Channels:          []cesium.ChannelKey{ch1},
						Authorities:       []control.Authority{control.Absolute - 2},
						ErrOnUnauthorized: config.False(),
						SendAuthErrors:    config.True(),
					}))
					streamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{math.MaxUint32},
					}))
					ctx, cancel := signal.Isolated()
					defer cancel()
					w1In, _ := confluence.Attach(w1, 2)
					w2In, w2Out := confluence.Attach(w2, 2)
					stIn, stOut := confluence.Attach(streamer, 2)
					w1.Flow(ctx)
					w2.Flow(ctx)
					streamer.Flow(ctx)
					By("Writing to the first writer")
					w1In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterWrite,
						Frame: core.NewFrame(
							[]cesium.ChannelKey{ch1},
							[]telem.Series{telem.NewSeriesV[int16](1, 2, 3)},
						),
					}

					By("Failing to write to the second writer")
					w2In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterWrite,
						Frame: core.NewFrame(
							[]cesium.ChannelKey{ch1},
							[]telem.Series{telem.NewSeriesV[int16](4, 5, 6)},
						),
					}
					var r cesium.WriterResponse
					Eventually(w2Out.Outlet()).Should(Receive(&r))
					Expect(r.Ack).To(BeFalse())
					w2In.Inlet() <- cesium.WriterRequest{Command: cesium.WriterError}
					Eventually(w2Out.Outlet()).Should(Receive(&r))
					Expect(r.Err).To(HaveOccurredAs(control.Unauthorized))

					By("Updating the second writer's authorities")
					w2In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterSetAuthority,
						Config: cesium.WriterConfig{
							Authorities: []control.Authority{control.Absolute - 1},
						},
					}

					By("Propagating the control transfer")
					Eventually(stOut.Outlet()).Should(Receive())

					By("Writing to the second writer")
					w2In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterWrite,
						Frame: core.NewFrame(
							[]cesium.ChannelKey{ch1},
							[]telem.Series{telem.NewSeriesV[int16](4, 5, 6)},
						),
					}
					By("Committing the second writer")
					w2In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterCommit,
					}
					Eventually(w2Out.Outlet()).Should(Receive(&r))
					Expect(r.Ack).To(BeTrue())

					By("Shutting down the writers")
					w1In.Close()
					w2In.Close()
					stIn.Close()
					Expect(ctx.Wait()).To(Succeed())

					By("Reading the data")
					f := MustSucceed(db.Read(
						ctx,
						start.SpanRange(10*telem.Second),
						ch1,
					))
					Expect(f.Series).To(HaveLen(1))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int16](1, 2, 3, 4, 5, 6).Data))
				})

				It("Should correctly hand off control authority when a writer is force closed via context cancellation", func() {
					var ch1 cesium.ChannelKey = 5
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: ch1, DataType: telem.Int16T, Rate: 1 * telem.Hz})).To(Succeed())
					start := telem.SecondTS * 10

					streamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{math.MaxUint32},
					}))

					stIn, stOut := confluence.Attach(streamer, 2)

					ctx2, cancel2 := signal.Isolated()
					defer cancel2()
					streamer.Flow(ctx2, confluence.CloseOutputInletsOnExit())

					runtime.Gosched()

					By("Opening the first writer")
					w1 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
						ControlSubject:    control.Subject{Name: "Writer One"},
						Start:             start,
						Channels:          []cesium.ChannelKey{ch1},
						Authorities:       []control.Authority{control.Absolute - 2},
						ErrOnUnauthorized: config.False(),
						SendAuthErrors:    config.True(),
					}))

					By("Opening the second writer")
					w2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
						Start:             start,
						ControlSubject:    control.Subject{Name: "Writer Two"},
						Channels:          []cesium.ChannelKey{ch1},
						Authorities:       []control.Authority{control.Absolute - 3},
						ErrOnUnauthorized: config.False(),
						SendAuthErrors:    config.True(),
					}))

					ctx1, cancel1 := signal.Isolated()

					w1In, _ := confluence.Attach(w1, 2)
					w2In, w2Out := confluence.Attach(w2, 2)

					w1.Flow(ctx1, confluence.CloseOutputInletsOnExit())
					w2.Flow(ctx2, confluence.CloseOutputInletsOnExit())

					runtime.Gosched()

					By("Writing to the first writer")
					w1In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterWrite,
						Frame: core.NewFrame(
							[]cesium.ChannelKey{ch1},
							[]telem.Series{telem.NewSeriesV[int16](1, 2, 3)},
						),
					}

					var res cesium.StreamerResponse
					Eventually(stOut.Outlet()).Should(Receive(&res))
					var d cesium.ControlUpdate
					Expect(json.Unmarshal(res.Frame.Series[0].Data, &d)).To(Succeed())
					Expect(d.Transfers).To(HaveLen(1))
					Expect(d.Transfers[0].To.Subject.Name).To(Equal("Writer One"))

					By("Force closing the first writer through context cancellation")
					cancel1()
					Expect(ctx1.Wait()).To(HaveOccurredAs(context.Canceled))

					By("Propagating the control transfer")
					Eventually(stOut.Outlet()).Should(Receive(&res))
					Expect(json.Unmarshal(res.Frame.Series[0].Data, &d)).To(Succeed())
					Expect(d.Transfers).To(HaveLen(1))
					Expect(d.Transfers[0].To.Subject.Name).To(Equal("Writer Two"))

					By("Writing to the second writer")
					w2In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterWrite,
						Frame: core.NewFrame(
							[]cesium.ChannelKey{ch1},
							[]telem.Series{telem.NewSeriesV[int16](4, 5, 6)},
						),
					}

					By("Committing the second writer")
					w2In.Inlet() <- cesium.WriterRequest{
						Command: cesium.WriterCommit,
					}
					var r cesium.WriterResponse
					Eventually(w2Out.Outlet()).Should(Receive(&r))
					Expect(r.Ack).To(BeTrue())

					By("Shutting down the second writer")
					w2In.Close()
					stIn.Close()
					Expect(ctx2.Wait()).To(Succeed())

					By("Reading the data")
					f := MustSucceed(db.Read(
						ctx,
						start.SpanRange(10*telem.Second),
						ch1,
					))
					Expect(f.Series).To(HaveLen(1))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int16](1, 2, 3, 4, 5, 6).Data))
				})
			})

			// Specs testing the control digest system correctly propagates control
			// changes between contending writers.
			Describe("Control digests", func() {
				It("Should propagate the control states of channels", func() {
					var k1, k2, k3 = GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(ctx,
						cesium.Channel{Key: k1, Virtual: true, DataType: telem.StringT},
						cesium.Channel{Key: k2, DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Key: k3, DataType: telem.Int64T, Index: k2},
					)).To(Succeed())
					w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Start:          0,
						Channels:       []core.ChannelKey{k1, k2},
						ControlSubject: control.Subject{Key: "1111", Name: "writer1"},
						Authorities:    []control.Authority{control.Absolute - 1},
					}))
					w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Start:          2,
						Channels:       []core.ChannelKey{k2, k3},
						ControlSubject: control.Subject{Key: "2222", Name: "writer2"},
						Authorities:    []control.Authority{control.Absolute},
					}))

					t := db.ControlStates().Transfers
					Expect(t).To(HaveLen(4))
					names := lo.Map(t, func(t controller.Transfer, _ int) string {
						return t.To.Subject.Name
					})
					Expect(names).To(ConsistOf("writer1", "writer2", "writer2", "cesium_internal_control_digest"))

					Expect(w1.Close()).To(Succeed())
					Expect(w2.Close()).To(Succeed())
				})

			})

			Describe("Error paths", func() {
				It("Should not allow control channel with key 0", func() {
					Expect(db.ConfigureControlUpdateChannel(ctx, 0)).To(MatchError(ContainSubstring("key:must be positive")))
				})
				It("Should not allow configuring a control channel with datatype not string", func() {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(db.ConfigureControlUpdateChannel(ctx, key)).To(MatchError(ContainSubstring("must be a string virtual")))
				})
			})

		})
	}
})
