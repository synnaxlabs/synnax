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
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/cockroachdb/pebble/v2/vfs/errorfs"
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

// errHalted is the error of each write after the engine stops.
var errHalted = errors.New("pebble: halted")

// halt fails each write and sync of Pebble from the first stop on, as the exit of the
// Core does, so the disk keeps what it held at the stop.
type halt struct {
	mu   sync.Mutex
	err  error
	done chan struct{}
}

func (h *halt) String() string { return "halt" }

func (h *halt) MaybeError(op errorfs.Op) error {
	if op.Kind.ReadOrWrite() != errorfs.OpIsWrite || op.Kind == errorfs.OpFileClose {
		return nil
	}
	if h.cause() != nil {
		return errHalted
	}
	return nil
}

// stop stops the engine with err, unless it stopped before.
func (h *halt) stop(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.err == nil {
		h.err = errors.Wrap(err, "pebble: stopped")
		close(h.done)
	}
}

// cause returns the error that stopped the engine, or nil.
func (h *halt) cause() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

// crashLogger writes the log of Pebble to the GinkgoWriter. Fatalf stops the engine,
// where the logger of the Core ends the process.
type crashLogger struct{ halt *halt }

func (crashLogger) Infof(format string, args ...any) {
	GinkgoWriter.Printf(format+"\n", args...)
}

func (crashLogger) Errorf(format string, args ...any) {
	GinkgoWriter.Printf(format+"\n", args...)
}

// Fatalf keeps the error in args as the cause, so Corrupt can find pebble.ErrCorruption
// in it.
func (l crashLogger) Fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	for _, arg := range args {
		if cause, ok := arg.(error); ok {
			prefix := strings.TrimSuffix(strings.TrimSuffix(msg, cause.Error()), ": ")
			l.halt.stop(errors.Wrap(cause, prefix))
			return
		}
	}
	l.halt.stop(errors.New(msg))
}

// stopOnDiskPanic turns a panic of Pebble on a failed write or sync into the error
// *err, and stops h. Pebble panics there by design, where the process of the Core ends.
// It panics again on any other value.
func stopOnDiskPanic(h *halt, err *error) {
	r := recover()
	if r == nil {
		return
	}
	cause, ok := r.(error)
	if !ok || !fromDisk(cause) {
		panic(r)
	}
	h.stop(cause)
	*err = errors.Combine(*err, h.cause())
}

// fromDisk reports if err is a failed write or sync of the disk.
func fromDisk(err error) bool {
	return errors.Is(err, syscall.EIO) ||
		errors.Is(err, syscall.ENOSPC) ||
		errors.Is(err, errHalted)
}

// keep is a Cleaner that removes no file. Pebble removes obsolete files on a goroutine
// of its own, at no fixed event, so a removal would make the crash images of a run key
// differ between runs. T is the file type of Pebble, which it does not export.
type keep[T any] struct{}

func (keep[T]) Clean(vfs.FS, T, string) error { return nil }

func keeper[T any](func(vfs.FS, T, string) error) keep[T] { return keep[T]{} }

// crashEngine adapts Pebble to the crash harness. One record is one key, the ID in
// big-endian order, with the payload as its value. A Durable commit goes through the
// kv.DB of pebblekv, which syncs the WAL as each commit of the Core does, and a Visible
// commit does not sync. Every fifth commit flushes the memtable, and every tenth
// compacts the keys, so a crash can land in each. After an error, the engine stops.
type crashEngine struct {
	pebble  *pebble.DB
	kv      kv.DB
	halt    *halt
	commits int
}

func openCrashEngine(_ context.Context, fs xfs.FS) (engine *crashEngine, err error) {
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
	}
	// An open that stops leaves the scheduler of Pebble registered, and its granter
	// running. A scheduler with no granter is enough for one compaction at a time.
	scheduler := pebble.NewConcurrencyLimitSchedulerWithNoPeriodicGrantingForTest()
	opts.Experimental.CompactionScheduler = scheduler
	db, err := pebble.Open("kv", opts)
	if err != nil {
		return nil, errors.Combine(err, h.cause())
	}
	return &crashEngine{pebble: db, kv: pebblekv.Wrap(db, logger), halt: h}, nil
}

func (e *crashEngine) Commit(
	ctx context.Context,
	r crash.Record,
	want crash.Ack,
) (ack crash.Ack, err error) {
	if err = e.halt.cause(); err != nil {
		return 0, err
	}
	defer stopOnDiskPanic(e.halt, &err)
	key := binary.BigEndian.AppendUint64(nil, r.ID)
	if want == crash.Durable {
		err = e.kv.Set(ctx, key, r.Payload)
	} else {
		err = e.pebble.Set(key, r.Payload, pebble.NoSync)
	}
	if err = errors.Combine(err, e.halt.cause()); err != nil {
		e.halt.stop(err)
		return 0, e.halt.cause()
	}
	e.commits++
	if e.commits%5 == 0 {
		e.flush()
	}
	if e.commits%10 == 0 {
		e.compact(ctx)
	}
	return want, nil
}

// compact compacts every key, and returns when the compaction ends or the engine stops.
// Pebble retries a failed compaction until its context ends.
func (e *crashEngine) compact(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-e.halt.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	if err := e.pebble.Compact(ctx, []byte{0}, []byte{0xff}, true); err != nil {
		e.halt.stop(err)
	}
}

// flush flushes the memtable, and returns when the flush ends or the engine stops.
func (e *crashEngine) flush() {
	done, err := e.pebble.AsyncFlush()
	if err != nil {
		e.halt.stop(err)
		return
	}
	select {
	case <-done:
	case <-e.halt.done:
	}
}

func (e *crashEngine) Recovered(context.Context) (records []crash.Record, err error) {
	if err = e.halt.cause(); err != nil {
		return nil, err
	}
	defer stopOnDiskPanic(e.halt, &err)
	iter, err := e.kv.OpenIterator(kv.IteratorOptions{})
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
	defer stopOnDiskPanic(e.halt, &err)
	return e.pebble.Close()
}

// acks counts the Durable acks of an engine. Only the engine of the workload commits.
type acks struct {
	crash.Engine
	count *int
}

func (a acks) Commit(
	ctx context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	ack, err := a.Engine.Commit(ctx, r, want)
	if err == nil && ack == crash.Durable {
		*a.count++
	}
	return ack, err
}

// run runs the harness on Pebble with one fault profile and one run key, and returns
// the count of Durable acks in the workload. Ordered keeps the directory changes in
// order.
func run(
	ctx context.Context,
	profile crash.Profile,
	runKey uint64,
	ordered bool,
) (int, error) {
	count := 0
	_, err := crash.Run(ctx, crash.Config{
		RunKey: runKey,
		Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
			e, err := openCrashEngine(ctx, fs)
			if err != nil {
				return nil, err
			}
			return acks{Engine: e, count: &count}, nil
		},
		Corrupt:        func(err error) bool { return errors.Is(err, pebble.ErrCorruption) },
		Commits:        20,
		Stride:         1,
		Profile:        profile,
		EntriesOrdered: ordered,
	})
	return count, err
}

// Each entry is what the harness finds in Pebble at FormatNewest, and why, with the
// count of read faults at the break. The workload of each run flushes 4 times and
// compacts twice.
var _ = DescribeTable("Crash harness",
	func(
		ctx SpecContext,
		profile crash.Profile,
		runKey uint64,
		rule crash.Rule,
		readFaults int,
		cause string,
	) {
		var v *crash.Violation
		_, err := run(ctx, profile, runKey, false)
		Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
		Expect(v.Rule).To(Equal(rule))
		Expect(v.ReadFaults).To(Equal(readFaults))
		Expect(v.Detail).To(ContainSubstring(cause))
	},
	// Pebble syncs the directory once, after it creates both the manifest and its
	// marker. Partial lets each directory change survive alone, so a crash can keep the
	// marker and lose the manifest, and the database does not open again.
	Entry("Partial: R4", crash.Partial, uint64(0), crash.R4, 0,
		"could not open manifest file"),
	Entry("Mixed: R4", crash.Mixed, uint64(0), crash.R4, 0,
		"could not open manifest file"),
	// One bad read of the manifest or of the OPTIONS file stops the open: Pebble reads
	// neither a second time.
	Entry("OneTimeReadFaults, manifest: R4", crash.OneTimeReadFaults, uint64(0),
		crash.R4, 1, "malformed manifest file"),
	Entry("OneTimeReadFaults, OPTIONS: R4", crash.OneTimeReadFaults, uint64(2),
		crash.R4, 1, "invalid key=value syntax"),
	Entry("RepeatedReadFaults, manifest: R4", crash.RepeatedReadFaults, uint64(0),
		crash.R4, 1, "malformed manifest file"),
	Entry("RepeatedReadFaults, OPTIONS: R4", crash.RepeatedReadFaults, uint64(2),
		crash.R4, 4, "invalid key=value syntax"),
	Entry("Mixed, manifest: R4", crash.Mixed, uint64(3), crash.R4, 1,
		"malformed manifest file"),
	Entry("Mixed, OPTIONS: R4", crash.Mixed, uint64(1), crash.R4, 1,
		"invalid key=value syntax"),
	// The OPTIONS file has no checksum, so a flipped byte in it stops the open with an
	// error that is not pebble.ErrCorruption.
	Entry("OldCorruption, OPTIONS: R4", crash.OldCorruption, uint64(0), crash.R4, 0,
		"merger name from file"),
	// A flipped bit in the last block of the WAL ends the replay there, with no error,
	// because Pebble looks for a synced offset only in later blocks.
	Entry("OldCorruption, WAL: R1", crash.OldCorruption, uint64(1), crash.R1, 0,
		"durable commit 1 is missing"),
)

// Each entry acks a Durable commit in its workload, so the rules check something. No
// run key of Mixed from 0 to 39 holds with an acked commit.
var _ = DescribeTable("Crash harness, holding",
	func(ctx SpecContext, profile crash.Profile, runKey uint64) {
		Expect(run(ctx, profile, runKey, false)).To(BeNumerically(">", 0))
	},
	Entry("Strict", crash.Strict, uint64(0)),
	Entry("SyncErrors, run key 4", crash.SyncErrors, uint64(4)),
	Entry("FullDisk", crash.FullDisk, uint64(0)),
	Entry("NewCorruption", crash.NewCorruption, uint64(0)),
	Entry("Partial, run key 1", crash.Partial, uint64(1)),
)

// With the directory changes in order, as ext4, XFS, APFS, and NTFS journals keep them,
// a crash keeps the manifest whenever it keeps the marker. Mixed still breaks R4 on its
// read faults, so no run key of it from 0 to 39 holds with an acked commit.
var _ = DescribeTable("Crash harness, ordered entries, holding",
	func(ctx SpecContext, profile crash.Profile, runKey uint64) {
		Expect(run(ctx, profile, runKey, true)).To(BeNumerically(">", 0))
	},
	Entry("Partial", crash.Partial, uint64(0)),
)

// flipped commits n durable records, with no flush, crashes, and flips one bit at the
// offset at of the newest WAL in the crash image, which it returns. A reopened store
// opens once before the commits, so its WAL has synced offsets.
func flipped(ctx context.Context, reopened bool, n uint64, at int64) *fault.FS {
	GinkgoHelper()
	disk := MustSucceed(fault.New(fault.Config{}))
	if reopened {
		Expect(MustSucceed(openCrashEngine(ctx, disk)).Close()).To(Succeed())
	}
	e := MustSucceed(openCrashEngine(ctx, disk))
	for id := range n {
		key := binary.BigEndian.AppendUint64(nil, id+1)
		Expect(e.kv.Set(ctx, key, make([]byte, 500))).To(Succeed())
	}
	image := MustSucceed(disk.Crash(fault.CrashConfig{}))
	Expect(e.Close()).To(Succeed())
	var wal string
	for _, info := range MustSucceed(image.List("kv")) {
		if strings.HasSuffix(info.Name(), ".log") {
			wal = max(wal, info.Name())
		}
	}
	f := MustSucceed(image.Open("kv/"+wal, os.O_RDWR))
	b := make([]byte, 1)
	Expect(f.ReadAt(b, at)).To(Equal(1))
	b[0] ^= 1
	Expect(f.WriteAt(b, at)).To(Equal(1))
	Expect(errors.Join(f.Sync(), f.Close())).To(Succeed())
	return image
}

var _ = Describe("Crash adapter", func() {
	It("Should stop on a panic of Pebble on a failed sync, and keep its cause",
		func() {
			h := &halt{done: make(chan struct{})}
			f := func() (err error) {
				defer stopOnDiskPanic(h, &err)
				panic(errors.Wrap(syscall.EIO, "sync"))
			}
			Expect(f()).To(MatchError(syscall.EIO))
			Expect(h.cause()).To(MatchError(syscall.EIO))
		},
	)

	It("Should panic again on a panic that is not a failed write or sync", func() {
		h := &halt{done: make(chan struct{})}
		f := func() (err error) {
			defer stopOnDiskPanic(h, &err)
			panic(errors.New("pebble: assertion failed"))
		}
		Expect(func() { Expect(f()).To(Succeed()) }).To(Panic())
		Expect(h.cause()).ToNot(HaveOccurred())
	})

	It("Should stop on a Fatalf and keep its error as the cause", func() {
		h := &halt{done: make(chan struct{})}
		cause := errors.Wrap(pebble.ErrCorruption, "block")
		crashLogger{halt: h}.Fatalf("MANIFEST sync failed: %v", cause)
		Expect(errors.Is(h.cause(), pebble.ErrCorruption)).To(BeTrue())
		Expect(h.cause()).To(MatchError(
			"pebble: stopped: MANIFEST sync failed: block: pebble: corruption",
		))
		Expect(h.done).To(BeClosed())
	})

	DescribeTable(
		"Should lose each commit from a flipped bit in the WAL on, with no error",
		func(ctx SpecContext, reopened bool, n uint64, at int64, kept int) {
			e := MustSucceed(openCrashEngine(ctx, flipped(ctx, reopened, n, at)))
			Expect(MustSucceed(e.Recovered(ctx))).To(HaveLen(kept))
			Expect(e.Close()).To(Succeed())
		},
		Entry("in the first WAL of a new store", false, uint64(10), int64(2500), 4),
		Entry("in the first block of the first WAL of a new store",
			false, uint64(100), int64(2500), 4),
		Entry("in the last block of a WAL with synced offsets",
			true, uint64(100), int64(40000), 73),
	)

	It("Should return each payload whole when the values come from sstables",
		func(ctx SpecContext) {
			disk := MustSucceed(fault.New(fault.Config{}))
			e := MustSucceed(openCrashEngine(ctx, disk))
			payload := func(id uint64) []byte {
				p := make([]byte, 300)
				for i := range p {
					p[i] = byte(id + uint64(i))
				}
				return p
			}
			for id := uint64(1); id <= 200; id++ {
				record := crash.Record{ID: id, Payload: payload(id)}
				Expect(e.Commit(ctx, record, crash.Durable)).To(Equal(crash.Durable))
			}
			Expect(e.pebble.Flush()).To(Succeed())
			records := MustSucceed(e.Recovered(ctx))
			Expect(e.Close()).To(Succeed())
			Expect(records).To(HaveLen(200))
			for _, r := range records {
				Expect(r.Payload).To(Equal(payload(r.ID)), "record %d", r.ID)
			}
		},
	)

	It("Should fail the open on a flipped bit in an earlier block of the WAL",
		func(ctx SpecContext) {
			_, err := openCrashEngine(ctx, flipped(ctx, true, 100, 2500))
			Expect(errors.Is(err, pebble.ErrCorruption)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("pebble/record: invalid chunk")))
		},
	)

	It("Should stop the open on a failed sync of the directory", func(ctx SpecContext) {
		disk := MustSucceed(fault.New(fault.Config{SyncErrorRate: 0.05}))
		Expect(openCrashEngine(ctx, disk)).Error().To(MatchError(
			"pebble: stopped: sync /kv: input/output error",
		))
	})

	It("Should stop on a failed sync of the manifest in a flush, and refuse each "+
		"later commit", func(ctx SpecContext) {
		cfg := fault.Config{RunKey: 8, SyncErrorRate: 0.02}
		e := MustSucceed(openCrashEngine(ctx, MustSucceed(fault.New(cfg))))
		for id := uint64(1); id <= 15; id++ {
			record := crash.Record{ID: id, Payload: []byte("taco")}
			Expect(e.Commit(ctx, record, crash.Durable)).To(Equal(crash.Durable))
		}
		record := crash.Record{ID: 16, Payload: []byte("cat")}
		_, err := e.Commit(ctx, record, crash.Durable)
		Expect(err).To(MatchError(syscall.EIO))
		Expect(err).To(MatchError("pebble: stopped: MANIFEST sync failed: " +
			"sync /kv/MANIFEST-000001: input/output error"))
		Expect(e.Commit(ctx, record, crash.Visible)).Error().To(MatchError(err))
		Expect(e.Recovered(ctx)).Error().To(MatchError(err))
		Expect(e.Close()).To(Succeed())
	})
})
