// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashgap3_test

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// dataStart is the offset of the first record. The block before it holds a header that
// each commit rewrites, so each Sync changes the header block and the tail block.
const dataStart = 4096

// headed is a log with a header block. dropAll is its bug: on any bad checksum it
// returns no record at all, in place of the records before the bad one.
type headed struct {
	f       xfs.File
	off     int64
	count   uint64
	dropAll bool
	// start is the offset of the first record. With no header it is 0.
	start int64
}

func openHeaded(dropAll bool) func(context.Context, xfs.FS) (crash.Engine, error) {
	return openLog(dropAll, dataStart)
}

func openLog(
	dropAll bool,
	start int64,
) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		if err = fs.SyncDir(""); err != nil {
			return nil, err
		}
		h := &headed{f: f, off: start, dropAll: dropAll, start: start}
		recs, err := h.Recovered(context.Background())
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			h.off += int64(16 + len(r.Payload))
		}
		h.count = uint64(len(recs))
		return h, nil
	}
}

func (h *headed) Commit(
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
	if _, err := h.f.WriteAt(buf, h.off); err != nil {
		return 0, err
	}
	h.off += int64(len(buf))
	h.count++
	if h.start > 0 {
		head := make([]byte, 8)
		binary.LittleEndian.PutUint64(head, h.count)
		if _, err := h.f.WriteAt(head, 0); err != nil {
			return 0, err
		}
	}
	if err := h.f.Sync(); err != nil {
		return 0, err
	}
	return want, nil
}

func (h *headed) Recovered(context.Context) ([]crash.Record, error) {
	info, err := h.f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() <= h.start {
		return nil, nil
	}
	data := make([]byte, info.Size()-h.start)
	if _, err = h.f.ReadAt(data, h.start); err != nil {
		return nil, err
	}
	var out []crash.Record
	for off := 0; off+16 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[off+8:]))
		end := off + 16 + size
		if size > 1<<20 || end > len(data) {
			break
		}
		if crc32.ChecksumIEEE(
			data[off:end-4],
		) != binary.LittleEndian.Uint32(
			data[end-4:],
		) {
			if h.dropAll {
				return nil, nil
			}
			break
		}
		out = append(out, crash.Record{
			ID:      binary.LittleEndian.Uint64(data[off:]),
			Payload: append([]byte(nil), data[off+12:end-4]...),
		})
		off = end
	}
	return out, nil
}

func (h *headed) Close() error { return h.f.Close() }

func caught(ctx context.Context, dropAll bool) int {
	return caughtAt(ctx, dropAll, dataStart)
}

func caughtAt(ctx context.Context, dropAll bool, start int64) int {
	n := 0
	for key := range uint64(32) {
		if _, err := crash.Run(ctx, crash.Config{
			RunKey:  key,
			Open:    openLog(dropAll, start),
			Commits: 40,
			Stride:  1,
			Profile: crash.NewCorruption,
		}); err != nil {
			n++
		}
	}
	return n
}

var _ = Describe("NewCorruption with a header block", func() {
	It(
		"Should pass a log that keeps the records before a stale block",
		func(ctx SpecContext) {
			Expect(caught(ctx, false)).To(Equal(0))
		},
	)

	It(
		"Should catch a log that drops every record at a stale block",
		func(ctx SpecContext) {
			Expect(caught(ctx, true)).To(BeNumerically(">", 0))
		},
	)

	It("Should catch the same bug in a log with no header", func(ctx SpecContext) {
		Expect(caughtAt(ctx, true, 0)).To(BeNumerically(">", 0))
	})
})
