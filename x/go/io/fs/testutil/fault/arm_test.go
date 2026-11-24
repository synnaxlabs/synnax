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
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arm", func() {
	var disk *fault.FS
	BeforeEach(func() { disk = MustSucceed(fault.New(fault.Config{})) })
	write := func(f xfs.File, b string) error {
		GinkgoHelper()
		_, err := f.Write([]byte(b))
		return err
	}

	It("Should fail the sync at SyncErrorAt, counted from the arm", func() {
		f := MustSucceed(disk.Open("a", os.O_CREATE|os.O_RDWR))
		Expect(write(f, "a")).To(Succeed())
		Expect(f.Sync()).To(Succeed())
		Expect(disk.Arm(fault.Config{SyncErrorAt: 2})).To(Succeed())
		Expect(write(f, "b")).To(Succeed())
		Expect(f.Sync()).To(Succeed())
		Expect(write(f, "c")).To(Succeed())
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(write(f, "d")).To(Succeed())
		Expect(f.Sync()).To(Succeed())
		Expect(f.Close()).To(Succeed())
	})

	It("Should not count a sync with no change toward SyncErrorAt", func() {
		f := MustSucceed(disk.Open("a", os.O_CREATE|os.O_RDWR))
		Expect(disk.Arm(fault.Config{SyncErrorAt: 1})).To(Succeed())
		Expect(f.Sync()).To(Succeed())
		Expect(write(f, "a")).To(Succeed())
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(f.Close()).To(Succeed())
	})

	It("Should refuse a config with a rate above 1", func() {
		Expect(disk.Arm(fault.Config{SyncErrorRate: 2})).
			To(MatchError(ContainSubstring("sync_error_rate")))
	})
})

var _ = Describe("Fill", func() {
	It("Should leave room bytes past what the disk holds", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		f := MustSucceed(disk.Open("a", os.O_CREATE|os.O_RDWR))
		MustSucceed(f.Write(make([]byte, 1<<20)))
		disk.Fill(3)
		n, err := f.Write([]byte("abcde"))
		Expect(err).To(MatchError(syscall.ENOSPC))
		Expect(n).To(Equal(3))
		Expect(f.Close()).To(Succeed())
	})

	It("Should cap a disk that holds no bytes at a room of zero", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		f := MustSucceed(disk.Open("a", os.O_CREATE|os.O_RDWR))
		disk.Fill(0)
		n, err := f.Write([]byte("a"))
		Expect(err).To(MatchError(syscall.ENOSPC))
		Expect(n).To(BeZero())
		Expect(f.Close()).To(Succeed())
	})
})

var _ = Describe("Unsynced", func() {
	It("Should report a written block or a created entry until a sync makes it "+
		"durable", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		Expect(disk.Unsynced()).To(BeFalse())
		f := MustSucceed(disk.Open("a", os.O_CREATE|os.O_RDWR))
		Expect(disk.Unsynced()).To(BeTrue())
		Expect(disk.SyncDir("")).To(Succeed())
		Expect(disk.Unsynced()).To(BeFalse())
		MustSucceed(f.Write([]byte("a")))
		Expect(disk.Unsynced()).To(BeTrue())
		Expect(f.Sync()).To(Succeed())
		Expect(disk.Unsynced()).To(BeFalse())
		Expect(f.Close()).To(Succeed())
	})

	It("Should report no change under a directory whose removal is durable", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		MustSucceed(disk.Sub("tmp"))
		f := MustSucceed(disk.Open("tmp/f", os.O_CREATE|os.O_RDWR))
		Expect(f.Close()).To(Succeed())
		Expect(disk.Remove("tmp")).To(Succeed())
		Expect(disk.SyncDir("")).To(Succeed())
		Expect(disk.Unsynced()).To(BeFalse())
	})

	It("Should keep the changes under a directory that a durable rename moved", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		MustSucceed(disk.Sub("tmp"))
		f := MustSucceed(disk.Open("tmp/f", os.O_CREATE|os.O_RDWR))
		Expect(f.Close()).To(Succeed())
		Expect(disk.Rename("tmp", "kept")).To(Succeed())
		Expect(disk.SyncDir("")).To(Succeed())
		Expect(disk.Unsynced()).To(BeTrue())
		Expect(disk.SyncDir("kept")).To(Succeed())
		Expect(disk.Unsynced()).To(BeFalse())
	})
})

var _ = Describe("ReadFaultAt", func() {
	It("Should flip one bit of the block read of that number, and give a fault",
		func() {
			disk := MustSucceed(fault.New(fault.Config{}))
			f := MustSucceed(disk.Open("a", os.O_CREATE|os.O_RDWR))
			MustSucceed(f.Write([]byte("hello world")))
			Expect(f.Sync()).To(Succeed())
			Expect(f.Close()).To(Succeed())
			Expect(disk.SyncDir("")).To(Succeed())
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				Image: fault.Config{ReadFaultAt: 2},
			}))
			g := MustSucceed(image.Open("a", os.O_RDONLY))
			b := make([]byte, 11)
			MustSucceed(g.ReadAt(b, 0))
			Expect(string(b)).To(Equal("hello world"))
			Expect(image.ReadFaults()).To(BeEmpty())
			MustSucceed(g.ReadAt(b, 0))
			Expect(string(b)).ToNot(Equal("hello world"))
			Expect(image.ReadFaults()).To(HaveLen(1))
			Expect(g.Close()).To(Succeed())
		})
})
