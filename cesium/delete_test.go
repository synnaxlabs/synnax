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
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"runtime"
)

var _ = Describe("Delete", Ordered, func() {
	var db *cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Delete a channel when there are no iterators/writers", func() {
		Specify("Deleting a nonexistent channel", func() {
			Expect(db.DeleteChannel(9)).To(MatchError(query.Error))
		})
		Specify("Simple unary channel", func() {
			By("Creating a channel")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 1, Rate: 10 * telem.Hz, DataType: telem.Float64T},
			)).To(Succeed())
			By("Creating a writer on it")
			w := MustSucceed(db.OpenWriter(
				ctx,
				cesium.WriterConfig{
					Channels: []cesium.ChannelKey{1},
					Start:    10 * telem.SecondTS,
				}))
			By("Trying to delete it")
			Expect(db.DeleteChannel(1).Error()).To(ContainSubstring("1 unclosed"))
			By("Closing the writer")
			Expect(w.Close()).To(Succeed())
			By("Trying to delete it again")
			Expect(db.DeleteChannel(1)).To(Succeed())
			By("Trying to retrieve the channel")
			_, err := db.RetrieveChannel(ctx, 1)
			Expect(err).To(MatchError(query.Error))
		})

		Specify("Simple virtual channel", func() {
			By("Creating a channel")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 1, DataType: telem.Float64T, Virtual: true},
			)).To(Succeed())
			By("Creating a writer on it")
			w := MustSucceed(db.OpenWriter(
				ctx,
				cesium.WriterConfig{
					Channels: []cesium.ChannelKey{1},
					Start:    10 * telem.SecondTS,
				}))
			By("Trying to delete it")
			Expect(db.DeleteChannel(1).Error()).To(ContainSubstring("1 unclosed"))
			By("Closing the writer")
			Expect(w.Close()).To(Succeed())
			By("Trying to delete it again")
			Expect(db.DeleteChannel(1)).To(Succeed())
			By("Trying to retrieve the channel")
			_, err := db.RetrieveChannel(ctx, 1)
			Expect(err).To(MatchError(query.Error))
		})

		Specify("Indexed unary channels", func() {
			By("Creating two channels")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 2, IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: 3, Index: 2, DataType: telem.Int64T},
			)).To(Succeed())
			By("Creating writers on them")
			w := MustSucceed(db.OpenWriter(
				ctx,
				cesium.WriterConfig{
					Channels: []cesium.ChannelKey{2, 3},
					Start:    10 * telem.SecondTS,
				}))

			By("Trying to delete them")
			Expect(db.DeleteChannel(2).Error()).To(ContainSubstring("1 unclosed"))
			Expect(db.DeleteChannel(3).Error()).To(ContainSubstring("1 unclosed"))

			By("Closing the writer")
			Expect(w.Close()).To(Succeed())
			By("Trying to delete them again")
			Expect(db.DeleteChannel(2)).To(Succeed())
			Expect(db.DeleteChannel(3)).To(Succeed())

			By("Trying to retrieve the channels")
			_, err2 := db.RetrieveChannel(ctx, 2)
			_, err3 := db.RetrieveChannel(ctx, 3)
			Expect(err2).To(MatchError(query.Error))
			Expect(err3).To(MatchError(query.Error))
		})

		Specify("Controlled streamwriters", func() {
			var (
				controlKey cesium.ChannelKey = 7
				basic3     cesium.ChannelKey = 8
			)
			Expect(db.ConfigureControlUpdateChannel(ctx, controlKey)).To(Succeed())
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: basic3, DataType: telem.Int64T, Rate: 1 * telem.Hz},
			)).To(Succeed())
			streamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
				Channels: []cesium.ChannelKey{controlKey},
			}))
			i, o := confluence.Attach(streamer, 1)
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			streamer.Flow(sCtx, confluence.CloseInletsOnExit())
			// Do a best effort schedule for the streamer to boot up
			runtime.Gosched()
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels:       []cesium.ChannelKey{basic3},
				ControlSubject: control.Subject{Name: "Writer"},
				Start:          10 * telem.SecondTS,
			}))
			Expect(db.DeleteChannel(basic3).Error()).To(ContainSubstring("1 unclosed"))
			var r cesium.StreamerResponse
			Eventually(o.Outlet()).Should(Receive(&r))
			Expect(r.Frame.Keys).To(HaveLen(1))
			u := MustSucceed(cesium.DecodeControlUpdate(ctx, r.Frame.Series[0]))
			t, ok := lo.Find(u.Transfers, func(t controller.Transfer) bool {
				return t.To.Resource == basic3
			})
			Expect(ok).To(BeTrue())
			Expect(t.To.Subject.Name).To(Equal("Writer"))
			Expect(w.Close()).To(Succeed())
			Eventually(o.Outlet()).Should(Receive(&r))
			Expect(r.Frame.Keys).To(HaveLen(1))
			i.Close()
			Expect(sCtx.Wait()).To(Succeed())

			Expect(db.DeleteChannel(basic3)).To(Succeed())
			Expect(db.DeleteChannel(controlKey).Error()).To(ContainSubstring("cannot delete update digest channel"))
		})
	})
})
