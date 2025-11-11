// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import (
	"github.com/synnaxlabs/x/io/fs"
)

type trackedWriteCloser struct {
	fs.File
	// offset stores the offset of the write cursor at the start of the write.
	offset int64
	// len stores the number of bytes written by the writer.
	len int64
}

// NewTrackedWriteCloser opens a new WriteCloser at the end of the file that tracks the
// number of bytes written to the file and the offset at which the write started.
func NewTrackedWriteCloser(f fs.File) (TrackedWriteCloser, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	offset := info.Size()
	return &trackedWriteCloser{File: f, offset: offset}, err
}

// Reset resets the tracked writer to the end of the file, setting Len() to zero and
// offset to the current end of the file.
func (o *trackedWriteCloser) Reset() {
	o.offset = o.offset + o.len
	o.len = 0
}

// Len returns the number of bytes written by the writer since it was opened or Reset()
// was called.
func (o *trackedWriteCloser) Len() int64 { return o.len }

// Offset returns the offset of the write cursor at the start of the current write i.e.
// when the writer was opened or Reset() was called.
func (o *trackedWriteCloser) Offset() int64 { return o.offset }

// Write implements the io.Writer interface.
func (o *trackedWriteCloser) Write(p []byte) (n int, err error) {
	n, err = o.File.Write(p)
	o.len += int64(n)
	return n, err
}
