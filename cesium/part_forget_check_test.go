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
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// segmentNames returns the names of the segment files of key in fs, in rising order.
func segmentNames(fs xfs.FS, key cesium.ChannelKey) []string {
	GinkgoHelper()
	var names []string
	for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
		if strings.HasSuffix(info.Name(), ".seg") {
			names = append(names, info.Name())
		}
	}
	slices.Sort(names)
	return names
}

// segmentFiles returns the number of segment files of key in fs.
func segmentFiles(fs xfs.FS, key cesium.ChannelKey) int {
	GinkgoHelper()
	return len(segmentNames(fs, key))
}

// framer is the part of an iterator that iterated reads.
type framer interface {
	Next(span telem.TimeSpan) bool
	Value() cesium.Frame
	Error() error
}

// iterated returns the values of key that iter reads from its position on.
func iterated(iter framer, key cesium.ChannelKey) []int64 {
	GinkgoHelper()
	var out []int64
	for iter.Next(cesium.AutoSpan) {
		for _, s := range iter.Value().Get(key).Series {
			for i := 0; i+8 <= len(s.Data); i += 8 {
				out = append(out, int64(binary.LittleEndian.Uint64(s.Data[i:])))
			}
		}
	}
	Expect(iter.Error()).ToNot(HaveOccurred())
	return out
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
		Expect(iterated(iter, data)).To(HaveLen(30))
		Expect(iter.Close()).To(Succeed())
		Expect(segmentFiles(fs, data)).To(Equal(2))
	})

	It("Should keep the file of a parted segment that two deletes empty while an "+
		"iterator that loaded its record reads", func(ctx SpecContext) {
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
		// The first parted segment has the lowest identity.
		first := segmentNames(fs, data)[0]
		// The first delete cuts the first parted segment, which then stays in the
		// state, and the second empties it. The cut can make the segment a candidate,
		// whose rewrite adds a file until the iterator closes or the copy goes stale,
		// so the check names the file that the iterator pins.
		for _, tr := range []telem.TimeRange{
			{Start: telem.SecondTS, End: 6 * telem.SecondTS},
			{Start: 6 * telem.SecondTS, End: 11 * telem.SecondTS},
		} {
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, tr)).To(Succeed())
		}
		Expect(segmentNames(fs, data)).
			To(ContainElement(first), "the iterator pins the segment")
		Expect(iterated(iter, data)).To(HaveLen(30))
		Expect(iter.Close()).To(Succeed())
		Eventually(func() []string { return segmentNames(fs, data) }).
			Should(SatisfyAll(HaveLen(2), Not(ContainElement(first))))
	})

	It("Should keep the file of a parted segment that a delete empties after a switch "+
		"writes its record to a new part, while an iterator that loaded the old part "+
		"reads", func(ctx SpecContext) {
		fs := headHoldFS{
			FS:      xfs.NewMem(),
			held:    make(chan struct{}, 1),
			release: make(chan struct{}),
			cleans:  &atomic.Int64{},
		}
		idx, data := closeParted(ctx, fs)
		db := partedDB(ctx, fs)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		release := sync.OnceFunc(func() { close(fs.release) })
		DeferCleanup(release)
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{data},
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		// The kills of the second segments of both keys leave part 1 half dead, so
		// switch 3 writes the first segments to a new part.
		Expect(db.DeleteTimeRange(
			ctx,
			[]cesium.ChannelKey{idx, data},
			telem.TimeRange{Start: 11 * telem.SecondTS, End: 21 * telem.SecondTS},
		)).To(Succeed())
		switched := make(chan error, 1)
		go func() { switched <- switchLog(db, fs, 3) }()
		Eventually(fs.held).WithTimeout(time.Minute).Should(Receive())
		cleans := fs.cleans.Load()
		release()
		Eventually(switched).WithTimeout(time.Minute).Should(Receive(Succeed()))
		// The checkpointer hands the head to the state before its clean, and a
		// writer's open waits for the state owner.
		Eventually(fs.cleans.Load).Should(BeNumerically(">", cleans))
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Start:    1_000 * telem.SecondTS,
		}))
		Expect(w.Close()).To(Succeed())
		Expect(parted(fs, 3, data)).To(Equal([]uint64{2, 3}))
		Expect(db.DeleteTimeRange(
			ctx,
			[]cesium.ChannelKey{data},
			telem.TimeRange{Start: telem.SecondTS, End: 11 * telem.SecondTS},
		)).To(Succeed())
		Expect(segmentFiles(fs, data)).To(Equal(3), "the iterator pins part 1")
		Expect(iter.Close()).To(Succeed())
		Expect(segmentFiles(fs, data)).To(Equal(1))
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
