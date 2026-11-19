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
	"math"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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

// strict returns fs after a crash that keeps only the durable state.
func strict(fs *fault.FS) *fault.FS {
	GinkgoHelper()
	return MustSucceed(fs.Crash(fault.CrashConfig{}))
}

var _ = Describe("Crash", func() {
	var fs *fault.FS
	BeforeEach(func() { fs = MustSucceed(fault.New(fault.Config{})) })

	Describe("File data", func() {
		BeforeEach(func() {
			save(fs, "f.txt", "tacocat")
			Expect(fs.SyncDir("")).To(Succeed())
		})

		It("Should keep the bytes that a Sync made durable", func() {
			Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
		})

		It("Should drop the bytes written after the last Sync", func() {
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
			Expect(f.WriteAt([]byte("burrito"), 0)).To(Equal(7))
			Expect(f.WriteAt([]byte("bowl"), 7)).To(Equal(4))
			Expect(read(fs, "f.txt")).To(Equal("burritobowl"))
			Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
		})

		It("Should keep a truncation only after a Sync", func() {
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
			Expect(f.Truncate(4)).To(Succeed())
			Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
			Expect(f.Sync()).To(Succeed())
			Expect(read(strict(fs), "f.txt")).To(Equal("taco"))
		})

		It("Should keep the durable bytes of a file cut and grown again", func() {
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
			Expect(f.Truncate(1)).To(Succeed())
			Expect(f.Truncate(7)).To(Succeed())
			Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
			Expect(f.Sync()).To(Succeed())
			Expect(read(strict(fs), "f.txt")).To(Equal("t\x00\x00\x00\x00\x00\x00"))
		})

		It("Should make each changed block durable on Sync", func() {
			data := bytes.Repeat([]byte("abcdefg"), 2000)
			save(fs, "f.bin", string(data))
			f := MustOpen(fs.Open("f.bin", os.O_WRONLY))
			Expect(f.WriteAt([]byte("BURRITO"), 5000)).To(Equal(7))
			Expect(f.WriteAt(bytes.Repeat([]byte("z"), 9000), 13990)).To(Equal(9000))
			Expect(f.Sync()).To(Succeed())
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(read(strict(fs), "f.bin")).To(Equal(read(fs, "f.bin")))
		})
	})

	Describe("Directory entries", func() {
		It("Should drop a synced file whose directory was not synced", func() {
			save(fs, "f.txt", "tacocat")
			Expect(strict(fs).Exists("f.txt")).To(BeFalse())
		})

		It("Should keep an empty file when only its entry is durable", func() {
			write(fs, "f.txt", "tacocat")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(read(strict(fs), "f.txt")).To(BeEmpty())
		})

		It("Should put a rename back, or keep it after SyncDir", func() {
			save(fs, "a.txt", "tacocat")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.Rename("a.txt", "b.txt")).To(Succeed())
			image := strict(fs)
			Expect(read(image, "a.txt")).To(Equal("tacocat"))
			Expect(image.Exists("b.txt")).To(BeFalse())
			Expect(fs.SyncDir("")).To(Succeed())
			image = strict(fs)
			Expect(read(image, "b.txt")).To(Equal("tacocat"))
			Expect(image.Exists("a.txt")).To(BeFalse())
		})

		It(
			"Should bring a removed file back, or keep it removed after SyncDir",
			func() {
				save(fs, "f.txt", "tacocat")
				Expect(fs.SyncDir("")).To(Succeed())
				Expect(fs.Remove("f.txt")).To(Succeed())
				Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
				Expect(fs.SyncDir("")).To(Succeed())
				Expect(strict(fs).Exists("f.txt")).To(BeFalse())
			},
		)

		It("Should drop a directory that its parent did not sync", func() {
			MustSucceed(fs.Sub("a"))
			save(fs, "a/f.txt", "tacocat")
			Expect(fs.SyncDir("a")).To(Succeed())
			Expect(strict(fs).Exists("a")).To(BeFalse())
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(read(strict(fs), "a/f.txt")).To(Equal("tacocat"))
		})

		It("Should sync only the directory that SyncDir names", func() {
			MustSucceed(fs.Sub("a/b"))
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.SyncDir("a")).To(Succeed())
			save(fs, "a/f.txt", "tacocat")
			save(fs, "a/b/f.txt", "tacocat")
			Expect(fs.SyncDir("a/b")).To(Succeed())
			image := strict(fs)
			Expect(image.Exists("a/f.txt")).To(BeFalse())
			Expect(read(image, "a/b/f.txt")).To(Equal("tacocat"))
		})

		It("Should sync a directory through the Sync of its handle", func() {
			save(fs, "f.txt", "tacocat")
			dir := MustOpen(fs.Open("", os.O_RDONLY))
			Expect(dir.Sync()).To(Succeed())
			Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
		})
	})

	Describe("Partial survival", func() {
		const (
			blockSize = 4096
			blocks    = 64
			size      = blocks * blockSize
		)
		var f xfs.File
		BeforeEach(func() {
			save(fs, "f.bin", strings.Repeat("a", size))
			Expect(fs.SyncDir("")).To(Succeed())
			f = MustOpen(fs.Open("f.bin", os.O_WRONLY))
		})
		// crash returns the bytes of the file after a crash with cfg.
		crash := func(cfg fault.CrashConfig) string {
			GinkgoHelper()
			return read(MustSucceed(fs.Crash(cfg)), "f.bin")
		}
		// each calls check on every whole block of data.
		each := func(data string, check func(block string)) {
			GinkgoHelper()
			Expect(len(data) % blockSize).To(BeZero())
			for i := 0; i < len(data); i += blockSize {
				check(data[i : i+blockSize])
			}
		}
		overwrite := func() {
			GinkgoHelper()
			Expect(f.WriteAt(bytes.Repeat([]byte("b"), size), 0)).To(Equal(size))
		}

		It("Should keep every unsynced byte at a survival rate of 1", func() {
			overwrite()
			Expect(f.WriteAt([]byte("burrito"), size+100)).To(Equal(7))
			Expect(crash(fault.CrashConfig{DataSurvivalRate: 1})).
				To(Equal(read(fs, "f.bin")))
		})

		It("Should keep each block whole or leave its durable bytes", func() {
			overwrite()
			kept := 0
			each(crash(fault.CrashConfig{DataSurvivalRate: 0.5}), func(block string) {
				Expect(block).To(Or(
					Equal(strings.Repeat("a", blockSize)),
					Equal(strings.Repeat("b", blockSize)),
				))
				kept += strings.Count(block[:1], "b")
			})
			Expect(kept).To(And(BeNumerically(">", 0), BeNumerically("<", blocks)))
		})

		It("Should fix the image by the run key", func() {
			overwrite()
			cfg := fault.CrashConfig{RunKey: 1, DataSurvivalRate: 0.5, TornRate: 0.5}
			image := crash(cfg)
			Expect(crash(cfg)).To(Equal(image))
			cfg.RunKey = 2
			Expect(crash(cfg)).ToNot(Equal(image))
		})

		It("Should keep only a prefix of a torn block", func() {
			overwrite()
			cfg := fault.CrashConfig{DataSurvivalRate: 1, TornRate: 1}
			torn := 0
			each(crash(cfg), func(block string) {
				prefix := len(block) - len(strings.TrimLeft(block, "b"))
				Expect(prefix).To(BeNumerically("<", blockSize))
				Expect(block[prefix:]).To(Equal(strings.Repeat("a", blockSize-prefix)))
				torn += min(prefix, 1)
			})
			Expect(torn).To(BeNumerically(">", blocks/2))
		})

		It("Should read a lost block under a surviving block as zeros", func() {
			Expect(f.WriteAt(bytes.Repeat([]byte("b"), size), size)).To(Equal(size))
			image := crash(fault.CrashConfig{DataSurvivalRate: 0.5})
			Expect(image[:size]).To(Equal(strings.Repeat("a", size)))
			lost, last := 0, ""
			each(image[size:], func(block string) {
				Expect(block).To(Or(
					Equal(strings.Repeat("\x00", blockSize)),
					Equal(strings.Repeat("b", blockSize)),
				))
				lost += strings.Count(block[:1], "\x00")
				last = block[:1]
			})
			Expect(lost).To(BeNumerically(">", 0))
			Expect(last).To(Equal("b"))
		})

		It("Should keep a cut or the durable length", func() {
			Expect(f.Truncate(blockSize)).To(Succeed())
			lengths := set.Set[int]{}
			for runKey := range uint64(32) {
				cfg := fault.CrashConfig{RunKey: runKey, DataSurvivalRate: 0.5}
				lengths.Add(len(crash(cfg)))
			}
			Expect(lengths.Slice()).To(ConsistOf(blockSize, size))
		})

		DescribeTable("Should fail a rate that is not from 0 to 1",
			func(cfg fault.CrashConfig) {
				Expect(fs.Crash(cfg)).Error().To(MatchError(validate.ErrValidation))
			},
			Entry("low survival", fault.CrashConfig{DataSurvivalRate: -0.1}),
			Entry("high survival", fault.CrashConfig{DataSurvivalRate: 1.1}),
			Entry("no number", fault.CrashConfig{DataSurvivalRate: math.NaN()}),
			Entry("low torn", fault.CrashConfig{TornRate: -0.1}),
			Entry("high torn", fault.CrashConfig{TornRate: 1.1}),
			Entry("low entry survival", fault.CrashConfig{EntrySurvivalRate: -0.1}),
			Entry("high entry survival", fault.CrashConfig{EntrySurvivalRate: 1.1}),
		)
	})

	Describe("Surviving directory changes", func() {
		// list returns the names in the root of image, each file with its bytes.
		list := func(image *fault.FS) string {
			GinkgoHelper()
			infos := MustSucceed(image.List(""))
			parts := make([]string, 0, len(infos))
			for _, info := range infos {
				part := info.Name()
				if !info.IsDir() {
					part += "=" + read(image, info.Name())
				}
				parts = append(parts, part)
			}
			return strings.Join(parts, " ")
		}
		// outcomes returns each distinct list that 32 crashes give when a change has
		// one chance in two to survive.
		outcomes := func() []string {
			GinkgoHelper()
			seen := set.Set[string]{}
			for runKey := range uint64(32) {
				cfg := fault.CrashConfig{RunKey: runKey, EntrySurvivalRate: 0.5}
				seen.Add(list(MustSucceed(fs.Crash(cfg))))
			}
			return seen.Slice()
		}
		all := fault.CrashConfig{EntrySurvivalRate: 1}

		It("Should keep every change at a survival rate of 1", func() {
			save(fs, "a.txt", "a")
			save(fs, "b.txt", "b")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.Rename("a.txt", "c.txt")).To(Succeed())
			Expect(fs.Remove("b.txt")).To(Succeed())
			save(fs, "d.txt", "d")
			MustSucceed(fs.Sub("e"))
			Expect(list(MustSucceed(fs.Crash(all)))).To(Equal("c.txt=a d.txt=d e"))
		})

		It("Should decide each change apart", func() {
			save(fs, "a.txt", "a")
			save(fs, "b.txt", "b")
			Expect(
				outcomes(),
			).To(ConsistOf("", "a.txt=a", "b.txt=b", "a.txt=a b.txt=b"))
		})

		It("Should keep a rename whole", func() {
			save(fs, "a.txt", "tacocat")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.Rename("a.txt", "b.txt")).To(Succeed())
			Expect(outcomes()).To(ConsistOf("a.txt=tacocat", "b.txt=tacocat"))
		})

		It("Should keep a rename onto a file whole", func() {
			save(fs, "final.txt", "old")
			save(fs, "tmp.txt", "new")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.Rename("tmp.txt", "final.txt")).To(Succeed())
			Expect(
				outcomes(),
			).To(ConsistOf("final.txt=old tmp.txt=new", "final.txt=new"))
		})

		It("Should lose a change with the change that it depends on", func() {
			save(fs, "a.txt", "tacocat")
			Expect(fs.Rename("a.txt", "b.txt")).To(Succeed())
			Expect(outcomes()).To(ConsistOf("", "a.txt=tacocat", "b.txt=tacocat"))
		})

		It("Should not replace a file whose remove was lost", func() {
			save(fs, "f.txt", "old")
			Expect(fs.SyncDir("")).To(Succeed())
			Expect(fs.Remove("f.txt")).To(Succeed())
			save(fs, "f.txt", "new")
			Expect(outcomes()).To(ConsistOf("f.txt=old", "", "f.txt=new"))
		})

		Describe("Rename across directories", func() {
			BeforeEach(func() {
				MustSucceed(fs.Sub("a"))
				MustSucceed(fs.Sub("b"))
				save(fs, "a/f.txt", "tacocat")
				for _, dir := range []string{"", "a", "b"} {
					Expect(fs.SyncDir(dir)).To(Succeed())
				}
				Expect(fs.Rename("a/f.txt", "b/f.txt")).To(Succeed())
			})

			It("Should keep both names when only the target is synced", func() {
				Expect(fs.SyncDir("b")).To(Succeed())
				image := strict(fs)
				Expect(read(image, "a/f.txt")).To(Equal("tacocat"))
				write(image, "a/f.txt", "burrito")
				Expect(read(image, "b/f.txt")).To(Equal("burrito"))
				image = MustSucceed(fs.Crash(all))
				Expect(image.Exists("a/f.txt")).To(BeFalse())
				Expect(read(image, "b/f.txt")).To(Equal("tacocat"))
			})

			It("Should lose the file when only the source is synced", func() {
				Expect(fs.SyncDir("a")).To(Succeed())
				image := strict(fs)
				Expect(image.Exists("a/f.txt")).To(BeFalse())
				Expect(image.Exists("b/f.txt")).To(BeFalse())
				Expect(read(MustSucceed(fs.Crash(all)), "b/f.txt")).To(Equal("tacocat"))
			})
		})
	})

	It("Should leave the FS that crashed as it was", func() {
		write(fs, "f.txt", "tacocat")
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY|os.O_APPEND))
		Expect(strict(fs).Exists("f.txt")).To(BeFalse())
		Expect(f.Write([]byte("s"))).To(Equal(1))
		Expect(read(fs, "f.txt")).To(Equal("tacocats"))
	})

	It("Should return an FS whose state is all durable", func() {
		save(fs, "f.txt", "tacocat")
		Expect(fs.SyncDir("")).To(Succeed())
		image := strict(fs)
		write(image, "g.txt", "burrito")
		again := strict(image)
		Expect(read(again, "f.txt")).To(Equal("tacocat"))
		Expect(again.Exists("g.txt")).To(BeFalse())
	})

	It("Should keep the root of a Sub in the image", func() {
		sub := MustSucceed(fs.Sub("a"))
		save(sub, "f.txt", "tacocat")
		Expect(sub.SyncDir("")).To(Succeed())
		Expect(fs.SyncDir("")).To(Succeed())
		Expect(read(strict(fs), "a/f.txt")).To(Equal("tacocat"))
	})
})
