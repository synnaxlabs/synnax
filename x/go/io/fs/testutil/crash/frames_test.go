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
	. "github.com/synnaxlabs/x/testutil"
)

// frameHeader is the size of the header of a frame: a checksum of the header, the
// length of the body, the ID, and a checksum of the body.
const frameHeader = 20

// frameLog is an append-only log of frames in one file. The header and the body of a
// frame each have a checksum that binds their place in the file. Recovery reads them in
// two calls, and calls a range bad only when two reads of it fail.
type frameLog struct {
	f       xfs.File
	dead    error
	records []crash.Record
	off     int64
}

// frameSum is the checksum of b at the offset off of the file.
func frameSum(off int64, b []byte) uint32 {
	place := binary.LittleEndian.AppendUint64(nil, uint64(off))
	return crc32.Update(crc32.Checksum(place, castagnoli), castagnoli, b)
}

func openFrameLog(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	exists, err := fs.Exists("log")
	if err != nil {
		return nil, err
	}
	f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	l := &frameLog{f: f}
	if exists {
		err = l.scan()
	} else {
		err = fs.SyncDir("")
	}
	if err != nil {
		return nil, errors.Combine(err, f.Close())
	}
	return l, nil
}

// read returns size bytes at off that pass good, from one of two reads.
func (l *frameLog) read(off, size int64, good func([]byte) bool) ([]byte, bool, error) {
	for range 2 {
		b := make([]byte, size)
		n, err := l.f.ReadAt(b, off)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, false, err
		}
		if int64(n) == size && good(b) {
			return b, true, nil
		}
	}
	return nil, false, nil
}

// scan reads the frames of the log and cuts a torn tail.
func (l *frameLog) scan() error {
	info, err := l.f.Stat()
	if err != nil {
		return err
	}
	for l.off < info.Size() {
		at := l.off
		head, ok, err := l.read(at, frameHeader, func(h []byte) bool {
			return binary.LittleEndian.Uint32(h) == frameSum(at, h[4:])
		})
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		size := int64(binary.LittleEndian.Uint32(head[4:]))
		body, ok, err := l.read(at+frameHeader, size, func(b []byte) bool {
			return binary.LittleEndian.Uint32(head[16:]) == frameSum(at+frameHeader, b)
		})
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		id := binary.LittleEndian.Uint64(head[8:])
		l.records = append(l.records, crash.Record{ID: id, Payload: body})
		l.off += frameHeader + size
	}
	if l.off == info.Size() {
		return nil
	}
	return errors.Join(l.f.Truncate(l.off), l.f.Sync())
}

func (l *frameLog) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	if l.dead != nil {
		return 0, l.dead
	}
	frame := make([]byte, frameHeader+len(r.Payload))
	binary.LittleEndian.PutUint32(frame[4:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint64(frame[8:], r.ID)
	copy(frame[frameHeader:], r.Payload)
	sum := frameSum(l.off+frameHeader, r.Payload)
	binary.LittleEndian.PutUint32(frame[16:], sum)
	binary.LittleEndian.PutUint32(frame, frameSum(l.off, frame[4:frameHeader]))
	if _, l.dead = l.f.WriteAt(frame, l.off); l.dead != nil {
		return 0, l.dead
	}
	if want == crash.Durable {
		if l.dead = l.f.Sync(); l.dead != nil {
			return 0, l.dead
		}
	}
	l.off += int64(len(frame))
	l.records = append(l.records, r)
	return want, nil
}

func (l *frameLog) Recovered(context.Context) ([]crash.Record, error) {
	return l.records, nil
}

func (l *frameLog) Close() error { return l.f.Close() }

var _ = DescribeTable(
	"Should pass a log that reads the header and the body of a frame in two calls",
	func(ctx SpecContext, profile crash.Profile, faulty bool) {
		var total crash.Result
		for runKey := range uint64(32) {
			result := MustSucceed(crash.Run(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openFrameLog,
				Commits: 20,
				Stride:  1,
				Profile: profile,
			}))
			total.Recoveries += result.Recoveries
			total.Refusals += result.Refusals
			total.Losses += result.Losses
			total.ReadFaults += result.ReadFaults
		}
		Expect(
			total.Recoveries - total.Refusals - total.Losses,
		).To(BeNumerically(">", 0))
		if faulty {
			Expect(total.ReadFaults).To(BeNumerically(">", 0))
		}
	},
	Entry("under Strict", crash.Strict, false),
	Entry("under Partial", crash.Partial, false),
	Entry("under SyncErrors", crash.SyncErrors, false),
	Entry("under OneTimeReadFaults", crash.OneTimeReadFaults, true),
	Entry("under RepeatedReadFaults", crash.RepeatedReadFaults, true),
)

var _ = It("Should catch a log that reads a frame in two calls and retries both",
	func(ctx SpecContext) {
		var found []string
		for runKey := range uint64(32) {
			err := violation(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openPlainLog(plainOpts{split: true}),
				Commits: 20,
				Stride:  1,
				Profile: crash.RepeatedReadFaults,
			})
			if err != nil {
				found = append(found, err.Error())
			}
		}
		Expect(found).ToNot(BeEmpty())
		Expect(found).To(HaveEach(MatchRegexp(`R1 broken .*, 1 in one cluster`)))
	},
)
