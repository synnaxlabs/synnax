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
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// openWrapped returns a faultless FS holding one file, "a.bin", containing "hello".
func openWrapped() (*FaultyFS, xfs.File) {
	fs := WrapFS(OpenMem(), Options{})
	f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
	MustSucceed(f.Write([]byte("hello")))
	return fs, f
}

var _ = Describe("FaultyFS", func() {
	It("Should pass every operation through when nothing fails", func() {
		fs, f := openWrapped()
		buf := make([]byte, 5)
		MustSucceed(f.ReadAt(buf, 0))
		Expect(string(buf)).To(Equal("hello"))
		Expect(f.Close()).To(Succeed())
		Expect(fs.Rename("a.bin", "b.bin")).To(Succeed())
		Expect(MustSucceed(fs.Stat("b.bin")).Size()).To(Equal(int64(5)))
		Expect(fs.Remove("b.bin")).To(Succeed())
		Expect(MustSucceed(fs.Exists("b.bin"))).To(BeFalse())
	})

	DescribeTable(
		"Should raise the fault in place of the operation",
		func(op FaultOp, run func(fs *FaultyFS, f xfs.File) error) {
			fs, f := openWrapped()
			DeferClose(f)
			fs.SetOptions(Options{Fail: []Failure{{Op: op, Name: "a.bin"}}})
			Expect(run(fs, f)).To(MatchError(ErrFault))
		},
		Entry(
			"open",
			FaultOpOpen,
			func(fs *FaultyFS, _ xfs.File) error {
				_, err := fs.Open("a.bin", os.O_RDONLY)
				return err
			},
		),
		Entry(
			"read at",
			FaultOpReadAt,
			func(_ *FaultyFS, f xfs.File) error {
				_, err := f.ReadAt(make([]byte, 5), 0)
				return err
			},
		),
		Entry(
			"write",
			FaultOpWrite,
			func(_ *FaultyFS, f xfs.File) error {
				_, err := f.Write([]byte("more"))
				return err
			},
		),
		Entry(
			"rename",
			FaultOpRename,
			func(fs *FaultyFS, _ xfs.File) error {
				return fs.Rename("a.bin", "b.bin")
			},
		),
		Entry(
			"stat",
			FaultOpStat,
			func(fs *FaultyFS, _ xfs.File) error {
				_, err := fs.Stat("a.bin")
				return err
			},
		),
		Entry(
			"remove",
			FaultOpRemove,
			func(fs *FaultyFS, _ xfs.File) error { return fs.Remove("a.bin") },
		),
	)

	It("Should raise the error the options name", func() {
		errCustom := errors.New("custom")
		fs, f := openWrapped()
		DeferClose(f)
		fs.SetOptions(Options{
			Fail: []Failure{{Op: FaultOpStat, Name: "a.bin"}},
			Err:  errCustom,
		})
		_, err := fs.Stat("a.bin")
		Expect(err).To(MatchError(errCustom))
	})

	It("Should fail an operation on every path when the failure names none", func() {
		fs, f := openWrapped()
		DeferClose(f)
		fs.SetOptions(Options{Fail: []Failure{{Op: FaultOpStat}}})
		_, err := fs.Stat("anything.bin")
		Expect(err).To(MatchError(ErrFault))
	})

	It("Should hold a failure back until the operation it follows has run", func() {
		fs, f := openWrapped()
		DeferClose(f)
		fs.SetOptions(Options{Fail: []Failure{{
			Op:    FaultOpStat,
			Name:  "a.bin",
			After: FaultOpRemove,
		}}})
		MustSucceed(fs.Stat("a.bin"))
		Expect(fs.Remove("b.bin")).To(Succeed())
		_, err := fs.Stat("a.bin")
		Expect(err).To(MatchError(ErrFault))
	})

	It("Should stop failing once its options are cleared", func() {
		fs, f := openWrapped()
		DeferClose(f)
		fs.SetOptions(Options{Fail: []Failure{{Op: FaultOpStat, Name: "a.bin"}}})
		_, err := fs.Stat("a.bin")
		Expect(err).To(MatchError(ErrFault))
		fs.SetOptions(Options{})
		Expect(MustSucceed(fs.Stat("a.bin")).Size()).To(Equal(int64(5)))
	})

	It("Should count the handles opened against it", func() {
		fs, f := openWrapped()
		Expect(fs.OpenFiles()).To(Equal(1))
		Expect(f.Close()).To(Succeed())
		Expect(fs.OpenFiles()).To(Equal(0))
	})

	It("Should carry its failures and handle count into a sub FS", func() {
		fs := WrapFS(
			OpenMem(),
			Options{Fail: []Failure{{Op: FaultOpOpen, Name: "a.bin"}}},
		)
		sub := MustSucceed(fs.Sub("nested"))
		f := MustSucceed(sub.Open("b.bin", os.O_CREATE|os.O_RDWR))
		Expect(fs.OpenFiles()).To(Equal(1))
		Expect(f.Close()).To(Succeed())
		_, err := sub.Open("a.bin", os.O_CREATE)
		Expect(err).To(MatchError(ErrFault))
	})
})
