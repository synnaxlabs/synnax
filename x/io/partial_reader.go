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
	"io"
)

// PartialReader wraps an io.ReaderAt the returns a Reader over a subset
// of the underlying data.
func PartialReader(r io.ReaderAt, offset, length int64) io.Reader {
	return &partialReader{internal: r, offset: offset, length: length}
}

// PartialReaderAt wraps an io.ReaderAt the returns a ReaderAt over a subset
// of the underlying data.
func PartialReaderAt(r io.ReaderAt, offset, length int64) io.ReaderAt {
	return &partialReader{internal: r, offset: offset, length: length}
}

type partialReader struct {
	offset   int64
	length   int64
	nRead    int64
	internal io.ReaderAt
}

// Read implements io.Reader.
func (r *partialReader) Read(p []byte) (n int, err error) {
	if r.nRead >= r.length {
		return 0, io.EOF
	}
	n, err = r.internal.ReadAt(p, r.offset+r.nRead)
	if err != nil {
		return
	}
	nextRead := r.nRead + int64(n)
	if nextRead >= r.length {
		err = io.EOF
		n = int(r.length - r.nRead)
	}
	r.nRead = nextRead
	return n, err
}

// ReadAt implements io.ReaderAt.
func (r *partialReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= r.length {
		return 0, io.EOF
	}
	adjusted := r.offset + off
	n, err = r.internal.ReadAt(p, adjusted)
	if err != nil {
		return
	}
	if off+int64(n) > r.length {
		err = io.EOF
		n = int(r.length - off)
	}
	return n, err
}
