// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import "io"

type ReaderAtCloser interface {
	io.ReaderAt
	io.Closer
}

type WriterAtCloser interface {
	io.WriterAt
	io.Closer
}

type ReaderAtWriterAtCloser interface {
	io.ReaderAt
	WriterAtCloser
}

type TrackedWriteCloser interface {
	OffsetWriter
	io.Closer
}

type OffsetWriter interface {
	// Reset resets the offset of the writer to the current offset.
	Reset()
	// Offset returns the offset of the write cursor.
	Offset() int64
	// Len returns the number of bytes written by the writer.
	Len() int64
	io.Writer
}
