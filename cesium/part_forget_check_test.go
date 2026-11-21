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
	"encoding/binary"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// closeParted writes three segments of samples of a data channel timed by an index to
// a database on fs, moves the first two to a part, and closes the database. A reopen
// then holds only the third in memory.
func closeParted(ctx context.Context, fs xfs.FS) (idx, data cesium.ChannelKey) {
	GinkgoHelper()
	db := partedDB(ctx, fs)
	stop := StartRun(db.Run)
	idx, data = GenerateChannelKey(), GenerateChannelKey()
	Expect(db.CreateChannel(
		ctx,
		cesium.Channel{
			Key:      idx,
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		},
		cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
	)).To(Succeed())
	writeParted(ctx, db, fs, idx, data, 3, 10)
	stop()
	Expect(db.Close()).To(Succeed())
	Expect(parted(fs, 2, data)).To(Equal([]uint64{1, 1, 0}))
	return idx, data
}

// segmentFiles returns the number of segment files of key in fs.
func segmentFiles(fs xfs.FS, key cesium.ChannelKey) int {
	GinkgoHelper()
	n := 0
	for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
		if strings.HasSuffix(info.Name(), ".seg") {
			n++
		}
	}
	return n
}

var _ = Describe("Check: segments that only parts hold", func() {
	It("Should keep the file of a parted segment that a delete empties while an "+
		"iterator that loaded it reads", func(ctx SpecContext) {
		fs := xfs.NewMem()
		_, data := closeParted(ctx, fs)
		db := partedDB(ctx, fs)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{data},
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		// Seconds 1 to 10 are the first parted segment.
		Expect(db.DeleteTimeRange(
			ctx,
			[]cesium.ChannelKey{data},
			telem.TimeRange{Start: telem.SecondTS, End: 11 * telem.SecondTS},
		)).To(Succeed())
		Expect(segmentFiles(fs, data)).To(Equal(3), "the iterator pins the segment")
		var read []int64
		for iter.Next(cesium.AutoSpan) {
			for _, s := range iter.Value().Get(data).Series {
				for i := 0; i+8 <= len(s.Data); i += 8 {
					read = append(read, int64(binary.LittleEndian.Uint64(s.Data[i:])))
				}
			}
		}
		Expect(iter.Error()).ToNot(HaveOccurred())
		Expect(read).To(HaveLen(30))
		Expect(iter.Close()).To(Succeed())
		Expect(segmentFiles(fs, data)).To(Equal(2))
	})

	It("Should drop in a sweep a key with no record whose parted samples a delete "+
		"took out", func(ctx SpecContext) {
		fs := xfs.NewMem()
		idx, data := closeParted(ctx, fs)
		db := partedDB(ctx, fs)
		stop := StartRun(db.Run)
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, telem.TimeRangeMax)).
			To(Succeed())
		Expect(int64s(ctx, db, data)).To(BeEmpty())
		stop()
		Expect(db.Close()).To(Succeed())
		reopened := partedDB(ctx, fs)
		DeferCleanup(func() { Expect(reopened.Close()).To(Succeed()) })
		DeferCleanup(StartRun(reopened.Run))
		kept, _ := MustSucceed2(reopened.Sweep(ctx, []cesium.ChannelKey{idx}))
		Expect(kept).To(BeEmpty())
	})
})
