// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import (
	"errors"
	"github.com/synnaxlabs/x/io/fs"
	"io"
	"os"
)

type offsetWriteCloser struct {
	fs.File
	// offset stores the offset of the write cursor at the start of the write.
	offset int64
	// len stores the number of bytes written by the writer.
	len int64
}

func NewOffsetWriteCloser(f fs.File, seek int) (OffsetWriteCloser, error) {
	var err error
	var offset int64
	switch seek {
	default:
		offset = 0
		err = errors.New("Whence error: not between 0 and 2")
	case io.SeekStart:
		offset = 0
	case io.SeekCurrent:
		offset = 0
	case io.SeekEnd:
		var info os.FileInfo
		info, err = f.Stat()
		offset = info.Size()
	}
	return &offsetWriteCloser{
		File:   f,
		offset: offset,
		len:    0,
	}, err
}

func (o *offsetWriteCloser) Reset() {
	o.offset = o.offset + o.len
	o.len = 0
}

func (o *offsetWriteCloser) Len() int64 { return o.len }

func (o *offsetWriteCloser) Offset() int64 { return o.offset }

func (o *offsetWriteCloser) Write(p []byte) (n int, err error) {
	n, err = o.WriteAt(p, o.offset+o.len)
	o.len += int64(n)
	return n, err
}
