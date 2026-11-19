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
	"strings"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/vfs/errorfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv/pebblekv"
)

// openCoreEngine is openCrashEngine with the Pebble options that core/pkg/storage
// openKV sets. It keeps the settings that fix the events of a run: no automatic
// compaction, no table stats, and no file removal.
func openCoreEngine(_ context.Context, fs xfs.FS) (engine *crashEngine, err error) {
	h := &halt{done: make(chan struct{})}
	defer stopOnDiskPanic(h, &err)
	logger := crashLogger{halt: h}
	opts := &pebble.Options{
		FS:                          errorfs.Wrap(pebblefs.New(fs), h),
		FormatMajorVersion:          pebble.FormatNewest,
		Logger:                      logger,
		EventListener:               &pebble.EventListener{BackgroundError: h.stop},
		Cleaner:                     keeper(pebble.DeleteCleaner{}.Clean),
		DisableAutomaticCompactions: true,
		DisableTableStats:           true,
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
		return nil, errors.Combine(err, h.cause())
	}
	return &crashEngine{pebble: db, kv: pebblekv.Wrap(db, logger), halt: h}, nil
}

func runCore(ctx context.Context, profile crash.Profile, runKey uint64) (int, error) {
	count := 0
	_, err := crash.Run(ctx, crash.Config{
		RunKey: runKey,
		Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
			e, err := openCoreEngine(ctx, fs)
			if err != nil {
				return nil, err
			}
			return acks{Engine: e, count: &count}, nil
		},
		Corrupt: func(err error) bool { return errors.Is(err, pebble.ErrCorruption) },
		Commits: 20,
		Stride:  1,
		Profile: profile,
	})
	return count, err
}

var _ = DescribeTable("Should break the same rule with the Core's Pebble options",
	func(ctx SpecContext, profile crash.Profile, runKey uint64) {
		_, a := run(ctx, profile, runKey, false)
		_, b := runCore(ctx, profile, runKey)
		if a == nil {
			Expect(b).To(BeNil())
			return
		}
		var va, vb *crash.Violation
		Expect(errors.As(a, &va)).To(BeTrue(), "%v", a)
		Expect(errors.As(b, &vb)).To(BeTrue(), "the Core's options hold: %v", a)
		Expect(vb.Rule).To(Equal(va.Rule), "%v | %v", a, b)
	},
	Entry("Partial", crash.Partial, uint64(0)),
	Entry("Partial, run key 1", crash.Partial, uint64(1)),
	Entry("Mixed", crash.Mixed, uint64(0)),
	Entry("Mixed, run key 3", crash.Mixed, uint64(3)),
	Entry("OneTimeReadFaults", crash.OneTimeReadFaults, uint64(0)),
	Entry("OneTimeReadFaults, run key 2", crash.OneTimeReadFaults, uint64(2)),
	Entry("RepeatedReadFaults", crash.RepeatedReadFaults, uint64(0)),
	Entry("RepeatedReadFaults, run key 2", crash.RepeatedReadFaults, uint64(2)),
	Entry("OldCorruption, run key 1", crash.OldCorruption, uint64(1)),
	Entry("Strict", crash.Strict, uint64(0)),
	Entry("SyncErrors, run key 4", crash.SyncErrors, uint64(4)),
	Entry("FullDisk", crash.FullDisk, uint64(0)),
	Entry("NewCorruption", crash.NewCorruption, uint64(0)),
)

// The OPTIONS text of the Core differs, so a fault in it lands elsewhere at a given run
// key. The cause is the same: OPTIONS has no checksum, and Pebble reads it once.
var _ = DescribeTable("Should break on the OPTIONS file with the Core's Pebble options",
	func(ctx SpecContext, profile crash.Profile, cause string) {
		var found []uint64
		for runKey := range uint64(16) {
			_, err := runCore(ctx, profile, runKey)
			var v *crash.Violation
			if errors.As(err, &v) && v.Rule == crash.R4 &&
				strings.Contains(v.Detail, cause) {
				found = append(found, runKey)
			}
		}
		Expect(found).ToNot(BeEmpty())
	},
	Entry("OldCorruption", crash.OldCorruption, "merger name from file"),
	Entry("Mixed", crash.Mixed, "comparer name from file"),
)
