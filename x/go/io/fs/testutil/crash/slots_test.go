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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

const (
	slotBytes = blockSize / 2
	slotCount = 32
)

// slotLog is a log in a file of zeros, which it writes and syncs before use. Each
// record goes in place into a slot of 2 KiB: a checksum, a length, an ID, and the
// payload. After each commit it writes the first slot again with the same bytes. A bad
// slot with a good slot after it is corruption, and with none it is a torn tail.
type slotLog struct {
	fs      xfs.FS
	f       xfs.File
	records []crash.Record
	o       slotOpts
}

// slotOpts holds the switches of a slotLog. The zero value is a correct log.
type slotOpts struct {
	// lenient takes each bad slot as a torn tail.
	lenient bool
	// tolerant returns a bad slot with a good slot after it as unreadable.
	tolerant bool
	// harm is what each recovery does to the disk after it reads the log.
	harm harm
}

// harm is a defect of a recovery that changes the disk after it reads the log.
type harm int

const (
	// rewritten copies the log to log.tmp with no Sync, renames the copy over the log,
	// and syncs the directory.
	rewritten harm = iota + 1
	// marked leaves a durable mark, and refuses each open that finds one.
	marked
	// spoiled flips a bit of the first slot for good, when a later slot holds a record.
	spoiled
)

func openSlotLog(o slotOpts) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		exists, err := fs.Exists("log")
		if err != nil {
			return nil, err
		}
		marked, err := fs.Exists("mark")
		if err != nil {
			return nil, err
		}
		if marked {
			return nil, errCorrupt
		}
		f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		l := &slotLog{fs: fs, f: f, o: o}
		if exists {
			err = l.scan()
		} else {
			err = errors.Join(put(f, make([]byte, slotBytes*slotCount)), fs.SyncDir(""))
		}
		if err == nil && exists {
			err = l.damage()
		}
		if err != nil {
			return nil, errors.Combine(err, l.f.Close())
		}
		return l, nil
	}
}

// damage does the harm of the log to the disk.
func (l *slotLog) damage() error {
	switch l.o.harm {
	case rewritten:
		data, err := xfs.ReadFile(l.fs, "log")
		if err != nil {
			return err
		}
		tmp, err := l.fs.Open("log.tmp", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return err
		}
		_, err = tmp.Write(data)
		err = errors.Join(err, tmp.Close(), l.f.Close(), l.fs.Rename("log.tmp", "log"))
		if err = errors.Join(err, l.fs.SyncDir("")); err != nil {
			return err
		}
		l.f, err = l.fs.Open("log", os.O_RDWR)
		return err
	case marked:
		mark, err := l.fs.Open("mark", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return err
		}
		return errors.Join(mark.Sync(), mark.Close(), l.fs.SyncDir(""))
	case spoiled:
		if len(l.records) < 2 {
			return nil
		}
		b := make([]byte, 1)
		if _, err := l.f.ReadAt(b, 8); err != nil {
			return err
		}
		b[0] ^= 1
		_, err := l.f.WriteAt(b, 8)
		return errors.Join(err, l.f.Sync())
	}
	return nil
}

// slot returns the record in the slot index. ok is false for a slot that is bad on two
// reads, or past the end of the log.
func (l *slotLog) slot(index int) (r crash.Record, ok bool, err error) {
	for range 2 {
		buf := make([]byte, slotBytes)
		_, err = l.f.ReadAt(buf, int64(index)*slotBytes)
		if errors.Is(err, io.EOF) {
			return r, false, nil
		}
		if err != nil {
			return r, false, err
		}
		place := binary.LittleEndian.AppendUint64(nil, uint64(index))
		sum := crc32.Update(crc32.Checksum(buf[4:], castagnoli), castagnoli, place)
		size := binary.LittleEndian.Uint32(buf[4:])
		if binary.LittleEndian.Uint32(buf) == sum && size > 0 {
			id := binary.LittleEndian.Uint64(buf[8:])
			return crash.Record{ID: id, Payload: buf[16 : 16+size]}, true, nil
		}
	}
	return r, false, nil
}

func (l *slotLog) scan() error {
	var (
		slots = make([]crash.Record, slotCount)
		good  = make([]bool, slotCount)
		end   = 0
	)
	for index := range slotCount {
		r, ok, err := l.slot(index)
		if err != nil {
			return err
		}
		if slots[index], good[index] = r, ok; ok {
			end = index + 1
		}
	}
	// Each bad slot before end has a good slot after it.
	for index := range end {
		switch {
		case good[index]:
			l.records = append(l.records, slots[index])
		case l.o.tolerant:
			unreadable := crash.Record{ID: uint64(index + 1), Unreadable: true}
			l.records = append(l.records, unreadable)
		case l.o.lenient:
			return nil
		default:
			return errCorrupt
		}
	}
	return nil
}

func (l *slotLog) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	index := len(l.records)
	buf := make([]byte, slotBytes)
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint64(buf[8:], r.ID)
	copy(buf[16:], r.Payload)
	place := binary.LittleEndian.AppendUint64(nil, uint64(index))
	sum := crc32.Update(crc32.Checksum(buf[4:], castagnoli), castagnoli, place)
	binary.LittleEndian.PutUint32(buf, sum)
	if _, err := l.f.WriteAt(buf, int64(index)*slotBytes); err != nil {
		return 0, err
	}
	if err := l.f.Sync(); err != nil {
		return 0, err
	}
	l.records = append(l.records, r)
	first := make([]byte, slotBytes)
	if _, err := l.f.ReadAt(first, 0); err != nil {
		return 0, err
	}
	if _, err := l.f.WriteAt(first, 0); err != nil {
		return 0, err
	}
	return crash.Durable, l.f.Sync()
}

func (l *slotLog) Recovered(context.Context) ([]crash.Record, error) {
	return l.records, nil
}

func (l *slotLog) Close() error { return l.f.Close() }

var _ = Describe("A log that writes in place in a file of zeros", func() {
	// broken returns the rules that the log breaks over 32 run keys.
	broken := func(ctx context.Context, lenient bool, profile crash.Profile) []crash.Rule {
		GinkgoHelper()
		var rules []crash.Rule
		for runKey := range uint64(32) {
			err := crash.Run(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openSlotLog(slotOpts{lenient: lenient}),
				Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
				Commits: 12,
				Stride:  1,
				Profile: profile,
			})
			if err != nil {
				rules = append(rules, broke(err))
			}
		}
		return rules
	}

	DescribeTable("Should pass",
		func(ctx SpecContext, profile crash.Profile) {
			Expect(broken(ctx, false, profile)).To(BeEmpty())
		},
		Entry("under Strict", crash.Strict),
		Entry("under OldCorruption", crash.OldCorruption),
		Entry("under NewCorruption", crash.NewCorruption),
	)

	It("Should catch a log that takes old corruption as a torn tail",
		func(ctx SpecContext) {
			Expect(broken(ctx, true, crash.NewCorruption)).To(BeEmpty())
			rules := broken(ctx, true, crash.OldCorruption)
			Expect(rules).ToNot(BeEmpty())
			Expect(rules).To(HaveEach(crash.R1))
		},
	)
})

var _ = Describe("A recovery that harms the disk", func() {
	// violations returns the error of each of 32 run keys on which Run fails.
	violations := func(ctx context.Context, o slotOpts, profile crash.Profile) []string {
		GinkgoHelper()
		var out []string
		for runKey := range uint64(32) {
			err := crash.Run(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openSlotLog(o),
				Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
				Commits: 12,
				Stride:  1,
				Profile: profile,
			})
			if err != nil {
				out = append(out, err.Error())
			}
		}
		return out
	}

	It(
		"Should pass a log that returns a bad slot as unreadable",
		func(ctx SpecContext) {
			Expect(violations(ctx, slotOpts{tolerant: true}, crash.OldCorruption)).
				To(BeEmpty())
		},
	)

	It("Should catch a recovery that renames a copy with no Sync over the log",
		func(ctx SpecContext) {
			found := violations(ctx, slotOpts{harm: rewritten}, crash.Strict)
			Expect(found).To(HaveLen(32))
			Expect(found).To(HaveEach(ContainSubstring("R1 broken")))
		},
	)

	DescribeTable(
		"Should catch a second open that refuses after a first that recovered",
		func(ctx SpecContext, profile crash.Profile) {
			found := violations(ctx, slotOpts{harm: marked}, profile)
			Expect(found).To(HaveLen(32))
			Expect(found).To(HaveEach(MatchRegexp(
				`R4 broken .*did not recover after a crash in recovery: .*corrupt`,
			)))
		},
		Entry("under Strict", crash.Strict),
		Entry("under OldCorruption", crash.OldCorruption),
	)

	It("Should catch a second recovery that cannot read what the first read",
		func(ctx SpecContext) {
			found := violations(
				ctx,
				slotOpts{tolerant: true, harm: spoiled},
				crash.OldCorruption,
			)
			Expect(found).To(HaveLen(32))
			Expect(found).To(ContainElement(MatchRegexp(
				`R1 broken .*commit 1 of the first recovery is unreadable after a crash`,
			)))
		},
	)
})
