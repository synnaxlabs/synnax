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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/validate"
)

var profiles = map[string]crash.Profile{
	"Strict":            crash.Strict,
	"Partial":           crash.Partial,
	"SyncErrors":        crash.SyncErrors,
	"FullDisk":          crash.FullDisk,
	"OneTimeReadFaults": crash.OneTimeReadFaults,
	"OldCorruption":     crash.OldCorruption,
	"NewCorruption":     crash.NewCorruption,
	"Mixed":             crash.Mixed,
}

// runKeys returns one table entry for each of 32 run keys.
func runKeys() []TableEntry {
	entries := make([]TableEntry, 32)
	for key := range entries {
		entries[key] = Entry(fmt.Sprintf("run key %d", key), uint64(key))
	}
	return entries
}

// run runs the harness on a reference engine with the bugs b.
func run(ctx context.Context, b bugs, profile crash.Profile, runKey uint64) error {
	return crash.Run(ctx, crash.Config{
		RunKey:  runKey,
		Open:    openRefLog(b),
		Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
		Commits: 40,
		Stride:  1,
		Profile: profile,
	})
}

// broke returns the rule that err says an engine broke.
func broke(err error) crash.Rule {
	GinkgoHelper()
	var v *crash.Violation
	Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
	return v.Rule
}

var _ = Describe("Run", func() {
	Describe("Reference engine", func() {
		for name, profile := range profiles {
			DescribeTable("Should hold under "+name,
				func(ctx SpecContext, runKey uint64) {
					Expect(run(ctx, bugs{}, profile, runKey)).To(Succeed())
				},
				runKeys(),
			)
		}
	})

	DescribeTable(
		"Should catch an engine that",
		func(ctx SpecContext, b bugs, profile crash.Profile, rule crash.Rule) {
			Expect(broke(run(ctx, b, profile, 0))).To(Equal(rule))
		},
		Entry(
			"skips a directory sync",
			bugs{skipDirSync: true},
			crash.Strict,
			crash.R1,
		),
		Entry("acks before the log sync",
			bugs{ackBeforeSync: true}, crash.Strict, crash.R1),
		Entry("trusts a retried fsync",
			bugs{trustRetriedSync: true}, crash.SyncErrors, crash.R1),
		Entry("truncates on any checksum error",
			bugs{truncateOnAnyError: true}, crash.OldCorruption, crash.R1),
		Entry("skips the checksum on read",
			bugs{skipChecksum: true}, crash.OneTimeReadFaults, crash.R2),
		Entry("skips a bad record and goes on",
			bugs{skipBadRecord: true}, crash.Partial, crash.R3),
		Entry(
			"refuses a torn tail",
			bugs{refuseTornTail: true},
			crash.Partial,
			crash.R4,
		),
	)

	It("Should give the same violation for the same run key", func(ctx SpecContext) {
		b := bugs{skipBadRecord: true}
		first := run(ctx, b, crash.Partial, 7)
		Expect(broke(first)).To(Equal(crash.R3))
		Expect(run(ctx, b, crash.Partial, 7)).To(MatchError(first.Error()))
		Expect(run(ctx, b, crash.Partial, 8)).ToNot(MatchError(first.Error()))
	})

	DescribeTable("Should fail a config that",
		func(ctx SpecContext, change func(cfg *crash.Config)) {
			cfg := crash.Config{
				Open:    openRefLog(bugs{}),
				Commits: 1,
				Stride:  1,
				Profile: crash.Strict,
			}
			change(&cfg)
			Expect(crash.Run(ctx, cfg)).To(MatchError(validate.ErrValidation))
		},
		Entry("has no Open", func(cfg *crash.Config) { cfg.Open = nil }),
		Entry("has no commit", func(cfg *crash.Config) { cfg.Commits = 0 }),
		Entry("has no stride", func(cfg *crash.Config) { cfg.Stride = 0 }),
		Entry("has no profile", func(cfg *crash.Config) { cfg.Profile = 0 }),
		Entry("has an unknown profile", func(cfg *crash.Config) { cfg.Profile = 9 }),
	)
})
