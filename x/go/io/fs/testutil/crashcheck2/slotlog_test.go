// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashcheck2_test

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

const (
	slotSize  = 1500
	frameHead = 16
	block     = 4096
)

var (
	castagnoli = crc32.MakeTable(crc32.Castagnoli)
	errCorrupt = errors.New("slotlog: corrupt")
)

// slotOpts holds the switches of a slotLog.
type slotOpts struct {
	// blockReads reads the log in aligned 4 KiB blocks, one ReadAt for each block, as
	// an engine with direct I/O does. A second read of a bad frame reads its blocks
	// again, one call each.
	blockReads bool
	// rewrite makes recovery write the good frames to log.tmp, with no Sync, rename it
	// over log, and sync the directory.
	rewrite bool
	// counted keeps the count of durable frames in a second file. Recovery refuses
	// when the log holds fewer frames than the count.
	counted bool
	// repairs makes recovery, on a bad frame under the count, truncate the log there
	// and sync it. It still returns each frame that it read before the cut.
	repairs bool
}

// slotLog is a log with one frame in each slot of 1,500 bytes. A frame is a checksum, a
// length, an ID, and the payload. The ID binds a frame to its slot.
type slotLog struct {
	fs      xfs.FS
	f       xfs.File
	count   xfs.File
	records []crash.Record
	o       slotOpts
}

func openSlotLog(o slotOpts) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		l := &slotLog{fs: fs, o: o}
		existed, err := fs.Exists("log")
		if err != nil {
			return nil, err
		}
		if l.f, err = fs.Open("log", os.O_CREATE|os.O_RDWR); err != nil {
			return nil, err
		}
		if o.counted {
			if l.count, err = fs.Open("count", os.O_CREATE|os.O_RDWR); err != nil {
				return nil, errors.Combine(err, l.f.Close())
			}
		}
		if !existed {
			err = fs.SyncDir("")
		} else {
			err = l.recover()
		}
		if err != nil {
			return nil, errors.Combine(err, l.Close())
		}
		return l, nil
	}
}

func encode(r crash.Record) []byte {
	buf := make([]byte, frameHead+len(r.Payload))
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint64(buf[8:], r.ID)
	copy(buf[frameHead:], r.Payload)
	binary.LittleEndian.PutUint32(buf, crc32.Checksum(buf[4:], castagnoli))
	return buf
}

// decode returns the frame of slot i from b, the bytes of the slot.
func decode(b []byte, i int) (crash.Record, bool) {
	if len(b) < frameHead {
		return crash.Record{}, false
	}
	n := frameHead + int(binary.LittleEndian.Uint32(b[4:]))
	if n > len(b) || binary.LittleEndian.Uint32(b) != crc32.Checksum(
		b[4:n],
		castagnoli,
	) {
		return crash.Record{}, false
	}
	id := binary.LittleEndian.Uint64(b[8:])
	if id != uint64(i+1) {
		return crash.Record{}, false
	}
	return crash.Record{ID: id, Payload: append([]byte(nil), b[frameHead:n]...)}, true
}

// readBlocks reads the aligned blocks that hold buf[from:to], one call each.
func (l *slotLog) readBlocks(buf []byte, from, to int64) error {
	size := int64(len(buf))
	for b := from / block * block; b < to; b += block {
		if _, err := l.f.ReadAt(
			buf[b:min(b+block, size)],
			b,
		); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	}
	return nil
}

// slot reads slot i two times at most, and returns its frame.
func (l *slotLog) slot(buf []byte, i int) (crash.Record, bool, error) {
	size := int64(len(buf))
	from, to := int64(i)*slotSize, min(int64(i+1)*slotSize, size)
	var (
		r  crash.Record
		ok bool
	)
	for try := 0; try < 2 && !ok; try++ {
		if l.o.blockReads {
			if try > 0 {
				if err := l.readBlocks(buf, from, to); err != nil {
					return r, false, err
				}
			}
		} else if _, err := l.f.ReadAt(
			buf[from:to],
			from,
		); err != nil && !errors.Is(err, io.EOF) {
			return r, false, err
		}
		r, ok = decode(buf[from:to], i)
	}
	return r, ok, nil
}

func (l *slotLog) confirmed() (int, error) {
	info, err := l.count.Stat()
	if err != nil || info.Size() == 0 {
		return 0, err
	}
	b := make([]byte, 12)
	if _, err = l.count.ReadAt(b, 0); err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	if binary.LittleEndian.Uint32(b[8:]) != crc32.Checksum(b[:8], castagnoli) {
		return 0, errCorrupt
	}
	return int(binary.LittleEndian.Uint64(b)), nil
}

func (l *slotLog) recover() error {
	info, err := l.f.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	buf := make([]byte, size)
	if l.o.blockReads {
		if err = l.readBlocks(buf, 0, size); err != nil {
			return err
		}
	}
	count := 0
	if l.o.counted {
		if count, err = l.confirmed(); err != nil {
			return err
		}
	}
	slots := int((size + slotSize - 1) / slotSize)
	if slots < count {
		return errCorrupt
	}
	type result struct {
		r  crash.Record
		ok bool
	}
	results := make([]result, slots)
	for i := range slots {
		if results[i].r, results[i].ok, err = l.slot(buf, i); err != nil {
			return err
		}
	}
	cut := -1
	for i, res := range results {
		r, ok := res.r, res.ok
		if ok {
			l.records = append(l.records, crash.Record{ID: r.ID, Payload: r.Payload})
			continue
		}
		if i >= count {
			if cut < 0 {
				cut = i
			}
			break
		}
		l.records = append(l.records, crash.Record{ID: uint64(i + 1), Unreadable: true})
		if l.o.repairs && cut < 0 {
			if err = l.f.Truncate(int64(i) * slotSize); err != nil {
				return err
			}
			if err = l.f.Sync(); err != nil {
				return err
			}
			cut = slots
		}
	}
	if cut >= 0 && cut < slots {
		if err = l.f.Truncate(int64(cut) * slotSize); err != nil {
			return err
		}
		if err = l.f.Sync(); err != nil {
			return err
		}
	}
	if l.o.rewrite {
		return l.rewriteLog()
	}
	return nil
}

func (l *slotLog) rewriteLog() error {
	tmp, err := l.fs.Open("log.tmp", os.O_CREATE|os.O_TRUNC|os.O_RDWR)
	if err != nil {
		return err
	}
	for i, r := range l.records {
		if _, err = tmp.WriteAt(encode(r), int64(i)*slotSize); err != nil {
			return errors.Combine(err, tmp.Close())
		}
	}
	if err = errors.Combine(tmp.Close(), l.f.Close()); err != nil {
		return err
	}
	if err = l.fs.Rename("log.tmp", "log"); err != nil {
		return err
	}
	if err = l.fs.SyncDir(""); err != nil {
		return err
	}
	l.f, err = l.fs.Open("log", os.O_RDWR)
	return err
}

func (l *slotLog) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	i := len(l.records)
	if _, err := l.f.WriteAt(encode(r), int64(i)*slotSize); err != nil {
		return 0, err
	}
	l.records = append(l.records, r)
	if want == crash.Visible {
		return want, nil
	}
	if err := l.f.Sync(); err != nil {
		return 0, err
	}
	if l.o.counted {
		b := make([]byte, 12)
		binary.LittleEndian.PutUint64(b, uint64(len(l.records)))
		binary.LittleEndian.PutUint32(b[8:], crc32.Checksum(b[:8], castagnoli))
		if _, err := l.count.WriteAt(b, 0); err != nil {
			return 0, err
		}
		if err := l.count.Sync(); err != nil {
			return 0, err
		}
	}
	return want, nil
}

func (l *slotLog) Recovered(context.Context) ([]crash.Record, error) {
	return append([]crash.Record(nil), l.records...), nil
}

func (l *slotLog) Close() error {
	err := l.f.Close()
	if l.count != nil {
		err = errors.Combine(err, l.count.Close())
	}
	return err
}
