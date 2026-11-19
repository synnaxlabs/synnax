// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashcheck2_test

import (
	"bytes"
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

func corrupt(err error) bool { return errors.Is(err, errCorrupt) }

// violations returns one line for each of 32 run keys on which Run fails.
func violations(
	ctx context.Context,
	open func(context.Context, xfs.FS) (crash.Engine, error),
	profile crash.Profile,
) []string {
	GinkgoHelper()
	var out []string
	for runKey := range uint64(32) {
		err := crash.Run(ctx, crash.Config{
			RunKey:  runKey,
			Open:    open,
			Corrupt: corrupt,
			Commits: 20,
			Stride:  1,
			Profile: profile,
		})
		if err != nil {
			out = append(out, err.Error())
		}
	}
	return out
}

var _ = Describe("Round 2", func() {
	It("Should give a good second read of a record read in two calls", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		f := MustSucceed(disk.Open("log", os.O_CREATE|os.O_RDWR))
		want := make([]byte, 2*4096)
		for i := range want {
			want[i] = byte(i * 7)
		}
		MustSucceed(f.WriteAt(want, 0))
		Expect(f.Sync()).To(Succeed())
		Expect(f.Close()).To(Succeed())
		Expect(disk.SyncDir("")).To(Succeed())
		// The record is [4080, 4200): a 16-byte header at the end of block 0, and a
		// body in block 1. The reader takes it in two calls, and reads it again when
		// it is bad.
		read := func(f xfs.File) []byte {
			head, body := make([]byte, 16), make([]byte, 104)
			MustSucceed(f.ReadAt(head, 4080))
			MustSucceed(f.ReadAt(body, 4096))
			return append(head, body...)
		}
		var bad []string
		for runKey := range uint64(64) {
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				RunKey: runKey,
				Image:  fault.Config{RunKey: runKey, ReadCorruptionRate: 0.5},
			}))
			f := MustSucceed(image.Open("log", os.O_RDONLY))
			if !bytes.Equal(read(f), want[4080:4200]) &&
				!bytes.Equal(read(f), want[4080:4200]) {
				bad = append(bad, fmt.Sprintf("run key %d", runKey))
			}
			Expect(f.Close()).To(Succeed())
		}
		Expect(bad).To(BeEmpty())
	})

	It(
		"Should pass a correct log that reads one slot in each call",
		func(ctx SpecContext) {
			Expect(violations(ctx, openSlotLog(slotOpts{}), crash.OneTimeReadFaults)).
				To(BeEmpty())
		},
	)

	It(
		"Should pass a correct log that reads aligned 4 KiB blocks",
		func(ctx SpecContext) {
			Expect(violations(
				ctx,
				openSlotLog(slotOpts{blockReads: true}),
				crash.OneTimeReadFaults,
			)).To(BeEmpty())
		},
	)

	It(
		"Should lose durable commits through a recovery that renames an unsynced rewrite",
		func(ctx SpecContext) {
			open := openSlotLog(slotOpts{rewrite: true})
			disk := MustSucceed(fault.New(fault.Config{}))
			e := MustSucceed(open(ctx, disk))
			for id := uint64(1); id <= 3; id++ {
				Expect(e.Commit(ctx, crash.Record{ID: id, Payload: []byte{byte(id)}},
					crash.Durable)).To(Equal(crash.Durable))
			}
			Expect(e.Close()).To(Succeed())
			image := MustSucceed(disk.Crash(fault.CrashConfig{}))
			e = MustSucceed(open(ctx, image))
			Expect(MustSucceed(e.Recovered(ctx))).To(HaveLen(3))
			Expect(e.Close()).To(Succeed())
			again := MustSucceed(image.Crash(fault.CrashConfig{}))
			e = MustSucceed(open(ctx, again))
			Expect(MustSucceed(e.Recovered(ctx))).To(BeEmpty())
			Expect(e.Close()).To(Succeed())
		},
	)

	It("Should catch a recovery that renames an unsynced rewrite over the log",
		func(ctx SpecContext) {
			Expect(violations(ctx, openSlotLog(slotOpts{}), crash.Strict)).To(BeEmpty())
			Expect(violations(ctx, openSlotLog(slotOpts{rewrite: true}), crash.Strict)).
				ToNot(BeEmpty())
		})

	It("Should pass a correct counted log under OldCorruption", func(ctx SpecContext) {
		Expect(
			violations(ctx, openSlotLog(slotOpts{counted: true}), crash.OldCorruption),
		).
			To(BeEmpty())
	})

	It("Should catch a recovery that cuts durable frames and then refuses",
		func(ctx SpecContext) {
			Expect(violations(
				ctx,
				openSlotLog(slotOpts{counted: true, repairs: true}),
				crash.OldCorruption,
			)).ToNot(BeEmpty())
		})
})
