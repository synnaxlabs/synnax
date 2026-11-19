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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// save replaces the file name in fs with data and syncs the file, but not its
// directory.
func save(fs xfs.FS, name, data string) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
	Expect(f.Write([]byte(data))).To(Equal(len(data)))
	Expect(f.Sync()).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("Crash", func() {
	var fs *fault.FS
	BeforeEach(func() { fs = fault.New() })

	Describe("File data", func() {
		BeforeEach(func() {
			save(fs, "f.txt", "tacocat")
			Expect(fs.SyncDir("")).To(Succeed())
		})

		It("Should keep the bytes that a Sync made durable", func() {
			Expect(read(fs.Crash(), "f.txt")).To(Equal("tacocat"))
		})

		It("Should drop the bytes written after the last Sync", func() {
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
			Expect(f.WriteAt([]byte("burrito"), 0)).To(Equal(7))
			Expect(f.WriteAt([]byte("bowl"), 7)).To(Equal(4))
			Expect(read(fs, "f.txt")).To(Equal("burritobowl"))
			Expect(read(fs.Crash(), "f.txt")).To(Equal("tacocat"))
		})

		It("Should keep a truncation only after a Sync", func() {
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
			Expect(f.Truncate(4)).To(Succeed())
			Expect(read(fs.Crash(), "f.txt")).To(Equal("tacocat"))
			Expect(f.Sync()).To(Succeed())
			Expect(read(fs.Crash(), "f.txt")).To(Equal("taco"))
		})

		It("Should keep the durable bytes of a file cut and grown again", func() {
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
			Expect(f.Truncate(1)).To(Succeed())
			Expect(f.Truncate(7)).To(Succeed())
			Expect(read(fs.Crash(), "f.txt")).To(Equal("tacocat"))
			Expect(f.Sync()).To(Succeed())
			Expect(read(fs.Crash(), "f.txt")).To(Equal("t\x00\x00\x00\x00\x00\x00"))
		})

		It("Should make each changed block durable on Sync", func() {
			data := bytes.Repeat([]byte("abcdefg"), 2000)
			save(fs, "f.bin", string(data))
			f := MustOpen(fs.Open("f.bin", os.O_WRONLY))
			Expect(f.WriteAt([]byte("BURRITO"), 5000)).To(Equal(7))
			Expect(f.WriteAt(bytes.Repeat([]byte("z"), 9000), 13990)).To(Equal(9000))
			Expect(f.Sync()).To(Succeed())
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(read(fs.Crash(), "f.bin")).To(Equal(read(fs, "f.bin")))
		})
	})

	Describe("Directory entries", func() {
		It("Should drop a synced file whose directory was not synced", func() {
			save(fs, "f.txt", "tacocat")
			Expect(fs.Crash().Exists("f.txt")).To(BeFalse())
		})

		It("Should keep an empty file when only its entry is durable", func() {
			write(fs, "f.txt", "tacocat")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(read(fs.Crash(), "f.txt")).To(BeEmpty())
		})

		It("Should put a rename back, or keep it after SyncDir", func() {
			save(fs, "a.txt", "tacocat")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.Rename("a.txt", "b.txt")).To(Succeed())
			image := fs.Crash()
			Expect(read(image, "a.txt")).To(Equal("tacocat"))
			Expect(image.Exists("b.txt")).To(BeFalse())
			Expect(fs.SyncDir("")).To(Succeed())
			image = fs.Crash()
			Expect(read(image, "b.txt")).To(Equal("tacocat"))
			Expect(image.Exists("a.txt")).To(BeFalse())
		})

		It(
			"Should bring a removed file back, or keep it removed after SyncDir",
			func() {
				save(fs, "f.txt", "tacocat")
				Expect(fs.SyncDir("")).To(Succeed())
				Expect(fs.Remove("f.txt")).To(Succeed())
				Expect(read(fs.Crash(), "f.txt")).To(Equal("tacocat"))
				Expect(fs.SyncDir("")).To(Succeed())
				Expect(fs.Crash().Exists("f.txt")).To(BeFalse())
			},
		)

		It("Should drop a directory that its parent did not sync", func() {
			MustSucceed(fs.Sub("a"))
			save(fs, "a/f.txt", "tacocat")
			Expect(fs.SyncDir("a")).To(Succeed())
			Expect(fs.Crash().Exists("a")).To(BeFalse())
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(read(fs.Crash(), "a/f.txt")).To(Equal("tacocat"))
		})

		It("Should sync only the directory that SyncDir names", func() {
			MustSucceed(fs.Sub("a/b"))
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.SyncDir("a")).To(Succeed())
			save(fs, "a/f.txt", "tacocat")
			save(fs, "a/b/f.txt", "tacocat")
			Expect(fs.SyncDir("a/b")).To(Succeed())
			image := fs.Crash()
			Expect(image.Exists("a/f.txt")).To(BeFalse())
			Expect(read(image, "a/b/f.txt")).To(Equal("tacocat"))
		})

		It("Should sync a directory through the Sync of its handle", func() {
			save(fs, "f.txt", "tacocat")
			dir := MustOpen(fs.Open("", os.O_RDONLY))
			Expect(dir.Sync()).To(Succeed())
			Expect(read(fs.Crash(), "f.txt")).To(Equal("tacocat"))
		})
	})

	It("Should leave the FS that crashed as it was", func() {
		write(fs, "f.txt", "tacocat")
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY|os.O_APPEND))
		Expect(fs.Crash().Exists("f.txt")).To(BeFalse())
		Expect(f.Write([]byte("s"))).To(Equal(1))
		Expect(read(fs, "f.txt")).To(Equal("tacocats"))
	})

	It("Should return an FS whose state is all durable", func() {
		save(fs, "f.txt", "tacocat")
		Expect(fs.SyncDir("")).To(Succeed())
		image := fs.Crash()
		write(image, "g.txt", "burrito")
		again := image.Crash()
		Expect(read(again, "f.txt")).To(Equal("tacocat"))
		Expect(again.Exists("g.txt")).To(BeFalse())
	})

	It("Should keep the root of a Sub in the image", func() {
		sub := MustSucceed(fs.Sub("a"))
		save(sub, "f.txt", "tacocat")
		Expect(sub.SyncDir("")).To(Succeed())
		Expect(fs.SyncDir("")).To(Succeed())
		Expect(read(fs.Crash(), "a/f.txt")).To(Equal("tacocat"))
	})
})
