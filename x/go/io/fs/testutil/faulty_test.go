// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// openWrapped returns a faultless FS holding one file, "a.bin", containing "hello".
func openWrapped() (*FaultyFS, xfs.File) {
	fs := WrapFaultyFS(OpenMem())
	f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
	Expect(f.Write([]byte("hello"))).To(Equal(5))
	return fs, f
}

var _ = Describe("FaultyFS", func() {
	It("Should pass every operation through when nothing fails", func() {
		fs, f := openWrapped()
		buf := make([]byte, 5)
		Expect(f.ReadAt(buf, 0)).To(Equal(5))
		Expect(string(buf)).To(Equal("hello"))
		Expect(f.Close()).To(Succeed())
		Expect(fs.Rename("a.bin", "b.bin")).To(Succeed())
		Expect(MustSucceed(fs.Stat("b.bin")).Size()).To(Equal(int64(5)))
		Expect(MustSucceed(fs.List(""))).To(HaveLen(1))
		Expect(fs.Remove("b.bin")).To(Succeed())
		Expect(MustSucceed(fs.Exists("b.bin"))).To(BeFalse())
	})

	DescribeTable(
		"Should raise the fault in place of the operation",
		func(opt FaultyFSOption, run func(fs *FaultyFS, f xfs.File) error) {
			fs, f := openWrapped()
			DeferClose(f)
			fs.SetOptions(opt)
			Expect(run(fs, f)).To(MatchError(ErrFault))
		},
		Entry(
			"open",
			WithFailOpen("a.bin"),
			func(fs *FaultyFS, _ xfs.File) error {
				_, err := fs.Open("a.bin", os.O_RDONLY)
				return err
			},
		),
		Entry(
			"read at",
			WithFailReadAt("a.bin"),
			func(_ *FaultyFS, f xfs.File) error {
				_, err := f.ReadAt(make([]byte, 5), 0)
				return err
			},
		),
		Entry(
			"write",
			WithFailWrite("a.bin"),
			func(_ *FaultyFS, f xfs.File) error {
				_, err := f.Write([]byte("more"))
				return err
			},
		),
		Entry(
			"rename",
			WithFailRename("a.bin"),
			func(fs *FaultyFS, _ xfs.File) error { return fs.Rename("a.bin", "b.bin") },
		),
		Entry(
			"stat",
			WithFailStat("a.bin"),
			func(fs *FaultyFS, _ xfs.File) error {
				_, err := fs.Stat("a.bin")
				return err
			},
		),
		Entry(
			"remove",
			WithFailRemove("a.bin"),
			func(fs *FaultyFS, _ xfs.File) error { return fs.Remove("a.bin") },
		),
	)

	It("Should fail an operation on every path when the failure names none", func() {
		fs, f := openWrapped()
		DeferClose(f)
		fs.SetOptions(WithFailStat())
		Expect(fs.Stat("anything.bin")).Error().To(MatchError(ErrFault))
	})

	It("Should hold a failure back until the operation it follows has run", func() {
		fs, f := openWrapped()
		DeferClose(f)
		fs.SetOptions(WithFailStat("a.bin"), WithFailAfter(FaultOpRemove))
		Expect(MustSucceed(fs.Stat("a.bin")).Name()).To(Equal("a.bin"))
		Expect(fs.Remove("b.bin")).To(Succeed())
		Expect(fs.Stat("a.bin")).Error().To(MatchError(ErrFault))
	})

	It("Should stop failing once its options are cleared", func() {
		fs, f := openWrapped()
		DeferClose(f)
		fs.SetOptions(WithFailStat("a.bin"))
		Expect(fs.Stat("a.bin")).Error().To(MatchError(ErrFault))
		fs.SetOptions()
		Expect(MustSucceed(fs.Stat("a.bin")).Size()).To(Equal(int64(5)))
	})

	It("Should count the handles opened against it", func() {
		fs, f := openWrapped()
		Expect(fs.OpenFiles()).To(Equal(1))
		Expect(f.Close()).To(Succeed())
		Expect(fs.OpenFiles()).To(Equal(0))
	})

	It("Should carry its failures and handle count into a sub FS", func() {
		fs := WrapFaultyFS(OpenMem(), WithFailOpen("a.bin"))
		sub := MustSucceed(fs.Sub("nested"))
		f := MustSucceed(sub.Open("b.bin", os.O_CREATE|os.O_RDWR))
		Expect(fs.OpenFiles()).To(Equal(1))
		Expect(f.Close()).To(Succeed())
		Expect(sub.Open("a.bin", os.O_CREATE)).Error().To(MatchError(ErrFault))
	})
})
