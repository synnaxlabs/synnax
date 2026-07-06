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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/alignment"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func virtualIndexChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:       key,
		Name:      name,
		DataType:  telem.TimeStampT,
		IsIndex:   true,
		Virtual:   true,
		Transient: true,
	}
}

func virtualDataChannel(key, index cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:       key,
		Name:      name,
		DataType:  telem.Int64T,
		Index:     index,
		Virtual:   true,
		Transient: true,
	}
}

var _ = Describe("Virtual Index Channels", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			ShouldNotLeakGoroutinesPerSpec()
			var db *cesium.DB
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				db = openDBOnFS(ctx, openFS())
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
			})

			Describe("Create Validation", func() {
				It("Should create a virtual index channel and a virtual data channel indexed by it", func(ctx SpecContext) {
					idx := GenerateChannelKey()
					data := GenerateChannelKey()
					Expect(db.CreateChannel(ctx,
						virtualIndexChannel(idx, "v_idx"),
						virtualDataChannel(data, idx, "v_data"),
					)).To(Succeed())
					ch := MustSucceed(db.RetrieveChannel(ctx, data))
					Expect(ch.Index).To(Equal(idx))
					idxCh := MustSucceed(db.RetrieveChannel(ctx, idx))
					Expect(idxCh.Index).To(Equal(idx))
				})

				It("Should reject a virtual index channel that is not a timestamp", func(ctx SpecContext) {
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      GenerateChannelKey(),
						Name:     "bad_dtype",
						DataType: telem.Int64T,
						IsIndex:  true,
						Virtual:  true,
					})).To(MatchError(ContainSubstring("index channel must be of type timestamp")))
				})

				It("Should reject a virtual channel indexed by a stored channel", func(ctx SpecContext) {
					storedIdx := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      storedIdx,
						Name:     "stored_idx",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(db.CreateChannel(ctx,
						virtualDataChannel(GenerateChannelKey(), storedIdx, "v_on_stored"),
					)).To(MatchError(ContainSubstring("virtual channel cannot be indexed by stored channel")))
				})

				It("Should reject a stored channel indexed by a virtual channel", func(ctx SpecContext) {
					idx := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, virtualIndexChannel(idx, "v_idx_stored_data"))).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      GenerateChannelKey(),
						Name:     "stored_on_virtual",
						DataType: telem.Int64T,
						Index:    idx,
					})).To(MatchError(ContainSubstring("stored channel cannot be indexed by virtual channel")))
				})

				It("Should reject a virtual channel indexed by a non-index virtual channel", func(ctx SpecContext) {
					nonIdx := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(nonIdx, "non_idx"))).To(Succeed())
					Expect(db.CreateChannel(ctx,
						virtualDataChannel(GenerateChannelKey(), nonIdx, "on_non_idx"),
					)).To(MatchError(ContainSubstring("is not an index")))
				})

				It("Should reject a virtual channel whose index does not exist", func(ctx SpecContext) {
					Expect(db.CreateChannel(ctx,
						virtualDataChannel(GenerateChannelKey(), GenerateChannelKey(), "dangling"),
					)).To(MatchError(ContainSubstring("does not exist")))
				})

				It("Should reject a persistent virtual channel indexed by a transient index", func(ctx SpecContext) {
					idx := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, virtualIndexChannel(idx, "transient_idx"))).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      GenerateChannelKey(),
						Name:     "persistent_member",
						DataType: telem.Int64T,
						Index:    idx,
						Virtual:  true,
					})).To(MatchError(ContainSubstring("persistent channel cannot be indexed by transient channel")))
				})
			})

			Describe("Delete", func() {
				It("Should not delete a virtual index that other virtual channels depend on", func(ctx SpecContext) {
					idx := GenerateChannelKey()
					data := GenerateChannelKey()
					Expect(db.CreateChannel(ctx,
						virtualIndexChannel(idx, "guarded_idx"),
						virtualDataChannel(data, idx, "dependent"),
					)).To(Succeed())
					Expect(db.DeleteChannel(idx)).To(MatchError(ContainSubstring("indexes data in channel")))
					Expect(db.DeleteChannel(data)).To(Succeed())
					Expect(db.DeleteChannel(idx)).To(Succeed())
				})
			})

			Describe("Write Alignment", func() {
				openGroup := func(ctx SpecContext) (idx, d1, d2 cesium.ChannelKey) {
					idx = GenerateChannelKey()
					d1 = GenerateChannelKey()
					d2 = GenerateChannelKey()
					Expect(db.CreateChannel(ctx,
						virtualIndexChannel(idx, "align_idx"),
						virtualDataChannel(d1, idx, "align_d1"),
						virtualDataChannel(d2, idx, "align_d2"),
					)).To(Succeed())
					return idx, d1, d2
				}

				streamInto := func(ctx SpecContext, keys ...cesium.ChannelKey) (
					confluence.Outlet[cesium.StreamerResponse],
					func(),
				) {
					r := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{Channels: keys}))
					i, o := confluence.Attach(r, 5)
					sCtx, cancel := signal.WithCancel(ctx)
					r.Flow(sCtx, confluence.CloseOutputInletsOnExit())
					return o, func() {
						i.Close()
						Expect(sCtx.Wait()).To(Succeed())
						cancel()
					}
				}

				It("Should stamp every series in an index group with the same alignment and advance on index writes", func(ctx SpecContext) {
					idx, d1, d2 := openGroup(ctx)
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{idx, d1, d2},
						Start:    10 * telem.SecondTS,
					}))
					o, stop := streamInto(ctx, idx, d1, d2)
					defer stop()

					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{idx, d1, d2},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12),
							telem.NewSeriesV[int64](1, 2, 3),
							telem.NewSeriesV[int64](4, 5, 6),
						},
					)))
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					first := res.Frame.SeriesAt(0).Alignment
					Expect(first.DomainIndex()).To(BeNumerically(">=", alignment.ZeroLeading))
					Expect(first.SampleIndex()).To(BeZero())
					for i := range res.Frame.Count() {
						Expect(res.Frame.SeriesAt(i).Alignment).To(Equal(first))
					}

					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{idx, d1},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(13, 14),
							telem.NewSeriesV[int64](7, 8),
						},
					)))
					Eventually(o.Outlet()).Should(Receive(&res))
					second := res.Frame.SeriesAt(0).Alignment
					Expect(second).To(Equal(first.AddSamples(3)))
					for i := range res.Frame.Count() {
						Expect(res.Frame.SeriesAt(i).Alignment).To(Equal(second))
					}
					Expect(w.Close()).To(Succeed())
				})

				It("Should not advance the group alignment on data-only writes", func(ctx SpecContext) {
					idx, d1, _ := openGroup(ctx)
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{idx, d1},
						Start:    10 * telem.SecondTS,
					}))
					o, stop := streamInto(ctx, idx, d1)
					defer stop()

					MustSucceed(w.Write(telem.UnaryFrame(d1, telem.NewSeriesV[int64](1, 2))))
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					first := res.Frame.SeriesAt(0).Alignment

					MustSucceed(w.Write(telem.UnaryFrame(d1, telem.NewSeriesV[int64](3, 4))))
					Eventually(o.Outlet()).Should(Receive(&res))
					Expect(res.Frame.SeriesAt(0).Alignment).To(Equal(first))
					Expect(w.Close()).To(Succeed())
				})

				It("Should allocate the group domain from the index channel even when the index is not in the writer's channel set", func(ctx SpecContext) {
					idx, d1, d2 := openGroup(ctx)
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{d1, d2},
						Start:    10 * telem.SecondTS,
					}))
					o, stop := streamInto(ctx, d1, d2)
					defer stop()

					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{d1, d2},
						[]telem.Series{
							telem.NewSeriesV[int64](1, 2),
							telem.NewSeriesV[int64](3, 4),
						},
					)))
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					// No gate is opened on the index channel, so the group's allocation
					// is the first domain drawn from the index DB's counter.
					Expect(res.Frame.SeriesAt(0).Alignment).To(Equal(alignment.Leading(1, 0)))
					Expect(res.Frame.SeriesAt(1).Alignment).To(Equal(alignment.Leading(1, 0)))
					Expect(w.Close()).To(Succeed())
					_ = idx
				})

				It("Should allocate a fresh domain for each writer on the same group", func(ctx SpecContext) {
					idx, d1, _ := openGroup(ctx)
					w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{idx, d1},
						Start:    10 * telem.SecondTS,
					}))
					w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{idx, d1},
						Start:    20 * telem.SecondTS,
					}))
					o, stop := streamInto(ctx, idx, d1)
					defer stop()

					MustSucceed(w1.Write(telem.UnaryFrame(idx, telem.NewSeriesSecondsTSV(10))))
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					a1 := res.Frame.SeriesAt(0).Alignment

					MustSucceed(w2.Write(telem.UnaryFrame(idx, telem.NewSeriesSecondsTSV(20))))
					Eventually(o.Outlet()).Should(Receive(&res))
					a2 := res.Frame.SeriesAt(0).Alignment

					Expect(a2.DomainIndex()).To(BeNumerically(">", a1.DomainIndex()))
					Expect(a2.SampleIndex()).To(BeZero())
					Expect(w1.Close()).To(Succeed())
					Expect(w2.Close()).To(Succeed())
				})

				It("Should keep alignments independent across separate index groups", func(ctx SpecContext) {
					idxA, dA, _ := openGroup(ctx)
					idxB, dB, _ := openGroup(ctx)
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{idxA, dA, idxB, dB},
						Start:    10 * telem.SecondTS,
					}))
					o, stop := streamInto(ctx, idxA, dA, idxB, dB)
					defer stop()

					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{idxA, dA, idxB},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11),
							telem.NewSeriesV[int64](1, 2),
							telem.NewSeriesSecondsTSV(10, 11, 12),
						},
					)))
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					groupA := res.Frame.SeriesAt(0).Alignment
					Expect(res.Frame.SeriesAt(1).Alignment).To(Equal(groupA))

					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{dA, dB},
						[]telem.Series{
							telem.NewSeriesV[int64](3, 4),
							telem.NewSeriesV[int64](5, 6),
						},
					)))
					Eventually(o.Outlet()).Should(Receive(&res))
					Expect(res.Frame.Get(dA).Series[0].Alignment).To(Equal(groupA.AddSamples(2)))
					Expect(res.Frame.Get(dB).Series[0].Alignment.SampleIndex()).To(Equal(uint32(3)))
					Expect(w.Close()).To(Succeed())
				})
			})
		})
	}

	Describe("Ungrouped virtual channels", func() {
		It("Should leave alignment behavior for index-free virtual channels unchanged", func(ctx SpecContext) {
			db := openDBOnFS(ctx, fs.NewMem())
			key := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, transientChannel(key, "ungrouped"))).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{key},
				Start:    10 * telem.SecondTS,
			}))
			r := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
				Channels: []cesium.ChannelKey{key},
			}))
			i, o := confluence.Attach(r, 1)
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			r.Flow(sCtx, confluence.CloseOutputInletsOnExit())

			MustSucceed(w.Write(telem.UnaryFrame(key, telem.NewSeriesV[int64](1, 2, 3))))
			var res cesium.StreamerResponse
			Eventually(o.Outlet()).Should(Receive(&res))
			Expect(res.Frame.SeriesAt(0).Alignment).To(Equal(alignment.Leading(1, 0)))

			MustSucceed(w.Write(telem.UnaryFrame(key, telem.NewSeriesV[int64](4, 5))))
			Eventually(o.Outlet()).Should(Receive(&res))
			Expect(res.Frame.SeriesAt(0).Alignment).To(Equal(alignment.Leading(1, 3)))

			i.Close()
			Expect(sCtx.Wait()).To(Succeed())
			Expect(w.Close()).To(Succeed())
			Expect(db.Close()).To(Succeed())
		})
	})
})
