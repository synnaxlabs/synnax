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
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// parted returns the part that holds each segment of key in the head of generation in
// fs, or 0 for a segment that the head holds.
func parted(fs xfs.FS, generation uint64, key cesium.ChannelKey) []uint64 {
	GinkgoHelper()
	h, s := MustSucceed2(checkpoint.Read(fs, generation))
	MustSucceed(checkpoint.LoadParts(fs, h.DB, h)).Merge(s)
	var parts []uint64
	for _, ch := range s.Channels {
		if ch.Key == key {
			for _, seg := range ch.Segments {
				parts = append(parts, seg.Part)
			}
		}
	}
	return parts
}

var _ = Describe("Check: claims over a parted segment that a delete cut", func() {
	It("Should take writes into the deleted range through its kills, then through the "+
		"dead marks of the switch that folds them, and refuse writes into what is left",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			db := partedDB(ctx, fs)
			stop := StartRun(db.Run)
			idx, data := GenerateChannelKey(), GenerateChannelKey()
			keys := []cesium.ChannelKey{idx, data}
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      data,
					Name:     "data",
					Index:    idx,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			// The samples sit at seconds 1 to 50 in five segments. Part 1 holds the
			// first
			// four of each key.
			writeParted(ctx, db, fs, idx, data, 5, 10)
			stop()
			Expect(db.Close()).To(Succeed())
			Expect(parted(fs, 2, data)).To(Equal([]uint64{1, 1, 1, 1, 0}))
			// writeAt writes one sample at each of seconds, and returns the error of
			// the
			// open or of the commit.
			writeAt := func(seconds ...telem.TimeStamp) error {
				w, err := db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Clock:    idx,
					Start:    seconds[0] * telem.SecondTS,
				})
				if err != nil {
					return err
				}
				values := make([]int64, len(seconds))
				for i, s := range seconds {
					values[i] = -int64(s)
				}
				_, _, err = w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(seconds...),
					telem.NewSeries(values),
				}), nil)
				if err == nil {
					_, _, err = w.Commit(ctx, nil)
				}
				return errors.Join(err, w.Close())
			}
			refused := SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("inside its stored data")),
			)

			db = partedDB(ctx, fs)
			stop = StartRun(db.Run)
			// The delete cuts segment 2 to seconds 11 and 12 and 17 to 20. Kills mark
			// its two records in part 1 dead, two of the eight that part 1 holds.
			Expect(db.DeleteTimeRange(
				ctx,
				keys,
				(13 * telem.SecondTS).Range(17*telem.SecondTS),
			)).To(Succeed())
			Expect(writeAt(13, 14)).To(Succeed())
			Expect(writeAt(5)).To(refused)
			Expect(writeAt(25)).To(refused)
			Expect(writeAt(18)).To(refused)
			// Switch 3 keeps part 1, with the two dead marks that the kills fold into.
			Expect(switchLog(db, fs, 3)).To(Succeed())
			waitHead(fs, 3)
			stop()
			Expect(db.Close()).To(Succeed())
			h, _ := MustSucceed2(checkpoint.Read(fs, 3))
			Expect(h.Parts).To(ContainElement(SatisfyAll(
				HaveField("ID", uint64(1)),
				HaveField("Dead", Not(BeEmpty())),
			)))

			db = partedDB(ctx, fs)
			DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
			DeferCleanup(StartRun(db.Run))
			Expect(writeAt(15, 16)).To(Succeed())
			Expect(writeAt(5)).To(refused)
			Expect(writeAt(25)).To(refused)
			Expect(writeAt(13)).To(refused)
			Expect(writeAt(19)).To(refused)
			want := make([]int64, 0, 50)
			for s := range int64(12) {
				want = append(want, s)
			}
			want = append(want, -13, -14, -15, -16)
			for s := int64(16); s < 50; s++ {
				want = append(want, s)
			}
			Expect(int64s(ctx, db, data)).To(Equal(want))
		})
})
