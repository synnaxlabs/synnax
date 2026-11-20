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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func writerConfig(
	keys []cesium.ChannelKey,
	auth control.Authority,
	subject string,
) writer.Config {
	return writer.Config{
		ControlSubject:   control.Subject{Key: subject},
		Keys:             channel.KeysFromUint32(keys),
		Start:            10 * telem.SecondTS,
		Authorities:      []control.Authority{auth},
		EnableAutoCommit: new(false),
		Sync:             new(true),
	}
}

func frameOf(f cesium.Frame) frame.Frame { return frame.NewFromStorage(f) }

var _ = Describe("One clock model check", func() {
	var (
		db             *testDB
		i1, d1, i2, d2 cesium.ChannelKey
		all            []cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		db = mustOpenDBOnFS(ctx, FileSystems["memFS"]())
		i1, d1 = GenerateChannelKey(), GenerateChannelKey()
		i2, d2 = GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      i1,
				Name:     "i1",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: d1, Name: "d1", Index: i1, DataType: telem.Int64T},
			cesium.Channel{
				Key:      i2,
				Name:     "i2",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: d2, Name: "d2", Index: i2, DataType: telem.Int64T},
		)).To(Succeed())
		all = []cesium.ChannelKey{i1, d1, i2, d2}
	})
	open := func(ctx SpecContext, cfg storageConfig) *storageWriter {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cfg))
		DeferCleanup(func() { _ = w.Close() })
		return w
	}
	ts := func(s ...int64) telem.Series {
		out := make([]telem.TimeStamp, len(s))
		for i, v := range s {
			out[i] = telem.TimeStamp(v) * telem.SecondTS
		}
		return telem.NewSeries(out)
	}
	both := func(a1, a2, v1, v2 []int64) cesium.Frame {
		return telem.MultiFrame(all, []telem.Series{
			ts(a1...), telem.NewSeries(v1), ts(a2...), telem.NewSeries(v2),
		})
	}
	read := func(ctx SpecContext, key cesium.ChannelKey) telem.MultiSeries {
		GinkgoHelper()
		return MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key)
	}

	It(
		"Should thread a frame of two clocks through both writers",
		func(ctx SpecContext) {
			s := open(ctx, storageConfig{Channels: all, Start: 10 * telem.SecondTS})
			written := MustSucceed(s.Write(ctx, both(
				[]int64{10, 11}, []int64{10, 11, 12}, []int64{1, 2}, []int64{5, 6, 7},
			)))
			Expect(written.KeysSlice()).To(ConsistOf(i1, d1, i2, d2))
			g := func(k cesium.ChannelKey) telem.Series { return written.Get(k).Series[0] }
			Expect(g(d1).Alignment).To(Equal(g(i1).Alignment))
			Expect(g(d2).Alignment).To(Equal(g(i2).Alignment))
			Expect(
				g(d1).TimeRange,
			).To(Equal((10 * telem.SecondTS).Range(11*telem.SecondTS + 1)))
			Expect(
				g(d2).TimeRange,
			).To(Equal((10 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
			Expect(MustSucceed(s.Commit(ctx))).To(Equal(12*telem.SecondTS + 1))
			Expect(read(ctx, d1).Len()).To(Equal(int64(2)))
			Expect(read(ctx, d2).Len()).To(Equal(int64(3)))
		},
	)

	It("Should write a frame that leaves one clock out", func(ctx SpecContext) {
		s := open(ctx, storageConfig{Channels: all, Start: 10 * telem.SecondTS})
		written := MustSucceed(s.Write(ctx, telem.MultiFrame(
			[]cesium.ChannelKey{i2, d2},
			[]telem.Series{ts(10, 11), telem.NewSeriesV[int64](1, 2)},
		)))
		Expect(written.KeysSlice()).To(ConsistOf(i2, d2))
		Expect(MustSucceed(s.Commit(ctx))).To(Equal(11*telem.SecondTS + 1))
		Expect(read(ctx, d1).Len()).To(BeZero())
		Expect(read(ctx, d2).Len()).To(Equal(int64(2)))
	})

	It(
		"Should commit one clock when the other is refused at commit",
		func(ctx SpecContext) {
			s := MustSucceed(db.svc.Open(ctx, writerConfig(all, 100, "s")))
			DeferCleanup(func() { _ = s.Close() })
			Expect(s.Write(frameOf(both(
				[]int64{10, 11}, []int64{10, 11, 12}, []int64{1, 2}, []int64{5, 6, 7},
			)))).To(BeTrue())
			<-db.writes.Outlet()
			open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "h"},
				Channels:       []cesium.ChannelKey{i1, d1},
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{200},
			})
			end, authorized, err := s.Commit()
			Expect(err).ToNot(HaveOccurred())
			Expect(authorized).To(BeFalse())
			Expect(end).To(BeZero())
			Expect(read(ctx, d1).Len()).To(BeZero())
			Expect(read(ctx, d2).Len()).To(Equal(int64(3)))
		},
	)

	It(
		"Should write one clock when the other is refused at write",
		func(ctx SpecContext) {
			open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "h"},
				Channels:       []cesium.ChannelKey{i1, d1},
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{200},
			})
			s := MustSucceed(db.svc.Open(ctx, writerConfig(all, 100, "s")))
			DeferCleanup(func() { _ = s.Close() })
			Expect(s.Write(frameOf(both(
				[]int64{10, 11}, []int64{10, 11, 12}, []int64{1, 2}, []int64{5, 6, 7},
			)))).To(BeFalse())
			res := <-db.writes.Outlet()
			Expect(res.Frame.ToStorage().KeysSlice()).To(ConsistOf(i2, d2))
			end, authorized, err := s.Commit()
			Expect(err).ToNot(HaveOccurred())
			Expect(authorized).To(BeFalse())
			Expect(end).To(Equal(12*telem.SecondTS + 1))
			Expect(read(ctx, d2).Len()).To(Equal(int64(3)))
		},
	)

	It("Should end the session on the error of one clock", func(ctx SpecContext) {
		s := open(ctx, storageConfig{Channels: all, Start: 10 * telem.SecondTS})
		_, err := s.Write(ctx, both(
			[]int64{10, 11}, []int64{12, 11}, []int64{1, 2}, []int64{5, 6},
		))
		Expect(err).To(MatchError(ContainSubstring("index out of order")))
		Expect(read(ctx, d1).Len()).To(BeZero())
		Expect(read(ctx, d2).Len()).To(BeZero())
	})

	It("Should end the domain of one clock of two", func(ctx SpecContext) {
		s := open(ctx, storageConfig{Channels: all, Start: 10 * telem.SecondTS})
		MustSucceed(s.Write(ctx, both(
			[]int64{10, 11}, []int64{10, 11}, []int64{1, 2}, []int64{5, 6},
		)))
		Expect(s.EndDomain(ctx, i1)).To(Succeed())
		Expect(read(ctx, d1).Len()).To(Equal(int64(2)))
		Expect(read(ctx, d2).Len()).To(BeZero())
		MustSucceed(s.Write(ctx, both(
			[]int64{12}, []int64{12}, []int64{3}, []int64{7},
		)))
		Expect(MustSucceed(s.Commit(ctx))).To(Equal(12*telem.SecondTS + 1))
		Expect(read(ctx, d1).Series).To(HaveLen(2))
		Expect(read(ctx, d2).Series).To(HaveLen(1))
		Expect(read(ctx, d2).Len()).To(Equal(int64(3)))
	})

	It("Should hand off one clock while the other clock writes", func(ctx SpecContext) {
		s := MustSucceed(db.svc.Open(ctx, writerConfig(all, 100, "s")))
		DeferCleanup(func() { _ = s.Close() })
		write := func(a1, a2, v1, v2 []int64) bool {
			GinkgoHelper()
			ok := MustSucceed(s.Write(frameOf(both(a1, a2, v1, v2))))
			<-db.writes.Outlet()
			return ok
		}
		Expect(write([]int64{10}, []int64{10}, []int64{1}, []int64{1})).To(BeTrue())
		end, authorized, err := s.Commit()
		Expect(err).ToNot(HaveOccurred())
		Expect(authorized).To(BeTrue())
		Expect(end).To(Equal(10*telem.SecondTS + 1))
		h := open(ctx, storageConfig{
			ControlSubject: control.Subject{Key: "h"},
			Channels:       []cesium.ChannelKey{i1, d1},
			Start:          11 * telem.SecondTS,
			Authorities:    []control.Authority{200},
		})
		Expect(write([]int64{11}, []int64{11}, []int64{2}, []int64{2})).To(BeFalse())
		MustSucceed(
			h.Write(ctx, telem.MultiFrame([]cesium.ChannelKey{i1, d1}, []telem.Series{
				ts(12), telem.NewSeriesV[int64](9),
			})),
		)
		Expect(MustSucceed(h.Commit(ctx))).To(Equal(12*telem.SecondTS + 1))
		end, authorized, err = s.Commit()
		Expect(err).ToNot(HaveOccurred())
		Expect(authorized).To(BeFalse())
		Expect(end).To(Equal(11*telem.SecondTS + 1))
		Expect(h.Close()).To(Succeed())
		Expect(write([]int64{13}, []int64{13}, []int64{3}, []int64{3})).To(BeTrue())
		end, authorized, err = s.Commit()
		Expect(err).ToNot(HaveOccurred())
		Expect(authorized).To(BeTrue())
		Expect(end).To(Equal(13*telem.SecondTS + 1))
		Expect(read(ctx, d1).Len()).To(Equal(int64(3)))
		Expect(read(ctx, d2).Len()).To(Equal(int64(3)))
	})

	It(
		"Should release the clocks it opened when another clock fails to open",
		func(ctx SpecContext) {
			h := open(
				ctx,
				storageConfig{
					Channels: []cesium.ChannelKey{i2, d2},
					Start:    10 * telem.SecondTS,
				},
			)
			MustSucceed(
				h.Write(
					ctx,
					telem.MultiFrame([]cesium.ChannelKey{i2, d2}, []telem.Series{
						ts(10, 11, 12), telem.NewSeriesV[int64](1, 2, 3),
					}),
				),
			)
			MustSucceed(h.Commit(ctx))
			Expect(h.Close()).To(Succeed())
			_, err := db.OpenWriter(
				ctx,
				storageConfig{Channels: all, Start: 11 * telem.SecondTS},
			)
			Expect(err).To(HaveOccurred())
			fill := open(ctx, storageConfig{
				Channels: []cesium.ChannelKey{i1, d1},
				Start:    11 * telem.SecondTS,
				Mode:     writer.ModePersistOnly,
			})
			MustSucceed(
				fill.Write(
					ctx,
					telem.MultiFrame([]cesium.ChannelKey{i1, d1}, []telem.Series{
						ts(11, 12), telem.NewSeriesV[int64](1, 2),
					}),
				),
			)
			Expect(MustSucceed(fill.Commit(ctx))).To(Equal(12*telem.SecondTS + 1))
		},
	)

	It(
		"Should write data whose index the session lacks beside another clock",
		func(ctx SpecContext) {
			h := open(
				ctx,
				storageConfig{
					Channels: []cesium.ChannelKey{i1},
					Start:    10 * telem.SecondTS,
				},
			)
			MustSucceed(h.Write(ctx, telem.UnaryFrame(i1, ts(10, 11, 12))))
			MustSucceed(h.Commit(ctx))
			Expect(h.Close()).To(Succeed())
			keys := []cesium.ChannelKey{d1, i2, d2}
			s := open(ctx, storageConfig{Channels: keys, Start: 10 * telem.SecondTS})
			MustSucceed(s.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesV[int64](
					1,
					2,
					3,
				),
				ts(20, 21),
				telem.NewSeriesV[int64](5, 6),
			})))
			Expect(MustSucceed(s.Commit(ctx))).To(Equal(21*telem.SecondTS + 1))
			Expect(read(ctx, d1).Len()).To(Equal(int64(3)))
			Expect(read(ctx, d2).Len()).To(Equal(int64(2)))
		},
	)
})
