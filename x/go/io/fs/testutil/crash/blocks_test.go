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
	"bytes"
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
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
)

const blockSize = 4096

// blockLog is a log with one record in each 4 KiB block of one file: a checksum, a
// length, an ID, and the payload. It reads a bad block a second time, and a block that
// stays bad ends the log as a torn tail. An error of the disk stops the log.
type blockLog struct {
	f       xfs.File
	dead    error
	records []crash.Record
	// unbound leaves the place of a block out of its checksum, so a block from another
	// place passes.
	unbound bool
}

func (l *blockLog) checksum(block []byte, index int) uint32 {
	sum := crc32.Checksum(block[4:], castagnoli)
	if l.unbound {
		return sum
	}
	return crc32.Update(
		sum,
		castagnoli,
		binary.LittleEndian.AppendUint64(nil, uint64(index)),
	)
}

func openBlockLog(unbound bool) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		l := &blockLog{f: f, unbound: unbound}
		if err = errors.Combine(fs.SyncDir(""), l.scan()); err != nil {
			return nil, errors.Combine(err, f.Close())
		}
		return l, nil
	}
}

func (l *blockLog) scan() error {
	info, err := l.f.Stat()
	if err != nil {
		return err
	}
	block := make([]byte, blockSize)
	for index := 0; int64(index) < info.Size()/blockSize; index++ {
		ok := false
		for try := 0; try < 2 && !ok; try++ {
			if _, err = l.f.ReadAt(block, int64(index)*blockSize); err != nil {
				return err
			}
			ok = binary.LittleEndian.Uint32(block) == l.checksum(block, index)
		}
		if !ok {
			break
		}
		size := binary.LittleEndian.Uint32(block[4:])
		l.records = append(l.records, crash.Record{
			ID:      binary.LittleEndian.Uint64(block[8:]),
			Payload: bytes.Clone(block[16 : 16+size]),
		})
	}
	return errors.Combine(
		l.f.Truncate(int64(len(l.records))*blockSize),
		l.f.Sync(),
	)
}

func (l *blockLog) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	if l.dead != nil {
		return 0, l.dead
	}
	block := make([]byte, blockSize)
	binary.LittleEndian.PutUint32(block[4:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint64(block[8:], r.ID)
	copy(block[16:], r.Payload)
	binary.LittleEndian.PutUint32(block, l.checksum(block, len(l.records)))
	if _, l.dead = l.f.WriteAt(block, int64(len(l.records))*blockSize); l.dead != nil {
		return 0, l.dead
	}
	if want == crash.Durable {
		if l.dead = l.f.Sync(); l.dead != nil {
			return 0, l.dead
		}
	}
	l.records = append(l.records, r)
	return want, nil
}

func (l *blockLog) Recovered(context.Context) ([]crash.Record, error) {
	return l.records, nil
}

func (l *blockLog) Close() error { return l.f.Close() }

// loose writes one block for each commit and syncs none. An engine on a crash image
// notes how much of the block of the first commit the image holds.
type loose struct {
	f       xfs.File
	records []crash.Record
	*looseBlocks
}

// looseBlocks is the state that the loose engines of one run share.
type looseBlocks struct {
	// kept holds each count of bytes that a crash image kept of the first block.
	kept    set.Set[int]
	written bool
}

func (b *looseBlocks) open(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	f, err := fs.Open("blocks", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	if _, image := fs.(*fault.FS); !image {
		return &loose{f: f, looseBlocks: b}, fs.SyncDir("")
	}
	block := make([]byte, blockSize)
	if _, err = f.ReadAt(block, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.Combine(err, f.Close())
	}
	if b.written {
		b.kept.Add(bytes.Count(block, []byte{1}))
	}
	return &loose{f: f, looseBlocks: b}, nil
}

func (l *loose) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	block := bytes.Repeat([]byte{1}, blockSize)
	if _, err := l.f.WriteAt(block, int64(len(l.records))*blockSize); err != nil {
		return 0, err
	}
	l.records, l.written = append(l.records, r), true
	return crash.Visible, nil
}

func (l *loose) Recovered(context.Context) ([]crash.Record, error) {
	return l.records, nil
}

func (l *loose) Close() error { return l.f.Close() }

var _ = Describe("Faults of a profile", func() {
	// broken returns the rules that the engine of open breaks over 32 run keys.
	broken := func(
		ctx context.Context,
		open func(context.Context, xfs.FS) (crash.Engine, error),
		profile crash.Profile,
	) []crash.Rule {
		GinkgoHelper()
		rules := set.Set[crash.Rule]{}
		for runKey := range uint64(32) {
			err := violation(ctx, crash.Config{
				RunKey:  runKey,
				Open:    open,
				Commits: 12,
				Stride:  1,
				Profile: profile,
			})
			if err != nil {
				rules.Add(broke(err))
			}
		}
		return rules.Slice()
	}

	DescribeTable("Should give a read the block of another place",
		func(ctx SpecContext, profile crash.Profile) {
			Expect(broken(ctx, openBlockLog(false), profile)).To(BeEmpty())
			// A block from another place shows as a commit out of order or as a hole.
			unbound := broken(ctx, openBlockLog(true), profile)
			Expect(unbound).ToNot(BeEmpty())
			Expect(unbound).To(HaveEach(BeElementOf(crash.R2, crash.R3)))
		},
		Entry("under OneTimeReadFaults", crash.OneTimeReadFaults),
		Entry("under Mixed", crash.Mixed),
	)

	It("Should draw the blocks that survive again for each crash image",
		func(ctx SpecContext) {
			b := &looseBlocks{kept: set.Set[int]{}}
			Expect(violation(ctx, crash.Config{
				Open:    b.open,
				Commits: 20,
				Stride:  1,
				Profile: crash.Partial,
			})).To(Succeed())
			Expect(len(b.kept)).To(BeNumerically(">", 2))
		},
	)
})
