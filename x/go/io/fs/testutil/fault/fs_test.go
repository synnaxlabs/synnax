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
	"os"
	"strconv"
	"sync"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// write replaces the file name in fs with data, and does not sync it.
func write(fs xfs.FS, name, data string) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
	Expect(f.Write([]byte(data))).To(Equal(len(data)))
	Expect(f.Close()).To(Succeed())
}

// read returns the contents of the file name in fs.
func read(fs xfs.FS, name string) string {
	GinkgoHelper()
	return string(MustSucceed(xfs.ReadFile(fs, name)))
}

var _ = Describe("FS", func() {
	var fs *fault.FS
	BeforeEach(func() { fs = MustSucceed(fault.New(fault.Config{})) })

	It("Should share one disk between an FS and its Sub", func() {
		sub := MustSucceed(fs.Sub("a/b"))
		write(sub, "f.txt", "tacocat")
		Expect(read(fs, "a/b/f.txt")).To(Equal("tacocat"))
		Expect(fs.Remove("a")).To(Succeed())
		Expect(sub.Exists("f.txt")).To(BeFalse())
	})

	It("Should fail to create a file that exists under O_EXCL", func() {
		write(fs, "f.txt", "tacocat")
		Expect(fs.Open("f.txt", os.O_CREATE|os.O_EXCL)).Error().
			To(MatchError(os.ErrExist))
	})

	It("Should fail a path under a file", func() {
		write(fs, "f.txt", "tacocat")
		Expect(fs.Sub("f.txt/a")).Error().To(MatchError(syscall.ENOTDIR))
		Expect(fs.List("f.txt")).Error().To(MatchError(syscall.ENOTDIR))
		Expect(fs.Stat("f.txt/a")).Error().To(MatchError(syscall.ENOTDIR))
		Expect(fs.Open("f.txt/a", os.O_RDONLY)).Error().
			To(MatchError(syscall.ENOTDIR))
	})

	DescribeTable("Should refuse to change the root",
		func(change func() error) {
			Expect(change()).To(MatchError(syscall.EINVAL))
		},
		Entry("remove", func() error { return fs.Remove("") }),
		Entry("rename from", func() error { return fs.Rename("", "a") }),
		Entry("rename to", func() error { return fs.Rename("a", "") }),
	)

	It("Should replace the target of a rename", func() {
		write(fs, "a.txt", "new")
		write(fs, "b.txt", "old")
		Expect(fs.Rename("a.txt", "b.txt")).To(Succeed())
		Expect(read(fs, "b.txt")).To(Equal("new"))
		Expect(fs.Exists("a.txt")).To(BeFalse())
	})

	It("Should give each change a later modification time", func() {
		write(fs, "f.txt", "a")
		first := MustSucceed(fs.Stat("f.txt"))
		write(fs, "f.txt", "b")
		Expect(MustSucceed(fs.Stat("f.txt")).ModTime()).
			To(BeTemporally(">", first.ModTime()))
		Expect(first.Size()).To(Equal(int64(1)))
	})

	It("Should take calls and crashes from many goroutines at once", func() {
		var wg sync.WaitGroup
		for i := range 8 {
			wg.Go(func() {
				defer GinkgoRecover()
				name := strconv.Itoa(i) + ".txt"
				for range 50 {
					save(fs, name, "tacocat")
					Expect(fs.SyncDir("")).To(Succeed())
					image := MustSucceed(fs.Crash(fault.CrashConfig{}))
					Expect(read(image, name)).To(Equal("tacocat"))
				}
			})
		}
		wg.Wait()
		Expect(MustSucceed(fs.List(""))).To(HaveLen(8))
	})

	Describe("File", func() {
		It("Should append at the end on every Write under O_APPEND", func() {
			write(fs, "f.txt", "one")
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY|os.O_APPEND))
			other := MustOpen(fs.Open("f.txt", os.O_WRONLY|os.O_APPEND))
			Expect(f.Write([]byte("two"))).To(Equal(3))
			Expect(other.Write([]byte("three"))).To(Equal(5))
			Expect(f.Write([]byte("four"))).To(Equal(4))
			Expect(read(fs, "f.txt")).To(Equal("onetwothreefour"))
		})

		It("Should fail to read a directory, or to open it for a write", func() {
			MustSucceed(fs.Sub("a"))
			f := MustOpen(fs.Open("a", os.O_RDONLY))
			Expect(f.Read(make([]byte, 1))).Error().To(MatchError(syscall.EISDIR))
			Expect(fs.Open("a/.", os.O_CREATE|os.O_WRONLY)).Error().
				To(MatchError(syscall.EISDIR))
			Expect(fs.List("a")).To(BeEmpty())
		})

		It(
			"Should return the bytes and io.EOF from a ReadAt that passes the end",
			func() {
				write(fs, "f.txt", "tacocat")
				f := MustOpen(fs.Open("f.txt", os.O_RDONLY))
				buf := make([]byte, 8)
				n, err := f.ReadAt(buf, 0)
				Expect(err).To(MatchError(io.EOF))
				Expect(string(buf[:n])).To(Equal("tacocat"))
			},
		)

		It("Should not grow a file on an empty write past its end", func() {
			f := MustOpen(fs.Open("f.txt", os.O_CREATE|os.O_WRONLY))
			Expect(f.WriteAt(nil, 100)).To(BeZero())
			Expect(MustSucceed(f.Stat()).Size()).To(BeZero())
		})

		DescribeTable("Should fail a negative offset or size",
			func(call func(f xfs.File) error) {
				f := MustOpen(fs.Open("f.txt", os.O_CREATE|os.O_RDWR))
				Expect(call(f)).To(MatchError(syscall.EINVAL))
			},
			Entry("ReadAt", func(f xfs.File) error {
				_, err := f.ReadAt(make([]byte, 1), -1)
				return err
			}),
			Entry("WriteAt", func(f xfs.File) error {
				_, err := f.WriteAt([]byte("a"), -1)
				return err
			}),
			Entry("Truncate", func(f xfs.File) error { return f.Truncate(-1) }),
		)

		DescribeTable("Should fail a call on a closed file",
			func(call func(f xfs.File) error) {
				f := MustSucceed(fs.Open("f.txt", os.O_CREATE|os.O_RDWR))
				Expect(f.Close()).To(Succeed())
				Expect(call(f)).To(MatchError(os.ErrClosed))
			},
			Entry("Close", func(f xfs.File) error { return f.Close() }),
			Entry("Read", func(f xfs.File) error {
				_, err := f.Read(make([]byte, 1))
				return err
			}),
			Entry("ReadAt", func(f xfs.File) error {
				_, err := f.ReadAt(make([]byte, 1), 0)
				return err
			}),
			Entry("Write", func(f xfs.File) error {
				_, err := f.Write([]byte("a"))
				return err
			}),
			Entry("WriteAt", func(f xfs.File) error {
				_, err := f.WriteAt([]byte("a"), 0)
				return err
			}),
			Entry("Truncate", func(f xfs.File) error { return f.Truncate(0) }),
			Entry("Stat", func(f xfs.File) error {
				_, err := f.Stat()
				return err
			}),
			Entry("Sync", func(f xfs.File) error { return f.Sync() }),
		)
	})
})
