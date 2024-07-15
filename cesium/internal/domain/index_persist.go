// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"os"
	"sync"
)

const indexFile = "index" + extension

type indexPersist struct {
	Config
	p   *pointerPersist
	idx *index
}

func openIndexPersist(idx *index, fs fs.FS) (*indexPersist, error) {
	p, err := openPointerPersist(fs)

	ip := &indexPersist{p: p, idx: idx}
	return ip, err
}

func (ip *indexPersist) load() ([]pointer, error) {
	return ip.p.load()
}

func (ip *indexPersist) prepare(start int) func() error {
	pointerEncoded := ip.p.encode(start, ip.idx.mu.pointers)
	lenOfPointers := len(ip.idx.mu.pointers)

	return func() error {
		ip.p.Lock()
		defer ip.p.Unlock()

		err := ip.p.Truncate(int64(lenOfPointers) * pointerByteSize)
		if err != nil {
			return err
		}
		_, err = ip.p.WriteAt(pointerEncoded, int64(start*pointerByteSize))
		return err
	}
}

type pointerPersist struct {
	fs.File
	sync.Mutex
	pointerEncoder
}

func openPointerPersist(fs fs.FS) (*pointerPersist, error) {
	f, err := fs.Open(indexFile, os.O_CREATE|os.O_RDWR)
	return &pointerPersist{File: f}, err
}

func (p *pointerPersist) load() ([]pointer, error) {
	info, err := p.Stat()
	size := info.Size()
	if err != nil {
		return nil, err
	}

	b := make([]byte, size)
	if len(b) != 0 {
		if _, err = p.ReadAt(b, 0); err != nil {
			return nil, err
		}
	}

	return p.decode(b), nil
}

var byteOrder = binary.LittleEndian

type pointerEncoder struct{}

func (f *pointerEncoder) encode(start int, ptrs []pointer) []byte {
	b := make([]byte, (len(ptrs)-start)*pointerByteSize)
	for i := start; i < len(ptrs); i++ {
		ptr := ptrs[i]
		base := (i - start) * pointerByteSize
		byteOrder.PutUint64(b[base:base+8], uint64(ptr.Start))
		byteOrder.PutUint64(b[base+8:base+16], uint64(ptr.End))
		byteOrder.PutUint16(b[base+16:base+18], ptr.fileKey)
		byteOrder.PutUint32(b[base+18:base+22], ptr.offset)
		byteOrder.PutUint32(b[base+22:base+26], ptr.length)
	}

	return b
}

func (f *pointerEncoder) decode(b []byte) []pointer {
	if len(b) == 0 {
		return []pointer{}
	}

	pointers := make([]pointer, len(b)/pointerByteSize)
	for i := 0; i < len(b)/pointerByteSize; i++ {
		base := i * pointerByteSize
		pointers[i] = pointer{
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(byteOrder.Uint64(b[base : base+8])),
				End:   telem.TimeStamp(byteOrder.Uint64(b[base+8 : base+16])),
			},
			fileKey: byteOrder.Uint16(b[base+16 : base+18]),
			offset:  byteOrder.Uint32(b[base+18 : base+22]),
			length:  byteOrder.Uint32(b[base+22 : base+26]),
		}
	}
	return pointers
}
