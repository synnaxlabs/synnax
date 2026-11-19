// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashgap2_test

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

var errCorrupt = errors.New("corrupt")

// bulk is a log that syncs each commit before its ack and reads the whole log in one
// call. checked is how many leading bytes of the read it checks for checksums; 0 checks
// every record.
type bulk struct {
	f       xfs.File
	off     int64
	checked int64
}

func openBulk(checked int64) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		if err = fs.SyncDir(""); err != nil {
			return nil, err
		}
		b := &bulk{f: f, checked: checked}
		recs, err := b.Recovered(context.Background())
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			b.off += int64(16 + len(r.Payload))
		}
		return b, nil
	}
}

func (b *bulk) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	buf := make([]byte, 16+len(r.Payload))
	binary.LittleEndian.PutUint64(buf, r.ID)
	binary.LittleEndian.PutUint32(buf[8:], uint32(len(r.Payload)))
	copy(buf[12:], r.Payload)
	binary.LittleEndian.PutUint32(
		buf[12+len(r.Payload):],
		crc32.ChecksumIEEE(buf[:12+len(r.Payload)]),
	)
	n, err := b.f.WriteAt(buf, b.off)
	b.off += int64(n)
	if err != nil {
		return 0, err
	}
	if err = b.f.Sync(); err != nil {
		return 0, err
	}
	return want, nil
}

// parse returns the records of data, whether a checked record had a bad sum, and
// whether the records stopped before the end of data.
func (b *bulk) parse(data []byte) ([]crash.Record, bool, bool) {
	var out []crash.Record
	for off := 0; off+16 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[off+8:]))
		end := off + 16 + size
		if size > 1<<20 || end > len(data) {
			return out, false, true
		}
		sum := binary.LittleEndian.Uint32(data[end-4:])
		if crc32.ChecksumIEEE(data[off:end-4]) != sum &&
			(b.checked == 0 || int64(off) < b.checked) {
			return out, true, false
		}
		out = append(out, crash.Record{
			ID:      binary.LittleEndian.Uint64(data[off:]),
			Payload: append([]byte(nil), data[off+12:end-4]...),
		})
		off = end
	}
	return out, false, false
}

func (b *bulk) Recovered(context.Context) ([]crash.Record, error) {
	info, err := b.f.Stat()
	if err != nil {
		return nil, err
	}
	var (
		recs         []crash.Record
		bad, stopped bool
	)
	for range 3 {
		data := make([]byte, info.Size())
		if len(data) > 0 {
			if _, err = b.f.ReadAt(data, 0); err != nil {
				return nil, err
			}
		}
		if recs, bad, stopped = b.parse(data); !bad && !stopped {
			return recs, nil
		}
	}
	if bad {
		return nil, errCorrupt
	}
	return recs, nil
}

func (b *bulk) Close() error { return b.f.Close() }

func caught(
	ctx context.Context,
	open func(context.Context, xfs.FS) (crash.Engine, error),
) int {
	n := 0
	for key := range uint64(32) {
		if crash.Run(ctx, crash.Config{
			RunKey:  key,
			Open:    open,
			Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
			Commits: 40,
			Stride:  1,
			Profile: crash.OneTimeReadFaults,
		}) != nil {
			n++
		}
	}
	return n
}

var _ = Describe("Read faults on a bulk read", func() {
	It(
		"Should pass a log that checks every record of a bulk read",
		func(ctx SpecContext) {
			Expect(caught(ctx, openBulk(0))).To(Equal(0))
		},
	)

	It(
		"Should catch a log that checks only the first block of a bulk read",
		func(ctx SpecContext) {
			Expect(caught(ctx, openBulk(4096))).To(BeNumerically(">", 0))
		},
	)
})
