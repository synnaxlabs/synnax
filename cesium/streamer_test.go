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
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func decodeControlUpdate(ctx context.Context, s telem.Series) (cesium.ControlUpdate, error) {
	var u cesium.ControlUpdate
	if err := (&binary.JSONCodec{}).Decode(ctx, s.Data, &u); err != nil {
		return cesium.ControlUpdate{}, err
	}
	return u, nil
}

var _ = Describe("Streamer Behavior", func() {
	for fsName, makeFS := range fileSystems {
		ShouldNotLeakRoutinesJustBeforeEach()
		Context("FS: "+fsName, Ordered, func() {
			var (
				db         *cesium.DB
				fs         fs.FS
				cleanUp    func() error
				controlKey cesium.ChannelKey = 5
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
				db = openDBOnFS(fs)
				Expect(db.ConfigureControlUpdateChannel(ctx, controlKey, "cesium_control")).To(Succeed())
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Happy Path", func() {
				It("Should subscribe to written frames for the given channels", func() {
					var basic1 cesium.ChannelKey = 1
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic1, Name: "Planck", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic1},
						Start:    10 * telem.SecondTS,
					}))
					r := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{basic1},
					}))
					i, o := confluence.Attach(r, 1)
					sCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					r.Flow(sCtx, confluence.CloseOutputInletsOnExit())

					d := telem.NewSeriesSecondsTSV(10, 11, 12)
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{basic1},
						[]telem.Series{d},
					)))

					f := <-o.Outlet()
					Expect(f.Frame.Count()).To(Equal(1))
					d.Alignment = alignment.Leading(1, 0)
					Expect(f.Frame.SeriesAt(0)).To(Equal(d))
					i.Close()
					Expect(sCtx.Wait()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Writer is in PersistOnly mode", func() {
				It("Should not receive any frames", func() {
					var basic2 cesium.ChannelKey = 3
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic2, Name: "Bohr", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic2},
						Start:    10 * telem.SecondTS,
						Mode:     cesium.WriterPersistOnly,
					}))
					r := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{basic2},
					}))
					i, o := confluence.Attach(r, 1)
					sCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					r.Flow(sCtx, confluence.CloseOutputInletsOnExit())

					d := telem.NewSeriesSecondsTSV(10, 11, 12)
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{basic2},
						[]telem.Series{d},
					)))

					Consistently(o.Outlet()).ShouldNot(Receive())
					i.Close()
					Expect(sCtx.Wait()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Virtual Channels", func() {
				It("Should subscribe to written frames for virtual channels", func() {
					var basic2 cesium.ChannelKey = 4
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic2, Name: "Heisenberg", DataType: telem.Int64T, Virtual: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic2},
						Start:    10 * telem.SecondTS,
					}))
					r := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{basic2},
					}))
					i, o := confluence.Attach(r, 1)
					sCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					r.Flow(sCtx, confluence.CloseOutputInletsOnExit())

					written := telem.NewSeriesV[int64](1, 2, 3)
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{basic2},
						[]telem.Series{written},
					)))
					var f cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&f))
					Expect(f.Frame.Count()).To(Equal(1))
					written.Alignment = alignment.Leading(1, 0)
					Expect(f.Frame.SeriesAt(0)).To(Equal(written))
					i.Close()
					Expect(sCtx.Wait()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Control Updates", func() {
				It("Should forward control updates to the streamer", func() {
					var basic3 cesium.ChannelKey = 6
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic3, Name: "Schrodinger", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					streamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
						Channels:    []cesium.ChannelKey{controlKey},
						SendOpenAck: true,
					}))
					i, o := confluence.Attach(streamer, 1)
					sCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
					// Do a best effort schedule for the streamer to boot up
					Eventually(o.Outlet()).Should(Receive())
					runtime.Gosched()
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:       []cesium.ChannelKey{basic3},
						ControlSubject: control.Subject{Name: "Writer"},
						Start:          10 * telem.SecondTS,
					}))
					var r cesium.StreamerResponse
					// Move this into an eventual closure, as we may be getting latent
					// control updates from other tests, so we just assert on updates
					// until we get one that matches.
					Eventually(func(g Gomega) {
						g.Eventually(o.Outlet()).Should(Receive(&r))
						g.Expect(r.Frame.Count()).To(Equal(1))
						u, err := decodeControlUpdate(ctx, r.Frame.SeriesAt(0))
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(u.Transfers).To(HaveLen(1))
						first := u.Transfers[0]
						g.Expect(first.Occurred()).To(BeTrue())
						g.Expect(first.IsAcquire()).To(BeTrue())
					}).Should(Succeed())

					Expect(w.Close()).To(Succeed())
					Eventually(o.Outlet()).Should(Receive(&r))
					Expect(r.Frame.Count()).To(Equal(1))
					i.Close()
					Expect(sCtx.Wait()).To(Succeed())
				})
			})

			Describe("Closed", func() {
				It("Should not allow opening a streamer on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Einstein",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.NewStreamer(ctx, cesium.StreamerConfig{Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(resource.NewErrClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})
			})
		})
	}
})
