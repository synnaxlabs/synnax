// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"encoding/binary"
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

const indexFile = "index" + extension

type indexPersist struct {
	p   *pointerPersist
	idx *index
	Config
}

func openIndexPersist(idx *index, fs fs.FS) (*indexPersist, error) {
	p, err := openPointerPersist(fs)

	ip := &indexPersist{p: p, idx: idx}
	return ip, err
}

func (ip *indexPersist) load() ([]pointer, error) {
	return ip.p.load()
}

// prepare encodes the pointers from start on, so the returned write can run after the
// pointers change. Prepared writes run in the order they prepare, so the caller must
// hold idx.mu and must run the returned write.
func (ip *indexPersist) prepare(start int) preparedWrite {
	buf, ok := ip.p.buffers.Get().(*[]byte)
	if !ok {
		buf = new([]byte)
	}
	*buf = ip.p.encode(*buf, start, ip.idx.mu.pointers)
	return preparedWrite{
		p:      ip.p,
		buf:    buf,
		start:  start,
		count:  len(ip.idx.mu.pointers),
		ticket: ip.p.tickets.Add(1) - 1,
	}
}

// preparedWrite is one encoded write of the index. It owns its buffer until run.
type preparedWrite struct {
	p      *pointerPersist
	buf    *[]byte
	start  int
	count  int
	ticket uint64
}

func (w preparedWrite) run() error {
	defer w.p.buffers.Put(w.buf)
	w.p.Lock()
	defer w.p.Unlock()
	for w.p.nextTicket != w.ticket {
		w.p.ticketDone.Wait()
	}
	defer func() {
		w.p.nextTicket++
		w.p.ticketDone.Broadcast()
	}()
	if err := w.p.Truncate(int64(w.count) * pointerByteSize); err != nil {
		return err
	}
	_, err := w.p.WriteAt(*w.buf, int64(w.start*pointerByteSize))
	return err
}

func (ip *indexPersist) Close() error {
	return ip.p.Close()
}

type pointerPersist struct {
	pointerCodec
	fs.File
	sync.Mutex
	// buffers holds *[]byte. Overlapping prepares each need their own buffer.
	buffers sync.Pool
	// tickets counts prepared writes. A run waits for nextTicket to reach its ticket,
	// so a stale snapshot cannot truncate a newer one.
	tickets    atomic.Uint64
	ticketDone sync.Cond
	nextTicket uint64
}

func openPointerPersist(fs fs.FS) (*pointerPersist, error) {
	f, err := fs.Open(indexFile, os.O_CREATE|os.O_RDWR)
	p := &pointerPersist{File: f}
	p.ticketDone.L = &p.Mutex
	return p, err
}

func (p *pointerPersist) load() ([]pointer, error) {
	info, err := p.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()

	b := make([]byte, size)
	if len(b) != 0 {
		if _, err = p.ReadAt(b, 0); err != nil {
			return nil, err
		}
	}
	return p.decode(b), nil
}

var byteOrder = binary.LittleEndian

type pointerCodec struct{}

// encode returns the encoded pointers from start on, in buf's memory when it fits.
func (f *pointerCodec) encode(buf []byte, start int, ptrs []pointer) []byte {
	size := (len(ptrs) - start) * pointerByteSize
	b := slices.Grow(buf[:0], size)[:size]
	for i := start; i < len(ptrs); i++ {
		ptr := ptrs[i]
		base := (i - start) * pointerByteSize
		byteOrder.PutUint64(b[base:base+8], uint64(ptr.Start))
		byteOrder.PutUint64(b[base+8:base+16], uint64(ptr.End))
		byteOrder.PutUint16(b[base+16:base+18], ptr.fileKey)
		byteOrder.PutUint32(b[base+18:base+22], ptr.offset)
		byteOrder.PutUint32(b[base+22:base+26], ptr.size)
	}

	return b
}

func (f *pointerCodec) decode(b []byte) []pointer {
	if len(b) == 0 {
		return []pointer{}
	}

	pointers := make([]pointer, len(b)/pointerByteSize)
	for i := range len(b) / pointerByteSize {
		base := i * pointerByteSize
		pointers[i] = pointer{
			Start:   telem.TimeStamp(byteOrder.Uint64(b[base : base+8])),
			End:     telem.TimeStamp(byteOrder.Uint64(b[base+8 : base+16])),
			fileKey: byteOrder.Uint16(b[base+16 : base+18]),
			offset:  byteOrder.Uint32(b[base+18 : base+22]),
			size:    byteOrder.Uint32(b[base+22 : base+26]),
		}
	}
	return pointers
}
