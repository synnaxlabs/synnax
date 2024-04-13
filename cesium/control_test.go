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
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"math"
)

var _ = Describe("Control", Ordered, func() {
	var db *cesium.DB
	BeforeAll(func() {
		db = openMemDB()
		Expect(db.ConfigureControlUpdateChannel(ctx, math.MaxUint32)).To(Succeed())
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Single Channel, Two Writer Contention", func() {
		It("Should work", func() {
			var ch1 cesium.ChannelKey = 1
			Expect(db.CreateChannel(ctx, cesium.Channel{Key: ch1, DataType: telem.Int16T, Rate: 1 * telem.Hz})).To(Succeed())
			start := telem.SecondTS * 10
			By("Opening the first writer")
			w1 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
				ControlSubject:    control.Subject{Name: "Writer One"},
				Start:             start,
				Channels:          []cesium.ChannelKey{ch1},
				Authorities:       []control.Authority{control.Absolute - 2},
				ErrOnUnauthorized: config.True(),
			}))
			By("Opening the second writer")
			w2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
				Start:             start,
				ControlSubject:    control.Subject{Name: "Writer Two"},
				Channels:          []cesium.ChannelKey{ch1},
				Authorities:       []control.Authority{control.Absolute - 2},
				ErrOnUnauthorized: config.True(),
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
			w2In.Inlet() <- cesium.WriterRequest{
				Command: cesium.WriterError,
			}
			Eventually(w2Out.Outlet()).Should(Receive(&r))
			Expect(errors.Is(r.Err, controller.Unauthorized("Writer Two", ch1))).To(BeTrue())

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
	})
	Describe("Creating update channel with key 0", func() {
		It("Should not allow it", func() {
			Expect(db.ConfigureControlUpdateChannel(ctx, 0).Error()).To(ContainSubstring("key must be positive"))
		})
	})
})
