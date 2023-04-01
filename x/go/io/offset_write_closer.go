// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

type offsetWriteCloser struct {
	WriteSeekCloser
	// offset stores the offset of the write cursor at the start of the write.
	offset int64
	// len stores the number of bytes written by the writer.
	len int64
}

func NewOffsetWriteCloser(f WriteSeekCloser, seek int) (OffsetWriteCloser, error) {
	off, err := f.Seek(0, seek)
	return &offsetWriteCloser{
		WriteSeekCloser: f,
		offset:          off,
		len:             0,
	}, err
}

func (o *offsetWriteCloser) Reset() {
	o.offset = o.offset + o.len
	o.len = 0
}

func (o *offsetWriteCloser) Len() int64 { return o.len }

func (o *offsetWriteCloser) Offset() int64 { return o.offset }

func (o *offsetWriteCloser) Write(p []byte) (n int, err error) {
	n, err = o.WriteSeekCloser.Write(p)
	o.len += int64(n)
	return n, err
}
