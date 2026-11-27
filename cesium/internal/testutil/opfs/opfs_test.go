// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package opfs_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/testutil/opfs"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var errInjected = errors.New("injected")

func create(fs *opfs.FS, name string) xfs.File {
	GinkgoHelper()
	return MustSucceed(fs.Open(name, os.O_CREATE|os.O_RDWR))
}

// touch writes 3 bytes at offset 2 of a new file name, syncs it and the root, and
// closes it.
func touch(fs *opfs.FS, name string) {
	GinkgoHelper()
	f := create(fs, name)
	Expect(f.WriteAt([]byte{1, 2, 3}, 2)).To(Equal(3))
	Expect(f.Sync()).To(Succeed())
	Expect(fs.SyncDir("")).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("FS", func() {
	It("Should record each operation of the named kinds in order", func() {
		fs := opfs.New(
			xfs.NewMem(),
			"open", "close", "write", "sync", "syncdir",
		)
		touch(fs, "a")
		Expect(fs.Ops()).To(Equal([]string{
			"open a", "write a 2 3", "sync a", "syncdir ", "close a",
		}))
	})

	It("Should record no operation of another kind, and each write size", func() {
		fs := opfs.New(xfs.NewMem(), "sync")
		touch(fs, "a")
		Expect(fs.Ops()).To(Equal([]string{"sync a"}))
		Expect(fs.Sizes()).To(Equal([]int{3}))
	})

	It("Should panic on Sub, which would skip its recorder", func() {
		fs := opfs.New(xfs.NewMem(), "sync")
		Expect(func() { MustSucceed(fs.Sub("a")) }).
			To(PanicWith(ContainSubstring("cannot give the file system of a")))
	})

	It("Should panic on a kind of operation that it cannot record", func() {
		Expect(func() { opfs.New(xfs.NewMem(), "truncate") }).
			To(PanicWith("opfs: an FS cannot record truncate"))
	})

	It("Should count the most files open at once", func() {
		fs := opfs.New(xfs.NewMem())
		a, b := create(fs, "a"), create(fs, "b")
		Expect(a.Close()).To(Succeed())
		c := create(fs, "c")
		Expect(b.Close()).To(Succeed())
		Expect(c.Close()).To(Succeed())
		Expect(fs.MaxOpen()).To(Equal(2))
		Expect(fs.Ops()).To(BeEmpty())
	})

	It("Should fail each sync of one file until cleared, and record none", func() {
		fs := opfs.New(xfs.NewMem(), "sync")
		a, b := DeferClose(create(fs, "a")), DeferClose(create(fs, "b"))
		fs.FailSync("a", errInjected)
		Expect(a.Sync()).To(MatchError(errInjected))
		Expect(b.Sync()).To(Succeed())
		fs.FailSync("a", nil)
		Expect(a.Sync()).To(Succeed())
		Expect(fs.Ops()).To(Equal([]string{"sync b", "sync a"}))
	})

	It("Should fail each directory sync until cleared, and record none", func() {
		fs := opfs.New(xfs.NewMem(), "syncdir")
		fs.FailSyncDir(errInjected)
		Expect(fs.SyncDir("")).To(MatchError(errInjected))
		fs.FailSyncDir(nil)
		Expect(fs.SyncDir("")).To(Succeed())
		Expect(fs.Ops()).To(Equal([]string{"syncdir "}))
	})

	It("Should fail each write until cleared, and write nothing", func() {
		fs := opfs.New(xfs.NewMem(), "write")
		a := DeferClose(create(fs, "a"))
		fs.FailWrite(errInjected)
		Expect(a.WriteAt([]byte{1}, 5)).Error().To(MatchError(errInjected))
		fs.FailWrite(nil)
		Expect(a.WriteAt([]byte{1, 2}, 0)).To(Equal(2))
		Expect(fs.Ops()).To(Equal([]string{"write a 0 2"}))
		Expect(fs.Sizes()).To(Equal([]int{2}))
		Expect(MustSucceed(a.Stat()).Size()).To(Equal(int64(2)))
	})

	It("Should hold each sync until a release, and count the syncs at once", func() {
		fs := opfs.New(xfs.NewMem(), "sync", "syncdir")
		a := DeferClose(create(fs, "a"))
		release := fs.Hold()
		done := make(chan error, 2)
		go func() { done <- a.Sync() }()
		go func() { done <- fs.SyncDir("") }()
		var first, second string
		Eventually(fs.Held()).Should(Receive(&first))
		Eventually(fs.Held()).Should(Receive(&second))
		Expect([]string{first, second}).To(ConsistOf("a", ""))
		Consistently(done).ShouldNot(Receive())
		fs.Unhold()
		close(release)
		Eventually(done).Should(Receive(BeNil()))
		Eventually(done).Should(Receive(BeNil()))
		Expect(a.Sync()).To(Succeed())
		Expect(fs.MaxSyncs()).To(Equal(2))
		Expect(fs.Ops()).To(ConsistOf("sync a", "syncdir ", "sync a"))
	})
})
