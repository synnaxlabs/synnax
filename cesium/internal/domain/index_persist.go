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
)

const indexFile = "index" + extension
const tombstoneFile = "tombstone" + extension

type indexPersist struct {
	Config
	p   *pointerPersist
	t   *tombstonePersist
	idx *index
}

func openIndexPersist(idx *index, fs fs.FS) (*indexPersist, error) {
	p, err := openPointerPersist(fs)
	if err != nil {
		return nil, err
	}

	t, err := openTombstonePersist(fs)
	if err != nil {
		return nil, err
	}

	ip := &indexPersist{p: p, t: t, idx: idx}
	return ip, nil
}

func (f *indexPersist) load() ([]pointer, map[uint16]uint32, error) {
	pointers, err := f.p.load()
	if err != nil {
		return nil, nil, err
	}

	tombstones, err := f.t.load()
	if err != nil {
		return nil, nil, err
	}

	return pointers, tombstones, nil
}

type pointerPersist struct {
	fs.File
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

type tombstonePersist struct {
	fs.File
	tombstoneEncoder
}

func openTombstonePersist(fs fs.FS) (*tombstonePersist, error) {
	f, err := fs.Open(tombstoneFile, os.O_CREATE|os.O_RDWR)
	return &tombstonePersist{File: f}, err
}

func (t *tombstonePersist) load() (map[uint16]uint32, error) {
	info, err := t.Stat()
	size := info.Size()
	if err != nil {
		return nil, err
	}

	b := make([]byte, size)
	if len(b) != 0 {
		if _, err = t.ReadAt(b, 0); err != nil {
			return nil, err
		}
	}

	return t.decode(b), nil
}

var byteOrder = binary.LittleEndian

type pointerEncoder struct{}

func (f *pointerEncoder) encode(start int, pointers []pointer) []byte {
	b := make([]byte, (len(pointers)-start)*pointerByteSize)
	for i := start; i < len(pointers); i++ {
		ptr := pointers[i]
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
	pointers := make([]pointer, len(b)/pointerByteSize)
	for i := 0; i < len(pointers); i++ {
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

type tombstoneEncoder struct{}

func (f *tombstoneEncoder) encode(tombstones map[uint16]uint32) []byte {
	var (
		b       = make([]byte, (len(tombstones))*tombstoneByteSize)
		counter = 0
	)
	for fileKey, tombstoneSize := range tombstones {
		base := counter * pointerByteSize
		byteOrder.PutUint16(b[base:base+2], fileKey)
		byteOrder.PutUint32(b[base+2:base+6], tombstoneSize)
	}
	return b
}

func (f *tombstoneEncoder) decode(b []byte) map[uint16]uint32 {
	tombstones := make(map[uint16]uint32)
	for i := 0; i < len(b); i += tombstoneByteSize {
		base := i * tombstoneByteSize
		tombstones[byteOrder.Uint16(b[base:base+2])] = byteOrder.Uint32(b[base+2 : base+6])
	}
	return tombstones
}
