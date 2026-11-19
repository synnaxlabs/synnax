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
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Misdirected reads", func() {
	const blockSize = 4096
	// open returns an FS with the faults of cfg and two durable files of two blocks:
	// f.bin holds a block of "a" and a block of "b", and g.bin holds "c" and "d".
	open := func(cfg fault.Config) *fault.FS {
		GinkgoHelper()
		base := MustSucceed(fault.New(fault.Config{}))
		save(
			base,
			"f.bin",
			strings.Repeat("a", blockSize)+strings.Repeat("b", blockSize),
		)
		save(
			base,
			"g.bin",
			strings.Repeat("c", blockSize)+strings.Repeat("d", blockSize),
		)
		Expect(base.SyncDir("")).To(Succeed())
		return MustSucceed(base.Crash(fault.CrashConfig{Image: cfg}))
	}
	// readAt returns size bytes of f.bin at off.
	readAt := func(fs *fault.FS, off int64, size int) string {
		GinkgoHelper()
		f := MustSucceed(fs.Open("f.bin", os.O_RDONLY))
		buf := make([]byte, size)
		Expect(f.ReadAt(buf, off)).To(Equal(size))
		Expect(f.Close()).To(Succeed())
		return string(buf)
	}
	always := fault.Config{MisdirectedReadRate: 1}

	It(
		"Should return one whole block from another place, and then a good read",
		func() {
			whole := strings.Repeat("a", blockSize) + strings.Repeat("b", blockSize)
			sources := set.Set[string]{}
			for runKey := range uint64(32) {
				fs := open(fault.Config{RunKey: runKey, MisdirectedReadRate: 1})
				first := readAt(fs, 0, 2*blockSize)
				Expect(first[blockSize:]).To(Equal(whole[blockSize:]))
				Expect(readAt(fs, 0, 2*blockSize)).To(Equal(whole))
				second := readAt(fs, blockSize, blockSize)
				Expect(readAt(fs, blockSize, blockSize)).To(Equal(whole[blockSize:]))
				for _, block := range []string{first[:blockSize], second} {
					Expect(block).To(Equal(strings.Repeat(block[:1], blockSize)))
					sources.Add(block[:1])
				}
				Expect(first[:1]).ToNot(Equal("a"))
				Expect(second[:1]).ToNot(Equal("b"))
			}
			Expect(sources.Slice()).To(ConsistOf("a", "b", "c", "d"))
			Expect(read(strict(open(always)), "f.bin")).To(Equal(whole))
		},
	)

	It("Should replace only the bytes of the block that a short read covers", func() {
		got := readAt(open(always), blockSize+50, 100)
		Expect(got).To(Equal(strings.Repeat(got[:1], 100)))
		Expect(got[:1]).ToNot(Equal("b"))
	})

	It("Should misdirect a Read as well, and read zeros past a short block", func() {
		base := MustSucceed(fault.New(fault.Config{}))
		save(base, "f.bin", strings.Repeat("a", blockSize))
		save(base, "g.bin", "tacocat")
		Expect(base.SyncDir("")).To(Succeed())
		fs := MustSucceed(base.Crash(fault.CrashConfig{Image: always}))
		f := MustOpen(fs.Open("f.bin", os.O_RDONLY))
		buf := make([]byte, blockSize)
		Expect(f.Read(buf)).To(Equal(blockSize))
		Expect(string(buf)).
			To(Equal("tacocat" + strings.Repeat("\x00", blockSize-7)))
	})

	It("Should keep the offset in the block for a read that is not aligned", func() {
		data := make([]byte, 2*blockSize)
		for i := range data {
			data[i] = byte(i % 251)
		}
		base := MustSucceed(fault.New(fault.Config{}))
		save(base, "f.bin", string(data))
		Expect(base.SyncDir("")).To(Succeed())
		fs := MustSucceed(base.Crash(fault.CrashConfig{Image: always}))
		Expect(readAt(fs, blockSize+50, 100)).To(Equal(string(data[50:150])))
	})

	It("Should misdirect the same reads whatever other files hold", func() {
		whole := strings.Repeat("a", blockSize) + strings.Repeat("b", blockSize)
		// hits says, for each of 32 run keys, if the first read of f.bin is
		// misdirected.
		hits := func(others bool) []bool {
			GinkgoHelper()
			out := make([]bool, 32)
			for runKey := range out {
				base := MustSucceed(fault.New(fault.Config{}))
				save(base, "f.bin", whole)
				if others {
					save(base, "g.bin", strings.Repeat("c", 3*blockSize))
				}
				Expect(base.SyncDir("")).To(Succeed())
				cfg := fault.Config{RunKey: uint64(runKey), MisdirectedReadRate: 0.5}
				image := MustSucceed(base.Crash(fault.CrashConfig{Image: cfg}))
				out[runKey] = readAt(image, 0, 2*blockSize) != whole
			}
			return out
		}
		Expect(hits(true)).To(Equal(hits(false)))
		Expect(hits(true)).To(And(ContainElement(true), ContainElement(false)))
	})

	It("Should leave the read of a disk with one block as it is", func() {
		base := MustSucceed(fault.New(fault.Config{}))
		save(base, "f.bin", "tacocat")
		Expect(base.SyncDir("")).To(Succeed())
		fs := MustSucceed(base.Crash(fault.CrashConfig{Image: always}))
		Expect(readAt(fs, 0, 7)).To(Equal("tacocat"))
	})

	It("Should fix the reads that it misdirects by the run key", func() {
		// results joins the first byte of each block in 32 reads of f.bin.
		results := func(runKey uint64) string {
			GinkgoHelper()
			fs := open(fault.Config{RunKey: runKey, MisdirectedReadRate: 0.5})
			var out strings.Builder
			for range 32 {
				got := readAt(fs, 0, 2*blockSize)
				out.WriteString(got[:1] + got[blockSize:blockSize+1])
			}
			return out.String()
		}
		Expect(results(1)).To(Equal(results(1)))
		Expect(results(1)).ToNot(Equal(results(2)))
		Expect(results(1)).To(ContainSubstring("abab"))
	})

	DescribeTable("Should fail a rate that is not from 0 to 1",
		func(rate float64) {
			Expect(fault.New(fault.Config{MisdirectedReadRate: rate})).Error().
				To(MatchError(validate.ErrValidation))
		},
		Entry("low", -0.1),
		Entry("high", 1.1),
	)
})
