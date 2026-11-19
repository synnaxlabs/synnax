// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv_test

import (
	"context"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv/pebblekv"
)

// openCoreEngine is openCrashEngine with the Pebble options that core/pkg/storage
// openKV sets, apart from the cache and the compaction scheduler.
func openCoreEngine(_ context.Context, fs xfs.FS) (engine crash.Engine, err error) {
	defer stopped(&err)
	ev := pebble.MakeLoggingEventListener(crashLogger{})
	opts := &pebble.Options{
		FS:                          pebblefs.New(fs),
		FormatMajorVersion:          pebble.FormatNewest,
		EventListener:               &ev,
		Logger:                      crashLogger{},
		MemTableSize:                128 << 20,
		MemTableStopWritesThreshold: 4,
		L0CompactionThreshold:       2,
		L0CompactionFileThreshold:   500,
		L0StopWritesThreshold:       12,
		MaxOpenFiles:                10000,
		BytesPerSync:                2 << 20,
		WALBytesPerSync:             0,
	}
	for i := range opts.Levels {
		opts.Levels[i].BlockSize = 32 << 10
		opts.Levels[i].IndexBlockSize = 256 << 10
		opts.Levels[i].FilterPolicy = bloom.FilterPolicy(10)
		opts.Levels[i].FilterType = pebble.TableFilter
	}
	opts.TargetFileSizes[0] = 64 << 20
	for i := 1; i < len(opts.TargetFileSizes); i++ {
		opts.TargetFileSizes[i] = opts.TargetFileSizes[i-1] * 10
	}
	opts.Experimental.CompactionScheduler = pebble.NewConcurrencyLimitSchedulerWithNoPeriodicGrantingForTest()
	db, err := pebble.Open("kv", opts)
	if err != nil {
		return nil, err
	}
	return &crashEngine{pebble: db, db: pebblekv.Wrap(db, crashLogger{})}, nil
}

func runCore(ctx context.Context, profile crash.Profile, runKey uint64) error {
	_, err := crash.Run(ctx, crash.Config{
		RunKey:  runKey,
		Open:    openCoreEngine,
		Corrupt: func(err error) bool { return errors.Is(err, pebble.ErrCorruption) },
		Commits: 20,
		Stride:  1,
		Profile: profile,
	})
	return err
}

var _ = DescribeTable("Should break the same rule with the Core's Pebble options",
	func(ctx SpecContext, profile crash.Profile, runKey uint64) {
		a, b := run(ctx, profile, runKey), runCore(ctx, profile, runKey)
		if a == nil {
			Expect(b).To(BeNil())
			return
		}
		var va, vb *crash.Violation
		Expect(errors.As(a, &va)).To(BeTrue())
		Expect(errors.As(b, &vb)).To(BeTrue(), "the Core's options hold: %v", a)
		// The event can differ, because the Core writes a different OPTIONS file.
		Expect(vb.Rule).To(Equal(va.Rule))
	},
	Entry("Strict", crash.Strict, uint64(0)),
	Entry("Partial", crash.Partial, uint64(0)),
	Entry("Partial, run key 1", crash.Partial, uint64(1)),
	Entry("SyncErrors", crash.SyncErrors, uint64(0)),
	Entry("FullDisk", crash.FullDisk, uint64(0)),
	Entry("OneTimeReadFaults", crash.OneTimeReadFaults, uint64(0)),
	Entry("RepeatedReadFaults", crash.RepeatedReadFaults, uint64(0)),
	Entry("OldCorruption", crash.OldCorruption, uint64(0)),
	Entry("NewCorruption", crash.NewCorruption, uint64(0)),
	Entry("Mixed", crash.Mixed, uint64(0)),
	Entry("Mixed, run key 1", crash.Mixed, uint64(1)),
)
