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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Breaker check", func() {
	It("Should keep the clean mark inside the log file after a delete fills "+
		"the last block before the end", func() {
		disk := MustSucceed(fault.New(fault.Config{Capacity: logSize}))
		fs := &opFS{FS: disk, held: make(chan struct{})}
		Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
		l := MustSucceed(log.Start(log.Config{
			FS:         fs,
			DB:         db,
			Generation: generation,
			Sequence:   5,
			Size:       logSize,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		ran := make(chan error, 1)
		go func() { ran <- l.Run(ctx) }()
		var e log.Entry
		for i := range 130_301 {
			e.Add(&log.Drop{Key: uint32(i + 1)})
		}
		entry := e.Bytes()
		end := int64(log.FramesStart + log.FrameHeaderSize + len(entry))
		Expect(end).To(Equal(int64(logSize - log.BlockSize - 8)))
		Expect(l.Append(entry, true).Wait()).To(Equal(uint64(5)))
		Eventually(func() []int64 { return writes(fs.Ops()) }).
			Should(Equal([]int64{log.FramesStart, logSize - log.BlockSize}))
		Expect(l.Close()).To(Succeed())
		Expect(<-ran).To(Succeed())
		Expect(writes(fs.Ops())).To(HaveEach(BeNumerically("<", logSize)))
		Expect(MustSucceed(fs.Stat(logName)).Size()).To(Equal(int64(logSize)))
	})

	It("Should keep the log file at the size its header records after a delete "+
		"fills the last block before the end", func() {
		fs := newOpFS()
		Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
		l := MustSucceed(log.Start(log.Config{
			FS:         fs,
			DB:         db,
			Generation: generation,
			Sequence:   5,
			Size:       logSize,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		ran := make(chan error, 1)
		go func() { ran <- l.Run(ctx) }()
		var e log.Entry
		for i := range 130_301 {
			e.Add(&log.Drop{Key: uint32(i + 1)})
		}
		Expect(l.Append(e.Bytes(), true).Wait()).To(Equal(uint64(5)))
		Eventually(func() []int64 { return writes(fs.Ops()) }).
			Should(Equal([]int64{log.FramesStart, logSize - log.BlockSize}))
		Expect(l.Close()).To(Succeed())
		Expect(<-ran).To(Succeed())
		h := MustSucceed(log.DecodeHeader(fileBytes(fs), 0))
		Expect(MustSucceed(fs.Stat(logName)).Size()).To(Equal(int64(h.Size)))
	})

	DescribeTable(
		"Should refuse a commit whose alignment or stamps no write can give",
		func(samples uint32, alignment uint64, first, last int64) {
			b := rawEntry(raw{3, concat(
				le32(1, samples),
				le64(alignment, uint64(first), uint64(last), 2),
				le32(24, 7),
			)})
			Expect(decodeAll(b)).Error().To(MatchError(place.ErrCorrupt))
		},
		Entry(
			"one sample with two stamps",
			uint32(1),
			uint64(5<<32),
			int64(10),
			int64(20),
		),
		Entry(
			"a sample index past 2^32 - 1",
			uint32(2),
			uint64(5<<32|0xFFFFFFFF),
			int64(10),
			int64(20),
		),
		Entry("alignment number 0", uint32(1), uint64(7), int64(10), int64(10)),
		Entry(
			"an alignment number of unstored data",
			uint32(1),
			uint64(1<<31)<<32,
			int64(10),
			int64(10),
		),
	)

	It("Should start every frame on the 8-byte grid that the torn tail scan reads",
		func() {
			fs := newOpFS()
			Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
			l := MustSucceed(log.Start(log.Config{
				FS:         fs,
				DB:         db,
				Generation: generation,
				Sequence:   1,
				Size:       logSize,
			}))
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			ran := make(chan error, 1)
			go func() { ran <- l.Run(ctx) }()
			odd := rawEntry(raw{1<<15 | 40, []byte{1, 2, 3}}, raw{7, le32(5, 0)})
			Expect(decodeAll(odd)).To(Equal([]any{log.Drop{Key: 5}}))
			release := fs.Hold()
			first := l.Append(odd, false)
			Eventually(fs.held).Should(Receive())
			second := l.Append(entryOf(&log.Drop{Key: 6}), false)
			fs.Unhold()
			release <- struct{}{}
			Expect(first.Wait()).To(Equal(uint64(1)))
			Expect(second.Wait()).To(Equal(uint64(2)))
			Expect(l.Close()).To(Succeed())
			Expect(<-ran).To(Succeed())
			Expect(writes(fs.Ops())).To(HaveEach(WithTransform(
				func(off int64) int64 { return off % 8 },
				BeZero(),
			)))
		})
})
