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
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// faulty returns an FS with the faults of cfg that holds one durable file, f.txt, with
// the bytes "tacocat".
func faulty(cfg fault.Config) *fault.FS {
	GinkgoHelper()
	base := MustSucceed(fault.New(fault.Config{}))
	save(base, "f.txt", "tacocat")
	Expect(base.SyncDir("")).To(Succeed())
	return MustSucceed(base.Crash(fault.CrashConfig{Image: cfg}))
}

var _ = Describe("Sync errors", func() {
	always := fault.Config{SyncErrorRate: 1}

	It("Should fail once, drop the changes, and succeed on the next Sync", func() {
		fs := faulty(always)
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(f.WriteAt([]byte("burrito"), 0)).To(Equal(7))
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(read(fs, "f.txt")).To(Equal("burrito"))
		Expect(f.Sync()).To(Succeed())
		Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
	})

	It("Should give the error of a failed Sync once to each open handle", func() {
		fs := faulty(always)
		writer := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		syncer := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(writer.WriteAt([]byte("burrito"), 0)).To(Equal(7))
		Expect(syncer.Sync()).To(MatchError(syscall.EIO))
		Expect(writer.Sync()).To(MatchError(syscall.EIO))
		Expect(writer.Sync()).To(Succeed())
		Expect(syncer.Sync()).To(Succeed())
		late := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(late.Sync()).To(Succeed())
		Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
	})

	It("Should draw again for a Sync that follows a new write", func() {
		fs := faulty(always)
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(f.WriteAt([]byte("burrito"), 0)).To(Equal(7))
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(f.WriteAt([]byte("bowl"), 7)).To(Equal(4))
		Expect(f.Sync()).To(MatchError(syscall.EIO))
	})

	It("Should make a dropped block durable when a later write touches it", func() {
		for runKey := range uint64(64) {
			fs := faulty(fault.Config{RunKey: runKey, SyncErrorRate: 0.5})
			f := MustSucceed(fs.Open("f.txt", os.O_WRONLY))
			Expect(f.WriteAt([]byte("burrito"), 0)).To(Equal(7))
			dropped := errors.Is(f.Sync(), syscall.EIO)
			Expect(f.WriteAt([]byte("B"), 0)).To(Equal(1))
			synced := f.Sync() == nil
			Expect(f.Close()).To(Succeed())
			if dropped && synced {
				Expect(read(strict(fs), "f.txt")).To(Equal("Burrito"))
				return
			}
		}
		Fail("no run key gave a failed Sync and then a good one")
	})

	It("Should fail a Sync that only shrinks a file", func() {
		fs := faulty(always)
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(f.Truncate(3)).To(Succeed())
		Expect(f.Sync()).To(MatchError(syscall.EIO))
	})

	It("Should read the durable bytes of an evicted block at once", func() {
		fs := faulty(fault.Config{SyncErrorRate: 1, EvictionRate: 1})
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(f.WriteAt([]byte("burritobowl"), 0)).To(Equal(11))
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(read(fs, "f.txt")).To(Equal("tacocat\x00\x00\x00\x00"))
	})

	It("Should keep a dropped tail as zeros once a later Sync succeeds", func() {
		fs := faulty(always)
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(f.WriteAt([]byte("bowl"), 7)).To(Equal(4))
		Expect(f.Sync()).To(MatchError(syscall.EIO))
		Expect(read(strict(fs), "f.txt")).To(Equal("tacocat"))
		Expect(f.Sync()).To(Succeed())
		Expect(read(strict(fs), "f.txt")).To(Equal("tacocat\x00\x00\x00\x00"))
	})

	It("Should not fail a Sync or a SyncDir with nothing to make durable", func() {
		fs := faulty(always)
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Expect(f.Sync()).To(Succeed())
		Expect(fs.SyncDir("")).To(Succeed())
	})

	It("Should not fail a Sync after a Sync made the changes durable", func() {
		fs := faulty(fault.Config{SyncErrorRate: 0.5})
		f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
		Eventually(func() error {
			Expect(f.WriteAt([]byte("burrito"), 0)).To(Equal(7))
			return f.Sync()
		}).Should(Succeed())
		for range 64 {
			Expect(f.Sync()).To(Succeed())
		}
	})

	It("Should fail a SyncDir and keep its changes for the next one", func() {
		fs := faulty(always)
		write(fs, "g.txt", "burrito")
		Expect(fs.SyncDir("")).To(MatchError(syscall.EIO))
		dir := MustOpen(fs.Open("", os.O_RDONLY))
		Expect(dir.Sync()).To(MatchError(syscall.EIO))
		Expect(strict(fs).Exists("g.txt")).To(BeFalse())
		image := MustSucceed(fs.Crash(fault.CrashConfig{EntrySurvivalRate: 1}))
		Expect(image.Exists("g.txt")).To(BeTrue())
	})

	It(
		"Should fail a SyncDir that makes a rename out of the directory durable",
		func() {
			base := MustSucceed(fault.New(fault.Config{}))
			MustSucceed(base.Sub("a"))
			MustSucceed(base.Sub("b"))
			Expect(base.SyncDir("")).To(Succeed())
			fs := MustSucceed(base.Crash(fault.CrashConfig{Image: always}))
			write(fs, "a/tmp", "burrito")
			Expect(fs.Rename("a/tmp", "b/tmp")).To(Succeed())
			Expect(fs.SyncDir("a")).To(MatchError(syscall.EIO))
			Expect(strict(fs).Exists("b/tmp")).To(BeFalse())
			image := MustSucceed(fs.Crash(fault.CrashConfig{EntrySurvivalRate: 1}))
			Expect(image.Exists("b/tmp")).To(BeTrue())
		},
	)

	It("Should let a SyncDir that failed succeed on a later try", func() {
		for runKey := range uint64(64) {
			fs := faulty(fault.Config{RunKey: runKey, SyncErrorRate: 0.5})
			write(fs, "g.txt", "burrito")
			if !errors.Is(fs.SyncDir(""), syscall.EIO) {
				continue
			}
			Eventually(func() error { return fs.SyncDir("") }).Should(Succeed())
			Expect(strict(fs).Exists("g.txt")).To(BeTrue())
			return
		}
		Fail("no run key gave a failed SyncDir")
	})

	It("Should fix the Syncs that fail by the run key", func() {
		// results has an E for each of 32 Syncs that fails, and a dot for the others.
		results := func(runKey uint64) string {
			GinkgoHelper()
			fs := faulty(fault.Config{RunKey: runKey, SyncErrorRate: 0.5})
			f := MustOpen(fs.Open("f.txt", os.O_WRONLY))
			var out strings.Builder
			for range 32 {
				Expect(f.WriteAt([]byte("b"), 0)).To(Equal(1))
				if errors.Is(f.Sync(), syscall.EIO) {
					out.WriteString("E")
				} else {
					out.WriteString(".")
				}
			}
			return out.String()
		}
		Expect(results(1)).To(Equal(results(1)))
		Expect(results(1)).ToNot(Equal(results(2)))
		Expect(results(1)).To(And(ContainSubstring("E"), ContainSubstring(".")))
	})

	DescribeTable("Should fail a rate that is not from 0 to 1",
		func(cfg fault.Config) {
			Expect(fault.New(cfg)).Error().To(MatchError(validate.ErrValidation))
			Expect(MustSucceed(fault.New(fault.Config{})).Crash(
				fault.CrashConfig{Image: cfg},
			)).Error().To(MatchError(validate.ErrValidation))
		},
		Entry("low sync error", fault.Config{SyncErrorRate: -0.1}),
		Entry("high sync error", fault.Config{SyncErrorRate: 1.1}),
		Entry("low eviction", fault.Config{EvictionRate: -0.1}),
		Entry("high eviction", fault.Config{EvictionRate: 1.1}),
	)
})
