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
	"encoding/binary"
	"hash/crc32"
	"io"
	"math"
	"os"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

// plainOpts holds the switches of a plainLog. The zero value is a correct log for the
// profiles with no lasting corruption.
type plainOpts struct {
	// lazy acks Durable before the sync. The sync comes at the start of the next
	// commit, and at the close.
	lazy bool
	// split reads the header and the body of a record in two calls.
	split bool
	// unacked returns a zero ack and no error.
	unacked bool
	// overcut cuts the last good record from the disk at each recovery, and keeps it in
	// memory.
	overcut bool
	// halted makes Recovered fail after an error stopped the log.
	halted bool
	// wiped cuts the whole log at a record that is bad on two reads.
	wiped bool
	// once reads each record one time.
	once bool
	// writeback makes a Durable commit durable with crash.Writeback, not a Sync.
	writeback bool
	// bulk reads the whole log in one call, two times at most, and keeps the read with
	// the longer good prefix. It checks the checksum of each record that starts before
	// bulk.
	bulk int64
}

// plainLog is an append-only log in one file. Each record is a checksum, a length, an
// ID, and the payload. A bad record ends the log as a torn tail.
type plainLog struct {
	fs      xfs.FS
	f       xfs.File
	dead    error
	records []crash.Record
	off     int64
	o       plainOpts
	pending bool
}

const plainHeader = 16

func openPlainLog(o plainOpts) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		existed, err := fs.Exists("log")
		if err != nil {
			return nil, err
		}
		f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		l := &plainLog{fs: fs, f: f, o: o}
		switch {
		case !existed:
			err = fs.SyncDir("")
		case o.bulk > 0:
			err = l.scanBulk()
		default:
			err = l.scan()
		}
		if err != nil {
			return nil, errors.Combine(err, f.Close())
		}
		return l, nil
	}
}

// readOnce returns the record at off and its length on disk. ok is false for a bad
// record.
func (l *plainLog) readOnce(
	off, size int64,
) (r crash.Record, n int64, ok bool, err error) {
	buf := make([]byte, size-off)
	if l.o.split {
		buf = buf[:min(plainHeader, len(buf))]
	}
	if _, err = l.f.ReadAt(buf, off); err != nil && !errors.Is(err, io.EOF) {
		return r, 0, false, err
	}
	if len(buf) < plainHeader {
		return r, 0, false, nil
	}
	n = plainHeader + int64(binary.LittleEndian.Uint32(buf[4:]))
	if off+n > size {
		return r, 0, false, nil
	}
	if l.o.split {
		body := make([]byte, n-plainHeader)
		if _, err = l.f.ReadAt(
			body,
			off+plainHeader,
		); err != nil &&
			!errors.Is(err, io.EOF) {
			return r, 0, false, err
		}
		buf = slices.Concat(buf, body)
	}
	buf = buf[:n]
	if binary.LittleEndian.Uint32(buf) != crc32.Checksum(buf[4:], castagnoli) {
		return r, 0, false, nil
	}
	r = crash.Record{
		ID:      binary.LittleEndian.Uint64(buf[8:]),
		Payload: buf[plainHeader:],
	}
	return r, n, true, nil
}

// scan reads the log and cuts a torn tail.
func (l *plainLog) scan() error {
	info, err := l.f.Stat()
	if err != nil {
		return err
	}
	last := int64(0)
	for size := info.Size(); l.off < size; {
		var (
			r  crash.Record
			n  int64
			ok bool
		)
		tries := 2
		if l.o.once {
			tries = 1
		}
		for try := 0; try < tries && !ok; try++ {
			if r, n, ok, err = l.readOnce(l.off, size); err != nil {
				return err
			}
		}
		if !ok {
			break
		}
		l.records = append(l.records, r)
		last = l.off
		l.off += n
	}
	cut := l.off
	if l.o.overcut && len(l.records) > 0 {
		cut = last
	}
	if l.o.wiped && cut < info.Size() {
		cut, l.off, l.records = 0, 0, nil
	}
	if cut == info.Size() {
		return nil
	}
	return errors.Join(l.f.Truncate(cut), l.f.Sync())
}

// scanBulk reads the whole log in one call, and cuts a torn tail.
func (l *plainLog) scanBulk() error {
	info, err := l.f.Stat()
	if err != nil {
		return err
	}
	torn, tries := true, 2
	if l.o.once {
		tries = 1
	}
	for try := 0; try < tries && torn; try++ {
		data := make([]byte, info.Size())
		if _, err = l.f.ReadAt(data, 0); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if records, off, cut := l.parse(data); try == 0 || off > l.off {
			l.records, l.off, torn = records, off, cut
		}
	}
	if l.off == info.Size() {
		return nil
	}
	return errors.Join(l.f.Truncate(l.off), l.f.Sync())
}

// parse returns the records of data, the end of the last one, and if bytes after it do
// not hold a whole record.
func (l *plainLog) parse(data []byte) (records []crash.Record, off int64, torn bool) {
	for size := int64(len(data)); off+plainHeader <= size; {
		n := plainHeader + int64(binary.LittleEndian.Uint32(data[off+4:]))
		if off+n > size {
			return records, off, true
		}
		rec := data[off : off+n]
		sum := crc32.Checksum(rec[4:], castagnoli)
		if off < l.o.bulk && binary.LittleEndian.Uint32(rec) != sum {
			return records, off, true
		}
		id := binary.LittleEndian.Uint64(rec[8:])
		records = append(records, crash.Record{ID: id, Payload: rec[plainHeader:]})
		off += n
	}
	return records, off, off < int64(len(data))
}

func (l *plainLog) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	if l.dead != nil {
		return 0, l.dead
	}
	if l.pending {
		l.pending = false
		if l.dead = l.f.Sync(); l.dead != nil {
			return 0, l.dead
		}
	}
	buf := make([]byte, plainHeader+len(r.Payload))
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint64(buf[8:], r.ID)
	copy(buf[plainHeader:], r.Payload)
	binary.LittleEndian.PutUint32(buf, crc32.Checksum(buf[4:], castagnoli))
	if _, l.dead = l.f.WriteAt(buf, l.off); l.dead != nil {
		return 0, l.dead
	}
	l.off += int64(len(buf))
	l.records = append(l.records, r)
	switch {
	case l.o.unacked:
		return 0, nil
	case want == crash.Durable && l.o.lazy:
		l.pending = true
	case want == crash.Durable && l.o.writeback:
		if l.dead = crash.Writeback(l.fs); l.dead != nil {
			return 0, l.dead
		}
	case want == crash.Durable:
		if l.dead = l.f.Sync(); l.dead != nil {
			l.records = l.records[:len(l.records)-1]
			return 0, l.dead
		}
	}
	return want, nil
}

func (l *plainLog) Recovered(context.Context) ([]crash.Record, error) {
	if l.o.halted && l.dead != nil {
		return nil, l.dead
	}
	return append([]crash.Record(nil), l.records...), nil
}

func (l *plainLog) Close() error {
	var err error
	if l.pending && l.dead == nil {
		err = l.f.Sync()
	}
	return errors.Combine(err, l.f.Close())
}

// memory holds each commit in memory and never changes the disk.
type memory struct{ records []crash.Record }

func (m *memory) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	m.records = append(m.records, r)
	return want, nil
}

func (m *memory) Recovered(
	context.Context,
) ([]crash.Record, error) {
	return m.records, nil
}

func (m *memory) Close() error { return nil }

var errBroken = errors.New("engine: broken")

// failing is an engine whose calls fail as its fields say, on a disk with no fault.
type failing struct {
	crash.Engine
	commit, read, close error
}

func (f failing) Commit(
	ctx context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	if f.commit != nil {
		return 0, f.commit
	}
	return f.Engine.Commit(ctx, r, want)
}

func (f failing) Recovered(ctx context.Context) ([]crash.Record, error) {
	if f.read != nil {
		return nil, f.read
	}
	return f.Engine.Recovered(ctx)
}

func (f failing) Close() error { return errors.Combine(f.Engine.Close(), f.close) }

var _ = Describe("Engines", func() {
	// broken returns the rules that the engine of open breaks over 32 run keys.
	broken := func(
		ctx context.Context,
		open func(context.Context, xfs.FS) (crash.Engine, error),
		profile crash.Profile,
	) []crash.Rule {
		GinkgoHelper()
		var rules []crash.Rule
		for runKey := range uint64(32) {
			err := violation(ctx, crash.Config{
				RunKey:  runKey,
				Open:    open,
				Commits: 20,
				Stride:  1,
				Profile: profile,
			})
			if err != nil {
				rules = append(rules, broke(err))
			}
		}
		return rules
	}

	DescribeTable("Should pass a plain log",
		func(ctx SpecContext, o plainOpts, profile crash.Profile) {
			Expect(broken(ctx, openPlainLog(o), profile)).To(BeEmpty())
		},
		Entry("under Strict", plainOpts{}, crash.Strict),
		Entry("under Partial", plainOpts{}, crash.Partial),
		Entry("under SyncErrors", plainOpts{}, crash.SyncErrors),
		Entry("under FullDisk", plainOpts{}, crash.FullDisk),
		Entry("under OneTimeReadFaults", plainOpts{}, crash.OneTimeReadFaults),
		Entry("under Mixed", plainOpts{}, crash.Mixed),
		Entry("that reads a record in two calls, under OneTimeReadFaults",
			plainOpts{split: true}, crash.OneTimeReadFaults),
		Entry("under RepeatedReadFaults", plainOpts{}, crash.RepeatedReadFaults),
		Entry("that reads the whole log in one call, under OneTimeReadFaults",
			plainOpts{bulk: math.MaxInt64}, crash.OneTimeReadFaults),
		Entry("that reads the whole log in one call, under Mixed",
			plainOpts{bulk: math.MaxInt64}, crash.Mixed),
		Entry("that stops its reads after a sync error",
			plainOpts{halted: true}, crash.SyncErrors),
		Entry("that stops its reads on a full disk",
			plainOpts{halted: true}, crash.FullDisk),
	)

	DescribeTable("Should catch on each run key",
		func(
			ctx SpecContext,
			open func(context.Context, xfs.FS) (crash.Engine, error),
			profile crash.Profile,
			rule crash.Rule,
		) {
			rules := broken(ctx, open, profile)
			Expect(rules).To(HaveLen(32))
			Expect(rules).To(HaveEach(rule))
		},
		Entry("a log that syncs a Durable commit only after its ack",
			openPlainLog(plainOpts{lazy: true}), crash.Strict, crash.R1),
		Entry(
			"an engine that acks Durable and never changes the disk",
			func(context.Context, xfs.FS) (crash.Engine, error) { return &memory{}, nil },
			crash.Strict,
			crash.R1,
		),
		Entry("a log that gives no ack and no error",
			openPlainLog(plainOpts{unacked: true}), crash.Strict, crash.R0),
		Entry("an engine that fails each commit with no fault",
			func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
				engine, err := openPlainLog(plainOpts{})(ctx, fs)
				return failing{Engine: engine, commit: errBroken}, err
			},
			crash.Strict, crash.R0),
		Entry("an engine that fails its close on each crash image",
			func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
				engine, err := openPlainLog(plainOpts{})(ctx, fs)
				return failing{Engine: engine, close: errBroken}, err
			},
			crash.Strict, crash.R4),
		Entry("an engine that fails its close before a crash, with no fault",
			func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
				// Only the engine of the workload gets the countedFS of the harness.
				engine, err := openPlainLog(plainOpts{})(ctx, fs)
				if _, image := fs.(*fault.FS); image || err != nil {
					return engine, err
				}
				return failing{Engine: engine, close: errBroken}, nil
			},
			crash.Strict, crash.R0),
		Entry("an engine that fails its read before a crash, with no fault",
			func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
				engine, err := openPlainLog(plainOpts{})(ctx, fs)
				if _, image := fs.(*fault.FS); image || err != nil {
					return engine, err
				}
				return failing{Engine: engine, read: errBroken}, nil
			},
			crash.Strict, crash.R0),
		Entry(
			"an engine that never opens, under Strict",
			func(context.Context, xfs.FS) (crash.Engine, error) { return nil, errBroken },
			crash.Strict,
			crash.R0,
		),
		Entry(
			"an engine that never opens, under SyncErrors",
			func(context.Context, xfs.FS) (crash.Engine, error) { return nil, errBroken },
			crash.SyncErrors,
			crash.R0,
		),
		Entry(
			"an engine that never opens, under FullDisk",
			func(context.Context, xfs.FS) (crash.Engine, error) { return nil, errBroken },
			crash.FullDisk,
			crash.R0,
		),
	)

	It("Should hold every rule after one read fault", func(ctx SpecContext) {
		var found []string
		for runKey := range uint64(32) {
			err := violation(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openPlainLog(plainOpts{once: true}),
				Commits: 20,
				Stride:  1,
				Profile: crash.RepeatedReadFaults,
			})
			if err != nil {
				found = append(found, err.Error())
			}
		}
		Expect(found).To(ContainElement(MatchRegexp(
			`R1 broken .*with 1 read faults, 1 in one cluster`,
		)))
	})

	It("Should hold every rule under one-time faults, whatever one read gets",
		func(ctx SpecContext) {
			var found []string
			for runKey := range uint64(32) {
				err := violation(ctx, crash.Config{
					RunKey:  runKey,
					Open:    openPlainLog(plainOpts{bulk: math.MaxInt64, once: true}),
					Commits: 20,
					Stride:  1,
					Profile: crash.OneTimeReadFaults,
				})
				if err != nil {
					found = append(found, err.Error())
				}
			}
			Expect(found).To(ContainElement(MatchRegexp(
				`R1 broken .*, ([2-9]|\d\d+) in one cluster`,
			)))
		},
	)

	It(
		"Should catch a log that cuts more than read faults free",
		func(ctx SpecContext) {
			var found []string
			for runKey := range uint64(32) {
				err := violation(ctx, crash.Config{
					RunKey:  runKey,
					Open:    openPlainLog(plainOpts{wiped: true}),
					Commits: 20,
					Stride:  1,
					Profile: crash.RepeatedReadFaults,
				})
				if err != nil {
					Expect(broke(err)).To(Equal(crash.R1))
					found = append(found, err.Error())
				}
			}
			Expect(
				found,
			).To(ContainElement(MatchRegexp(`with ([2-9]|\d\d+) read faults`)))
		},
	)

	It("Should catch a log that checks only the first block of a read of the whole log",
		func(ctx SpecContext) {
			rules := broken(
				ctx,
				openPlainLog(plainOpts{bulk: 4096}),
				crash.OneTimeReadFaults,
			)
			Expect(rules).ToNot(BeEmpty())
			Expect(rules).To(HaveEach(crash.R2))
		},
	)

	It(
		"Should catch a recovery that cuts a record with no durable ack that it showed",
		func(ctx SpecContext) {
			found := make([]string, 0, 32)
			for runKey := range uint64(32) {
				err := violation(ctx, crash.Config{
					RunKey:  runKey,
					Open:    openPlainLog(plainOpts{overcut: true}),
					Commits: 20,
					Stride:  1,
					Profile: crash.Partial,
				})
				Expect(err).To(MatchError(ContainSubstring("broken at event")))
				found = append(found, err.Error())
			}
			Expect(found).To(ContainElement(MatchRegexp(
				`R1 broken .*commit \d+ of the first recovery is missing after a crash`,
			)))
		},
	)

	It(
		"Should catch a recovery that cuts a good record from the disk",
		func(ctx SpecContext) {
			rules := broken(ctx, openPlainLog(plainOpts{overcut: true}), crash.Partial)
			Expect(rules).ToNot(BeEmpty())
			Expect(rules).To(HaveEach(Or(Equal(crash.R1), Equal(crash.R3))))
		},
	)

	It(
		"Should name the run, the event, and the rule in a violation",
		func(ctx SpecContext) {
			err := violation(ctx, crash.Config{
				RunKey:  7,
				Open:    openPlainLog(plainOpts{lazy: true}),
				Commits: 20,
				Stride:  1,
				Profile: crash.Strict,
			})
			var v *crash.Violation
			Expect(errors.As(err, &v)).To(BeTrue())
			Expect(v.RunKey).To(BeEquivalentTo(7))
			Expect(v.Profile).To(Equal(crash.Strict))
			Expect(v.Rule).To(Equal(crash.R1))
			Expect(v.Event).To(BeNumerically(">", 0))
			Expect(v.Detail).To(MatchRegexp(`^durable commit \d+ is missing$`))
			Expect(err).To(MatchError(ContainSubstring("R1 broken at event")))
		},
	)
})
