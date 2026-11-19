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
	f       xfs.File
	records []crash.Record
	// lenient takes each bad slot as a torn tail.
	lenient bool
}

func openSlotLog(lenient bool) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		exists, err := fs.Exists("log")
		if err != nil {
			return nil, err
		}
		f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		l := &slotLog{f: f, lenient: lenient}
		if exists {
			err = l.scan()
		} else {
			err = errors.Join(put(f, make([]byte, slotBytes*slotCount)), fs.SyncDir(""))
		}
		if err != nil {
			return nil, errors.Combine(err, f.Close())
		}
		return l, nil
	}
}

// slot returns the record in the slot index. ok is false for a slot that is bad on two
// reads.
func (l *slotLog) slot(index int) (r crash.Record, ok bool, err error) {
	buf := make([]byte, slotBytes)
	for range 2 {
		if _, err = l.f.ReadAt(buf, int64(index)*slotBytes); err != nil {
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
	bad := false
	for index := range slotCount {
		r, ok, err := l.slot(index)
		if err != nil {
			return err
		}
		if ok && bad && !l.lenient {
			return errCorrupt
		}
		if bad = bad || !ok; !bad {
			l.records = append(l.records, r)
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
				Open:    openSlotLog(lenient),
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
