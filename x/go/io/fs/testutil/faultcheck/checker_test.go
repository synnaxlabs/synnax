// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package faultcheck_test

import (
	"bytes"
	"io"
	"os"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// pattern returns n bytes that differ at each offset inside a 4 KiB block.
func pattern(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

var _ = Describe("Misdirected reads", func() {
	It("Should not change with the order of writes to other files", func() {
		for key := range uint64(32) {
			read := func(otherFirst bool) []byte {
				fs := MustSucceed(fault.New(fault.Config{
					RunKey:              key,
					MisdirectedReadRate: 1,
				}))
				a := create(fs, "a.bin")
				Expect(a.Write(pattern(2 * 4096))).To(Equal(2 * 4096))
				writeOther := func() {
					b := create(fs, "b.bin")
					Expect(
						b.Write(bytes.Repeat([]byte("B"), 3*4096)),
					).To(Equal(3 * 4096))
				}
				if otherFirst {
					writeOther()
				}
				got := make([]byte, 2*4096)
				Expect(a.ReadAt(got, 0)).To(Equal(2 * 4096))
				if !otherFirst {
					writeOther()
				}
				return got
			}
			Expect(read(true)).To(Equal(read(false)), "run key %d", key)
		}
	})

	It("Should read the same offset of the other block for an unaligned read", func() {
		fs := MustSucceed(fault.New(fault.Config{MisdirectedReadRate: 1}))
		data := pattern(2 * 4096)
		a := create(fs, "a.bin")
		Expect(a.Write(data)).To(Equal(len(data)))
		got := make([]byte, 100)
		Expect(a.ReadAt(got, 4096+50)).To(Equal(100))
		Expect(got).To(Equal(data[50:150]))
	})
})

var _ = Describe("Calls the specs leave out", func() {
	It("Should return io.EOF from a ReadAt one byte past the end", func() {
		fs := MustSucceed(fault.New(fault.Config{}))
		f := create(fs, "f.bin")
		Expect(f.Write([]byte("tacocat"))).To(Equal(7))
		n, err := f.ReadAt(make([]byte, 8), 0)
		Expect(n).To(Equal(7))
		Expect(err).To(MatchError(io.EOF))
	})

	It("Should fail a Sync that only shrinks a file", func() {
		fs := MustSucceed(fault.New(fault.Config{SyncErrorRate: 1}))
		f := create(fs, "f.bin")
		Expect(f.Write([]byte("tacocat"))).To(Equal(7))
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(f.Truncate(3)).To(Succeed())
		Expect(f.Sync()).To(MatchError(syscall.EIO))
	})

	It("Should let a SyncDir that failed succeed on a later try", func() {
		for key := range uint64(64) {
			fs := MustSucceed(fault.New(fault.Config{RunKey: key, SyncErrorRate: 0.5}))
			Expect(create(fs, "f.bin").Close()).To(Succeed())
			if fs.SyncDir("") == nil {
				continue
			}
			for range 8 {
				if fs.SyncDir("") == nil {
					return
				}
			}
		}
		Fail("no run key gave a failed SyncDir and then a good one")
	})

	It("Should return ENOTDIR for a path under a file", func() {
		fs := MustSucceed(fault.New(fault.Config{}))
		Expect(xfs.WriteFile(fs, "f.txt", []byte("x"))).To(Succeed())
		Expect(fs.Stat("f.txt/child")).Error().To(MatchError(syscall.ENOTDIR))
		Expect(fs.Open("f.txt/child", os.O_RDONLY)).Error().
			To(MatchError(syscall.ENOTDIR))
	})
})
