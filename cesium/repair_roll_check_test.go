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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// rollChunk is the most samples of one Write of a roll spec.
const rollChunk = 1 << 20

// fill writes the samples from position p to p+n to keys, a clock and a key of type dt,
// in frames of at most rollChunk samples.
func fill(
	ctx context.Context,
	w *cesium.Writer,
	keys []cesium.ChannelKey,
	dt telem.DataType,
	p, n int,
) {
	GinkgoHelper()
	density := int(dt.Density())
	for from := p; from < p+n; from += rollChunk {
		m := min(rollChunk, p+n-from)
		stamps, data := make([]telem.TimeStamp, m), make([]byte, m*density)
		for i := range m {
			stamps[i], data[i*density] = repairStamp(from+i), byte(from+i)
		}
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			{DataType: dt, Data: data},
		}), nil))
	}
}

// rollRoot returns a new root with an index and a key of type dt that it indexes.
func rollRoot(
	ctx context.Context,
	dt telem.DataType,
) (repairRoot, []cesium.ChannelKey) {
	GinkgoHelper()
	r := newRepairRoot(ctx)
	keys := []cesium.ChannelKey{GenerateChannelKey(), GenerateChannelKey()}
	Expect(r.db.CreateChannel(
		ctx,
		cesium.Channel{
			Key:      keys[0],
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		},
		cesium.Channel{Key: keys[1], Name: "data", Index: keys[0], DataType: dt},
	)).To(Succeed())
	return r, keys
}

// Each spec holds over 1 GiB of segments in memory.
var _ = Describe("Check: repair of a roll in the gap", func() {
	It("Should place the new segment of a key that rolled without its clock through "+
		"the clock of its own segments", func(ctx SpecContext) {
		// A key of 16-byte samples fills its 1 GiB segment at 2^26 samples, when its
		// clock holds half of that.
		const rolled, lost = 1 << 26, 5000
		r, keys := rollRoot(ctx, telem.UUIDT)
		w := MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
		))
		fill(ctx, w, keys, telem.UUIDT, 0, rolled)
		r.synced(ctx, w, 0)
		// The new segment holds a full block of the lost commit.
		fill(ctx, w, keys, telem.UUIDT, rolled, lost)
		bad := r.synced(ctx, w, rolled)
		Expect(w.Close()).To(Succeed())
		// The next writer seals both open segments that it takes over, so no Commit
		// names the new segment.
		w = MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(rolled + lost)},
		))
		fill(ctx, w, keys, telem.UUIDT, rolled+lost, 16)
		r.synced(ctx, w, rolled+lost)
		Expect(w.Close()).To(Succeed())
		id := r.spoil(bad)
		clocks, data := segmentsOf(r.fs, keys[0]), segmentsOf(r.fs, keys[1])
		Expect(clocks).To(HaveLen(2))
		Expect(data).To(HaveLen(3))
		rep := repairOf(ctx, r.fs, id)
		tr := telem.TimeRange{
			Start: repairStamp(rolled),
			End:   repairStamp(rolled+lost-1) + 1,
		}
		Expect(rep.Unrecovered).To(BeEmpty())
		Expect(rep.Rebuilt).To(ConsistOf(
			cesium.Range{
				Key:       keys[0],
				Segment:   clocks[0],
				Bytes:     8 * lost,
				Samples:   lost,
				TimeRange: tr,
			},
			cesium.Range{
				Key:       keys[1],
				Segment:   data[1],
				Bytes:     16 * lost,
				Samples:   lost,
				TimeRange: tr,
			},
		))
		Expect(rep.Unplaced).To(BeEmpty())
		Expect(rep.Tails).To(BeEmpty())
	})

	It("Should leave unplaced every range of a clock write whose clock rolled in the "+
		"bad frame", func(ctx SpecContext) {
		// A clock fills its 1 GiB segment at 2^27 stamps.
		const rolled = 1 << 27
		r, keys := rollRoot(ctx, telem.Uint8T)
		w := MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
		))
		fill(ctx, w, keys, telem.Uint8T, 0, 16)
		r.synced(ctx, w, 0)
		// The commit that fills the old segment of the clock, its Seal, and a commit
		// of a full block to the new segment share the bad frame.
		fill(ctx, w, keys, telem.Uint8T, 16, rolled-16)
		MustSucceed2(w.Commit(ctx, nil))
		fill(ctx, w, keys, telem.Uint8T, rolled, commits)
		bad := r.synced(ctx, w, rolled)
		Expect(commitAt(r.fs, r.db.Identity(), repairStamp(16))).To(Equal(bad))
		fill(ctx, w, keys, telem.Uint8T, rolled+commits, 16)
		r.synced(ctx, w, rolled+commits)
		Expect(w.Close()).To(Succeed())
		id := r.spoil(bad)
		clocks := segmentsOf(r.fs, keys[0])
		Expect(clocks).To(HaveLen(2))
		rep := repairOf(ctx, r.fs, id)
		two := "its clock has more than one segment with bytes in the gap"
		Expect(rep.Rebuilt).To(BeEmpty())
		Expect(rep.Unplaced).To(ConsistOf(
			cesium.Range{
				Key:     keys[0],
				Segment: clocks[0],
				Bytes:   8 * (rolled - 16),
				Samples: rolled - 16,
				Open:    true,
				Reason:  two,
			},
			cesium.Range{
				Key:     keys[0],
				Segment: clocks[1],
				Bytes:   8 * commits,
				Samples: commits,
				Reason:  two,
			},
			cesium.Range{
				Key:     keys[1],
				Segment: onlySegment(r.fs, keys[1]),
				Bytes:   rolled - 16 + commits,
				Samples: rolled - 16 + commits,
				Reason:  two,
			},
		))
		Expect(rep.Tails).To(BeEmpty())
		Expect(rep.Unrecovered).To(BeEmpty())
	})
})
