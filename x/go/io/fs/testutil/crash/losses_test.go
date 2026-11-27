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
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	// dropperBlocks is the size of the log of a dropper, in blocks.
	dropperBlocks = 8
	// dropperHeader is the size of the ID, the length, and the checksum that start each
	// record.
	dropperHeader = 16
	// dropperTries is the count of reads of a record before a dropper takes the log to
	// end there.
	dropperTries = 60
)

// dropper keeps its commits in a synced log and acks each as Visible. Its first
// recovery leaves a synced mark. A recovery that finds the mark drops its last commit
// when two of its reads of the same bytes fault, which the rules allow.
type dropper struct {
	fs      xfs.FS
	f       xfs.File
	off     int64
	records []crash.Record
	second  bool
}

func openDropper(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	d := &dropper{fs: fs, f: f}
	info, err := f.Stat()
	if err == nil && info.Size() < dropperBlocks*blockSize {
		_, err = f.WriteAt(make([]byte, dropperBlocks*blockSize), 0)
		err = errors.Join(err, f.Sync(), fs.SyncDir(""))
	}
	if err == nil {
		d.second, err = fs.Exists("mark")
	}
	if err == nil {
		err = d.scan()
	}
	if err != nil {
		return nil, errors.Combine(err, f.Close())
	}
	return d, nil
}

// record returns the record at off and its size in the log. A read can fault, so it
// reads the record up to dropperTries times until its checksum passes there, and
// returns false when none does.
func (d *dropper) record(off int64) (crash.Record, int64, bool, error) {
	header := make([]byte, dropperHeader)
	for range dropperTries {
		if _, err := d.f.ReadAt(header, off); err != nil {
			return crash.Record{}, 0, false, err
		}
		id := binary.LittleEndian.Uint64(header)
		size := int64(binary.LittleEndian.Uint32(header[8:]))
		if id == 0 || off+dropperHeader+size > dropperBlocks*blockSize {
			continue
		}
		payload := make([]byte, size)
		if _, err := d.f.ReadAt(payload, off+dropperHeader); err != nil {
			return crash.Record{}, 0, false, err
		}
		if binary.LittleEndian.Uint32(header[12:]) == dropperSum(off, header, payload) {
			r := crash.Record{ID: id, Payload: payload}
			return r, dropperHeader + size, true, nil
		}
	}
	return crash.Record{}, 0, false, nil
}

// dropperSum returns the checksum of the record at off with header and payload, which
// binds the record to its offset.
func dropperSum(off int64, header, payload []byte) uint32 {
	at := binary.LittleEndian.AppendUint64(nil, uint64(off))
	sum := crc32.Update(crc32.ChecksumIEEE(at), crc32.IEEETable, header[:12])
	return crc32.Update(sum, crc32.IEEETable, payload)
}

// scan reads the records of the log, up to the first offset that holds none.
func (d *dropper) scan() error {
	for {
		r, size, ok, err := d.record(d.off)
		if err != nil || !ok {
			return err
		}
		d.records = append(d.records, r)
		d.off += size
	}
}

func (d *dropper) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	b := make([]byte, dropperHeader+len(r.Payload))
	binary.LittleEndian.PutUint64(b, r.ID)
	binary.LittleEndian.PutUint32(b[8:], uint32(len(r.Payload)))
	binary.LittleEndian.PutUint32(b[12:], dropperSum(d.off, b, r.Payload))
	copy(b[dropperHeader:], r.Payload)
	if _, err := d.f.WriteAt(b, d.off); err != nil {
		return 0, err
	}
	if err := d.f.Sync(); err != nil {
		return 0, err
	}
	d.off += int64(len(b))
	d.records = append(d.records, r)
	return crash.Visible, nil
}

func (d *dropper) Recovered(context.Context) ([]crash.Record, error) {
	disk, ok := d.fs.(*fault.FS)
	if !ok {
		return d.records, nil
	}
	if !d.second {
		mark, err := d.fs.Open("mark", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		err = errors.Join(mark.Sync(), mark.Close(), d.fs.SyncDir(""))
		return d.records, err
	}
	for range 40 {
		if _, err := d.f.ReadAt(make([]byte, 64), 0); err != nil {
			return nil, err
		}
	}
	faults := disk.ReadFaults()
	for i, a := range faults {
		for _, b := range faults[:i] {
			if a.File == b.File && a.Read == b.Read && len(d.records) > 0 {
				return d.records[:len(d.records)-1], nil
			}
		}
	}
	return d.records, nil
}

func (d *dropper) Close() error { return d.f.Close() }

var _ = Describe("Losses", func() {
	It("Should count a commit that a second recovery lost as the rules allow, after "+
		"two faults of one read", func(ctx SpecContext) {
		losses := 0
		for runKey := range uint64(8) {
			losses += MustSucceed(crash.Run(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openDropper,
				Commits: 6,
				Stride:  1,
				Profile: crash.RepeatedReadFaults,
			})).Losses
		}
		Expect(losses).To(BeNumerically(">", 0))
	})
})
