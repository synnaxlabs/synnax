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
	"encoding/binary"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: data against stored stamps across a clock roll", func() {
	It("Should write one run across a roll of the clock's segment, and grow a run "+
		"across two parts", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithPartSize(1),
		))
		stop := StartRun(db.Run)
		defer func() {
			stop()
			Expect(db.Close()).To(Succeed())
		}()
		idx, d1, d2 := cesium.ChannelKey(1), cesium.ChannelKey(2), cesium.ChannelKey(3)
		clockKeys := []cesium.ChannelKey{idx}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: d1, Name: "d1", Index: idx, DataType: telem.Int64T},
			cesium.Channel{Key: d2, Name: "d2", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		// Sample i of the clock has stamp 2(i + 1). Its segment reaches 2^30 bytes, and
		// rolls inside the stretch, after 2^27 samples.
		const (
			chunk = 1 << 22
			roll  = 1 << 27
		)
		stamp := func(i int) telem.TimeStamp { return telem.TimeStamp(2 * (i + 1)) }
		c := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: clockKeys,
			Clock:    idx,
			Start:    stamp(0),
		}))
		stamps := make([]telem.TimeStamp, chunk)
		var first telem.Alignment
		write := func(from, n int) {
			GinkgoHelper()
			for i := range n {
				stamps[i] = stamp(from + i)
			}
			fr, _ := MustSucceed2(c.Write(ctx, telem.MultiFrame(
				clockKeys,
				[]telem.Series{telem.NewSeries(stamps[:n])},
			), nil))
			if from == 0 {
				first = fr.Get(idx).Series[0].Alignment
			}
			MustSucceed2(c.Commit(ctx, nil))
		}
		// The clock writer stays open with its stamps up to stamp(roll - 2).
		for from := 0; from+chunk < roll; from += chunk {
			write(from, chunk)
		}
		write(roll-chunk, chunk-1)
		// read returns the values and the alignment of the one series of key.
		read := func(key cesium.ChannelKey) ([]int64, telem.Alignment) {
			GinkgoHelper()
			series := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key).Series
			Expect(series).To(HaveLen(1))
			got := make([]int64, 0, series[0].Len())
			for i := 0; i+8 <= len(series[0].Data); i += 8 {
				got = append(got, int64(binary.LittleEndian.Uint64(series[0].Data[i:])))
			}
			return got, series[0].Alignment
		}
		open := func(key cesium.ChannelKey) *cesium.Writer {
			GinkgoHelper()
			return MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{key},
				Clock:    idx,
				Start:    stamp(roll - 4),
			}))
		}
		// streams writes values to key with w in one Write, and checks that they stream
		// as one series with alignment a.
		streams := func(
			w *cesium.Writer,
			key cesium.ChannelKey,
			a telem.Alignment,
			values ...int64,
		) {
			GinkgoHelper()
			fr, _ := MustSucceed2(w.Write(ctx, telem.MultiFrame(
				[]cesium.ChannelKey{key},
				[]telem.Series{telem.NewSeries(values)},
			), nil))
			Expect(fr.Get(key).Series).To(HaveLen(1))
			Expect(fr.Get(key).Series[0].Alignment).To(Equal(a))
		}
		// d2 takes a run whose stored stamps end at stamp(roll - 2).
		w2 := open(d2)
		streams(w2, d2, first.AddSamples(roll-4), 0, 1)
		// The clock passes the roll: segment A seals at 2^27 stamps, and B holds the
		// rest.
		write(roll-1, 17)
		Expect(c.Sync(ctx)).To(Succeed())
		// M8: one Write across the roll, with A and B in the head, is one series.
		w1 := open(d1)
		streams(w1, d1, first.AddSamples(roll-4), 0, 1, 2, 3, 4, 5, 6, 7)
		Expect(MustSucceed2(w1.Commit(ctx, nil))).To(Equal(stamp(roll+3) + 1))
		Expect(w1.Close()).To(Succeed())
		values, a := read(d1)
		Expect(values).To(Equal([]int64{0, 1, 2, 3, 4, 5, 6, 7}))
		Expect(a).To(Equal(first.AddSamples(roll - 4)))
		// switched fills the log with Intends until log next exists, and returns the
		// head that the switch to the log before it wrote.
		switched := func(next uint64) checkpoint.Header {
			GinkgoHelper()
			for range 20_000 {
				if MustSucceed(fs.Exists(log.Name(next))) {
					break
				}
				Expect(db.Intend([]cesium.ChannelKey{999})).To(Succeed())
			}
			var h checkpoint.Header
			Eventually(func() error {
				var err error
				h, _, err = checkpoint.Read(fs, next-1)
				return err
			}).WithTimeout(time.Minute).Should(Succeed())
			return h
		}
		// M15: a switch parts A. The clock writer then fills B to 2^27 stamps, so B
		// seals in turn, and a second switch parts B into a later part, which the first
		// bounds of d2's run miss.
		parted := switched(3)
		Expect(parted.Parts).ToNot(BeEmpty())
		for from := roll + 16; from < 2*roll+16; from += chunk {
			write(from, chunk)
		}
		Expect(c.Close()).To(Succeed())
		// The clock holds A and B, in rising order. The commit that fills B seals it,
		// and no write follows to start another.
		var segments []uint64
		for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(idx), 10))) {
			if hex, ok := strings.CutSuffix(info.Name(), ".seg"); ok {
				segments = append(segments, MustSucceed(strconv.ParseUint(hex, 16, 64)))
			}
		}
		slices.Sort(segments)
		Expect(segments).To(HaveLen(2))
		Expect(len(switched(4).Parts)).To(BeNumerically(">", len(parted.Parts)))
		_, head := MustSucceed2(checkpoint.Read(fs, 3))
		var inHead []uint64
		for _, ch := range head.Channels {
			if ch.Key == idx {
				for _, seg := range ch.Segments {
					inHead = append(inHead, seg.ID)
				}
			}
		}
		Expect(inHead).ToNot(ContainElement(segments[1]), "segment B is in the head")
		streams(w2, d2, first.AddSamples(roll-2), 2, 3, 4, 5, 6, 7)
		Expect(MustSucceed2(w2.Commit(ctx, nil))).To(Equal(stamp(roll+3) + 1))
		Expect(w2.Close()).To(Succeed())
		values, a = read(d2)
		Expect(values).To(Equal([]int64{0, 1, 2, 3, 4, 5, 6, 7}))
		Expect(a).To(Equal(first.AddSamples(roll - 4)))
	})
})
