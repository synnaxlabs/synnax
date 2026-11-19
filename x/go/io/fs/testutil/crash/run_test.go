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
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
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

	Describe("Audit", func() {
		It("Should audit each recovery", func(ctx SpecContext) {
			result, calls := MustSucceed2(runAudited(ctx, nil, always))
			Expect(calls).To(Equal(result.Recoveries))
		})

		// Each image gets a first recovery and then a second, after a crash in the
		// first, so the odd calls of Audit are first recoveries.
		DescribeTable("Should break R5 on a failed audit",
			func(ctx SpecContext, fails func(call int) bool) {
				_, _, err := runAudited(ctx, errors.New("taco"), fails)
				var v *crash.Violation
				Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
				Expect(v.Rule).To(Equal(crash.R5))
				Expect(v.Detail).To(Equal("the audit failed: taco"))
			},
			Entry("in a first recovery", func(call int) bool { return call%2 == 1 }),
			Entry("in a second recovery", func(call int) bool { return call%2 == 0 }),
		)
	})

	Describe("Exhaustive", func() {
		It(
			"Should check each crash image and list each violation",
			func(ctx SpecContext) {
				calls := 0
				result, err := crash.Run(ctx, crash.Config{
					Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
						e, err := openPlainLog(plainOpts{})(ctx, fs)
						if err != nil {
							return nil, err
						}
						return audited{
							Engine: e,
							err:    errors.New("taco"),
							fails:  always,
							calls:  &calls,
						}, nil
					},
					Commits:    4,
					Stride:     1,
					Profile:    crash.Strict,
					Exhaustive: true,
				})
				var v *crash.Violation
				Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
				Expect(result.Violations).To(HaveLen(result.Recoveries))
				Expect(result.Violations[0]).To(Equal(*v))
				for i, got := range result.Violations {
					Expect(got.Rule).To(Equal(crash.R5))
					if i > 0 {
						Expect(
							got.Event,
						).To(BeNumerically(">", result.Violations[i-1].Event))
					}
				}
			},
		)

		It("Should list the violations in the order of their events, and return the "+
			"earliest", func(ctx SpecContext) {
			result, err := crash.Run(ctx, crash.Config{
				Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
					e, err := openPlainLog(plainOpts{})(ctx, fs)
					if err != nil {
						return nil, err
					}
					return dropFirst{Engine: e}, nil
				},
				Commits:    4,
				Stride:     1,
				Profile:    crash.Strict,
				Exhaustive: true,
			})
			var v *crash.Violation
			Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
			Expect(v.Event).To(BeZero())
			Expect(result.Violations[0]).To(Equal(*v))
			Expect(
				slices.IsSortedFunc(result.Violations, func(a, b crash.Violation) int {
					return a.Event - b.Event
				}),
			).To(BeTrue())
		})

		It("Should list no violation in a run that is not exhaustive",
			func(ctx SpecContext) {
				result, _, err := runAudited(ctx, errors.New("taco"), always)
				Expect(err).To(MatchError(ContainSubstring("the audit failed: taco")))
				Expect(result.Violations).To(BeEmpty())
			},
		)
	})

	Describe("Writeback", func() {
		It("Should keep each Durable commit that a writeback made durable",
			func(ctx SpecContext) {
				Expect(violation(ctx, crash.Config{
					Open:    openPlainLog(plainOpts{writeback: true}),
					Commits: 20,
					Stride:  1,
					Profile: crash.Strict,
				})).To(Succeed())
			},
		)

		It("Should fail on an FS that Run did not give", func() {
			Expect(crash.Writeback(MustSucceed(fault.New(fault.Config{})))).
				To(MatchError("crash: writeback needs the FS of the workload"))
		})
	})

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

// audited is an engine whose Audit counts its calls, and returns err on each call for
// which fails is true.
type audited struct {
	crash.Engine
	err   error
	fails func(call int) bool
	calls *int
}

func (a audited) Audit(context.Context) error {
	*a.calls++
	if a.fails(*a.calls) {
		return a.err
	}
	return nil
}

// runAudited runs the harness under Strict on a plain log whose Audit returns audit on
// each call for which fails is true, and returns the count of Audit calls.
func runAudited(
	ctx context.Context,
	audit error,
	fails func(call int) bool,
) (crash.Result, int, error) {
	calls := 0
	result, runErr := crash.Run(ctx, crash.Config{
		Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
			e, err := openPlainLog(plainOpts{})(ctx, fs)
			if err != nil {
				return nil, err
			}
			return audited{Engine: e, err: audit, fails: fails, calls: &calls}, nil
		},
		Commits: 4,
		Stride:  1,
		Profile: crash.Strict,
	})
	return result, calls, runErr
}

// dropFirst is an engine whose every read loses its first record.
type dropFirst struct{ crash.Engine }

func (d dropFirst) Recovered(ctx context.Context) ([]crash.Record, error) {
	records, err := d.Engine.Recovered(ctx)
	if len(records) > 0 {
		records = records[1:]
	}
	return records, err
}

// always fails each call of Audit.
func always(int) bool { return true }

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
