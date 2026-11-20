// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Log end check", func() {
	// frameEnd returns the frame end of an entry of n Drop records, the first frame of
	// a
	// log: the frame header, the entry header, and 16 bytes per record.
	frameEnd := func(n int) int64 {
		return log.FramesStart + log.FrameHeaderSize + 8 + 16*int64(n)
	}

	DescribeTable(
		"Should keep the log file at its size, with a clean last frame, after a delete "+
			"near the end",
		func(n int, full bool) {
			var inner xfs.FS = xfs.NewMem()
			if full {
				inner = MustSucceed(fault.New(fault.Config{Capacity: logSize}))
			}
			fs := newOpFS(inner)
			Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
			l := MustSucceed(log.Start(log.Config{
				FS:         fs,
				Executor:   goExec,
				DB:         db,
				Generation: generation,
				Sequence:   5,
			}))
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			ran := make(chan error, 1)
			go func() { ran <- l.Run(ctx) }()
			var e log.Entry
			for i := range n {
				e.Add(&log.Drop{Key: uint32(i + 1)})
			}
			if _, err := l.Append(&e).Wait(); err == nil {
				// The idle mark after the frame.
				Eventually(func() int { return len(walk(fileBytes(fs))) }).
					Should(Equal(2))
			} else {
				Expect(err).To(MatchError(log.ErrFull))
			}
			Expect(l.Close()).To(Succeed())
			Eventually(ran).Should(Receive(BeNil()))
			Expect(MustSucceed(fs.Stat(logName)).Size()).To(Equal(int64(logSize)))
			frames := walk(fileBytes(fs))
			Expect(frames[len(frames)-1].Clean).To(BeTrue())
		},
		func(n int, full bool) string {
			return fmt.Sprintf("frame end %d bytes before the file end, full disk %t",
				logSize-frameEnd(n), full)
		},
		Entry(nil, 129_789, false),
		Entry(nil, 130_045, false),
		Entry(nil, 130_046, false),
		Entry(nil, 130_301, false),
		Entry(nil, 130_045, true),
		Entry(nil, 130_301, true),
	)

	It("Should refuse a skippable record whose body is not a multiple of 8 bytes",
		func() {
			var d log.Decoder
			_, err := d.Reset(rawEntry(raw{1<<15 | 40, []byte{1, 2, 3}}))
			if err == nil {
				_, _, err = d.Next()
			}
			Expect(err).To(MatchError(place.ErrCorrupt))
		},
	)
})
