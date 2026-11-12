// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("CompileConsistent", func() {
	createChannel := func(ctx context.Context, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, Virtual: true, DataType: telem.Float32T}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
		})
		return ch
	}

	It("Should compile and check once when no lookup changes", func(ctx SpecContext) {
		var compiles, checks int
		Expect(channel.CompileConsistent(ctx, db, func(
			_ gorp.Snapshot,
			lookups *channel.Lookups,
		) (int, error) {
			compiles++
			lookups.Record(func(context.Context, gorp.Snapshot) (bool, error) {
				checks++
				return true, nil
			})
			return 42, nil
		})).To(Equal(42))
		Expect(compiles).To(Equal(1))
		Expect(checks).To(Equal(1))
	})

	It("Should fail after five changed tries with no lock held", func(
		ctx SpecContext,
	) {
		var snaps []gorp.Snapshot
		Expect(channel.CompileConsistent(ctx, db, func(
			snap gorp.Snapshot,
			lookups *channel.Lookups,
		) (int, error) {
			snaps = append(snaps, snap)
			lookups.Record(func(context.Context, gorp.Snapshot) (bool, error) {
				return false, nil
			})
			return len(snaps), nil
		})).Error().To(MatchError(channel.ErrLookupsChanged))
		Expect(snaps).To(HaveLen(5))
		Expect(snaps).To(HaveEach(BeIdenticalTo(db)))
	})

	It("Should stop waiting between tries when the context is cancelled", func(
		ctx SpecContext,
	) {
		cancelCtx, cancel := context.WithCancel(ctx)
		var compiles int
		Expect(channel.CompileConsistent(cancelCtx, db, func(
			_ gorp.Snapshot,
			lookups *channel.Lookups,
		) (int, error) {
			compiles++
			cancel()
			lookups.Record(func(context.Context, gorp.Snapshot) (bool, error) {
				return false, nil
			})
			return 42, nil
		})).Error().To(MatchError(context.Canceled))
		Expect(compiles).To(Equal(1))
	})

	It("Should return the error of a check", func(ctx SpecContext) {
		Expect(channel.CompileConsistent(ctx, db, func(
			_ gorp.Snapshot,
			lookups *channel.Lookups,
		) (int, error) {
			lookups.Record(func(context.Context, gorp.Snapshot) (bool, error) {
				return false, errors.New("check failed")
			})
			return 42, nil
		})).Error().To(MatchError("check failed"))
	})

	It("Should compile again when a resolved channel is renamed before the check", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx, "checked_rename_ch")
		var compiles int
		Expect(channel.CompileConsistent(ctx, db, func(
			snap gorp.Snapshot,
			lookups *channel.Lookups,
		) (*symbol.Symbol, error) {
			compiles++
			sym, err := svc.NewCheckedResolver(snap, lookups).
				Resolve(ctx, "checked_rename_ch")
			if compiles == 1 {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).
						Rename(ctx, ch.Key(), "checked_rename_ch_2", false)
				})).To(Succeed())
			}
			return sym, err
		})).Error().To(MatchError(query.ErrNotFound))
		Expect(compiles).To(Equal(2))
	})

	It("Should compile again when a missing channel is created before the check", func(
		ctx SpecContext,
	) {
		var (
			compiles int
			ch       channel.Channel
		)
		sym := MustSucceed(channel.CompileConsistent(ctx, db, func(
			snap gorp.Snapshot,
			lookups *channel.Lookups,
		) (*symbol.Symbol, error) {
			compiles++
			sym, err := svc.NewCheckedResolver(snap, lookups).
				Resolve(ctx, "checked_later_ch")
			if compiles == 1 {
				ch = createChannel(ctx, "checked_later_ch")
			}
			return sym, err
		}))
		Expect(sym.ID).To(Equal(int(ch.Key())))
		Expect(compiles).To(Equal(2))
	})
})

var _ = Describe("Lookups", func() {
	DescribeTable(
		"Unchanged",
		func(ctx SpecContext, answers []bool, unchanged bool) {
			lookups := &channel.Lookups{}
			for _, same := range answers {
				lookups.Record(func(context.Context, gorp.Snapshot) (bool, error) {
					return same, nil
				})
			}
			Expect(lookups.Unchanged(ctx, db)).To(Equal(unchanged))
		},
		Entry("Should hold with no lookups", nil, true),
		Entry(
			"Should hold when each lookup gives the same answer",
			[]bool{true, true},
			true,
		),
		Entry(
			"Should not hold when a lookup gives another answer",
			[]bool{true, false},
			false,
		),
	)

	It("Should return the error of a lookup", func(ctx SpecContext) {
		lookups := &channel.Lookups{}
		lookups.Record(func(context.Context, gorp.Snapshot) (bool, error) {
			return false, errors.New("lookup failed")
		})
		Expect(lookups.Unchanged(ctx, db)).Error().To(MatchError("lookup failed"))
	})
})

var _ = Describe("RetryChanged", func() {
	// changing returns a function that returns channel.ErrLookupsChanged for its first
	// changes calls, and counts each call in calls.
	changing := func(changes int, calls *int) func() (int, error) {
		return func() (int, error) {
			*calls++
			if *calls <= changes {
				return 0, channel.ErrLookupsChanged
			}
			return 42, nil
		}
	}

	It("Should call again until the lookups hold", func(ctx SpecContext) {
		var calls int
		Expect(channel.RetryChanged(ctx, changing(2, &calls))).To(Equal(42))
		Expect(calls).To(Equal(3))
	})

	It("Should fail after three changed calls", func(ctx SpecContext) {
		var calls int
		Expect(channel.RetryChanged(ctx, changing(3, &calls))).Error().
			To(MatchError(channel.ErrLookupsChanged))
		Expect(calls).To(Equal(3))
	})

	It("Should return another error at once", func(ctx SpecContext) {
		var calls int
		Expect(channel.RetryChanged(ctx, func() (int, error) {
			calls++
			return 0, query.ErrNotFound
		})).Error().To(MatchError(query.ErrNotFound))
		Expect(calls).To(Equal(1))
	})

	It("Should stop waiting when the context is cancelled", func(ctx SpecContext) {
		cancelCtx, cancel := context.WithCancel(ctx)
		var calls int
		Expect(channel.RetryChanged(cancelCtx, func() (int, error) {
			calls++
			cancel()
			return 0, channel.ErrLookupsChanged
		})).Error().To(MatchError(context.Canceled))
		Expect(calls).To(Equal(1))
	})
})
