// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblefscheck_test

import (
	"fmt"
	"io"
	"slices"

	"github.com/cockroachdb/errors/oserror"
	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	. "github.com/synnaxlabs/x/testutil"
)

// target is one vfs.FS and the directory that a case works in.
type target struct {
	fs  vfs.FS
	dir string
}

func (t target) path(name string) string { return t.fs.PathJoin(t.dir, name) }

// targets returns the reference file systems of Pebble and pebblefs on a fault disk.
func targets() map[string]target {
	GinkgoHelper()
	mem := vfs.NewMem()
	Expect(mem.MkdirAll("r", 0o755)).To(Succeed())
	adapted := pebblefs.New(MustSucceed(fault.New(fault.Config{})))
	Expect(adapted.MkdirAll("r", 0o755)).To(Succeed())
	return map[string]target{
		"default":  {fs: vfs.Default, dir: GinkgoT().TempDir()},
		"mem":      {fs: mem, dir: "r"},
		"pebblefs": {fs: adapted, dir: "r"},
	}
}

// results runs c on each target and returns what it saw, by the name of the target.
func results(c func(t target) string) map[string]string {
	GinkgoHelper()
	out := map[string]string{}
	for name, t := range targets() {
		out[name] = c(t)
	}
	return out
}

// errClass names the kind of err, so the errors of two file systems compare.
func errClass(err error) string {
	switch {
	case err == nil:
		return "nil"
	case oserror.IsNotExist(err):
		return "not exist"
	case oserror.IsExist(err):
		return "exist"
	}
	return "error"
}

func writeFile(t target, name, data string) {
	GinkgoHelper()
	f := MustSucceed(t.fs.Create(t.path(name), vfs.WriteCategoryUnspecified))
	Expect(f.Write([]byte(data))).To(Equal(len(data)))
	Expect(f.Close()).To(Succeed())
}

func readFile(fs vfs.FS, name string) string {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	return string(MustSucceed(io.ReadAll(f)))
}

func syncDir(fs vfs.FS, name string) {
	GinkgoHelper()
	d := MustSucceed(fs.OpenDir(name))
	Expect(d.Sync()).To(Succeed())
	Expect(d.Close()).To(Succeed())
}

// crashTarget is a file system that can crash, with a crash that keeps only what was
// synced.
type crashTarget struct {
	fs    vfs.FS
	crash func() vfs.FS
}

func crashTargets() map[string]crashTarget {
	mem := vfs.NewCrashableMem()
	disk := MustSucceed(fault.New(fault.Config{}))
	return map[string]crashTarget{
		"mem": {fs: mem, crash: func() vfs.FS {
			return mem.CrashClone(vfs.CrashCloneCfg{})
		}},
		"pebblefs": {fs: pebblefs.New(disk), crash: func() vfs.FS {
			return pebblefs.New(MustSucceed(disk.Crash(fault.CrashConfig{})))
		}},
	}
}

func crashResults(c func(t crashTarget) string) map[string]string {
	GinkgoHelper()
	out := map[string]string{}
	for name, t := range crashTargets() {
		fs := t.fs
		Expect(fs.MkdirAll("r", 0o755)).To(Succeed())
		syncDir(fs, "")
		out[name] = c(t)
	}
	return out
}

var _ = Describe("pebblefs against the file systems of Pebble", func() {
	It("Should create a missing file with OpenReadWrite", func() {
		r := results(func(t target) string {
			f, err := t.fs.OpenReadWrite(t.path("a"), vfs.WriteCategoryUnspecified)
			if err != nil {
				return errClass(err)
			}
			Expect(f.Write([]byte("taco"))).To(Equal(4))
			Expect(f.Close()).To(Succeed())
			return readFile(t.fs, t.path("a"))
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
		Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
	})

	It("Should give Create a new file, so a reader of the old file keeps it", func() {
		r := results(func(t target) string {
			writeFile(t, "a", "taco")
			old := MustSucceed(t.fs.Open(t.path("a")))
			writeFile(t, "a", "xy")
			buf := make([]byte, 8)
			n, err := old.ReadAt(buf, 0)
			Expect(old.Close()).To(Succeed())
			return fmt.Sprintf("%q %s", buf[:n], errClass(err))
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
		Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
	})

	It(
		"Should keep the old file after a crash when Create was not made durable",
		func() {
			r := crashResults(func(t crashTarget) string {
				f := MustSucceed(t.fs.Create("r/a", vfs.WriteCategoryUnspecified))
				Expect(f.Write([]byte("taco"))).To(Equal(4))
				Expect(f.Sync()).To(Succeed())
				Expect(f.Close()).To(Succeed())
				syncDir(t.fs, "r")
				f = MustSucceed(t.fs.Create("r/a", vfs.WriteCategoryUnspecified))
				Expect(f.Write([]byte("xy"))).To(Equal(2))
				Expect(f.Sync()).To(Succeed())
				Expect(f.Close()).To(Succeed())
				return readFile(t.crash(), "r/a")
			})
			Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
		},
	)

	It("Should share one offset between Read and Write on a handle", func() {
		r := results(func(t target) string {
			writeFile(t, "a", "taco")
			f := MustSucceed(
				t.fs.OpenReadWrite(t.path("a"), vfs.WriteCategoryUnspecified),
			)
			Expect(f.Read(make([]byte, 2))).To(Equal(2))
			Expect(f.Write([]byte("XY"))).To(Equal(2))
			Expect(f.Close()).To(Succeed())
			return readFile(t.fs, t.path("a"))
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
		Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
	})

	It("Should make SyncTo of a directory a no-op that reports no full sync", func() {
		r := results(func(t target) string {
			d := MustSucceed(t.fs.OpenDir(t.dir))
			full, err := d.SyncTo(0)
			Expect(d.Close()).To(Succeed())
			return fmt.Sprintf("full=%t err=%s", full, errClass(err))
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
		Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
	})

	It("Should not make the entries of a directory durable with SyncTo", func() {
		r := crashResults(func(t crashTarget) string {
			f := MustSucceed(t.fs.Create("r/a", vfs.WriteCategoryUnspecified))
			Expect(f.Sync()).To(Succeed())
			Expect(f.Close()).To(Succeed())
			d := MustSucceed(t.fs.OpenDir("r"))
			_, err := d.SyncTo(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(d.Close()).To(Succeed())
			names := MustSucceed(t.crash().List("r"))
			slices.Sort(names)
			return fmt.Sprint(names)
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
	})

	It("Should truncate a lock file that holds bytes", func() {
		r := results(func(t target) string {
			writeFile(t, "LOCK", "taco")
			lock := MustSucceed(t.fs.Lock(t.path("LOCK")))
			Expect(lock.Close()).To(Succeed())
			return fmt.Sprint(MustSucceed(t.fs.Stat(t.path("LOCK"))).Size())
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
		Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
	})

	It("Should sync a directory handle after the directory is renamed", func() {
		r := results(func(t target) string {
			Expect(t.fs.MkdirAll(t.path("d"), 0o755)).To(Succeed())
			d := MustSucceed(t.fs.OpenDir(t.path("d")))
			Expect(t.fs.Rename(t.path("d"), t.path("e"))).To(Succeed())
			err := d.Sync()
			Expect(d.Close()).To(Succeed())
			return errClass(err)
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
		Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
	})

	DescribeTable(
		"Should match Pebble's file systems",
		func(c func(t target) string) {
			r := results(c)
			Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
			Expect(r).To(HaveKeyWithValue("pebblefs", r["mem"]))
		},
		Entry("Stat of a file and a directory", func(t target) string {
			writeFile(t, "a", "taco")
			Expect(t.fs.MkdirAll(t.path("d"), 0o755)).To(Succeed())
			a := MustSucceed(t.fs.Stat(t.path("a")))
			d := MustSucceed(t.fs.Stat(t.path("d")))
			return fmt.Sprint(a.Size(), a.IsDir(), d.IsDir(), a.Name())
		}),
		Entry("Stat and Open of a missing file", func(t target) string {
			_, err := t.fs.Stat(t.path("a"))
			_, openErr := t.fs.Open(t.path("a"))
			return errClass(err) + " " + errClass(openErr)
		}),
		Entry("List of a missing directory", func(t target) string {
			_, err := t.fs.List(t.path("d"))
			return errClass(err)
		}),
		Entry("Remove of a missing file", func(t target) string {
			return errClass(t.fs.Remove(t.path("a")))
		}),
		Entry("Remove of a directory with entries", func(t target) string {
			Expect(t.fs.MkdirAll(t.path("d/e"), 0o755)).To(Succeed())
			err := t.fs.Remove(t.path("d"))
			return fmt.Sprint(err != nil, MustSucceed(t.fs.List(t.dir)))
		}),
		Entry("RemoveAll of a tree and of a missing name", func(t target) string {
			Expect(t.fs.MkdirAll(t.path("d/e"), 0o755)).To(Succeed())
			writeFile(t, "d/e/a", "taco")
			first := errClass(t.fs.RemoveAll(t.path("d")))
			second := errClass(t.fs.RemoveAll(t.path("x/y")))
			return fmt.Sprint(first, second, len(MustSucceed(t.fs.List(t.dir))))
		}),
		Entry("Rename onto an existing file", func(t target) string {
			writeFile(t, "a", "taco")
			writeFile(t, "b", "cat")
			Expect(t.fs.Rename(t.path("a"), t.path("b"))).To(Succeed())
			names := MustSucceed(t.fs.List(t.dir))
			slices.Sort(names)
			return fmt.Sprint(names, readFile(t.fs, t.path("b")))
		}),
		Entry("ReuseForWrite of a missing file", func(t target) string {
			_, err := t.fs.ReuseForWrite(
				t.path("a"),
				t.path("b"),
				vfs.WriteCategoryUnspecified,
			)
			return errClass(err)
		}),
		Entry(
			"ReuseForWrite writes from the start, with no cut",
			func(t target) string {
				writeFile(t, "a", "tacocat")
				f := MustSucceed(t.fs.ReuseForWrite(
					t.path("a"),
					t.path("b"),
					vfs.WriteCategoryUnspecified,
				))
				Expect(f.Write([]byte("XY"))).To(Equal(2))
				Expect(f.Close()).To(Succeed())
				return readFile(t.fs, t.path("b"))
			},
		),
		Entry("Create on an existing file cuts it", func(t target) string {
			writeFile(t, "a", "tacocat")
			writeFile(t, "a", "xy")
			return readFile(t.fs, t.path("a"))
		}),
		Entry("ReadAt past the end and a short ReadAt", func(t target) string {
			writeFile(t, "a", "taco")
			f := MustSucceed(t.fs.Open(t.path("a")))
			n1, err1 := f.ReadAt(make([]byte, 2), 10)
			n2, err2 := f.ReadAt(make([]byte, 8), 2)
			n3, err3 := f.ReadAt(make([]byte, 2), 2)
			Expect(f.Close()).To(Succeed())
			return fmt.Sprint(n1, err1 == io.EOF, n2, err2 == io.EOF, n3, err3)
		}),
		Entry("Read to the end", func(t target) string {
			writeFile(t, "a", "taco")
			f := MustSucceed(t.fs.Open(t.path("a")))
			n1, err1 := f.Read(make([]byte, 3))
			n2, err2 := f.Read(make([]byte, 3))
			n3, err3 := f.Read(make([]byte, 3))
			Expect(f.Close()).To(Succeed())
			return fmt.Sprint(n1, err1, n2, err2, n3, err3 == io.EOF)
		}),
		Entry("WriteAt past the end fills zeros", func(t target) string {
			f := MustSucceed(t.fs.Create(t.path("a"), vfs.WriteCategoryUnspecified))
			Expect(f.WriteAt([]byte("x"), 3)).To(Equal(1))
			Expect(f.Close()).To(Succeed())
			return fmt.Sprintf("%q", readFile(t.fs, t.path("a")))
		}),
		Entry("Lock twice, and again after Close", func(t target) string {
			lock := MustSucceed(t.fs.Lock(t.path("LOCK")))
			_, err := t.fs.Lock(t.path("LOCK"))
			Expect(lock.Close()).To(Succeed())
			again := MustSucceed(t.fs.Lock(t.path("LOCK")))
			Expect(again.Close()).To(Succeed())
			return fmt.Sprint(err != nil)
		}),
		Entry("MkdirAll on an existing directory and on a file", func(t target) string {
			Expect(t.fs.MkdirAll(t.path("d"), 0o755)).To(Succeed())
			Expect(t.fs.MkdirAll(t.path("d"), 0o755)).To(Succeed())
			writeFile(t, "a", "taco")
			return fmt.Sprint(t.fs.MkdirAll(t.path("a/b"), 0o755) != nil)
		}),
		Entry("Paths with trailing slashes", func(t target) string {
			return fmt.Sprint(
				t.fs.PathBase("db/a/"),
				t.fs.PathDir("db/a/"),
				t.fs.PathDir("a"),
				t.fs.PathJoin("db", "", "a"),
			)
		}),
	)

	It("Should fail to rename a file onto a directory with entries", func() {
		r := results(func(t target) string {
			writeFile(t, "a", "taco")
			Expect(t.fs.MkdirAll(t.path("d"), 0o755)).To(Succeed())
			writeFile(t, "d/x", "cat")
			err := t.fs.Rename(t.path("a"), t.path("d"))
			names := MustSucceed(t.fs.List(t.dir))
			slices.Sort(names)
			return fmt.Sprintf("%s %v", errClass(err), names)
		})
		Expect(r).To(HaveKeyWithValue("pebblefs", r["default"]))
	})
})
