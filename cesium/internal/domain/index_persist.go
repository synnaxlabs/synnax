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
	"context"
	"encoding/binary"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"os"
)

type indexPersist struct {
	Config
	indexEncoder
	fs.File
	idx *index
}

const indexFile = "index"

func openIndexPersist(idx *index, cfg Config) (*indexPersist, error) {
	f, err := cfg.FS.Open(fileName(indexFile), os.O_CREATE|os.O_RDWR)
	ip := &indexPersist{File: f, idx: idx}
	idx.OnChange(ip.onChange)
	return ip, err
}

func (f indexPersist) onChange(ctx context.Context, update indexUpdate) {
	ctx, span := f.T.Bench(ctx, "onChange")
	var encoded []byte
	f.idx.mu.RLock()
	encoded = f.encode(update.afterIndex, f.idx.mu.pointers)
	f.idx.mu.RUnlock()
	var err error
	if len(encoded) != 0 {
		_, err = f.WriteAt(encoded, int64(update.afterIndex*pointerByteSize))
	}

	_ = span.EndWith(err)
	if err != nil {
		f.L.Error("failed to write index update", zap.Error(err))
	}
}

func (f *indexPersist) load() ([]pointer, error) {
	info, err := f.File.Stat()
	size := info.Size()
	if err != nil {
		return nil, err
	}

	b := make([]byte, size)
	if len(b) != 0 {
		if _, err = f.ReadAt(b, 0); err != nil {
			return nil, err
		}
	}
	return f.decode(b), nil
}

type indexEncoder struct{}

var byteOrder = binary.LittleEndian

func (f *indexEncoder) encode(start int, pointers []pointer) []byte {
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

func (f *indexEncoder) decode(b []byte) []pointer {
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
