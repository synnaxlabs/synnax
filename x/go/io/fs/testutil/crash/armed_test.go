// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash_test

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	. "github.com/synnaxlabs/x/testutil"
)

// openPadded opens a plain log after it writes and syncs a file of 1 MiB, more than
// any room that a full disk leaves.
func openPadded(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
	if ok, err := fs.Exists("pad"); err != nil || !ok {
		f, err := fs.Open("pad", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		if _, err = f.Write(make([]byte, 1<<20)); err == nil {
			err = f.Sync()
		}
		if err = errors.Combine(err, f.Close()); err != nil {
			return nil, err
		}
	}
	return openPlainLog(plainOpts{})(ctx, fs)
}

var _ = Describe("Faults after the open", func() {
	DescribeTable("Should fail a run in which no fault of its profile fired",
		func(ctx SpecContext, profile crash.Profile) {
			Expect(violation(ctx, crash.Config{
				Open: func(context.Context, xfs.FS) (crash.Engine, error) {
					return &visible{}, nil
				},
				Commits: 8,
				Stride:  1,
				Profile: profile,
			})).To(MatchError(ContainSubstring("crash: no fault of the profile fired")))
		},
		Entry("under Partial", crash.Partial),
		Entry("under SyncErrors", crash.SyncErrors),
		Entry("under FullDisk", crash.FullDisk),
		Entry("under OneTimeReadFaults", crash.OneTimeReadFaults),
		Entry("under OldCorruption", crash.OldCorruption),
		Entry("under Mixed", crash.Mixed),
	)

	It("Should pass a run with no fault under Strict", func(ctx SpecContext) {
		MustSucceed(crash.Run(ctx, crash.Config{
			Open: func(context.Context, xfs.FS) (crash.Engine, error) {
				return &visible{}, nil
			},
			Commits: 8,
			Stride:  1,
			Profile: crash.Strict,
		}))
	})

	DescribeTable("Should fire each fault of the workload after an open that fills "+
		"more than the room of a full disk",
		func(ctx SpecContext, profile crash.Profile) {
			for runKey := range uint64(8) {
				Expect(violation(ctx, crash.Config{
					RunKey:  runKey,
					Open:    openPadded,
					Commits: 8,
					Stride:  1,
					Profile: profile,
				})).To(Succeed(), "run key %d", runKey)
			}
		},
		Entry("under SyncErrors", crash.SyncErrors),
		Entry("under FullDisk", crash.FullDisk),
		Entry("under Mixed", crash.Mixed),
	)
})
