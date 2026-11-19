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
	"io"
	"math/bits"
	"os"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
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
