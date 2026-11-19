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
	"encoding/binary"
	"os"
	"slices"

	"github.com/cockroachdb/pebble/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	. "github.com/synnaxlabs/x/testutil"
)

// crashLogger drops each message, and panics on Fatalf, where the logger of the Core
// ends the process.
type crashLogger struct{}

func (crashLogger) Infof(string, ...any) {}

func (crashLogger) Errorf(string, ...any) {}

func (crashLogger) Fatalf(format string, args ...any) {
	panic(errors.Newf(format, args...))
}

// stopped turns a panic in Pebble into the error *err. Pebble panics, or calls Fatalf,
// where the process of the Core ends, so the engine stops there. On a disk with no
// fault, the harness takes the error as a break of R0.
func stopped(err *error) {
	if r := recover(); r != nil {
		*err = errors.Combine(*err, errors.Newf("pebble: stopped: %v", r))
	}
}

// crashEngine adapts Pebble, through the kv.DB of pebblekv, to the crash harness. One
// record is one key, the ID in big-endian order, with the payload as its value. Each
// commit syncs the WAL, as each commit of the Core does, so each ack is Durable. After
// an error, the engine stops.
type crashEngine struct {
	pebble *pebble.DB
	db     kv.DB
	dead   error
}

func openCrashEngine(_ context.Context, fs xfs.FS) (engine crash.Engine, err error) {
	defer stopped(&err)
	opts := &pebble.Options{
		FS:                 pebblefs.New(fs),
		FormatMajorVersion: pebble.FormatNewest,
		Logger:             crashLogger{},
	}
	// An open that stops leaves the scheduler of Pebble registered, and its granter
	// running. The workload runs no compaction, so a scheduler with no granter is
	// enough.
	scheduler := pebble.NewConcurrencyLimitSchedulerWithNoPeriodicGrantingForTest()
	opts.Experimental.CompactionScheduler = scheduler
	db, err := pebble.Open("kv", opts)
	if err != nil {
		return nil, err
	}
	return &crashEngine{pebble: db, db: pebblekv.Wrap(db, crashLogger{})}, nil
}

func (e *crashEngine) Commit(
	ctx context.Context,
	r crash.Record,
	_ crash.Ack,
) (ack crash.Ack, err error) {
	if e.dead != nil {
		return 0, e.dead
	}
	defer func() {
		if err != nil {
			ack, e.dead = 0, err
		}
	}()
	defer stopped(&err)
	key := binary.BigEndian.AppendUint64(nil, r.ID)
	if err = e.db.Set(ctx, key, r.Payload); err != nil {
		return 0, err
	}
	return crash.Durable, nil
}

func (e *crashEngine) Recovered(context.Context) (records []crash.Record, err error) {
	if e.dead != nil {
		return nil, e.dead
	}
	defer stopped(&err)
	iter, err := e.db.OpenIterator(kv.IteratorOptions{})
	if err != nil {
		return nil, err
	}
	for ok := iter.First(); ok; ok = iter.Next() {
		records = append(records, crash.Record{
			ID:      binary.BigEndian.Uint64(iter.Key()),
			Payload: slices.Clone(iter.Value()),
		})
	}
	return records, errors.Combine(iter.Error(), iter.Close())
}

func (e *crashEngine) Close() (err error) {
	defer stopped(&err)
	return e.pebble.Close()
}

// run runs the harness on Pebble with one fault profile and one run key.
func run(ctx context.Context, profile crash.Profile, runKey uint64) error {
	_, err := crash.Run(ctx, crash.Config{
		RunKey:  runKey,
		Open:    openCrashEngine,
		Corrupt: func(err error) bool { return errors.Is(err, pebble.ErrCorruption) },
		Commits: 20,
		Stride:  1,
		Profile: profile,
	})
	return err
}

// Each entry is what the harness finds in Pebble, as the Core configures its commits,
// and why.
var _ = DescribeTable("Crash harness",
	func(
		ctx SpecContext,
		profile crash.Profile,
		runKey uint64,
		rule crash.Rule,
		cause string,
	) {
		var v *crash.Violation
		err := run(ctx, profile, runKey)
		Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
		Expect(v.Rule).To(Equal(rule))
		Expect(v.Detail).To(ContainSubstring(cause))
	},
	// A crash in the first open can keep the marker of the manifest and lose the
	// manifest, so the database does not open again.
	Entry("Partial: R4", crash.Partial, uint64(0), crash.R4,
		"could not open manifest file"),
	Entry("Mixed: R4", crash.Mixed, uint64(0), crash.R4,
		"could not open manifest file"),
	// One bad read of the manifest or of the options file stops the open: Pebble reads
	// neither a second time.
	Entry("OneTimeReadFaults: R4", crash.OneTimeReadFaults, uint64(0), crash.R4,
		"malformed manifest file"),
	Entry("RepeatedReadFaults: R4", crash.RepeatedReadFaults, uint64(0), crash.R4,
		"invalid key=value syntax"),
	// A flipped bit in an older record of the WAL ends the replay there, so Pebble
	// loses each durable commit from that record on, with no error.
	Entry("OldCorruption: R1", crash.OldCorruption, uint64(0), crash.R1,
		"durable commit 3 is missing"),
)

var _ = DescribeTable("Crash harness, holding",
	func(ctx SpecContext, profile crash.Profile, runKey uint64) {
		Expect(run(ctx, profile, runKey)).To(Succeed())
	},
	Entry("Strict", crash.Strict, uint64(0)),
	Entry("SyncErrors", crash.SyncErrors, uint64(0)),
	Entry("FullDisk", crash.FullDisk, uint64(0)),
	Entry("NewCorruption", crash.NewCorruption, uint64(0)),
	Entry("Partial, run key 1", crash.Partial, uint64(1)),
	Entry("Mixed, run key 1", crash.Mixed, uint64(1)),
)

var _ = Describe("Crash adapter", func() {
	It("Should lose each commit from a flipped bit in the WAL on, with no error",
		func(ctx SpecContext) {
			disk := MustSucceed(fault.New(fault.Config{}))
			e := MustSucceed(openCrashEngine(ctx, disk))
			for id := uint64(1); id <= 10; id++ {
				record := crash.Record{ID: id, Payload: make([]byte, 500)}
				Expect(e.Commit(ctx, record, crash.Durable)).To(Equal(crash.Durable))
			}
			image := MustSucceed(disk.Crash(fault.CrashConfig{}))
			Expect(e.Close()).To(Succeed())
			f := MustSucceed(image.Open("kv/000002.log", os.O_RDWR))
			b := make([]byte, 1)
			Expect(f.ReadAt(b, 2500)).To(Equal(1))
			b[0] ^= 1
			Expect(f.WriteAt(b, 2500)).To(Equal(1))
			Expect(errors.Join(f.Sync(), f.Close())).To(Succeed())
			e = MustSucceed(openCrashEngine(ctx, image))
			Expect(MustSucceed(e.Recovered(ctx))).To(HaveLen(4))
			Expect(e.Close()).To(Succeed())
		},
	)

	It("Should stop after a failed sync of the WAL and refuse each later commit",
		func(ctx SpecContext) {
			found := false
			for runKey := uint64(0); runKey < 64 && !found; runKey++ {
				cfg := fault.Config{RunKey: runKey, SyncErrorRate: 0.05}
				e, err := openCrashEngine(ctx, MustSucceed(fault.New(cfg)))
				if err != nil {
					continue
				}
				var first error
				for id := uint64(1); id <= 20 && first == nil; id++ {
					record := crash.Record{ID: id, Payload: []byte("taco")}
					_, first = e.Commit(ctx, record, crash.Durable)
				}
				if first == nil {
					Expect(e.Close()).To(Succeed())
					continue
				}
				found = true
				Expect(first).To(MatchError(ContainSubstring(
					"pebble: stopped: failed to sync the WAL",
				)))
				record := crash.Record{ID: 99, Payload: []byte("cat")}
				Expect(
					e.Commit(ctx, record, crash.Durable),
				).Error().
					To(MatchError(first))
				Expect(e.Recovered(ctx)).Error().To(MatchError(first))
				Expect(e.Close()).To(MatchError(ContainSubstring("input/output error")))
			}
			Expect(found).To(BeTrue())
		},
	)
})
