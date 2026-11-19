// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fault_test

import (
	"bytes"
	"io"
	"math/bits"
	"os"
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Full disk", func() {
	var fs *fault.FS
	BeforeEach(func() {
		fs = MustSucceed(fault.New(fault.Config{Capacity: 10}))
		write(fs, "f.txt", "tacocat")
	})
	// open returns a handle that appends to f.txt.
	open := func() xfs.File {
		GinkgoHelper()
		return MustOpen(fs.Open("f.txt", os.O_RDWR|os.O_APPEND))
	}

	It("Should write the bytes that fit and fail with ENOSPC", func() {
		f := open()
		n, err := f.Write([]byte("burrito"))
		Expect(err).To(MatchError(syscall.ENOSPC))
		Expect(n).To(Equal(3))
		Expect(read(fs, "f.txt")).To(Equal("tacocatbur"))
		Expect(f.Write([]byte("s"))).Error().To(MatchError(syscall.ENOSPC))
	})

	It("Should write over bytes that the file holds on a full disk", func() {
		f := open()
		Expect(f.WriteAt([]byte("cat"), 7)).To(Equal(3))
		Expect(f.WriteAt([]byte("burrito"), 0)).To(Equal(7))
		Expect(read(fs, "f.txt")).To(Equal("burritocat"))
	})

	It("Should not grow a file to a write that starts past the space", func() {
		Expect(open().WriteAt([]byte("s"), 10)).Error().
			To(MatchError(syscall.ENOSPC))
		Expect(read(fs, "f.txt")).To(Equal("tacocat"))
	})

	It("Should fail to grow a file past the space with Truncate", func() {
		f := open()
		Expect(f.Truncate(11)).To(MatchError(syscall.ENOSPC))
		Expect(read(fs, "f.txt")).To(Equal("tacocat"))
		Expect(f.Truncate(10)).To(Succeed())
	})

	It("Should count the bytes of every file", func() {
		MustSucceed(fs.Sub("a"))
		other := MustOpen(fs.Open("a/g.txt", os.O_CREATE|os.O_WRONLY))
		n, err := other.Write([]byte("burrito"))
		Expect(err).To(MatchError(syscall.ENOSPC))
		Expect(n).To(Equal(3))
	})

	DescribeTable("Should give space back",
		func(free func()) {
			free()
			other := MustOpen(fs.Open("g.txt", os.O_CREATE|os.O_WRONLY))
			Expect(other.Write([]byte("burritos"))).To(Equal(8))
		},
		Entry("after a remove", func() { Expect(fs.Remove("f.txt")).To(Succeed()) }),
		Entry("after a truncate", func() { Expect(open().Truncate(2)).To(Succeed()) }),
		Entry("after an open that truncates", func() { write(fs, "f.txt", "ta") }),
	)

	It("Should hold the space of a removed file until its last handle closes", func() {
		f := MustSucceed(fs.Open("f.txt", os.O_RDONLY))
		Expect(fs.Remove("f.txt")).To(Succeed())
		other := MustOpen(fs.Open("g.txt", os.O_CREATE|os.O_WRONLY))
		n, err := other.Write([]byte("burritos"))
		Expect(err).To(MatchError(syscall.ENOSPC))
		Expect(n).To(Equal(3))
		Expect(f.Close()).To(Succeed())
		Expect(other.Write([]byte("ritos"))).To(Equal(5))
	})

	It("Should write over the bytes of a file that is past the space", func() {
		Expect(open().Sync()).To(Succeed())
		Expect(fs.SyncDir("")).To(Succeed())
		cfg := fault.CrashConfig{Image: fault.Config{Capacity: 4}}
		image := MustSucceed(fs.Crash(cfg))
		g := MustOpen(image.Open("f.txt", os.O_WRONLY))
		n, err := g.WriteAt([]byte("burritos"), 0)
		Expect(err).To(MatchError(syscall.ENOSPC))
		Expect(n).To(Equal(7))
		Expect(read(image, "f.txt")).To(Equal("burrito"))
	})

	It("Should fail New for a capacity under zero", func() {
		Expect(fault.New(fault.Config{Capacity: -1})).Error().
			To(MatchError(validate.ErrValidation))
	})
})

// flipped returns the count of bits that differ between got and want.
func flipped(got []byte, want string) int {
	GinkgoHelper()
	Expect(got).To(HaveLen(len(want)))
	count := 0
	for i := range got {
		count += bits.OnesCount8(got[i] ^ want[i])
	}
	return count
}

var _ = Describe("Read corruption", func() {
	It("Should flip one bit of a read and leave the disk as it was", func() {
		fs := faulty(fault.Config{ReadCorruptionRate: 1})
		f := MustOpen(fs.Open("f.txt", os.O_RDONLY))
		buf := make([]byte, 7)
		Expect(f.Read(buf)).To(Equal(7))
		Expect(flipped(buf, "tacocat")).To(Equal(1))
		Expect(f.Read(buf)).Error().To(MatchError(io.EOF))
		Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
	})

	It("Should give one read fault for a block at most", func() {
		fs := faulty(fault.Config{ReadCorruptionRate: 1, MisdirectedReadRate: 1})
		f := MustOpen(fs.Open("f.txt", os.O_RDONLY))
		buf := make([]byte, 7)
		for _, want := range []int{1, 0, 0} {
			Expect(f.ReadAt(buf, 0)).To(Equal(7))
			Expect(flipped(buf, "tacocat")).To(Equal(want))
		}
		Expect(f.ReadAt(buf[:3], 4)).To(Equal(3))
		Expect(flipped(buf[:3], "cat")).To(BeZero())
	})

	It("Should give a fault to each block on the first read that covers it", func() {
		fs := MustSucceed(fault.New(fault.Config{ReadCorruptionRate: 1}))
		whole := strings.Repeat("a", 2*4096)
		write(fs, "f.bin", whole)
		f := MustOpen(fs.Open("f.bin", os.O_RDONLY))
		buf := make([]byte, 4096)
		Expect(f.ReadAt(buf, 4096)).To(Equal(4096))
		Expect(flipped(buf, whole[4096:])).To(Equal(1))
		buf = make([]byte, 2*4096)
		Expect(f.ReadAt(buf, 0)).To(Equal(2 * 4096))
		Expect(flipped(buf[:4096], whole[:4096])).To(Equal(1))
		Expect(flipped(buf[4096:], whole[4096:])).To(BeZero())
		Expect(f.ReadAt(buf, 0)).To(Equal(2 * 4096))
		Expect(flipped(buf, whole)).To(BeZero())
	})

	It("Should give a good second read of a record read in two calls", func() {
		want := make([]byte, 2*4096)
		for i := range want {
			want[i] = byte(i * 7)
		}
		base := MustSucceed(fault.New(fault.Config{}))
		save(base, "f.bin", string(want))
		Expect(base.SyncDir("")).To(Succeed())
		// The record is a header at the end of block 0 and a body in block 1.
		read := func(f xfs.File) []byte {
			GinkgoHelper()
			record := make([]byte, 120)
			Expect(f.ReadAt(record[:16], 4080)).To(Equal(16))
			Expect(f.ReadAt(record[16:], 4096)).To(Equal(104))
			return record
		}
		bad := 0
		for runKey := range uint64(64) {
			cfg := fault.Config{RunKey: runKey, ReadCorruptionRate: 0.5}
			image := MustSucceed(base.Crash(fault.CrashConfig{Image: cfg}))
			f := MustOpen(image.Open("f.bin", os.O_RDONLY))
			if !bytes.Equal(read(f), want[4080:4200]) {
				bad++
				Expect(read(f)).To(Equal(want[4080:4200]), "run key %d", runKey)
			}
		}
		Expect(bad).To(BeNumerically(">", 0))
	})

	It("Should draw again for each read of a block until the first fault", func() {
		firsts := set.Set[int]{}
		for runKey := range uint64(32) {
			cfg := fault.Config{RunKey: runKey, ReadCorruptionRate: 0.5}
			f := MustOpen(faulty(cfg).Open("f.txt", os.O_RDONLY))
			buf := make([]byte, 7)
			faulted := false
			for i := range 8 {
				Expect(f.ReadAt(buf, 0)).To(Equal(7))
				if faulted {
					Expect(flipped(buf, "tacocat")).To(BeZero())
				} else if flipped(buf, "tacocat") == 1 {
					firsts.Add(i)
					faulted = true
				}
			}
		}
		Expect(len(firsts)).To(BeNumerically(">", 2))
	})

	It("Should give a good second read of bytes in another file after a fault",
		func() {
			for runKey := range uint64(32) {
				base := MustSucceed(fault.New(fault.Config{}))
				save(base, "a.txt", "tacocat")
				save(base, "b.txt", "burrito")
				Expect(base.SyncDir("")).To(Succeed())
				cfg := fault.Config{RunKey: runKey, ReadCorruptionRate: 0.5}
				image := MustSucceed(base.Crash(fault.CrashConfig{Image: cfg}))
				a := MustOpen(image.Open("a.txt", os.O_RDONLY))
				b := MustOpen(image.Open("b.txt", os.O_RDONLY))
				buf := make([]byte, 7)
				// A record of two files, read until both files read whole.
				for try := 0; ; try++ {
					Expect(a.ReadAt(buf, 0)).To(Equal(7))
					good := flipped(buf, "tacocat") == 0
					Expect(b.ReadAt(buf, 0)).To(Equal(7))
					if good && flipped(buf, "burrito") == 0 {
						break
					}
					Expect(try).To(BeZero(), "run key %d", runKey)
				}
			}
		},
	)

	It(
		"Should fix the fault of a first read by the run key and the block, not by "+
			"the order of reads",
		func() {
			const blocks = 8
			whole := strings.Repeat("a", blocks*4096)
			// faulty says, for each block, if its first read had a flipped bit.
			faulty := func(runKey uint64, rising bool) []bool {
				GinkgoHelper()
				base := MustSucceed(fault.New(fault.Config{}))
				save(base, "f.bin", whole)
				Expect(base.SyncDir("")).To(Succeed())
				cfg := fault.Config{RunKey: runKey, ReadCorruptionRate: 0.5}
				image := MustSucceed(base.Crash(fault.CrashConfig{Image: cfg}))
				f := MustOpen(image.Open("f.bin", os.O_RDONLY))
				out := make([]bool, blocks)
				for i := range blocks {
					block := i
					if !rising {
						block = blocks - 1 - i
					}
					buf := make([]byte, 4096)
					Expect(f.ReadAt(buf, int64(block)*4096)).To(Equal(4096))
					out[block] = flipped(buf, whole[:4096]) == 1
				}
				return out
			}
			for runKey := range uint64(8) {
				Expect(faulty(runKey, true)).To(Equal(faulty(runKey, false)))
				Expect(
					faulty(runKey, true),
				).To(And(ContainElement(true), ContainElement(false)))
			}
			Expect(faulty(0, true)).ToNot(Equal(faulty(1, true)))
		},
	)

	It("Should fix the reads that it corrupts by the run key", func() {
		// results has the count of flipped bits in the first read of 32 disks, whose
		// run keys start at first.
		results := func(first uint64) []int {
			GinkgoHelper()
			counts := make([]int, 32)
			for i := range counts {
				cfg := fault.Config{RunKey: first + uint64(i), ReadCorruptionRate: 0.5}
				f := MustSucceed(faulty(cfg).Open("f.txt", os.O_RDONLY))
				buf := make([]byte, 7)
				Expect(f.ReadAt(buf, 0)).To(Equal(7))
				Expect(f.Close()).To(Succeed())
				counts[i] = flipped(buf, "tacocat")
			}
			return counts
		}
		Expect(results(0)).To(Equal(results(0)))
		Expect(results(0)).ToNot(Equal(results(100)))
		Expect(results(0)).To(And(ContainElement(0), ContainElement(1)))
	})

	DescribeTable("Should fail a rate that is not from 0 to 1",
		func(rate float64) {
			Expect(fault.New(fault.Config{ReadCorruptionRate: rate})).Error().
				To(MatchError(validate.ErrValidation))
		},
		Entry("low", -0.1),
		Entry("high", 1.1),
	)
})

var _ = Describe("File ID", func() {
	var fs *fault.FS
	BeforeEach(func() { fs = MustSucceed(fault.New(fault.Config{})) })
	id := func(fs xfs.FS, name string) fault.ID {
		GinkgoHelper()
		stat, ok := MustSucceed(fs.Stat(name)).Sys().(fault.Stat)
		Expect(ok).To(BeTrue())
		return stat.ID
	}

	It("Should give each file its own ID", func() {
		write(fs, "a.txt", "taco")
		write(fs, "b.txt", "cat")
		Expect(id(fs, "a.txt")).ToNot(Equal(id(fs, "b.txt")))
	})

	It("Should give a new file at the path of a removed file a new ID", func() {
		write(fs, "a.txt", "taco")
		old := id(fs, "a.txt")
		Expect(fs.Remove("a.txt")).To(Succeed())
		write(fs, "a.txt", "taco")
		Expect(id(fs, "a.txt")).ToNot(Equal(old))
	})

	It("Should keep the ID of a file through a rename, a write, and a list", func() {
		write(fs, "a.txt", "taco")
		old := id(fs, "a.txt")
		Expect(fs.Rename("a.txt", "b.txt")).To(Succeed())
		write(fs, "b.txt", "burrito")
		Expect(id(fs, "b.txt")).To(Equal(old))
		Expect(MustSucceed(fs.List(""))[0].Sys()).To(Equal(fault.Stat{ID: old}))
		f := MustOpen(fs.Open("b.txt", os.O_RDONLY))
		Expect(MustSucceed(f.Stat()).Sys()).To(Equal(fault.Stat{ID: old}))
	})

	It("Should keep the ID of a file in a crash image", func() {
		save(fs, "a.txt", "taco")
		save(fs, "b.txt", "cat")
		Expect(fs.SyncDir("")).To(Succeed())
		image := strict(fs)
		Expect(id(image, "a.txt")).To(Equal(id(fs, "a.txt")))
		Expect(id(image, "b.txt")).To(Equal(id(fs, "b.txt")))
		write(image, "c.txt", "new")
		Expect(id(image, "c.txt")).ToNot(BeElementOf(id(fs, "a.txt"), id(fs, "b.txt")))
	})
})

var _ = Describe("Sync state", func() {
	const blockSize = 4096
	var (
		fs *fault.FS
		f  xfs.File
	)
	BeforeEach(func() {
		fs = MustSucceed(fault.New(fault.Config{}))
		f = MustOpen(fs.Open("f.bin", os.O_CREATE|os.O_RDWR))
	})
	stat := func() fault.Stat {
		GinkgoHelper()
		s, ok := MustSucceed(f.Stat()).Sys().(fault.Stat)
		Expect(ok).To(BeTrue())
		return s
	}
	put := func(data []byte, off int64) {
		GinkgoHelper()
		Expect(f.WriteAt(data, off)).To(Equal(len(data)))
	}
	ones := func(n int) []byte { return bytes.Repeat([]byte{1}, n) }

	It("Should report the ranges that a Sync made durable", func() {
		put(make([]byte, 200), 0)
		Expect(f.Sync()).To(Succeed())
		put(ones(10), 0)
		put(ones(20), 100)
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Syncs).To(BeEquivalentTo(1))
		Expect(stat().Synced).To(Equal([]fault.Span{{0, 10}, {100, 120}}))
	})

	It("Should take the zeros that a write past the end adds as written", func() {
		put(ones(10), 0)
		put(ones(1), 100)
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Synced).To(Equal([]fault.Span{{0, 101}}))
	})

	It("Should not count a Sync of bytes equal to the durable bytes", func() {
		put(ones(10), 0)
		Expect(f.Sync()).To(Succeed())
		put(ones(10), 0)
		Expect(f.Sync()).To(Succeed())
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Syncs).To(BeEquivalentTo(1))
		Expect(stat().Synced).To(Equal([]fault.Span{{0, 10}}))
	})

	It("Should not count a Sync of zeros past the durable end", func() {
		put(make([]byte, 2*blockSize), 0)
		Expect(f.Truncate(3 * blockSize)).To(Succeed())
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Syncs).To(BeZero())
		Expect(stat().Synced).To(BeEmpty())
	})

	It("Should leave out a block whose bytes did not change", func() {
		put(ones(2*blockSize), 0)
		Expect(f.Sync()).To(Succeed())
		put(append(ones(blockSize), make([]byte, 10)...), 0)
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Syncs).To(BeEquivalentTo(2))
		Expect(stat().Synced).To(Equal([]fault.Span{{blockSize, blockSize + 10}}))
	})

	It("Should keep each range in a changed block, also of equal bytes", func() {
		put(ones(blockSize), 0)
		Expect(f.Sync()).To(Succeed())
		put(ones(10), 0)
		put([]byte{2}, 100)
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Synced).To(Equal([]fault.Span{{0, 10}, {100, 101}}))
	})

	It("Should report only the bytes in the file after a cut", func() {
		put(ones(100), 0)
		Expect(f.Truncate(10)).To(Succeed())
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Synced).To(Equal([]fault.Span{{0, 10}}))
	})

	It("Should not count a failed Sync", func() {
		fs = MustSucceed(fault.New(fault.Config{SyncErrorRate: 1}))
		f = MustOpen(fs.Open("f.bin", os.O_CREATE|os.O_RDWR))
		put(ones(10), 0)
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(stat().Syncs).To(BeZero())
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Syncs).To(BeZero())
	})

	It("Should keep the writes of a failed Sync for the next Sync of their block",
		func() {
			found := false
			for runKey := range uint64(64) {
				cfg := fault.Config{RunKey: runKey, SyncErrorRate: 0.5}
				fs = MustSucceed(fault.New(cfg))
				f = MustOpen(fs.Open("f.bin", os.O_CREATE|os.O_RDWR))
				put(ones(10), 0)
				if f.Sync() == nil {
					continue
				}
				put([]byte{2}, 100)
				if f.Sync() != nil {
					continue
				}
				// The write at 100 also adds zeros from 10 on.
				Expect(stat().Synced).To(Equal([]fault.Span{{0, 101}}))
				found = true
				break
			}
			Expect(found).To(BeTrue())
		},
	)

	It("Should cut a range at the end of a changed block", func() {
		put(ones(3*blockSize), 0)
		Expect(f.Sync()).To(Succeed())
		changed := append(bytes.Repeat([]byte{2}, 96), ones(blockSize+100)...)
		put(changed, blockSize-96)
		Expect(f.Sync()).To(Succeed())
		Expect(stat().Synced).To(Equal([]fault.Span{{blockSize - 96, blockSize}}))
	})

	It("Should start a crash image with no Sync state", func() {
		put(ones(10), 0)
		Expect(f.Sync()).To(Succeed())
		Expect(fs.SyncDir("")).To(Succeed())
		image := strict(fs)
		s, ok := MustSucceed(image.Stat("f.bin")).Sys().(fault.Stat)
		Expect(ok).To(BeTrue())
		Expect(s).To(Equal(fault.Stat{ID: stat().ID}))
	})

	It("Should not let a caller change the reported ranges", func() {
		put(ones(10), 0)
		Expect(f.Sync()).To(Succeed())
		stat().Synced[0].To = 99
		Expect(stat().Synced).To(Equal([]fault.Span{{0, 10}}))
	})
})
