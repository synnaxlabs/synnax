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
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

const (
	// slotSize is the space of one record in a segment. It does not divide a 4 KiB
	// block, so records cross block edges.
	slotSize   = 1500
	headerSize = 24
	// slots is the count of records in one segment file.
	slots = 8
)

var errCorrupt = errors.New("reflog: corrupt log")

// bugs holds the known bugs of the canaries. The zero value is the correct engine.
type bugs struct {
	dirSyncSkipped, ackedBeforeSync, retriedSyncTrusted, cutOnAnyError    bool
	checksumSkipped, badRecordSkipped, tornTailRefused, writeErrorIgnored bool
	// retryTrusted takes the second read of a bad slot with no checksum.
	retryTrusted bool
}

// refLog is a small log store: records in slots of segment files, a checksum that binds
// each record to its segment and offset, and in each record the count of records that
// were durable when it was written.
type refLog struct {
	fs      xfs.FS
	file    xfs.File
	dead    error
	records []crash.Record
	bugs    bugs
	// segment and next give the slot of the next record.
	segment, next int
	// synced is the count of records, from the first segment, that are durable.
	synced uint64
}

func segmentName(n int) string { return fmt.Sprintf("%06d.log", n) }

// openRefLog returns an Open function for a refLog with the bugs b.
func openRefLog(b bugs) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		l := &refLog{fs: fs, bugs: b}
		if err := l.recover(); err != nil {
			return nil, err
		}
		return l, nil
	}
}

func (l *refLog) encode(r crash.Record) []byte {
	buf := make([]byte, headerSize+len(r.Payload))
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint64(buf[8:], r.ID)
	binary.LittleEndian.PutUint64(buf[16:], l.synced)
	copy(buf[headerSize:], r.Payload)
	binary.LittleEndian.PutUint32(buf, checksum(l.segment, l.next, buf[4:]))
	return buf
}

func checksum(segment, slot int, body []byte) uint32 {
	place := binary.LittleEndian.AppendUint64(nil, uint64(segment*slots+slot))
	return crc32.Update(crc32.Checksum(place, castagnoli), castagnoli, body)
}

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// read returns the record in the slot of f, and the durable count that it carries. It
// reads a bad slot a second time before it calls the slot bad.
func (l *refLog) read(
	f xfs.File,
	segment, slot int,
) (r crash.Record, synced uint64, ok bool, err error) {
	for try := range 2 {
		buf := make([]byte, slotSize)
		n, err := f.ReadAt(buf, int64(slot)*slotSize)
		if err != nil && !errors.Is(err, io.EOF) {
			return crash.Record{}, 0, false, err
		}
		if n < headerSize {
			continue
		}
		size := int(binary.LittleEndian.Uint32(buf[4:]))
		if headerSize+size > n {
			continue
		}
		body := buf[4 : headerSize+size]
		trusted := l.bugs.checksumSkipped || l.bugs.retryTrusted && try == 1
		if !trusted &&
			binary.LittleEndian.Uint32(buf) != checksum(segment, slot, body) {
			continue
		}
		r = crash.Record{ID: binary.LittleEndian.Uint64(buf[8:]), Payload: body[20:]}
		return r, binary.LittleEndian.Uint64(buf[16:]), true, nil
	}
	return crash.Record{}, 0, false, nil
}

// recover reads each segment. A bad record that a later record proves durable, or that
// is in a sealed segment, is corruption. Any other bad record is a torn tail.
func (l *refLog) recover() error {
	infos, err := l.fs.List("")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		if strings.HasSuffix(info.Name(), ".log") {
			names = append(names, info.Name())
		}
	}
	if len(names) == 0 {
		return l.open(0)
	}
	for i, name := range names {
		if cut, err := l.scan(name, names[i+1:]); err != nil || cut {
			return err
		}
	}
	l.synced = uint64(l.segment*slots + l.next)
	return nil
}

// scan reads the records of the segment name, which the segments later follow. It
// reports if it cut a torn tail, which ends the recovery.
func (l *refLog) scan(name string, later []string) (cut bool, err error) {
	if _, err = fmt.Sscanf(name, "%06d.log", &l.segment); err != nil {
		return false, err
	}
	if l.file, err = l.fs.Open(name, os.O_RDWR); err != nil {
		return false, err
	}
	info, err := l.file.Stat()
	if err != nil {
		return false, errors.Combine(err, l.file.Close())
	}
	count := int((info.Size() + slotSize - 1) / slotSize)
	for l.next = 0; l.next < count; l.next++ {
		r, _, ok, err := l.read(l.file, l.segment, l.next)
		if err != nil {
			return false, errors.Combine(err, l.file.Close())
		}
		if ok {
			l.records = append(l.records, r)
			continue
		}
		if l.bugs.badRecordSkipped {
			continue
		}
		promised, err := l.promised(count)
		if err != nil {
			return false, errors.Combine(err, l.file.Close())
		}
		corrupt := len(later) > 0 || promised || l.bugs.tornTailRefused
		if corrupt && !l.bugs.cutOnAnyError {
			return false, errors.Combine(errCorrupt, l.file.Close())
		}
		return true, l.cut(later)
	}
	if len(later) > 0 {
		return false, l.file.Close()
	}
	return false, nil
}

// promised reports if a record after the slot next says that the slot was durable.
func (l *refLog) promised(count int) (bool, error) {
	for slot := l.next + 1; slot < count; slot++ {
		_, synced, ok, err := l.read(l.file, l.segment, slot)
		if err != nil {
			return false, err
		}
		if ok && synced > uint64(l.segment*slots+l.next) {
			return true, nil
		}
	}
	return false, nil
}

// cut drops the torn tail from the slot next, and each segment after it.
func (l *refLog) cut(later []string) error {
	if err := l.file.Truncate(int64(l.next) * slotSize); err != nil {
		return err
	}
	if err := l.file.Sync(); err != nil {
		return err
	}
	for _, name := range later {
		if err := l.fs.Remove(name); err != nil {
			return err
		}
	}
	l.synced = uint64(l.segment*slots + l.next)
	return l.fs.SyncDir("")
}

// open makes the segment n the place of the next record.
func (l *refLog) open(n int) (err error) {
	l.segment, l.next = n, 0
	if l.file, err = l.fs.Open(segmentName(n), os.O_CREATE|os.O_RDWR); err != nil {
		return err
	}
	if l.bugs.dirSyncSkipped {
		return nil
	}
	return l.fs.SyncDir("")
}

// seal makes the full segment durable and opens the next one.
func (l *refLog) seal() error {
	if err := l.file.Sync(); err != nil {
		return err
	}
	l.synced = uint64(l.segment*slots + l.next)
	if err := l.file.Close(); err != nil {
		return err
	}
	return l.open(l.segment + 1)
}

func (l *refLog) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	if l.dead != nil {
		return 0, l.dead
	}
	if l.next == slots {
		if l.dead = l.seal(); l.dead != nil {
			return 0, l.dead
		}
	}
	_, err := l.file.WriteAt(l.encode(r), int64(l.next)*slotSize)
	if err != nil && !l.bugs.writeErrorIgnored {
		return 0, err
	}
	l.next++
	if want == crash.Visible {
		l.records = append(l.records, r)
		return crash.Visible, nil
	}
	if !l.bugs.ackedBeforeSync {
		err := l.file.Sync()
		if err != nil && l.bugs.retriedSyncTrusted {
			err = l.file.Sync()
		}
		if l.dead = err; err != nil {
			return 0, err
		}
		l.synced = uint64(l.segment*slots + l.next)
	}
	l.records = append(l.records, r)
	return crash.Durable, nil
}

func (l *refLog) Recovered(context.Context) ([]crash.Record, error) {
	return append([]crash.Record(nil), l.records...), nil
}

func (l *refLog) Close() error { return l.file.Close() }
