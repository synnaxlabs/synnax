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
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

const logSize = 2 << 20

var logName = log.Name(generation)

// fileBytes returns the bytes of the log file in fs.
func fileBytes(fs xfs.FS) []byte {
	GinkgoHelper()
	f := MustSucceed(fs.Open(logName, os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	return MustSucceed(io.ReadAll(f))
}

// walk returns the frames of a log file in order. It looks for each next frame at the
// end of the last one, then at the next block boundary, and checks the sequence and
// the link of each.
func walk(b []byte) []log.Frame {
	GinkgoHelper()
	h := MustSucceed(log.DecodeHeader(b, 0))
	var (
		frames []log.Frame
		at     = int64(log.FramesStart)
		seq    = h.Sequence
		link   = binary.LittleEndian.Uint32(b[60:])
	)
	for {
		f, err := log.DecodeFrame(b[at:], db, generation, uint64(at))
		if err != nil {
			aligned := (at + log.BlockSize - 1) &^ (log.BlockSize - 1)
			if aligned == at {
				return frames
			}
			at = aligned
			if f, err = log.DecodeFrame(
				b[at:],
				db,
				generation,
				uint64(at),
			); err != nil {
				return frames
			}
		}
		Expect(f.Sequence).To(Equal(seq), "frame at %d", at)
		Expect(f.PrevCRC).To(Equal(link), "frame at %d", at)
		frames = append(frames, f)
		seq += uint64(f.Count)
		link = f.CRC
		at += int64(f.Size())
	}
}

// writes returns the offsets of the frame writes that ops record after the second
// header copy.
func writes(ops []string) []int64 {
	var out []int64
	for i, op := range ops {
		if op == fmt.Sprintf(
			"write %s %d %d",
			logName,
			log.HeaderCopy,
			log.HeaderSize,
		) {
			out = nil
			ops = ops[i+1:]
			break
		}
	}
	for _, op := range ops {
		var (
			name string
			off  int64
			n    int
		)
		if _, err := fmt.Sscanf(op, "write %s %d %d", &name, &off, &n); err == nil &&
			off >= log.FramesStart {
			out = append(out, off)
		}
	}
	return out
}

var _ = Describe("Log", func() {
	var (
		fs  *opFS
		l   *log.Log
		ran chan error
	)

	open := func(size int64) {
		GinkgoHelper()
		Expect(log.Prepare(fs, generation, size)).To(Succeed())
		l = MustSucceed(log.Start(log.Config{
			FS:         fs,
			DB:         db,
			Generation: generation,
			Sequence:   5,
			PrevCRC:    0xBEEF,
			Size:       size,
		}))
		ran = make(chan error, 1)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { ran <- l.Run(ctx) }()
		DeferCleanup(cancel)
	}

	BeforeEach(func() { fs = newOpFS() })

	Describe("Prepare", func() {
		It("Should make a synced file of zeros and sync its entry", func() {
			Expect(log.Prepare(fs, generation, 3<<20)).To(Succeed())
			Expect(fileBytes(fs)).To(Equal(make([]byte, 3<<20)))
			Expect(fs.Ops()).To(HaveExactElements(
				"write "+logName+" 0 1048576",
				"write "+logName+" 1048576 1048576",
				"write "+logName+" 2097152 1048576",
				"sync "+logName,
				"syncdir ",
			))
		})
	})

	It("Should name a log by its generation in 16 hex digits", func() {
		Expect(log.Name(0xAB)).To(Equal("00000000000000ab.log"))
	})

	It("Should write and sync both header copies at start", func() {
		open(logSize)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		b := fileBytes(fs)
		want := log.Header{
			DB:         db,
			Generation: generation,
			Sequence:   5,
			Size:       logSize,
			PrevCRC:    0xBEEF,
		}
		Expect(log.DecodeHeader(b, 0)).To(Equal(want))
		Expect(log.DecodeHeader(b[log.HeaderCopy:], log.HeaderCopy)).To(Equal(want))
		Expect(fs.Ops()[len(fs.Ops())-1]).To(Equal("sync " + logName))
	})

	It("Should ack an entry with its sequence once its frame is synced", func() {
		open(logSize)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		entry := entryOf(&log.Drop{Key: 3})
		Expect(l.Append(entry, false).Wait()).To(Equal(uint64(5)))
		ops := fs.Ops()
		at := len(ops) - 1
		for ops[at] != fmt.Sprintf("write %s 8192 %d", logName, 32+len(entry)) {
			at--
		}
		Expect(ops[at+1]).To(Equal("sync " + logName))
		f := walk(fileBytes(fs))[0]
		Expect(f.Count).To(Equal(uint32(1)))
		Expect(f.Entries).To(Equal(entry))
	})

	It("Should mark an isolated frame at the next block, and pack the next frame "+
		"after the mark", func() {
		open(logSize)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		entry := entryOf(&log.Drop{Key: 3})
		Expect(l.Append(entry, false).Wait()).To(Equal(uint64(5)))
		Eventually(func() []int64 { return writes(fs.Ops()) }).
			Should(Equal([]int64{8_192, 12_288}))
		Expect(l.Append(entry, false).Wait()).To(Equal(uint64(6)))
		Eventually(func() []int64 { return writes(fs.Ops()) }).
			Should(Equal([]int64{8_192, 12_288, 12_320, 16_384}))
		frames := walk(fileBytes(fs))
		Expect(frames).To(HaveLen(4))
		Expect(frames[1].Count).To(BeZero())
		Expect(frames[1].Sequence).To(Equal(uint64(6)))
	})

	It("Should write every entry queued during a sync as one frame", func() {
		open(logSize)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		release := fs.Hold()
		first := l.Append(entryOf(&log.Drop{Key: 1}), false)
		Eventually(fs.held).Should(Receive())
		second := l.Append(entryOf(&log.Drop{Key: 2}), false)
		third := l.Append(entryOf(&log.Drop{Key: 3}), false)
		fs.Unhold()
		release <- struct{}{}
		Expect(first.Wait()).To(Equal(uint64(5)))
		Expect(second.Wait()).To(Equal(uint64(6)))
		Expect(third.Wait()).To(Equal(uint64(7)))
		frames := walk(fileBytes(fs))
		Expect(frames[0].Count).To(Equal(uint32(1)))
		Expect(frames[1].Count).To(Equal(uint32(2)))
		Expect(decodeAll(frames[1].Entries)).To(Equal([]any{log.Drop{Key: 2}}))
	})

	It("Should start a frame at the next block when its block holds a confirmed "+
		"frame", func() {
		open(logSize)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		entry := entryOf(&log.Drop{Key: 3})
		size := int64(32 + len(entry))
		release := fs.Hold()
		a := l.Append(entry, false)
		Eventually(fs.held).Should(Receive())
		b := l.Append(entry, false)
		release <- struct{}{}
		Expect(a.Wait()).To(Equal(uint64(5)))
		Eventually(fs.held).Should(Receive())
		c := l.Append(entry, false)
		fs.Unhold()
		release <- struct{}{}
		Expect(b.Wait()).To(Equal(uint64(6)))
		Expect(c.Wait()).To(Equal(uint64(7)))
		Expect(writes(fs.Ops())[:3]).To(Equal([]int64{8_192, 8_192 + size, 12_288}))
	})

	It("Should split a batch larger than a frame into frames", func() {
		open(32 << 20)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		release := fs.Hold()
		first := l.Append(entryOf(&log.Restart{}), false)
		Eventually(fs.held).Should(Receive())
		big := make([]byte, 10<<20)
		second := l.Append(big, false)
		third := l.Append(big, false)
		fs.Unhold()
		release <- struct{}{}
		Expect(first.Wait()).To(Equal(uint64(5)))
		Expect(second.Wait()).To(Equal(uint64(6)))
		Expect(third.Wait()).To(Equal(uint64(7)))
		frames := walk(fileBytes(fs))
		Expect(frames[1].Count).To(Equal(uint32(1)))
		Expect(frames[2].Count).To(Equal(uint32(1)))
	})

	It(
		"Should refuse an entry past the delete reserve, and take a delete there",
		func() {
			open(logSize)
			DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
			entry := make([]byte, logSize-log.DeleteReserve-log.FramesStart-31)
			Expect(l.Append(entry, false).Wait()).Error().To(SatisfyAll(
				MatchError(log.ErrFull),
				MatchError(ContainSubstring("does not fit in log 3")),
			))
			Expect(l.Append(entry, true).Wait()).To(Equal(uint64(5)))
		},
	)

	It("Should refuse an entry larger than a frame", func() {
		open(logSize)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		Expect(l.Append(make([]byte, log.MaxFrame), true).Wait()).Error().
			To(MatchError(log.ErrFull))
	})

	DescribeTable(
		"Should fail every entry after a failed write or sync, until a restart",
		func(fail func(error)) {
			open(logSize)
			fault := errors.New("disk fault")
			fail(fault)
			Expect(l.Append([]byte{1}, false).Wait()).Error().To(SatisfyAll(
				MatchError(fault),
				MatchError(ContainSubstring("no entry is durable until a restart")),
			))
			fail(nil)
			Expect(l.Append([]byte{1}, false).Wait()).Error().To(MatchError(fault))
			Expect(l.Close()).To(MatchError(fault))
		},
		Entry("sync", func(err error) { fs.FailSync(err) }),
		Entry("write", func(err error) { fs.FailWrite(err) }),
	)

	It("Should write queued entries and a clean mark at Close, then refuse "+
		"entries", func() {
		open(logSize)
		Expect(l.Append(entryOf(&log.Drop{Key: 1}), false).Wait()).To(Equal(uint64(5)))
		Expect(l.Close()).To(Succeed())
		Expect(<-ran).To(Succeed())
		frames := walk(fileBytes(fs))
		last := frames[len(frames)-1]
		Expect(last.Clean).To(BeTrue())
		Expect(last.Sequence).To(Equal(uint64(6)))
		Expect(l.Append([]byte{1}, false).Wait()).Error().To(MatchError(log.ErrClosed))
	})

	It("Should fail queued and later entries when the context of Run ends", func() {
		Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
		l = MustSucceed(log.Start(log.Config{
			FS:         fs,
			DB:         db,
			Generation: generation,
			Sequence:   1,
			Size:       logSize,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		ran = make(chan error, 1)
		go func() { ran <- l.Run(ctx) }()
		cancel()
		Expect(<-ran).To(MatchError(context.Canceled))
		Expect(l.Append([]byte{1}, false).Wait()).Error().
			To(MatchError(context.Canceled))
		Expect(l.Close()).To(Succeed())
	})

	It("Should give each of many concurrent entries one sequence, in linked "+
		"frames", func() {
		open(8 << 20)
		DeferCleanup(func() { Expect(l.Close()).To(Succeed()) })
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			seqs []uint64
		)
		for w := range 8 {
			wg.Go(func() {
				defer GinkgoRecover()
				for i := range 100 {
					seq := MustSucceed(l.Append(
						entryOf(&log.Drop{Key: uint32(w*100 + i)}),
						false,
					).Wait())
					mu.Lock()
					seqs = append(seqs, seq)
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		Expect(seqs).To(HaveLen(800))
		Expect(seqs).To(ContainElements(uint64(5), uint64(804)))
		var entries uint32
		for _, f := range walk(fileBytes(fs)) {
			entries += f.Count
		}
		Expect(entries).To(Equal(uint32(800)))
	})
})
