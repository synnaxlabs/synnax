// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashcheck_test

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

const headerSize = 16

var (
	errCorrupt = errors.New("log: corrupt")
	castagnoli = crc32.MakeTable(crc32.Castagnoli)
)

// opts holds the behaviors of a log. The zero value is a correct log.
type opts struct {
	// lazySync acks Durable before the sync, and syncs at the start of the next call.
	lazySync bool
	// splitRead reads the header and the body of a record in two calls.
	splitRead bool
	// tries is the count of reads of a record before the log gives up. Zero is two.
	tries int
	// deadReads makes Recovered fail after an error stops the log.
	deadReads bool
	// noAck returns a zero ack and no error, and never syncs.
	noAck bool
	// refuse refuses each crash image that holds a record.
	refuse bool
	// cutBack cuts a torn tail at the start of the last good record, and keeps that
	// record in memory.
	cutBack bool
}

// log is an append-only log in one file. Each record is a header of a checksum, a
// length, and an ID, then the payload. A bad record ends the log as a torn tail.
type log struct {
	fs      xfs.FS
	f       xfs.File
	dead    error
	records []crash.Record
	o       opts
	off     int64
	pending bool
}

func openLog(o opts) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		l := &log{fs: fs, o: o}
		existed, err := fs.Exists("log")
		if err != nil {
			return nil, err
		}
		if l.f, err = fs.Open("log", os.O_CREATE|os.O_RDWR); err != nil {
			return nil, err
		}
		if !existed {
			if err := fs.SyncDir(""); err != nil {
				return nil, errors.Combine(err, l.f.Close())
			}
			return l, nil
		}
		if err := l.scan(); err != nil {
			return nil, errors.Combine(err, l.f.Close())
		}
		if o.refuse && len(l.records) > 0 {
			return nil, errors.Combine(errCorrupt, l.f.Close())
		}
		return l, nil
	}
}

func encode(r crash.Record) []byte {
	buf := make([]byte, headerSize+len(r.Payload))
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint64(buf[8:], r.ID)
	copy(buf[headerSize:], r.Payload)
	binary.LittleEndian.PutUint32(buf, crc32.Checksum(buf[4:], castagnoli))
	return buf
}

// readOnce reads the record at off in one call, or in two with splitRead.
func (l *log) readOnce(off, size int64) (crash.Record, int64, bool) {
	var buf []byte
	if l.o.splitRead {
		hdr := make([]byte, headerSize)
		if n, _ := l.f.ReadAt(hdr, off); n < headerSize {
			return crash.Record{}, 0, false
		}
		length := int64(binary.LittleEndian.Uint32(hdr[4:]))
		if off+headerSize+length > size {
			return crash.Record{}, 0, false
		}
		body := make([]byte, length)
		if n, _ := l.f.ReadAt(body, off+headerSize); int64(n) < length {
			return crash.Record{}, 0, false
		}
		buf = append(hdr, body...)
	} else {
		buf = make([]byte, size-off)
		n, err := l.f.ReadAt(buf, off)
		if err != nil && !errors.Is(err, io.EOF) || n < headerSize {
			return crash.Record{}, 0, false
		}
		length := int64(binary.LittleEndian.Uint32(buf[4:]))
		if headerSize+length > int64(n) {
			return crash.Record{}, 0, false
		}
		buf = buf[:headerSize+length]
	}
	if binary.LittleEndian.Uint32(buf) != crc32.Checksum(buf[4:], castagnoli) {
		return crash.Record{}, 0, false
	}
	r := crash.Record{
		ID:      binary.LittleEndian.Uint64(buf[8:]),
		Payload: buf[headerSize:],
	}
	return r, int64(len(buf)), true
}

func (l *log) read(off, size int64) (crash.Record, int64, bool) {
	tries := l.o.tries
	if tries == 0 {
		tries = 2
	}
	for range tries {
		if r, n, ok := l.readOnce(off, size); ok {
			return r, n, true
		}
	}
	return crash.Record{}, 0, false
}

func (l *log) scan() error {
	info, err := l.f.Stat()
	if err != nil {
		return err
	}
	size, prev := info.Size(), int64(0)
	for l.off < size {
		r, n, ok := l.read(l.off, size)
		if !ok {
			cut := l.off
			if l.o.cutBack {
				cut = prev
			}
			if err := l.f.Truncate(cut); err != nil {
				return err
			}
			return l.f.Sync()
		}
		l.records = append(l.records, r)
		prev = l.off
		l.off += n
	}
	return nil
}

func (l *log) Commit(
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
	buf := encode(r)
	if _, l.dead = l.f.WriteAt(buf, l.off); l.dead != nil {
		return 0, l.dead
	}
	l.off += int64(len(buf))
	if l.o.noAck {
		l.records = append(l.records, r)
		return 0, nil
	}
	if want == crash.Durable {
		if l.o.lazySync {
			l.pending = true
		} else if l.dead = l.f.Sync(); l.dead != nil {
			return 0, l.dead
		}
	}
	l.records = append(l.records, r)
	return want, nil
}

func (l *log) Recovered(context.Context) ([]crash.Record, error) {
	if l.o.deadReads && l.dead != nil {
		return nil, l.dead
	}
	return append([]crash.Record(nil), l.records...), nil
}

func (l *log) Close() error {
	var err error
	if l.pending && l.dead == nil {
		err = l.f.Sync()
	}
	return errors.Combine(err, l.f.Close())
}
