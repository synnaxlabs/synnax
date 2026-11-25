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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// violation returns the error of a run of cfg.
func violation(ctx context.Context, cfg crash.Config) error {
	_, err := crash.Run(ctx, cfg)
	return err
}

var profiles = map[string]crash.Profile{
	"Strict":             crash.Strict,
	"Partial":            crash.Partial,
	"SyncErrors":         crash.SyncErrors,
	"FullDisk":           crash.FullDisk,
	"OneTimeReadFaults":  crash.OneTimeReadFaults,
	"OldCorruption":      crash.OldCorruption,
	"NewCorruption":      crash.NewCorruption,
	"Mixed":              crash.Mixed,
	"RepeatedReadFaults": crash.RepeatedReadFaults,
}

// runRef runs the harness on a reference engine with the bugs b.
func runRef(
	ctx context.Context,
	b bugs,
	profile crash.Profile,
	runKey uint64,
) (crash.Result, error) {
	return crash.Run(ctx, crash.Config{
		RunKey:  runKey,
		Open:    openRefLog(b),
		Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
		Commits: 40,
		Stride:  1,
		Profile: profile,
	})
}

// run returns the error of runRef.
func run(ctx context.Context, b bugs, profile crash.Profile, runKey uint64) error {
	_, err := runRef(ctx, b, profile, runKey)
	return err
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
			It("Should hold under "+name+", and check a recovery in full",
				func(ctx SpecContext) {
					var total crash.Result
					for runKey := range uint64(32) {
						result, err := runRef(ctx, bugs{}, profile, runKey)
						Expect(err).ToNot(HaveOccurred(), "run key %d", runKey)
						total.Recoveries += result.Recoveries
						total.Refusals += result.Refusals
						total.Losses += result.Losses
						total.ReadFaults += result.ReadFaults
					}
					GinkgoWriter.Printf("%s: %+v\n", name, total)
					full := total.Recoveries - total.Refusals - total.Losses
					Expect(full).To(BeNumerically(">", 0))
					faults := Equal(0)
					if profile == crash.OneTimeReadFaults ||
						profile == crash.RepeatedReadFaults || profile == crash.Mixed {
						faults = BeNumerically(">", 0)
					}
					Expect(total.ReadFaults).To(faults)
				},
			)
		}
	})

	// Each entry keeps a run key that caught its bug.
	DescribeTable(
		"Should catch an engine that",
		func(
			ctx SpecContext,
			b bugs,
			profile crash.Profile,
			runKey uint64,
			rule crash.Rule,
		) {
			Expect(broke(run(ctx, b, profile, runKey))).To(Equal(rule))
		},
		Entry("skips a directory sync",
			bugs{dirSyncSkipped: true}, crash.Strict, uint64(0), crash.R1),
		Entry("acks before the log sync",
			bugs{ackedBeforeSync: true}, crash.Strict, uint64(0), crash.R1),
		Entry("trusts a retried fsync",
			bugs{retriedSyncTrusted: true}, crash.SyncErrors, uint64(0), crash.R1),
		Entry("truncates on any checksum error",
			bugs{cutOnAnyError: true}, crash.OldCorruption, uint64(0), crash.R1),
		Entry("skips the checksum on read",
			bugs{checksumSkipped: true}, crash.OneTimeReadFaults, uint64(0), crash.R2),
		Entry("skips a bad record and goes on",
			bugs{badRecordSkipped: true}, crash.Partial, uint64(0), crash.R3),
		Entry("refuses a torn tail",
			bugs{tornTailRefused: true}, crash.Partial, uint64(0), crash.R4),
		Entry("ignores a failed write",
			bugs{writeErrorIgnored: true}, crash.FullDisk, uint64(1), crash.R1),
		Entry("trusts the retry of a read with no checksum",
			bugs{retryTrusted: true}, crash.RepeatedReadFaults, uint64(1), crash.R2),
		Entry("reads a bad slot a second time only once in a recovery",
			bugs{retryRationed: true}, crash.RepeatedReadFaults, uint64(4), crash.R1),
		Entry("reads a bad slot a second time only once in a recovery, and refuses",
			bugs{retryRationed: true}, crash.RepeatedReadFaults, uint64(9), crash.R4),
		Entry(
			"retries the header and the body of a slot together, with one fault on each",
			bugs{
				partsRetriedTogether: true,
			},
			crash.RepeatedReadFaults,
			uint64(10),
			crash.R1,
		),
	)

	It("Should catch a whole-slot retry only through clusters of one fault",
		func(ctx SpecContext) {
			Expect(
				run(
					ctx,
					bugs{partsRetriedTogether: true},
					crash.RepeatedReadFaults,
					10,
				),
			).
				To(MatchError(MatchRegexp(`, 1 in one cluster:`)))
			for runKey := range uint64(32) {
				Expect(
					run(
						ctx,
						bugs{partsRetriedTogether: true},
						crash.OneTimeReadFaults,
						runKey,
					),
				).
					To(Succeed(), "run key %d", runKey)
			}
		},
	)

	It("Should pass an engine that trusts its retry under OneTimeReadFaults",
		func(ctx SpecContext) {
			for runKey := range uint64(32) {
				Expect(
					run(ctx, bugs{retryTrusted: true}, crash.OneTimeReadFaults, runKey),
				).
					To(Succeed(), "run key %d", runKey)
			}
		},
	)

	It("Should give the same violation for the same run key", func(ctx SpecContext) {
		b := bugs{badRecordSkipped: true}
		first := run(ctx, b, crash.Partial, 7)
		Expect(broke(first)).To(Equal(crash.R3))
		Expect(run(ctx, b, crash.Partial, 7)).To(MatchError(first.Error()))
		Expect(run(ctx, b, crash.Partial, 8)).ToNot(MatchError(first.Error()))
	})

	DescribeTable("Should fail a config that",
		func(ctx SpecContext, field string, change func(cfg *crash.Config)) {
			cfg := crash.Config{
				Open:    openRefLog(bugs{}),
				Commits: 1,
				Stride:  1,
				Profile: crash.Strict,
			}
			change(&cfg)
			err := violation(ctx, cfg)
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(field)))
		},
		Entry("has no Open", "open", func(cfg *crash.Config) { cfg.Open = nil }),
		Entry("has no commit", "commits", func(cfg *crash.Config) { cfg.Commits = 0 }),
		Entry("has no stride", "stride", func(cfg *crash.Config) { cfg.Stride = 0 }),
		Entry("has no profile", "profile", func(cfg *crash.Config) { cfg.Profile = 0 }),
		Entry("has an unknown profile", "profile",
			func(cfg *crash.Config) { cfg.Profile = 10 }),
	)

	It("Should catch a skipped checksum on a disk of one block", func(ctx SpecContext) {
		rules := set.Set[crash.Rule]{}
		for runKey := range uint64(32) {
			err := violation(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openRefLog(bugs{checksumSkipped: true}),
				Commits: 2,
				Stride:  1,
				Profile: crash.OneTimeReadFaults,
			})
			if err != nil {
				rules.Add(broke(err))
			}
		}
		Expect(rules.Slice()).To(ConsistOf(crash.R2))
	})
})

// BenchmarkRun measures one pass of 200 commits on the reference engine, with a crash
// image and a recovery after each disk change.
func BenchmarkRun(b *testing.B) {
	cfg := crash.Config{
		Open:    openRefLog(bugs{}),
		Commits: 200,
		Stride:  1,
		Profile: crash.Strict,
	}
	for b.Loop() {
		if err := violation(b.Context(), cfg); err != nil {
			b.Fatal(err)
		}
	}
}
