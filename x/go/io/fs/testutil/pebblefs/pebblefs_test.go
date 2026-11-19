// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblefs_test

import (
	"io"
	"os"
	"syscall"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/internal/invariants"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv/pebblekv"
	. "github.com/synnaxlabs/x/testutil"
)

// strict returns disk after a crash that keeps only its durable state.
func strict(disk *fault.FS) *fault.FS {
	GinkgoHelper()
	return MustSucceed(disk.Crash(fault.CrashConfig{}))
}

// open opens Pebble in the directory db of disk.
func open(disk xfs.FS) *pebble.DB {
	GinkgoHelper()
	return MustSucceed(pebble.Open("db", &pebble.Options{
		FS:     pebblefs.New(disk),
		Logger: pebblekv.NewNoopLogger(),
	}))
}

// get returns the value of key in db, or nil when db has no such key.
func get(db *pebble.DB, key string) []byte {
	GinkgoHelper()
	value, closer, err := db.Get([]byte(key))
	if errors.Is(err, pebble.ErrNotFound) {
		return nil
	}
	Expect(err).ToNot(HaveOccurred())
	defer func() { Expect(closer.Close()).To(Succeed()) }()
	return append([]byte(nil), value...)
}

var _ = Describe("Pebble FS", func() {
	var (
		disk *fault.FS
		fs   vfs.FS
	)
	BeforeEach(func() {
		disk = MustSucceed(fault.New(fault.Config{}))
		fs = pebblefs.New(disk)
	})

	Describe("Pebble", func() {
		It("Should keep a write that it synced through a crash", func() {
			db := open(disk)
			Expect(db.Set([]byte("taco"), []byte("cat"), pebble.Sync)).To(Succeed())
			Expect(
				db.Set([]byte("burrito"), []byte("dog"), pebble.NoSync),
			).To(Succeed())
			image := strict(disk)
			Expect(db.Close()).To(Succeed())
			db = open(image)
			Expect(get(db, "taco")).To(Equal([]byte("cat")))
			Expect(get(db, "burrito")).To(BeNil())
			Expect(db.Close()).To(Succeed())
		})

		It("Should open again on the same disk and read its writes", func() {
			db := open(disk)
			Expect(db.Set([]byte("taco"), []byte("cat"), pebble.NoSync)).To(Succeed())
			Expect(db.Close()).To(Succeed())
			db = open(disk)
			Expect(get(db, "taco")).To(Equal([]byte("cat")))
			Expect(db.Close()).To(Succeed())
		})
	})

	Describe("OpenDir", func() {
		It("Should make the entries of the directory durable with a Sync", func() {
			Expect(fs.MkdirAll("d", 0o755)).To(Succeed())
			d := MustSucceed(fs.OpenDir(""))
			Expect(d.Sync()).To(Succeed())
			f := MustSucceed(fs.Create("d/a", vfs.WriteCategoryUnspecified))
			Expect(f.Sync()).To(Succeed())
			Expect(f.Close()).To(Succeed())
			Expect(MustSucceed(strict(disk).Exists("d/a"))).To(BeFalse())
			sub := MustSucceed(fs.OpenDir("d"))
			Expect(sub.Sync()).To(Succeed())
			Expect(MustSucceed(strict(disk).Exists("d/a"))).To(BeTrue())
			Expect(sub.Close()).To(Succeed())
			Expect(d.Close()).To(Succeed())
		})

		It("Should fail on a file", func() {
			Expect(MustSucceed(fs.Create("a", vfs.WriteCategoryUnspecified)).Close()).
				To(Succeed())
			Expect(fs.OpenDir("a")).Error().To(MatchError(syscall.ENOTDIR))
		})

		It("Should fail to read or write a directory", func() {
			d := MustSucceed(fs.OpenDir(""))
			Expect(d.Read(make([]byte, 1))).Error().To(MatchError(syscall.EISDIR))
			Expect(d.Write([]byte("a"))).Error().To(MatchError(syscall.EISDIR))
			Expect(d.Fd()).To(Equal(vfs.InvalidFd))
			Expect(MustSucceed(d.Stat()).IsDir()).To(BeTrue())
		})
	})

	Describe("Files", func() {
		It("Should make the bytes of a file durable with SyncData and SyncTo", func() {
			f := MustSucceed(fs.Create("a", vfs.WriteCategoryUnspecified))
			Expect(MustSucceed(fs.OpenDir("")).Sync()).To(Succeed())
			Expect(f.Write([]byte("taco"))).To(Equal(4))
			Expect(f.SyncData()).To(Succeed())
			Expect(
				MustSucceed(xfs.ReadFile(strict(disk), "a")),
			).To(Equal([]byte("taco")))
			Expect(f.Write([]byte("cat"))).To(Equal(3))
			Expect(f.SyncTo(7)).To(BeTrue())
			Expect(
				MustSucceed(xfs.ReadFile(strict(disk), "a")),
			).To(Equal([]byte("tacocat")))
			Expect(f.Preallocate(0, 100)).To(Succeed())
			Expect(f.Prefetch(0, 100)).To(Succeed())
			Expect(f.Fd()).To(Equal(vfs.InvalidFd))
			Expect(MustSucceed(f.Stat()).Size()).To(BeEquivalentTo(7))
			Expect(f.Close()).To(Succeed())
		})

		It("Should cut a file with Create, and keep it with OpenReadWrite", func() {
			f := MustSucceed(fs.Create("a", vfs.WriteCategoryUnspecified))
			Expect(f.Write([]byte("taco"))).To(Equal(4))
			Expect(f.Close()).To(Succeed())
			f = MustSucceed(fs.OpenReadWrite("a", vfs.WriteCategoryUnspecified))
			Expect(f.WriteAt([]byte("T"), 0)).To(Equal(1))
			Expect(f.Close()).To(Succeed())
			Expect(MustSucceed(xfs.ReadFile(disk, "a"))).To(Equal([]byte("Taco")))
			Expect(MustSucceed(fs.Create("a", vfs.WriteCategoryUnspecified)).Close()).
				To(Succeed())
			Expect(MustSucceed(xfs.ReadFile(disk, "a"))).To(BeEmpty())
		})

		It("Should open a file to read only", func() {
			Expect(MustSucceed(fs.Create("a", vfs.WriteCategoryUnspecified)).Close()).
				To(Succeed())
			f := MustSucceed(fs.Open("a"))
			Expect(
				f.Write([]byte("a")),
			).Error().
				To(MatchError(invariants.ErrAccessDenied))
			Expect(f.Read(make([]byte, 1))).Error().To(MatchError(io.EOF))
			Expect(f.Close()).To(Succeed())
		})

		It("Should rename a file for reuse and open it to write", func() {
			f := MustSucceed(fs.Create("old", vfs.WriteCategoryUnspecified))
			Expect(f.Write([]byte("taco"))).To(Equal(4))
			Expect(f.Close()).To(Succeed())
			f = MustSucceed(
				fs.ReuseForWrite("old", "new", vfs.WriteCategoryUnspecified),
			)
			Expect(f.WriteAt([]byte("T"), 0)).To(Equal(1))
			Expect(f.Close()).To(Succeed())
			Expect(MustSucceed(fs.List(""))).To(Equal([]string{"new"}))
			Expect(MustSucceed(xfs.ReadFile(disk, "new"))).To(Equal([]byte("Taco")))
		})
	})

	Describe("Remove", func() {
		It("Should fail on a name that does not exist", func() {
			Expect(fs.Remove("a")).To(MatchError(os.ErrNotExist))
		})

		It("Should fail on a directory with entries, and remove an empty one", func() {
			Expect(fs.MkdirAll("d/e", 0o755)).To(Succeed())
			Expect(fs.Remove("d")).To(MatchError(syscall.ENOTEMPTY))
			Expect(fs.Remove("d/e")).To(Succeed())
			Expect(fs.Remove("d")).To(Succeed())
			Expect(MustSucceed(fs.List(""))).To(BeEmpty())
		})

		It("Should remove a directory and its entries with RemoveAll", func() {
			Expect(fs.MkdirAll("d/e", 0o755)).To(Succeed())
			Expect(fs.RemoveAll("d")).To(Succeed())
			Expect(fs.RemoveAll("d")).To(Succeed())
			Expect(MustSucceed(fs.List(""))).To(BeEmpty())
		})
	})

	Describe("Lock", func() {
		It("Should hold a lock until it closes", func() {
			lock := MustSucceed(fs.Lock("LOCK"))
			Expect(fs.Lock("LOCK")).Error().To(MatchError(syscall.EAGAIN))
			Expect(lock.Close()).To(Succeed())
			Expect(MustSucceed(fs.Lock("LOCK")).Close()).To(Succeed())
			Expect(MustSucceed(fs.Stat("LOCK")).IsDir()).To(BeFalse())
		})

		It("Should stop a second Pebble on the same FS", func() {
			db := MustSucceed(pebble.Open("db", &pebble.Options{
				FS:     fs,
				Logger: pebblekv.NewNoopLogger(),
			}))
			Expect(pebble.Open("db", &pebble.Options{
				FS:     fs,
				Logger: pebblekv.NewNoopLogger(),
			})).Error().To(MatchError(syscall.EAGAIN))
			Expect(db.Close()).To(Succeed())
		})
	})

	Describe("Link", func() {
		It("Should fail, so that LinkOrCopy copies", func() {
			f := MustSucceed(fs.Create("a", vfs.WriteCategoryUnspecified))
			Expect(f.Write([]byte("taco"))).To(Equal(4))
			Expect(f.Close()).To(Succeed())
			Expect(
				fs.Link("a", "b"),
			).To(MatchError("link a b: pebblefs: no hard links"))
			Expect(vfs.LinkOrCopy(fs, "a", "b")).To(Succeed())
			Expect(MustSucceed(xfs.ReadFile(disk, "b"))).To(Equal([]byte("taco")))
		})
	})

	Describe("Paths and space", func() {
		It("Should join and split paths with slashes", func() {
			Expect(fs.PathJoin("db", "000001.log")).To(Equal("db/000001.log"))
			Expect(fs.PathBase("db/000001.log")).To(Equal("000001.log"))
			Expect(fs.PathDir("db/000001.log")).To(Equal("db"))
			Expect(fs.Unwrap()).To(BeNil())
		})

		It("Should report free space", func() {
			usage := MustSucceed(fs.GetDiskUsage(""))
			Expect(usage.AvailBytes).To(BeNumerically(">", 0))
			Expect(usage.AvailBytes).To(Equal(usage.TotalBytes))
		})

		It("Should give each file the same device", func() {
			Expect(fs.MkdirAll("d", 0o755)).To(Succeed())
			Expect(MustSucceed(fs.Stat("d")).DeviceID()).To(Equal(vfs.DeviceID{}))
		})
	})
})
